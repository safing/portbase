package varint

import "errors"

// Common errors.
var (
	errEmptyBuf = errors.New("buffer empty")
	errTooSmall = errors.New("buffer too small")
)

type valueExceededError struct {
	max string
}

func (e *valueExceededError) Error() string {
	return "varint: encoded integer greater than " + e.max
}
