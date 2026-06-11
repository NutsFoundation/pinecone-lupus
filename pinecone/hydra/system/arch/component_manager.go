package arch

import "context"

type ComponentManager interface {
	Register(component Component) error
	Get(name string) (Component, bool)
	List() []Component
	VitalizeAll(ctx context.Context) error
	ShutdownAll(ctx context.Context) error
}
