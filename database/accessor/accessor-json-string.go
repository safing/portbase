package accessor

import (
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// JSONAccessor is a json string with get functions.
type JSONAccessor struct {
	json *string
}

// NewJSONAccessor adds the Accessor interface to a JSON string.
func NewJSONAccessor(json *string) *JSONAccessor {
	return &JSONAccessor{
		json: json,
	}
}

// Set sets the value identified by key.
func (ja *JSONAccessor) Set(key string, value interface{}) error {
	result := gjson.Get(*ja.json, key)
	if result.Exists() {
		switch value.(type) {
		case string:
			if result.Type != gjson.String {
				return fmt.Errorf("tried to set field %s (%s) to a %T value", key, result.Type.String(), value)
			}
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			if result.Type != gjson.Number {
				return fmt.Errorf("tried to set field %s (%s) to a %T value", key, result.Type.String(), value)
			}
		case bool:
			if result.Type != gjson.True && result.Type != gjson.False {
				return fmt.Errorf("tried to set field %s (%s) to a %T value", key, result.Type.String(), value)
			}
		}
	}

	new, err := sjson.Set(*ja.json, key, value)
	if err != nil {
		return err
	}
	*ja.json = new
	return nil
}

// GetString returns the string found by the given json key and whether it could be successfully extracted.
func (ja *JSONAccessor) GetString(key string) (value string, ok bool) {
	result := gjson.Get(*ja.json, key)
	if !result.Exists() || result.Type != gjson.String {
		return emptyString, false
	}
	return result.String(), true
}

// GetInt returns the int found by the given json key and whether it could be successfully extracted.
func (ja *JSONAccessor) GetInt(key string) (value int64, ok bool) {
	result := gjson.Get(*ja.json, key)
	if !result.Exists() || result.Type != gjson.Number {
		return 0, false
	}
	return result.Int(), true
}

// GetFloat returns the float found by the given json key and whether it could be successfully extracted.
func (ja *JSONAccessor) GetFloat(key string) (value float64, ok bool) {
	result := gjson.Get(*ja.json, key)
	if !result.Exists() || result.Type != gjson.Number {
		return 0, false
	}
	return result.Float(), true
}

// GetBool returns the bool found by the given json key and whether it could be successfully extracted.
func (ja *JSONAccessor) GetBool(key string) (value bool, ok bool) {
	result := gjson.Get(*ja.json, key)
	switch {
	case !result.Exists():
		return false, false
	case result.Type == gjson.True:
		return true, true
	case result.Type == gjson.False:
		return false, true
	default:
		return false, false
	}
}

// Exists returns the whether the given key exists.
func (ja *JSONAccessor) Exists(key string) bool {
	result := gjson.Get(*ja.json, key)
	return result.Exists()
}

// Type returns the accessor type as a string.
func (ja *JSONAccessor) Type() string {
	return "JSONAccessor"
}
