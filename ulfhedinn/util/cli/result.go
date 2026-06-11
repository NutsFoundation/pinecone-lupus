package cli

type Result struct {
	ExitCode int
	Message  string
}

func Success(message string) Result {
	return Result{ExitCode: ExitSuccess, Message: message}
}

func Failure(exitCode int, message string) Result {
	if exitCode == ExitSuccess {
		exitCode = ExitGenericError
	}
	return Result{ExitCode: exitCode, Message: message}
}
