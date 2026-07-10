package entity

type MushroomConfig struct {
	Runtime      RuntimeConfig
	MaxBodyBytes int64
}

type RuntimeConfig struct {
	Enable               bool
	Host                 string
	Port                 int
	ToolchainName        string
	CGIExecutable        string
	LazyStart            bool
	StartupTimeoutMillis int64
	RequestTimeoutMillis int64
	MaxRequestBodyBytes  int64
	MaxResponseBodyBytes int64
}
