package storage

import "errors"

// Errors for storages.
var (
	ErrNotFound       = errors.New("storage entry not found")
	ErrQueryTimeout   = errors.New("query timeout")
	ErrInvalidKey     = errors.New("invalid key")
	ErrNotImplemented = errors.New("not implemented")
	ErrUnknownType    = errors.New("unknown storage type")
	ErrExists         = errors.New("factory already registered")
)
