package core

import (
	"context"

	servicearch "github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/lodestone/service/arch"
)

type GenericServiceLodestone struct {
	name              string
	lifecyclePolicy   servicearch.LifecyclePolicy
	serviceClient     servicearch.ServiceClient
	serviceIdentifier string
	declaration       *servicearch.ServiceDeclaration
	legionary         servicearch.CollectiveServiceLegionary
	prepare           func(context.Context) error
}

var _ servicearch.ServiceLodestone = (*GenericServiceLodestone)(nil)

func NewGenericServiceLodestone(
	name string,
	lifecyclePolicy servicearch.LifecyclePolicy,
	serviceClient servicearch.ServiceClient,
	serviceIdentifier string,
	declaration *servicearch.ServiceDeclaration,
) *GenericServiceLodestone {
	if lifecyclePolicy == "" {
		lifecyclePolicy = servicearch.LifecyclePolicyManual
	}
	return &GenericServiceLodestone{
		name:              name,
		lifecyclePolicy:   lifecyclePolicy,
		serviceClient:     serviceClient,
		serviceIdentifier: serviceIdentifier,
		declaration:       declaration,
	}
}

func (l *GenericServiceLodestone) Name() string {
	if l == nil {
		return ""
	}
	return l.name
}

func (l *GenericServiceLodestone) LifecyclePolicy() servicearch.LifecyclePolicy {
	if l == nil {
		return servicearch.LifecyclePolicyManual
	}
	return l.lifecyclePolicy
}

func (l *GenericServiceLodestone) Legionary() servicearch.CollectiveServiceLegionary {
	if l == nil {
		return nil
	}
	return l.legionary
}

func (l *GenericServiceLodestone) SetLegionary(legionary servicearch.CollectiveServiceLegionary) {
	if l == nil {
		return
	}
	l.legionary = legionary
}

func (l *GenericServiceLodestone) SetPrepare(prepare func(context.Context) error) {
	if l == nil {
		return
	}
	l.prepare = prepare
}

func (l *GenericServiceLodestone) ServiceClient() servicearch.ServiceClient {
	if l == nil {
		return nil
	}
	return l.serviceClient
}

func (l *GenericServiceLodestone) ServiceIdentifier() string {
	if l == nil {
		return ""
	}
	return l.serviceIdentifier
}

func (l *GenericServiceLodestone) Declaration() *servicearch.ServiceDeclaration {
	if l == nil {
		return nil
	}
	return l.declaration
}

func (l *GenericServiceLodestone) Start(ctx context.Context) error {
	if l == nil {
		return nil
	}
	if l.prepare != nil {
		if err := l.prepare(ctx); err != nil {
			return err
		}
	}
	if l.legionary == nil {
		return nil
	}
	if err := l.legionary.StartService(ctx); err != nil {
		return err
	}
	_, err := l.legionary.JoinRegiment(ctx)
	return err
}

func (l *GenericServiceLodestone) Stop(ctx context.Context, reason string) error {
	if l == nil {
		return nil
	}
	if l.legionary != nil {
		err := l.legionary.Deregister(ctx, reason)
		l.legionary.TerminateService(ctx, reason)
		return err
	}
	if l.serviceClient != nil {
		return l.serviceClient.TerminateService(ctx)
	}
	return nil
}
