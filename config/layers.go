package config

import (
	"errors"
	"sync"
	"fmt"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var (
	configLock sync.RWMutex

	userConfig    = ""
	defaultConfig = ""

	// ErrInvalidJSON is returned by SetConfig and SetDefaultConfig if they receive invalid json.
	ErrInvalidJSON = errors.New("json string invalid")

	// ErrInvalidOptionType is returned by SetConfigOption and SetDefaultConfigOption if given an unsupported option type.
	ErrInvalidOptionType = errors.New("invalid option value type")
)

// SetConfig sets the (prioritized) user defined config.
func SetConfig(json string) error {
	if !gjson.Valid(json) {
		return ErrInvalidJSON
	}

	configLock.Lock()
	defer configLock.Unlock()
	userConfig = json
	resetValidityFlag()

	return nil
}

// SetDefaultConfig sets the (fallback) default config.
func SetDefaultConfig(json string) error {
	if !gjson.Valid(json) {
		return ErrInvalidJSON
	}

	configLock.Lock()
	defer configLock.Unlock()
	defaultConfig = json
	resetValidityFlag()

	return nil
}

func validateValue(name string, value interface{}) error {
	optionsLock.RLock()
	defer optionsLock.RUnlock()

	option, ok := options[name]
	if !ok {
		switch value.(type) {
		case string:
			return nil
		case []string:
			return nil
		case int:
			return nil
		case bool:
			return nil
		default:
			return ErrInvalidOptionType
		}
	}

	switch v := value.(type) {
	case string:
		if option.OptType != OptTypeString {
			return fmt.Errorf("expected type string for option %s, got type %T", name, v)
		}
		if option.compiledRegex != nil {
			if !option.compiledRegex.MatchString(v) {
				return fmt.Errorf("validation failed: string \"%s\" did not match regex for option %s", v, name)
			}
		}
		return nil
	case []string:
		if option.OptType != OptTypeStringArray {
			return fmt.Errorf("expected type string for option %s, got type %T", name, v)
		}
		if option.compiledRegex != nil {
			for pos, entry := range v {
				if !option.compiledRegex.MatchString(entry) {
					return fmt.Errorf("validation failed: string \"%s\" at index %d did not match regex for option %s", entry, pos, name)
				}
			}
		}
		return nil
	case int:
		if option.OptType != OptTypeInt {
			return fmt.Errorf("expected type int for option %s, got type %T", name, v)
		}
		return nil
	case bool:
		if option.OptType != OptTypeBool {
			return fmt.Errorf("expected type bool for option %s, got type %T", name, v)
		}
		return nil
	default:
		return ErrInvalidOptionType
	}
}

// SetConfigOption sets a single value in the (prioritized) user defined config.
func SetConfigOption(name string, value interface{}) error {
	configLock.Lock()
	defer configLock.Unlock()

	var err error
	var newConfig string

	if value == nil {
		newConfig, err = sjson.Delete(userConfig, name)
	} else {
		err = validateValue(name, value)
		if err == nil {
			newConfig, err = sjson.Set(userConfig, name, value)
		}
	}

	if err == nil {
		userConfig = newConfig
		resetValidityFlag()
	}

	return err
}

// SetDefaultConfigOption sets a single value in the (fallback) default config.
func SetDefaultConfigOption(name string, value interface{}) error {
	configLock.Lock()
	defer configLock.Unlock()

	var err error
	var newConfig string

	if value == nil {
		newConfig, err = sjson.Delete(defaultConfig, name)
	} else {
		err = validateValue(name, value)
		if err == nil {
			newConfig, err = sjson.Set(defaultConfig, name, value)
		}
	}

	if err == nil {
		defaultConfig = newConfig
		resetValidityFlag()
	}

	return err
}

// findValue find the correct value in the user or default config.
func findValue(name string) (result gjson.Result) {
	configLock.RLock()
	defer configLock.RUnlock()

	result = gjson.Get(userConfig, name)
	if !result.Exists() {
		result = gjson.Get(defaultConfig, name)
	}
	return result
}

// findStringValue validates and returns the value with the given name.
func findStringValue(name string, fallback string) (value string) {
	result := findValue(name)
	if !result.Exists() {
		return fallback
	}
	if result.Type != gjson.String {
		return fallback
	}
	return result.String()
}

// findStringArrayValue validates and returns the value with the given name.
func findStringArrayValue(name string, fallback []string) (value []string) {
	result := findValue(name)
	if !result.Exists() {
		return fallback
	}
	if !result.IsArray() {
		return fallback
	}
	results := result.Array()
	for _, r := range results {
		if r.Type != gjson.String {
			return fallback
		}
		value = append(value, r.String())
	}
	return value
}

// findIntValue validates and returns the value with the given name.
func findIntValue(name string, fallback int64) (value int64) {
	result := findValue(name)
	if !result.Exists() {
		return fallback
	}
	if result.Type != gjson.Number {
		return fallback
	}
	return result.Int()
}

// findBoolValue validates and returns the value with the given name.
func findBoolValue(name string, fallback bool) (value bool) {
	result := findValue(name)
	if !result.Exists() {
		return fallback
	}
	switch result.Type {
	case gjson.True:
		return true
	case gjson.False:
		return false
	default:
		return fallback
	}
}
