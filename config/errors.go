package config

import (
	"errors"
	"fmt"
)

// Common error definitions.
var (
	ErrInvalidData     = errors.New("invalid data")
	ErrUnknownOption   = errors.New("unknown option")
	ErrUnsupportedType = errors.New("type not supported")

	// TODO(ppacher): the following error code are actually unused.

	// ErrInvalidJSON is returned by SetConfig and SetDefaultConfig if they receive invalid json.
	ErrInvalidJSON = errors.New("json string invalid")

	// ErrInvalidOptionType is returned by SetConfigOption and SetDefaultConfigOption if given an unsupported option type.
	ErrInvalidOptionType = errors.New("invalid option value type")
)

// InvalidOptionError describes an error encountered while
// registering a new option.
type InvalidOptionError struct {
	Msg string
	Err error
}

func (ioe *InvalidOptionError) Error() string {
	return fmt.Sprintf("failed to register option: %s", ioe.Msg)
}

func (ioe *InvalidOptionError) Unwrap() error {
	return ioe.Err
}

func newInvalidOptionError(msg string, err error) *InvalidOptionError {
	return &InvalidOptionError{
		Msg: msg,
		Err: err,
	}
}

// InvalidValueError describes a validation error for the options
// value.
type InvalidValueError struct {
	Option string
	Value  interface{}
	Msg    string
}

func (ive *InvalidValueError) Error() string {
	msg := fmt.Sprintf("%s: invalid value %+v", ive.Option, ive.Value)
	if ive.Msg != "" {
		msg += ": " + ive.Msg
	}
	return msg
}

func newInvalidValueError(option string, value interface{}, msg string) *InvalidValueError {
	return &InvalidValueError{
		Option: option,
		Value:  value,
		Msg:    msg,
	}
}
