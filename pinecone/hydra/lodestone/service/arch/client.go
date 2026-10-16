package arch

import "context"

type ServiceClient interface {
	ClientID() int64
	StartService(ctx context.Context) error
	TerminateService(ctx context.Context) error
	Lifecycle() ServiceLifecyclePort
	Meta() ServiceMetaPort
	RegisterStateSynchronizedHandler(handler ServiceClientStateSynchronizedHandler)
	DeregisterStateSynchronizedHandler(handler ServiceClientStateSynchronizedHandler)
	RegisterManipulationHandler(handler ServiceClientManipulationHandler)
	DeregisterManipulationHandler(handler ServiceClientManipulationHandler)
	RegisterConnectionStateHandler(handler ServiceClientConnectionStateHandler)
	DeregisterConnectionStateHandler(handler ServiceClientConnectionStateHandler)
}

type ServiceLifecyclePort interface {
	Register(ctx context.Context, request ServiceJoinRequest) (*ServiceJoinResponse, error)
	Deregister(ctx context.Context, instruction ServiceDeregisterInstruction) (*ServiceDeregisterResult, error)
}

type ServiceMetaPort interface {
	QueryServiceMetaByPath(ctx context.Context, path string) (*ServiceMeta, error)
	QueryServiceMetaByGUID(ctx context.Context, guid string) (*ServiceMeta, error)
	CreateNewService(ctx context.Context, parentAppPath string, meta ServiceMeta) (string, error)
}

type ServiceClientStateSynchronizedHandler func(reason string)

type ServiceClientManipulationHandler func(instruction ServiceShutdownInstruction)

type ServiceClientConnectionStateHandler func(state string, reason string)
