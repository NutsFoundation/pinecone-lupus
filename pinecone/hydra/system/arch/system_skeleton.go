package arch

import "context"

type SystemSkeleton interface {
	Hydraco
	Vitalize(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
