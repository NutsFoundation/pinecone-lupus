package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	servicearch "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/arch"
	lifecyclepb "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/redqueen/protocol/lifecycle"
	metapb "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/redqueen/protocol/meta"
	guidarch "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/arch"
	guidcore "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ServiceClient struct {
	config    Config
	allocator guidarch.GuidAllocator
	factory   *frameFactory
	waiter    *correlationWaiter

	conn          *grpc.ClientConn
	controlClient lifecyclepb.ServiceControlClient
	metaClient    metapb.ServiceMetaClient
	stream        lifecyclepb.ServiceControl_ControlClient
	controlCancel context.CancelFunc

	sessionGUID  string
	instanceGUID string
	ready        bool
	terminated   bool

	mu                   sync.Mutex
	stateHandlers        []servicearch.ServiceClientStateSynchronizedHandler
	manipulationHandlers []servicearch.ServiceClientManipulationHandler
	connectionHandlers   []servicearch.ServiceClientConnectionStateHandler
	heartbeatCancel      context.CancelFunc
	reconnectRequested   bool
	reconnecting         bool
	reconnectReason      string
}

var _ servicearch.ServiceClient = (*ServiceClient)(nil)
var _ servicearch.ServiceLifecyclePort = (*ServiceClient)(nil)
var _ servicearch.ServiceMetaPort = (*ServiceClient)(nil)

func NewServiceClient(config Config, allocator guidarch.GuidAllocator) *ServiceClient {
	config = config.Normalize()
	if allocator == nil {
		allocator = guidcore.NewUUIDV7GuidAllocator()
	}
	return &ServiceClient{
		config:    config,
		allocator: allocator,
		factory:   newFrameFactory(allocator),
		waiter:    newCorrelationWaiter(),
	}
}

func (c *ServiceClient) ClientID() int64 {
	if c == nil {
		return 0
	}
	return c.config.ClientID
}

func (c *ServiceClient) StartService(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	c.terminated = false
	c.mu.Unlock()
	c.notifyConnectionState("CONNECTING", "Startup")
	c.logInfof("[RedQueenServiceClient] [Start] (Address:`%s`, ClientId:`%d`) <Start>", c.config.Address, c.ClientID())
	if err := c.synchronizeControlState(ctx, "Startup"); err != nil {
		c.notifyConnectionState("OFFLINE", err.Error())
		c.logErrorf("[RedQueenServiceClient] [Start] (Address:`%s`, ClientId:`%d`) <Failure> %v", c.config.Address, c.ClientID(), err)
		return err
	}
	c.startHeartbeat()
	c.logInfof("[RedQueenServiceClient] [Start] (Address:`%s`, ClientId:`%d`) <Done>", c.config.Address, c.ClientID())
	return nil
}

func (c *ServiceClient) TerminateService(ctx context.Context) error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	c.terminated = true
	c.reconnectRequested = false
	if c.heartbeatCancel != nil {
		c.heartbeatCancel()
		c.heartbeatCancel = nil
	}
	if c.controlCancel != nil {
		c.controlCancel()
		c.controlCancel = nil
	}
	stream := c.stream
	conn := c.conn
	c.stream = nil
	c.conn = nil
	c.ready = false
	c.sessionGUID = ""
	c.connectionHandlers = nil
	c.mu.Unlock()

	if stream != nil {
		_ = stream.CloseSend()
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}

func (c *ServiceClient) Lifecycle() servicearch.ServiceLifecyclePort {
	return c
}

func (c *ServiceClient) Meta() servicearch.ServiceMetaPort {
	return c
}

func (c *ServiceClient) Register(ctx context.Context, request servicearch.ServiceJoinRequest) (*servicearch.ServiceJoinResponse, error) {
	if c == nil {
		return nil, nil
	}
	if err := c.ensureControlReady(ctx); err != nil {
		return nil, err
	}
	c.mu.Lock()
	sessionGUID := c.sessionGUID
	c.mu.Unlock()
	frame := c.factory.registerFrame(c.ClientID(), sessionGUID, request)
	response, err := c.sendAndAwait(ctx, frame, c.config.CommandTimeout)
	if err != nil {
		return nil, err
	}
	if response == nil || response.FrameType != lifecyclepb.ServiceControlFrameType_REGISTER_ACCEPTED {
		return nil, unexpectedFrameError("REGISTER_ACCEPTED", response)
	}
	result := toJoinResponse(response.GetRegisterAccepted())
	if result != nil {
		c.mu.Lock()
		c.instanceGUID = result.InstanceGUID
		c.mu.Unlock()
	}
	return result, nil
}

func (c *ServiceClient) Deregister(ctx context.Context, instruction servicearch.ServiceDeregisterInstruction) (*servicearch.ServiceDeregisterResult, error) {
	if c == nil {
		return nil, nil
	}
	if !c.isControlReady() {
		return toDeregisterResult(instruction, nil), nil
	}
	c.mu.Lock()
	sessionGUID := c.sessionGUID
	c.mu.Unlock()
	frame := c.factory.deregisterFrame(c.ClientID(), sessionGUID, instruction)
	response, err := c.sendAndAwait(ctx, frame, c.config.CommandTimeout)
	if err != nil {
		return nil, err
	}
	if response == nil || response.FrameType != lifecyclepb.ServiceControlFrameType_DEREGISTERED {
		return nil, unexpectedFrameError("DEREGISTERED", response)
	}
	c.mu.Lock()
	c.instanceGUID = ""
	c.mu.Unlock()
	return toDeregisterResult(instruction, response.GetDeregistered()), nil
}

func (c *ServiceClient) QueryServiceMetaByPath(ctx context.Context, path string) (*servicearch.ServiceMeta, error) {
	if err := c.ensureMetaClient(ctx); err != nil {
		return nil, err
	}
	reply, err := c.metaClient.QueryServiceMetaByPath(ctx, &metapb.QueryServiceMetaByPathRequest{Path: path})
	if err != nil {
		return nil, err
	}
	return toServiceMetaDTO(reply.GetMeta()), nil
}

func (c *ServiceClient) QueryServiceMetaByGUID(ctx context.Context, guid string) (*servicearch.ServiceMeta, error) {
	if err := c.ensureMetaClient(ctx); err != nil {
		return nil, err
	}
	reply, err := c.metaClient.QueryServiceMetaByGuid(ctx, &metapb.QueryServiceMetaByGuidRequest{Guid: guid})
	if err != nil {
		return nil, err
	}
	return toServiceMetaDTO(reply.GetMeta()), nil
}

func (c *ServiceClient) CreateNewService(ctx context.Context, parentAppPath string, meta servicearch.ServiceMeta) (string, error) {
	if err := c.ensureMetaClient(ctx); err != nil {
		return "", err
	}
	reply, err := c.metaClient.CreateNewService(ctx, &metapb.CreateNewServiceRequest{
		ParentAppPath: parentAppPath,
		Meta:          toProtocolMeta(meta),
	})
	if err != nil {
		return "", err
	}
	return reply.GetGuid(), nil
}

func (c *ServiceClient) RegisterStateSynchronizedHandler(handler servicearch.ServiceClientStateSynchronizedHandler) {
	if c == nil || handler == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stateHandlers = append(c.stateHandlers, handler)
}

func (c *ServiceClient) DeregisterStateSynchronizedHandler(handler servicearch.ServiceClientStateSynchronizedHandler) {
	if c == nil || handler == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stateHandlers = nil
}

func (c *ServiceClient) RegisterManipulationHandler(handler servicearch.ServiceClientManipulationHandler) {
	if c == nil || handler == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.manipulationHandlers = append(c.manipulationHandlers, handler)
}

func (c *ServiceClient) DeregisterManipulationHandler(handler servicearch.ServiceClientManipulationHandler) {
	if c == nil || handler == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.manipulationHandlers = nil
}

func (c *ServiceClient) RegisterConnectionStateHandler(handler servicearch.ServiceClientConnectionStateHandler) {
	if c == nil || handler == nil {
		return
	}
	c.mu.Lock()
	c.connectionHandlers = append(c.connectionHandlers, handler)
	c.mu.Unlock()
}

func (c *ServiceClient) DeregisterConnectionStateHandler(handler servicearch.ServiceClientConnectionStateHandler) {
	if c == nil || handler == nil {
		return
	}
	c.mu.Lock()
	c.connectionHandlers = nil
	c.mu.Unlock()
}

func (c *ServiceClient) synchronizeControlState(ctx context.Context, reason string) error {
	syncCtx, cancel := context.WithTimeout(ctx, c.config.ControlSyncTimeout)
	defer cancel()
	c.mu.Lock()
	c.ready = false
	c.mu.Unlock()
	if err := c.ensureControlStream(syncCtx); err != nil {
		return err
	}
	frame := c.factory.clientMusterFrame(c.ClientID(), c.config.ClientName, c.config)
	response, err := c.sendAndAwait(syncCtx, frame, c.config.ControlSyncTimeout)
	if err != nil {
		c.markDisconnected()
		return err
	}
	if response == nil || response.FrameType != lifecyclepb.ServiceControlFrameType_CLIENT_READY {
		c.markDisconnected()
		return unexpectedFrameError("CLIENT_READY", response)
	}
	c.mu.Lock()
	c.sessionGUID = response.GetClientReady().GetSessionGuid()
	c.ready = true
	handlers := append([]servicearch.ServiceClientStateSynchronizedHandler{}, c.stateHandlers...)
	c.mu.Unlock()
	c.logInfof("[RedQueenServiceClient] [ControlReady] (ClientId:`%d`, Session:`%s`, Reason:`%s`) <Done>", c.ClientID(), c.sessionGUID, reason)
	c.notifyConnectionState("ONLINE", reason)
	for _, handler := range handlers {
		handler(reason)
	}
	return nil
}

func (c *ServiceClient) ensureControlReady(ctx context.Context) error {
	if c.isControlReady() {
		return nil
	}
	return c.synchronizeControlState(ctx, "StreamRecovered")
}

func (c *ServiceClient) ensureControlStream(ctx context.Context) error {
	c.mu.Lock()
	if c.stream != nil {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	if err := c.ensureMetaClient(ctx); err != nil {
		return err
	}
	stream, controlCancel, err := c.openControlStream()
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.stream = stream
	c.controlCancel = controlCancel
	c.mu.Unlock()
	go c.receiveLoop(stream)
	return nil
}

func (c *ServiceClient) openControlStream() (lifecyclepb.ServiceControl_ControlClient, context.CancelFunc, error) {
	controlCtx, controlCancel := context.WithCancel(context.Background())
	stream, err := c.controlClient.Control(controlCtx)
	if err != nil {
		controlCancel()
		return nil, nil, err
	}
	return stream, controlCancel, nil
}

func (c *ServiceClient) ensureMetaClient(ctx context.Context) error {
	c.mu.Lock()
	if c.conn != nil {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	if c.config.Address == "" {
		return errors.New("redqueen gRPC address is required")
	}
	dialCtx := ctx
	cancel := func() {}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		dialCtx, cancel = context.WithTimeout(ctx, c.config.InitialConnectTimeout)
	}
	defer cancel()
	c.logInfof("[RedQueenServiceClient] [Connect] (Address:`%s`, Timeout:`%s`, ClientId:`%d`) <Start>", c.config.Address, c.config.InitialConnectTimeout, c.ClientID())
	conn, err := grpc.DialContext(dialCtx, c.config.Address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		c.logErrorf("[RedQueenServiceClient] [Connect] (Address:`%s`, ClientId:`%d`) <Failure> %v", c.config.Address, c.ClientID(), err)
		return err
	}
	c.mu.Lock()
	c.conn = conn
	c.controlClient = lifecyclepb.NewServiceControlClient(conn)
	c.metaClient = metapb.NewServiceMetaClient(conn)
	c.mu.Unlock()
	c.logInfof("[RedQueenServiceClient] [Connect] (Address:`%s`, ClientId:`%d`) <Done>", c.config.Address, c.ClientID())
	return nil
}

func (c *ServiceClient) sendAndAwait(ctx context.Context, frame *lifecyclepb.ServiceControlFrame, timeout time.Duration) (*lifecyclepb.ServiceControlFrame, error) {
	if frame == nil {
		return nil, errors.New("redqueen service control frame is nil")
	}
	if timeout <= 0 {
		timeout = c.config.CommandTimeout
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	c.waiter.prepare(frame.FrameGuid)
	c.mu.Lock()
	stream := c.stream
	c.mu.Unlock()
	if stream == nil {
		return nil, errors.New("redqueen service control stream is nil")
	}
	if err := stream.Send(frame); err != nil {
		c.waiter.completeExceptionally(frame.FrameGuid, err)
		return nil, err
	}
	return c.waiter.await(waitCtx, frame.FrameGuid)
}

func (c *ServiceClient) receiveLoop(stream lifecyclepb.ServiceControl_ControlClient) {
	for {
		frame, err := stream.Recv()
		if err != nil {
			if err != io.EOF {
				c.handleStreamBroken(err)
			}
			return
		}
		c.acceptServerFrame(frame)
	}
}

func (c *ServiceClient) acceptServerFrame(frame *lifecyclepb.ServiceControlFrame) {
	if frame == nil {
		return
	}
	if frame.FrameType == lifecyclepb.ServiceControlFrameType_ERROR {
		err := fmt.Errorf("redqueen service control error: %s %s", frame.GetError().GetCode(), frame.GetError().GetMessage())
		c.waiter.completeExceptionally(frame.CorrelationGuid, err)
		return
	}
	if frame.FrameType == lifecyclepb.ServiceControlFrameType_SHUTDOWN_SERVICE {
		c.acceptShutdownServiceFrame(frame)
		return
	}
	c.waiter.complete(frame.CorrelationGuid, frame)
}

func (c *ServiceClient) acceptShutdownServiceFrame(frame *lifecyclepb.ServiceControlFrame) {
	instruction := toShutdownInstruction(frame.GetShutdownService())
	c.mu.Lock()
	handlers := append([]servicearch.ServiceClientManipulationHandler{}, c.manipulationHandlers...)
	sessionGUID := c.sessionGUID
	stream := c.stream
	c.mu.Unlock()
	for _, handler := range handlers {
		handler(instruction)
	}
	if stream != nil {
		_ = stream.Send(c.factory.shutdownAcceptedFrame(c.ClientID(), sessionGUID, frame))
	}
}

func (c *ServiceClient) handleStreamBroken(err error) {
	c.mu.Lock()
	if c.terminated {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()
	c.logErrorf("[RedQueenServiceClient] [StreamBroken] (ClientId:`%d`, Reason:`%s`) <Disconnected>", c.ClientID(), err)
	c.notifyConnectionState("OFFLINE", err.Error())
	c.markDisconnected()
	c.waiter.completeAllExceptionally(err)
	if c.config.AutoReconnect {
		c.logInfof("[RedQueenServiceClient] [StreamBroken] (ClientId:`%d`) <ReconnectRequested>", c.ClientID())
		c.requestControlReconnect("StreamError")
	}
}

func (c *ServiceClient) markDisconnected() {
	c.mu.Lock()
	if c.controlCancel != nil {
		c.controlCancel()
		c.controlCancel = nil
	}
	c.stream = nil
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
		c.controlClient = nil
		c.metaClient = nil
	}
	c.sessionGUID = ""
	c.ready = false
	c.mu.Unlock()
}

func (c *ServiceClient) requestControlReconnect(reason string) {
	c.mu.Lock()
	if c.terminated {
		c.mu.Unlock()
		return
	}
	c.reconnectRequested = true
	c.reconnectReason = reason
	if c.reconnecting {
		c.mu.Unlock()
		return
	}
	c.reconnecting = true
	c.mu.Unlock()

	go c.runControlReconnectLoop()
}

func (c *ServiceClient) runControlReconnectLoop() {
	failures := 0
	defer func() {
		c.mu.Lock()
		c.reconnecting = false
		shouldRestart := c.reconnectRequested && !c.terminated
		reason := c.reconnectReason
		c.mu.Unlock()
		if shouldRestart {
			c.requestControlReconnect(reason)
		}
	}()

	for {
		reason, ok := c.consumeControlReconnectReason()
		if !ok {
			return
		}
		for {
			if err := c.synchronizeControlState(context.Background(), reason); err != nil {
				if c.isTerminated() {
					return
				}
				failures++
				delay := controlReconnectDelay(failures)
				c.logErrorf("[RedQueenServiceClient] [ControlReconnect] (ClientId:`%d`, Attempt:`%d`, Delay:`%s`) <Failure> %v", c.ClientID(), failures, delay, err)
				time.Sleep(delay)
				if c.isTerminated() {
					return
				}
				if nextReason, ok := c.consumeControlReconnectReason(); ok {
					reason = nextReason
				}
				continue
			}
			c.logInfof("[RedQueenServiceClient] [ControlReconnect] (ClientId:`%d`, Reason:`%s`) <Done>", c.ClientID(), reason)
			failures = 0
			break
		}
	}
}

func (c *ServiceClient) consumeControlReconnectReason() (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.reconnectRequested || c.terminated {
		return "", false
	}
	c.reconnectRequested = false
	return c.reconnectReason, true
}

func (c *ServiceClient) isTerminated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.terminated
}

func (c *ServiceClient) isControlReady() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ready && c.stream != nil
}

func (c *ServiceClient) startHeartbeat() {
	if !c.config.EnableHeartbeat {
		return
	}
	c.mu.Lock()
	if c.heartbeatCancel != nil {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.heartbeatCancel = cancel
	interval := c.config.HeartbeatInterval
	c.mu.Unlock()
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.sendHeartbeatQuietly()
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (c *ServiceClient) sendHeartbeatQuietly() {
	if !c.isControlReady() {
		return
	}
	c.mu.Lock()
	stream := c.stream
	sessionGUID := c.sessionGUID
	instanceGUID := c.instanceGUID
	c.mu.Unlock()
	if stream == nil || instanceGUID == "" {
		return
	}
	if err := stream.Send(c.factory.heartbeatFrame(c.ClientID(), sessionGUID, instanceGUID)); err != nil {
		c.logErrorf("[RedQueenServiceClient] [Heartbeat] (ClientId:`%d`) <Failure> %v", c.ClientID(), err)
		c.handleStreamBroken(err)
	}
}

func (c *ServiceClient) logInfof(format string, args ...any) {
	if c != nil && c.config.Logger != nil {
		c.config.Logger.Infof(format, args...)
	}
}

func (c *ServiceClient) logErrorf(format string, args ...any) {
	if c != nil && c.config.Logger != nil {
		c.config.Logger.Errorf(format, args...)
	}
}

func (c *ServiceClient) notifyConnectionState(state string, reason string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	handlers := append([]servicearch.ServiceClientConnectionStateHandler{}, c.connectionHandlers...)
	c.mu.Unlock()
	for _, handler := range handlers {
		handler(state, reason)
	}
}

func controlReconnectDelay(failures int) time.Duration {
	if failures <= 1 {
		return 500 * time.Millisecond
	}
	if failures == 2 {
		return time.Second
	}
	if failures == 3 {
		return 2 * time.Second
	}
	return 5 * time.Second
}

func unexpectedFrameError(expected string, frame *lifecyclepb.ServiceControlFrame) error {
	if frame == nil {
		return fmt.Errorf("expected `%s`, but got nil frame", expected)
	}
	return fmt.Errorf("expected `%s`, but got `%s`", expected, frame.FrameType.String())
}
