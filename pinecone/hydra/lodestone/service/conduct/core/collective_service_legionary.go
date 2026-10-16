package core

import (
	"context"
	"errors"
	"sync"
	"time"

	servicearch "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/arch"
)

const (
	rejoinRetryDelayMillis0 = 500 * time.Millisecond
	rejoinRetryDelayMillis1 = 1000 * time.Millisecond
	rejoinRetryDelayMillis2 = 2000 * time.Millisecond
)

type CollectiveServiceLegionary struct {
	name          string
	serviceClient servicearch.ServiceClient
	joinRequest   servicearch.ServiceJoinRequest

	serviceGUID  string
	instanceGUID string
	state        servicearch.ServiceLegionaryState
	logger       servicearch.Logger

	rejoinMu        sync.Mutex
	rejoinRequested bool
	rejoining       bool
	terminated      bool
	rejoinReason    string

	stateSynchronizedHandler servicearch.ServiceClientStateSynchronizedHandler
	manipulationHandler      servicearch.ServiceClientManipulationHandler
}

var _ servicearch.CollectiveServiceLegionary = (*CollectiveServiceLegionary)(nil)

func NewCollectiveServiceLegionary(
	name string,
	serviceClient servicearch.ServiceClient,
	joinRequest servicearch.ServiceJoinRequest,
	loggers ...servicearch.Logger,
) *CollectiveServiceLegionary {
	legionary := &CollectiveServiceLegionary{
		name:          name,
		serviceClient: serviceClient,
		joinRequest:   joinRequest,
		serviceGUID:   joinRequest.ServiceGUID,
		state:         servicearch.ServiceLegionaryStateNew,
	}
	if len(loggers) > 0 {
		legionary.logger = loggers[0]
	}
	return legionary
}

func (l *CollectiveServiceLegionary) Name() string {
	if l == nil {
		return ""
	}
	return l.name
}

func (l *CollectiveServiceLegionary) ClientID() int64 {
	if l == nil || l.serviceClient == nil {
		return 0
	}
	return l.serviceClient.ClientID()
}

func (l *CollectiveServiceLegionary) ServiceGUID() string {
	if l == nil {
		return ""
	}
	return l.serviceGUID
}

func (l *CollectiveServiceLegionary) InstanceGUID() string {
	if l == nil {
		return ""
	}
	return l.instanceGUID
}

func (l *CollectiveServiceLegionary) State() servicearch.ServiceLegionaryState {
	if l == nil {
		return servicearch.ServiceLegionaryStateTerminated
	}
	return l.state
}

func (l *CollectiveServiceLegionary) ServiceClient() servicearch.ServiceClient {
	if l == nil {
		return nil
	}
	return l.serviceClient
}

func (l *CollectiveServiceLegionary) JoinRequest() servicearch.ServiceJoinRequest {
	if l == nil {
		return servicearch.ServiceJoinRequest{}
	}
	return l.joinRequest
}

func (l *CollectiveServiceLegionary) StartService(ctx context.Context) error {
	if l == nil {
		return nil
	}
	if l.serviceClient == nil {
		return errors.New("lodestone service client is nil")
	}
	l.registerManipulationHandler()
	l.state = servicearch.ServiceLegionaryStateStarting
	if err := l.serviceClient.StartService(ctx); err != nil {
		l.state = servicearch.ServiceLegionaryStateOffline
		return err
	}
	l.state = servicearch.ServiceLegionaryStateControlReady
	l.registerStateSynchronizedHandler()
	return nil
}

func (l *CollectiveServiceLegionary) JoinRegiment(ctx context.Context) (*servicearch.ServiceJoinResponse, error) {
	if l == nil {
		return nil, nil
	}
	if l.serviceClient == nil || l.serviceClient.Lifecycle() == nil {
		return nil, errors.New("lodestone service lifecycle port is nil")
	}
	if l.joinRequest.ServiceGUID == "" {
		return nil, errors.New("lodestone service guid is required")
	}
	l.state = servicearch.ServiceLegionaryStateJoining
	request := l.joinRequest
	request.ClientID = l.ClientID()
	request.InstanceGUID = l.instanceGUID
	response, err := l.serviceClient.Lifecycle().Register(ctx, request)
	if err != nil {
		l.state = servicearch.ServiceLegionaryStateOffline
		return nil, err
	}
	if response != nil {
		l.serviceGUID = response.ServiceGUID
		l.instanceGUID = response.InstanceGUID
	}
	l.state = servicearch.ServiceLegionaryStateOnline
	l.logInfof("[LodestoneServiceLegionary] [JoinRegiment] (Name:`%s`, ClientId:`%d`, Service:`%s`, Instance:`%s`) <Done>", l.name, l.ClientID(), l.serviceGUID, l.instanceGUID)
	return response, nil
}

func (l *CollectiveServiceLegionary) RequestRejoinRegiment(reason string) {
	if l == nil {
		return
	}
	l.rejoinMu.Lock()
	if l.terminated || l.joinRequest.ServiceGUID == "" {
		l.rejoinMu.Unlock()
		return
	}
	l.rejoinRequested = true
	l.rejoinReason = reason
	if l.rejoining {
		l.rejoinMu.Unlock()
		return
	}
	l.rejoining = true
	l.rejoinMu.Unlock()

	go l.runRejoinRegimentLoop()
}

func (l *CollectiveServiceLegionary) runRejoinRegimentLoop() {
	failures := 0
	defer func() {
		l.rejoinMu.Lock()
		l.rejoining = false
		shouldRestart := l.rejoinRequested && !l.terminated
		reason := l.rejoinReason
		l.rejoinMu.Unlock()
		if shouldRestart {
			l.RequestRejoinRegiment(reason)
		}
	}()

	for {
		reason, ok := l.consumeRejoinReason()
		if !ok {
			return
		}
		if l.rejoinRegimentOnce(reason) {
			failures = 0
			continue
		}
		time.Sleep(rejoinRetryDelay(failures))
		failures++
		l.RequestRejoinRegiment(reason)
		return
	}
}

func (l *CollectiveServiceLegionary) consumeRejoinReason() (string, bool) {
	l.rejoinMu.Lock()
	defer l.rejoinMu.Unlock()
	if !l.rejoinRequested || l.terminated {
		return "", false
	}
	l.rejoinRequested = false
	return l.rejoinReason, true
}

func (l *CollectiveServiceLegionary) rejoinRegimentOnce(reason string) bool {
	l.state = servicearch.ServiceLegionaryStateRejoining
	if _, err := l.JoinRegiment(context.Background()); err != nil {
		l.state = servicearch.ServiceLegionaryStateOffline
		l.logErrorf("[LodestoneServiceLegionary] [RejoinRegiment] (Reason:`%s`, Name:`%s`, ClientId:`%d`) <Failure> %v", reason, l.name, l.ClientID(), err)
		return false
	}
	l.logInfof("[LodestoneServiceLegionary] [RejoinRegiment] (Reason:`%s`, Name:`%s`, ClientId:`%d`) <Done>", reason, l.name, l.ClientID())
	return true
}

func (l *CollectiveServiceLegionary) logInfof(format string, args ...any) {
	if l != nil && l.logger != nil {
		l.logger.Infof(format, args...)
	}
}

func (l *CollectiveServiceLegionary) logErrorf(format string, args ...any) {
	if l != nil && l.logger != nil {
		l.logger.Errorf(format, args...)
	}
}

func rejoinRetryDelay(failures int) time.Duration {
	if failures <= 0 {
		return rejoinRetryDelayMillis0
	}
	if failures == 1 {
		return rejoinRetryDelayMillis1
	}
	return rejoinRetryDelayMillis2
}

func (l *CollectiveServiceLegionary) Deregister(ctx context.Context, reason string) error {
	if l == nil || l.instanceGUID == "" {
		if l != nil {
			l.state = servicearch.ServiceLegionaryStateOffline
		}
		return nil
	}
	if l.serviceClient == nil || l.serviceClient.Lifecycle() == nil {
		return errors.New("lodestone service lifecycle port is nil")
	}
	_, err := l.serviceClient.Lifecycle().Deregister(ctx, servicearch.ServiceDeregisterInstruction{
		InstanceGUID: l.instanceGUID,
		Reason:       reason,
	})
	if err != nil {
		return err
	}
	l.instanceGUID = ""
	l.state = servicearch.ServiceLegionaryStateOffline
	return nil
}

func (l *CollectiveServiceLegionary) TerminateService(ctx context.Context, reason string) {
	if l == nil {
		return
	}
	l.rejoinMu.Lock()
	l.terminated = true
	l.rejoinRequested = false
	l.rejoinMu.Unlock()

	if err := l.Deregister(ctx, reason); err != nil {
		l.logErrorf("[LodestoneServiceLegionary] [Terminate] (Name:`%s`, ClientId:`%d`) <DeregisterFailure> %v", l.name, l.ClientID(), err)
	}
	if l.stateSynchronizedHandler != nil && l.serviceClient != nil {
		l.serviceClient.DeregisterStateSynchronizedHandler(l.stateSynchronizedHandler)
		l.stateSynchronizedHandler = nil
	}
	if l.manipulationHandler != nil && l.serviceClient != nil {
		l.serviceClient.DeregisterManipulationHandler(l.manipulationHandler)
		l.manipulationHandler = nil
	}
	if l.serviceClient != nil {
		_ = l.serviceClient.TerminateService(ctx)
	}
	l.state = servicearch.ServiceLegionaryStateTerminated
}

func (l *CollectiveServiceLegionary) registerStateSynchronizedHandler() {
	if l.stateSynchronizedHandler != nil || l.serviceClient == nil {
		return
	}
	l.stateSynchronizedHandler = func(reason string) {
		l.state = servicearch.ServiceLegionaryStateOffline
		l.RequestRejoinRegiment(reason)
	}
	l.serviceClient.RegisterStateSynchronizedHandler(l.stateSynchronizedHandler)
}

func (l *CollectiveServiceLegionary) registerManipulationHandler() {
	if l.manipulationHandler != nil || l.serviceClient == nil {
		return
	}
	l.manipulationHandler = func(instruction servicearch.ServiceShutdownInstruction) {
		if instruction.InstanceGUID != "" && l.instanceGUID != "" && instruction.InstanceGUID != l.instanceGUID {
			l.logInfof("[LodestoneServiceLegionary] [PassiveShutdown] (Name:`%s`, ClientId:`%d`, CurrentInstance:`%s`, RequestedInstance:`%s`) <Ignored>", l.name, l.ClientID(), l.instanceGUID, instruction.InstanceGUID)
			return
		}
		go l.TerminateService(context.Background(), instruction.Reason)
	}
	l.serviceClient.RegisterManipulationHandler(l.manipulationHandler)
}
