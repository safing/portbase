package config

import (
	"github.com/safing/portbase/log"
)

type (
	// StringOption defines the returned function by GetAsString.
	StringOption func() string
	// StringArrayOption defines the returned function by GetAsStringArray.
	StringArrayOption func() []string
	// IntOption defines the returned function by GetAsInt.
	IntOption func() int64
	// BoolOption defines the returned function by GetAsBool.
	BoolOption func() bool
)

// GetAsString returns a function that returns the wanted string with high performance.
func GetAsString(name string, fallback string) StringOption {
	valid := getValidityFlag()
	value := findStringValue(name, fallback)
	return func() string {
		if !valid.IsSet() {
			valid = getValidityFlag()
			value = findStringValue(name, fallback)
		}
		return value
	}
}

// GetAsStringArray returns a function that returns the wanted string with high performance.
func GetAsStringArray(name string, fallback []string) StringArrayOption {
	valid := getValidityFlag()
	value := findStringArrayValue(name, fallback)
	return func() []string {
		if !valid.IsSet() {
			valid = getValidityFlag()
			value = findStringArrayValue(name, fallback)
		}
		return value
	}
}

// GetAsInt returns a function that returns the wanted int with high performance.
func GetAsInt(name string, fallback int64) IntOption {
	valid := getValidityFlag()
	value := findIntValue(name, fallback)
	return func() int64 {
		if !valid.IsSet() {
			valid = getValidityFlag()
			value = findIntValue(name, fallback)
		}
		return value
	}
}

// GetAsBool returns a function that returns the wanted int with high performance.
func GetAsBool(name string, fallback bool) BoolOption {
	valid := getValidityFlag()
	value := findBoolValue(name, fallback)
	return func() bool {
		if !valid.IsSet() {
			valid = getValidityFlag()
			value = findBoolValue(name, fallback)
		}
		return value
	}
}

// findValue find the correct value in the user or default config.
func findValue(key string) interface{} {
	optionsLock.RLock()
	option, ok := options[key]
	optionsLock.RUnlock()
	if !ok {
		log.Errorf("config: request for unregistered option: %s", key)
		return nil
	}

	// lock option
	option.Lock()
	defer option.Unlock()

	if option.ReleaseLevel <= getReleaseLevel() && option.activeValue != nil {
		return option.activeValue
	}

	if option.activeDefaultValue != nil {
		return option.activeDefaultValue
	}

	return option.DefaultValue
}

// findStringValue validates and returns the value with the given key.
func findStringValue(key string, fallback string) (value string) {
	result := findValue(key)
	if result == nil {
		return fallback
	}
	v, ok := result.(string)
	if ok {
		return v
	}
	return fallback
}

// findStringArrayValue validates and returns the value with the given key.
func findStringArrayValue(key string, fallback []string) (value []string) {
	result := findValue(key)
	if result == nil {
		return fallback
	}

	v, ok := result.([]interface{})
	if ok {
		new := make([]string, len(v))
		for i, val := range v {
			s, ok := val.(string)
			if ok {
				new[i] = s
			} else {
				return fallback
			}
		}
		return new
	}

	return fallback
}

// findIntValue validates and returns the value with the given key.
func findIntValue(key string, fallback int64) (value int64) {
	result := findValue(key)
	if result == nil {
		return fallback
	}
	switch v := result.(type) {
	case int:
		return int64(v)
	case int8:
		return int64(v)
	case int16:
		return int64(v)
	case int32:
		return int64(v)
	case int64:
		return v
	case uint:
		return int64(v)
	case uint8:
		return int64(v)
	case uint16:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		return int64(v)
	case float32:
		return int64(v)
	case float64:
		return int64(v)
	}
	return fallback
}

// findBoolValue validates and returns the value with the given key.
func findBoolValue(key string, fallback bool) (value bool) {
	result := findValue(key)
	if result == nil {
		return fallback
	}
	v, ok := result.(bool)
	if ok {
		return v
	}
	return fallback
}
