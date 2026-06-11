package arch

import "fmt"

type URIError struct {
	Message string
}

func (e URIError) Error() string {
	return e.Message
}

type EngineError struct {
	Operation string
	Bucket    string
	Key       string
	Cause     error
}

func (e EngineError) Error() string {
	if e.Key == "" {
		return fmt.Sprintf("%s red object %s: %v", e.Operation, e.Bucket, e.Cause)
	}
	return fmt.Sprintf("%s red object %s/%s: %v", e.Operation, e.Bucket, e.Key, e.Cause)
}

func (e EngineError) Unwrap() error {
	return e.Cause
}
