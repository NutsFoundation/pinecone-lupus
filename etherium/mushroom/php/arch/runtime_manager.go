package arch

import "context"

type RuntimeManager interface {
	EnsureReady(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
