package arch

import "context"

type SystemHooks interface {
	Prepare(ctx context.Context) error
	Load(ctx context.Context) error
	Wire(ctx context.Context) error
	Mount(ctx context.Context) error
	AfterVitalize(ctx context.Context) error
	BeforeShutdown(ctx context.Context) error
}
