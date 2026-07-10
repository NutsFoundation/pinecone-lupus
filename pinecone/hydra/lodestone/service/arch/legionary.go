package arch

import "context"

type CollectiveServiceLegionary interface {
	Name() string
	ClientID() int64
	ServiceGUID() string
	InstanceGUID() string
	State() ServiceLegionaryState
	ServiceClient() ServiceClient
	JoinRequest() ServiceJoinRequest

	StartService(ctx context.Context) error
	JoinRegiment(ctx context.Context) (*ServiceJoinResponse, error)
	RequestRejoinRegiment(reason string)
	Deregister(ctx context.Context, reason string) error
	TerminateService(ctx context.Context, reason string)
}
