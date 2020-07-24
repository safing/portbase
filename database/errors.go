package database

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/safing/portbase/database/record"
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

// UnexpectedRecordTypeError is the common error of receiving an record of an unknown
// or unsupported type.
type UnexpectedRecordTypeError struct {
	Expected string
	Actual   string
}

func (urte *UnexpectedRecordTypeError) Error() string {
	return fmt.Sprintf("expected record of type %s but got %s", urte.Expected, urte.Actual)
}

// NewUnexpectedRecordTypeErr returns a new unexpected record type error.
func NewUnexpectedRecordTypeErr(expected string, r record.Record) *UnexpectedRecordTypeError {
	return &UnexpectedRecordTypeError{
		Expected: expected,
		Actual:   reflect.TypeOf(r).String(),
	}
}
