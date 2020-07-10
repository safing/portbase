package accessor

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/tidwall/gjson"
)

// Common error defintions.
var (
	ErrUnknownField = errors.New("struct field does not exist")
	ErrImmutable    = errors.New("field or struct is immutable")
)

// InvalidValueTypeError describes an error when trying to set a value
// of an invalid type to a field.
type InvalidValueTypeError struct {
	FieldName string
	FieldKind string
	ValueKind string
}

func (ivte *InvalidValueTypeError) Error() string {
	return fmt.Sprintf("tried to set field %s (%s) to a %s value", ivte.FieldName, ivte.FieldKind, ivte.ValueKind)
}

func newInvalidValueTypeError(key string, field, val reflect.Value) *InvalidValueTypeError {
	return &InvalidValueTypeError{
		FieldName: key,
		FieldKind: field.Kind().String(),
		ValueKind: val.Kind().String(),
	}
}

func newInvalidJSONValueTypeError(key string, field gjson.Result, value interface{}) *InvalidValueTypeError {
	return &InvalidValueTypeError{
		FieldName: key,
		FieldKind: field.Type.String(),
		ValueKind: reflect.ValueOf(value).Kind().String(),
	}
}

// OverflowError describes an error when value would overflow the type of a field.
type OverflowError struct {
	FieldName string
	FieldKind string
	Value     interface{}
}

func (oe *OverflowError) Error() string {
	return fmt.Sprintf("setting field %s (%s) to %d would overflow", oe.FieldName, oe.FieldKind, oe.Value)
}

func newOverflowError(key string, field reflect.Value, value interface{}) *OverflowError {
	return &OverflowError{
		FieldName: key,
		FieldKind: field.Kind().String(),
		Value:     value,
	}
}
