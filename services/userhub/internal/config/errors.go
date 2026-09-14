package config

import "fmt"

type configError struct {
	Msg string
	Err error
}

func (e *configError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("config: %s - %v", e.Msg, e.Err)
	}

	return fmt.Sprintf("config: %s", e.Msg)
}

func (e *configError) Unwrap() error {
	return e.Err
}

func newError(msg string) *configError {
	return &configError{Msg: msg, Err: nil}
}

func wrapError(err error, msg string) *configError {
	return &configError{Msg: msg, Err: err}
}
