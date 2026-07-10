package arch

type ExecutableSpec struct {
	ExecutablePath string
	InstallPath    string
	WorkPath       string
}

type ExecutableResolver interface {
	ResolvePHPFastCGI(toolchainName string) (*ExecutableSpec, error)
}
