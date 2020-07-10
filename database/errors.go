package database

import (
	"errors"
)

// Errors.
var (
	ErrNotFound               = errors.New("database entry not found")
	ErrPermissionDenied       = errors.New("access to database record denied")
	ErrReadOnly               = errors.New("database is read only")
	ErrShuttingDown           = errors.New("database system is shutting down")
	ErrNotImplemented         = errors.New("not implemented")
	ErrNotInitialized         = errors.New("database not initialized")
	ErrInitialized            = errors.New("database already initialized")
	ErrLoaded                 = errors.New("database already loaded")
	ErrNotRegistered          = errors.New("database not registered")
	ErrInvalidStorageType     = errors.New("invalid database storage type")
	ErrMalformedWrappedRecord = errors.New("record is malformed (reports to be wrapped but is not of type *record.Wrapper)")
	ErrTimeout                = errors.New("timeout")
	ErrBatchClosed            = errors.New("batch already closed")
	ErrInvalidScope           = errors.New("invalid database scope")
	ErrInvalidName            = errors.New("database name must only contain alphanumeric and `_-` characters and must be at least 4 characters long")
)
