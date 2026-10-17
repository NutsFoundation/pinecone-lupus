package client

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	servicearch "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/arch"
	lifecyclepb "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/redqueen/protocol/lifecycle"
	metapb "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/redqueen/protocol/meta"
	guidarch "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/arch"
	guidcore "github.com/NutsFoundation/pinecone-lupus/pinecone/util/guid/core"
	"google.golang.org/grpc"
)

var errServiceClientTerminated = errors.New("redqueen service client terminated")
var errControlStreamClosed = errors.New("redqueen service control stream closed")

type controlSendError struct{ error }

func (e controlSendError) Unwrap() error { return e.error }

type controlSession struct {
	stream   lifecyclepb.ServiceControl_ControlClient
	cancel   context.CancelFunc
	waiter   *correlationWaiter
	sendGate chan struct{}
}

func bindClientLifetime(ctx, lifetime context.Context) (context.Context, context.CancelFunc) {
	bound, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(lifetime, cancel)
	if lifetime.Err() != nil {
		cancel()
	}
	return bound, func() { stop(); cancel() }
}

type ServiceClient struct {
	config    Config
	allocator guidarch.GuidAllocator
	factory   *frameFactory

	conn          *grpc.ClientConn
	controlClient lifecyclepb.ServiceControlClient
	metaClient    metapb.ServiceMetaClient
	control       *controlSession

	sessionGUID  string
	instanceGUID string
	ready        bool
	terminated   bool

	mu                   sync.Mutex
	connectMu            sync.Mutex
	syncMu               sync.Mutex
	lifetimeCtx          context.Context
	lifetimeCancel       context.CancelFunc
	stateHandlers        []servicearch.ServiceClientStateSynchronizedHandler
	manipulationHandlers []servicearch.ServiceClientManipulationHandler
	connectionHandlers   []servicearch.ServiceClientConnectionStateHandler
	connectionState      string
	heartbeatCancel      context.CancelFunc
	reconnectRequested   bool
	reconnecting         bool
	reconnectReason      string
	reconnectLifetime    context.Context
}

var _ servicearch.ServiceClient = (*ServiceClient)(nil)
var _ servicearch.ServiceLifecyclePort = (*ServiceClient)(nil)
var _ servicearch.ServiceMetaPort = (*ServiceClient)(nil)

func NewServiceClient(config Config, allocator guidarch.GuidAllocator) *ServiceClient {
	config = config.Normalize()
	if allocator == nil {
		allocator = guidcore.NewUUIDV7GuidAllocator()
	}
	lifetimeCtx, lifetimeCancel := context.WithCancel(context.Background())
	return &ServiceClient{
		config:         config,
		allocator:      allocator,
		factory:        newFrameFactory(allocator),
		lifetimeCtx:    lifetimeCtx,
		lifetimeCancel: lifetimeCancel,
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
	if c.terminated {
		c.lifetimeCtx, c.lifetimeCancel = context.WithCancel(context.Background())
		c.terminated = false
	} else if c.ready && c.control != nil {
		c.mu.Unlock()
		return nil
	}
	lifetime := c.lifetimeCtx
	c.mu.Unlock()
	startCtx, cancel := bindClientLifetime(ctx, lifetime)
	defer cancel()
	c.notifyConnectionState("CONNECTING", "Startup", lifetime, nil)
	c.logInfof("[RedQueenServiceClient] [Start] (Address:`%s`, ClientId:`%d`) <Start>", c.config.Address, c.ClientID())
	if err := c.synchronizeControlState(startCtx, "Startup"); err != nil {
		c.notifyConnectionState("OFFLINE", err.Error(), lifetime, nil)
		c.logErrorf("[RedQueenServiceClient] [Start] (Address:`%s`, ClientId:`%d`) <Failure> %v", c.config.Address, c.ClientID(), err)
		return err
	}
	c.startHeartbeat(lifetime)
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
	lifetimeCancel := c.lifetimeCancel
	if c.heartbeatCancel != nil {
		c.heartbeatCancel()
		c.heartbeatCancel = nil
	}
	session := c.control
	conn := c.conn
	c.control = nil
	c.conn = nil
	c.controlClient = nil
	c.metaClient = nil
	c.ready = false
	c.sessionGUID = ""
	c.connectionHandlers = nil
	if session != nil {
		session.waiter.completeAllExceptionally(errServiceClientTerminated)
	}
	c.mu.Unlock()

	lifetimeCancel()
	if session != nil {
		session.cancel()
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
	session := c.control
	c.mu.Unlock()
	frame := c.factory.registerFrame(c.ClientID(), sessionGUID, request)
	response, err := c.sendAndAwait(ctx, session, frame, c.config.CommandTimeout)
	if err != nil {
		c.handleSendFailure(session, err)
		return nil, err
	}
	if response == nil || response.FrameType != lifecyclepb.ServiceControlFrameType_REGISTER_ACCEPTED {
		return nil, unexpectedFrameError("REGISTER_ACCEPTED", response)
	}
	result := toJoinResponse(response.GetRegisterAccepted())
	if result != nil {
		c.mu.Lock()
		if c.control == session && !c.terminated {
			c.instanceGUID = result.InstanceGUID
		}
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
	session := c.control
	c.mu.Unlock()
	frame := c.factory.deregisterFrame(c.ClientID(), sessionGUID, instruction)
	response, err := c.sendAndAwait(ctx, session, frame, c.config.CommandTimeout)
	if err != nil {
		c.handleSendFailure(session, err)
		return nil, err
	}
	if response == nil || response.FrameType != lifecyclepb.ServiceControlFrameType_DEREGISTERED {
		return nil, unexpectedFrameError("DEREGISTERED", response)
	}
	c.mu.Lock()
	if c.control == session && !c.terminated {
		c.instanceGUID = ""
	}
	c.mu.Unlock()
	return toDeregisterResult(instruction, response.GetDeregistered()), nil
}

func (c *ServiceClient) QueryServiceMetaByPath(ctx context.Context, path string) (*servicearch.ServiceMeta, error) {
	client, err := c.serviceMetaClient(ctx)
	if err != nil {
		return nil, err
	}
	reply, err := client.QueryServiceMetaByPath(ctx, &metapb.QueryServiceMetaByPathRequest{Path: path})
	if err != nil {
		return nil, err
	}
	return toServiceMetaDTO(reply.GetMeta()), nil
}

func (c *ServiceClient) QueryServiceMetaByGUID(ctx context.Context, guid string) (*servicearch.ServiceMeta, error) {
	client, err := c.serviceMetaClient(ctx)
	if err != nil {
		return nil, err
	}
	reply, err := client.QueryServiceMetaByGuid(ctx, &metapb.QueryServiceMetaByGuidRequest{Guid: guid})
	if err != nil {
		return nil, err
	}
	return toServiceMetaDTO(reply.GetMeta()), nil
}

func (c *ServiceClient) CreateNewService(ctx context.Context, parentAppPath string, meta servicearch.ServiceMeta) (string, error) {
	client, err := c.serviceMetaClient(ctx)
	if err != nil {
		return "", err
	}
	reply, err := client.CreateNewService(ctx, &metapb.CreateNewServiceRequest{
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
	c.mu.Lock()
	lifetime := c.lifetimeCtx
	c.mu.Unlock()
	syncCtx, cancel := bindClientLifetime(ctx, lifetime)
	defer cancel()
	c.syncMu.Lock()
	session, err := c.synchronizeControlStateOnce(syncCtx, lifetime)
	c.syncMu.Unlock()
	if err != nil {
		c.handleStreamBroken(session, err)
		return err
	}
	if session == nil {
		return nil
	}
	c.mu.Lock()
	if c.terminated || c.control != session || !c.ready {
		c.mu.Unlock()
		return errControlStreamClosed
	}
	sessionGUID := c.sessionGUID
	handlers := append([]servicearch.ServiceClientStateSynchronizedHandler{}, c.stateHandlers...)
	c.mu.Unlock()
	c.logInfof("[RedQueenServiceClient] [ControlReady] (ClientId:`%d`, Session:`%s`, Reason:`%s`) <Done>", c.ClientID(), sessionGUID, reason)
	c.notifyConnectionState("ONLINE", reason, lifetime, session)
	for _, handler := range handlers {
		c.mu.Lock()
		current := !c.terminated && c.control == session && c.ready
		c.mu.Unlock()
		if !current {
			return errControlStreamClosed
		}
		handler(reason)
	}
	c.mu.Lock()
	current := !c.terminated && c.control == session && c.ready
	c.mu.Unlock()
	if !current {
		return errControlStreamClosed
	}
	return nil
}

func (c *ServiceClient) synchronizeControlStateOnce(ctx, lifetime context.Context) (*controlSession, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c.mu.Lock()
	if c.terminated || c.lifetimeCtx != lifetime {
		c.mu.Unlock()
		return nil, errServiceClientTerminated
	}
	if c.ready && c.control != nil {
		c.mu.Unlock()
		return nil, nil
	}
	c.mu.Unlock()
	syncCtx, cancel := context.WithTimeout(ctx, c.config.ControlSyncTimeout)
	defer cancel()
	session, err := c.ensureControlStream(syncCtx, lifetime)
	if err != nil {
		return nil, err
	}
	frame := c.factory.clientMusterFrame(c.ClientID(), c.config.ClientName, c.config)
	response, err := c.sendAndAwait(syncCtx, session, frame, c.config.ControlSyncTimeout)
	if err != nil {
		return session, err
	}
	if response == nil || response.FrameType != lifecyclepb.ServiceControlFrameType_CLIENT_READY {
		err = unexpectedFrameError("CLIENT_READY", response)
		return session, err
	}
	c.mu.Lock()
	if c.terminated || c.control != session {
		c.mu.Unlock()
		return session, errControlStreamClosed
	}
	c.sessionGUID = response.GetClientReady().GetSessionGuid()
	c.ready = true
	c.mu.Unlock()
	return session, nil
}

func (c *ServiceClient) ensureControlReady(ctx context.Context) error {
	if c.isControlReady() {
		return nil
	}
	return c.synchronizeControlState(ctx, "StreamRecovered")
}

func (c *ServiceClient) ensureControlStream(ctx, lifetime context.Context) (*controlSession, error) {
	c.mu.Lock()
	if c.terminated || c.lifetimeCtx != lifetime {
		c.mu.Unlock()
		return nil, errServiceClientTerminated
	}
	if c.control != nil {
		session := c.control
		c.mu.Unlock()
		return session, nil
	}
	c.mu.Unlock()
	if err := c.ensureMetaClient(ctx); err != nil {
		return nil, err
	}
	c.mu.Lock()
	client := c.controlClient
	current := !c.terminated && c.lifetimeCtx == lifetime && client != nil
	c.mu.Unlock()
	if !current {
		return nil, errControlStreamClosed
	}
	controlCtx, controlCancel := context.WithCancel(lifetime)
	stopStartupCancel := context.AfterFunc(ctx, controlCancel)
	stream, err := client.Control(controlCtx)
	stopStartupCancel()
	if err != nil {
		controlCancel()
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		controlCancel()
		return nil, err
	}
	session := &controlSession{stream: stream, cancel: controlCancel, waiter: newCorrelationWaiter(), sendGate: make(chan struct{}, 1)}
	c.mu.Lock()
	if c.terminated || c.lifetimeCtx != lifetime || c.controlClient != client {
		c.mu.Unlock()
		controlCancel()
		return nil, errControlStreamClosed
	}
	if c.control != nil {
		existing := c.control
		c.mu.Unlock()
		controlCancel()
		return existing, nil
	}
	c.control = session
	c.mu.Unlock()
	go c.receiveLoop(session)
	return session, nil
}

func (c *ServiceClient) ensureMetaClient(ctx context.Context) error {
	c.mu.Lock()
	lifetime := c.lifetimeCtx
	c.mu.Unlock()
	operationCtx, operationCancel := bindClientLifetime(ctx, lifetime)
	defer operationCancel()
	c.connectMu.Lock()
	defer c.connectMu.Unlock()
	if err := operationCtx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	if c.terminated || c.lifetimeCtx != lifetime {
		c.mu.Unlock()
		return errServiceClientTerminated
	}
	if c.conn != nil {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()
	if c.config.Address == "" {
		return errors.New("redqueen gRPC address is required")
	}
	transportCredentials, err := c.config.transportCredentials()
	if err != nil {
		return err
	}
	dialCtx, cancel := context.WithTimeout(operationCtx, c.config.InitialConnectTimeout)
	defer cancel()
	c.logInfof("[RedQueenServiceClient] [Connect] (Address:`%s`, Timeout:`%s`, ClientId:`%d`) <Start>", c.config.Address, c.config.InitialConnectTimeout, c.ClientID())
	conn, err := grpc.DialContext(dialCtx, c.config.Address, grpc.WithTransportCredentials(transportCredentials), grpc.WithBlock())
	if err != nil {
		c.logErrorf("[RedQueenServiceClient] [Connect] (Address:`%s`, ClientId:`%d`) <Failure> %v", c.config.Address, c.ClientID(), err)
		return err
	}
	c.mu.Lock()
	if c.terminated || c.lifetimeCtx != lifetime || dialCtx.Err() != nil {
		c.mu.Unlock()
		_ = conn.Close()
		return errServiceClientTerminated
	}
	c.conn = conn
	c.controlClient = lifecyclepb.NewServiceControlClient(conn)
	c.metaClient = metapb.NewServiceMetaClient(conn)
	c.mu.Unlock()
	c.logInfof("[RedQueenServiceClient] [Connect] (Address:`%s`, ClientId:`%d`) <Done>", c.config.Address, c.ClientID())
	return nil
}

func (c *ServiceClient) serviceMetaClient(ctx context.Context) (metapb.ServiceMetaClient, error) {
	if err := c.ensureMetaClient(ctx); err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.terminated || c.metaClient == nil {
		return nil, errControlStreamClosed
	}
	return c.metaClient, nil
}

func (c *ServiceClient) sendAndAwait(ctx context.Context, session *controlSession, frame *lifecyclepb.ServiceControlFrame, timeout time.Duration) (*lifecyclepb.ServiceControlFrame, error) {
	if frame == nil {
		return nil, errors.New("redqueen service control frame is nil")
	}
	if timeout <= 0 {
		timeout = c.config.CommandTimeout
	}
	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	c.mu.Lock()
	if c.terminated || session == nil || c.control != session {
		c.mu.Unlock()
		return nil, errControlStreamClosed
	}
	pending := session.waiter.prepare(frame.FrameGuid)
	c.mu.Unlock()
	defer session.waiter.remove(pending)
	if err := c.sendFrame(waitCtx, session, frame); err != nil {
		return nil, err
	}
	return session.waiter.await(waitCtx, pending)
}

func (c *ServiceClient) sendControlFrame(ctx context.Context, session *controlSession, frame *lifecyclepb.ServiceControlFrame) error {
	err := c.sendFrame(ctx, session, frame)
	c.handleSendFailure(session, err)
	return err
}

func (c *ServiceClient) handleSendFailure(session *controlSession, err error) {
	var sendErr controlSendError
	if errors.As(err, &sendErr) {
		c.handleStreamBroken(session, sendErr.error)
	}
}

func (c *ServiceClient) sendFrame(ctx context.Context, session *controlSession, frame *lifecyclepb.ServiceControlFrame) error {
	select {
	case session.sendGate <- struct{}{}:
		defer func() { <-session.sendGate }()
	case <-ctx.Done():
		return ctx.Err()
	case <-session.stream.Context().Done():
		return errControlStreamClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	current := !c.terminated && c.control == session
	c.mu.Unlock()
	if !current {
		return errControlStreamClosed
	}
	// gRPC cannot cancel an individual Send, so its deadline cancels this stream.
	cancelFinished := make(chan struct{})
	stopSendCancel := context.AfterFunc(ctx, func() {
		session.cancel()
		close(cancelFinished)
	})
	err := session.stream.Send(frame)
	if !stopSendCancel() {
		<-cancelFinished
	}
	if ctx.Err() != nil {
		return controlSendError{ctx.Err()}
	}
	if err != nil {
		return controlSendError{err}
	}
	return nil
}

func (c *ServiceClient) receiveLoop(session *controlSession) {
	for {
		frame, err := session.stream.Recv()
		if err != nil {
			c.handleStreamBroken(session, err)
			return
		}
		c.acceptServerFrame(session, frame)
	}
}

func (c *ServiceClient) acceptServerFrame(session *controlSession, frame *lifecyclepb.ServiceControlFrame) {
	if frame == nil {
		return
	}
	c.mu.Lock()
	current := !c.terminated && c.control == session
	c.mu.Unlock()
	if !current {
		return
	}
	if frame.FrameType == lifecyclepb.ServiceControlFrameType_ERROR {
		err := fmt.Errorf("redqueen service control error: %s %s", frame.GetError().GetCode(), frame.GetError().GetMessage())
		session.waiter.completeExceptionally(frame.CorrelationGuid, err)
		return
	}
	if frame.FrameType == lifecyclepb.ServiceControlFrameType_SHUTDOWN_SERVICE {
		c.acceptShutdownServiceFrame(session, frame)
		return
	}
	session.waiter.complete(frame.CorrelationGuid, frame)
}

func (c *ServiceClient) acceptShutdownServiceFrame(session *controlSession, frame *lifecyclepb.ServiceControlFrame) {
	instruction := toShutdownInstruction(frame.GetShutdownService())
	c.mu.Lock()
	handlers := append([]servicearch.ServiceClientManipulationHandler{}, c.manipulationHandlers...)
	sessionGUID := c.sessionGUID
	c.mu.Unlock()
	for _, handler := range handlers {
		handler(instruction)
	}
	_ = c.sendControlFrame(context.Background(), session, c.factory.shutdownAcceptedFrame(c.ClientID(), sessionGUID, frame))
}

func (c *ServiceClient) handleStreamBroken(session *controlSession, err error) {
	c.mu.Lock()
	if c.terminated || session == nil || c.control != session {
		c.mu.Unlock()
		return
	}
	conn := c.conn
	lifetime := c.lifetimeCtx
	c.control = nil
	c.conn = nil
	c.controlClient = nil
	c.metaClient = nil
	c.sessionGUID = ""
	c.ready = false
	c.connectionState = "OFFLINE"
	session.waiter.completeAllExceptionally(err)
	handlers := append([]servicearch.ServiceClientConnectionStateHandler{}, c.connectionHandlers...)
	c.mu.Unlock()
	session.cancel()
	if conn != nil {
		_ = conn.Close()
	}
	c.logErrorf("[RedQueenServiceClient] [StreamBroken] (ClientId:`%d`, Reason:`%s`) <Disconnected>", c.ClientID(), err)
	for _, handler := range handlers {
		c.mu.Lock()
		current := !c.terminated && c.lifetimeCtx == lifetime && c.control == nil && c.connectionState == "OFFLINE"
		c.mu.Unlock()
		if !current {
			break
		}
		handler("OFFLINE", err.Error())
	}
	if c.config.AutoReconnect {
		c.requestControlReconnect("StreamError", lifetime)
	}
}

func (c *ServiceClient) requestControlReconnect(reason string, lifetime context.Context) {
	c.mu.Lock()
	if c.terminated || c.lifetimeCtx != lifetime || !c.config.AutoReconnect {
		c.mu.Unlock()
		return
	}
	c.reconnectRequested = true
	c.reconnectReason = reason
	if c.reconnecting && c.reconnectLifetime == lifetime {
		c.mu.Unlock()
		return
	}
	c.reconnecting = true
	c.reconnectLifetime = lifetime
	c.mu.Unlock()

	go c.runControlReconnectLoop(lifetime)
}

func (c *ServiceClient) runControlReconnectLoop(lifetime context.Context) {
	failures := 0
	defer func() {
		c.mu.Lock()
		if c.reconnectLifetime != lifetime {
			c.mu.Unlock()
			return
		}
		c.reconnecting = false
		shouldRestart := c.reconnectRequested && !c.terminated && c.lifetimeCtx == lifetime
		reason := c.reconnectReason
		c.mu.Unlock()
		if shouldRestart {
			c.requestControlReconnect(reason, lifetime)
		}
	}()

	for {
		reason, ok := c.consumeControlReconnectReason(lifetime)
		if !ok {
			return
		}
		for {
			if err := c.synchronizeControlState(lifetime, reason); err != nil {
				if lifetime.Err() != nil {
					return
				}
				failures++
				delay := controlReconnectDelay(failures)
				c.logErrorf("[RedQueenServiceClient] [ControlReconnect] (ClientId:`%d`, Attempt:`%d`, Delay:`%s`) <Failure> %v", c.ClientID(), failures, delay, err)
				timer := time.NewTimer(delay)
				select {
				case <-lifetime.Done():
					timer.Stop()
					return
				case <-timer.C:
				}
				if nextReason, ok := c.consumeControlReconnectReason(lifetime); ok {
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

func (c *ServiceClient) consumeControlReconnectReason(lifetime context.Context) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.reconnectRequested || c.terminated || c.lifetimeCtx != lifetime {
		return "", false
	}
	c.reconnectRequested = false
	return c.reconnectReason, true
}

func (c *ServiceClient) isControlReady() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ready && c.control != nil && !c.terminated
}

func (c *ServiceClient) startHeartbeat(lifetime context.Context) {
	if !c.config.EnableHeartbeat {
		return
	}
	c.mu.Lock()
	if c.heartbeatCancel != nil || c.terminated || c.lifetimeCtx != lifetime {
		c.mu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(c.lifetimeCtx)
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
	session := c.control
	sessionGUID := c.sessionGUID
	instanceGUID := c.instanceGUID
	c.mu.Unlock()
	if session == nil || instanceGUID == "" {
		return
	}
	if err := c.sendControlFrame(context.Background(), session, c.factory.heartbeatFrame(c.ClientID(), sessionGUID, instanceGUID)); err != nil {
		c.logErrorf("[RedQueenServiceClient] [Heartbeat] (ClientId:`%d`) <Failure> %v", c.ClientID(), err)
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

func (c *ServiceClient) notifyConnectionState(state string, reason string, lifetime context.Context, session *controlSession) {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.connectionState == state || c.terminated || c.lifetimeCtx != lifetime || (session != nil && c.control != session) || (state == "OFFLINE" && c.control != nil) {
		c.mu.Unlock()
		return
	}
	c.connectionState = state
	handlers := append([]servicearch.ServiceClientConnectionStateHandler{}, c.connectionHandlers...)
	c.mu.Unlock()
	for _, handler := range handlers {
		c.mu.Lock()
		current := !c.terminated && c.lifetimeCtx == lifetime && c.connectionState == state && (session == nil || c.control == session)
		c.mu.Unlock()
		if !current {
			break
		}
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
