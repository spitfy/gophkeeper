package user

import "errors"

var (
	ErrNotFound     = errors.New("user not found")
	ErrInvalidAuth  = errors.New("invalid credentials")
	ErrInvalidInput = errors.New("invalid input")
)

type DomainError struct {
	Err     error
	Message string
	Code    string
}

func (e *DomainError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

func (e *DomainError) Unwrap() error {
	return e.Err
}
