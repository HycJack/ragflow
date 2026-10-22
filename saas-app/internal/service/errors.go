package service

import "fmt"

// ValidationError marks an error caused by caller-supplied input rather than a
// server fault. Handlers map it to HTTP 400 and surface the message verbatim,
// which keeps user-facing text free of internal prefixes.
type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string { return e.Message }

// invalid builds a ValidationError.
func invalid(format string, args ...any) error {
	return ValidationError{Message: fmt.Sprintf(format, args...)}
}
