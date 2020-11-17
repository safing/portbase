package osdetail

import "errors"

var (
	ErrNotSupported = errors.New("not supported")
	ErrNotFound     = errors.New("not found")
	ErrEmptyOutput  = errors.New("command succeeded with empty output")
)
