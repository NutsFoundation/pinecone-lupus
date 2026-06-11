package cli

const (
	ExitSuccess        = 0
	ExitGenericError   = 1
	ExitArgumentError  = 2
	ExitUnsupported    = 3
	ExitNetworkFailure = 4
	ExitTLSFailure     = 5
	ExitResolveFailure = 6
	ExitConnectFailure = 7
	ExitHTTPFailure    = 8
	ExitWriteFailure   = 23
	ExitTimeout        = 28
)
