package arch

import "context"

type Component interface {
	Name() string
	Vitalize(ctx context.Context) error
	Shutdown(ctx context.Context) error
}
