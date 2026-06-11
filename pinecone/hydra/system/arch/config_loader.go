package arch

type ConfigLoader[C any] interface {
	Load(path string) (C, error)
}
