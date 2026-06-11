package arch

type System interface {
	Name() string
	Components() ComponentManager
}
