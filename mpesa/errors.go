package mpesa

import "fmt"

// ErrorKind identifies the category of error.
type ErrorKind string

const (
	ErrorKindValidation     ErrorKind = "validation_error"
	ErrorKindAuthentication ErrorKind = "authentication_error"
	ErrorKindNetwork        ErrorKind = "network_error"
	ErrorKindProvider       ErrorKind = "provider_error"
)

// Error represents an error returned by M-Pesa operations.
type Error struct {
	Kind       ErrorKind
	Message    string
	StatusCode int
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Kind, e.Message, e.Err)
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("[%s] %s (HTTP %d)", e.Kind, e.Message, e.StatusCode)
	}
	return fmt.Sprintf("[%s] %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}
