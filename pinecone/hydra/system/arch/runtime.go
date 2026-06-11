package arch

type Runtime interface {
	RootPath() string
	SetupPath() string
	LogsPath() string
	TempPath() string
}
