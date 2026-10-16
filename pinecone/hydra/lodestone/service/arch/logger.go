package arch

type Logger interface {
	Infof(format string, args ...any)
	Errorf(format string, args ...any)
}
