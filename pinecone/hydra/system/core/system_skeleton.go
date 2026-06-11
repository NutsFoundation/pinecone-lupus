package core

import (
	"context"

	"github.com/NutsFoundation/pinecone-lupus/pinecone/hydra/system/arch"
)

type GenericSystemSkeleton struct {
	system arch.Hydraco
	hooks  arch.SystemHooks
}

var _ arch.SystemSkeleton = (*GenericSystemSkeleton)(nil)

func NewGenericSystemSkeleton(system arch.Hydraco, hooks arch.SystemHooks) *GenericSystemSkeleton {
	return &GenericSystemSkeleton{
		system: system,
		hooks:  hooks,
	}
}

func (s *GenericSystemSkeleton) Name() string {
	if s == nil || s.system == nil {
		return ""
	}
	return s.system.Name()
}

func (s *GenericSystemSkeleton) Components() arch.ComponentManager {
	if s == nil || s.system == nil {
		return nil
	}
	return s.system.Components()
}

func (s *GenericSystemSkeleton) Vitalize(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.hooks != nil {
		if err := s.hooks.Prepare(ctx); err != nil {
			return err
		}
		if err := s.hooks.Load(ctx); err != nil {
			return err
		}
		if err := s.hooks.Wire(ctx); err != nil {
			return err
		}
	}
	if manager := s.Components(); manager != nil {
		if err := manager.VitalizeAll(ctx); err != nil {
			return err
		}
	}
	if s.hooks != nil {
		if err := s.hooks.Mount(ctx); err != nil {
			return err
		}
		if err := s.hooks.AfterVitalize(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *GenericSystemSkeleton) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	var firstErr error
	if s.hooks != nil {
		if err := s.hooks.BeforeShutdown(ctx); err != nil {
			firstErr = err
		}
	}
	if manager := s.Components(); manager != nil {
		if err := manager.ShutdownAll(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
