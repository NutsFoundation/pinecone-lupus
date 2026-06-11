package arch

import "context"

type Morph interface {
	Name() string
	Run(ctx context.Context) error
	Shutdown(ctx context.Context) error
}

type Morpher interface {
	Morph(system System, ctx MorphContext) (Morph, error)
}
