package config

import (
	"errors"
	"sync"
	"fmt"

	"github.com/Safing/portbase/log"
)

var (
	configLock sync.RWMutex

	userConfig    = make(map[string]interface{})
	defaultConfig = make(map[string]interface{})

	// ErrInvalidJSON is returned by SetConfig and SetDefaultConfig if they receive invalid json.
	ErrInvalidJSON = errors.New("json string invalid")

	// ErrInvalidOptionType is returned by SetConfigOption and SetDefaultConfigOption if given an unsupported option type.
	ErrInvalidOptionType = errors.New("invalid option value type")

	changedSignal = make(chan struct{}, 0)
)

// Changed signals if any config option was changed.
func Changed() <-chan struct{} {
	configLock.RLock()
	defer configLock.RUnlock()
	return changedSignal
}

// triggerChange signals listeners that a config option was changed.
func triggerChange() {
	// must be locked!
	close(changedSignal)
	changedSignal = make(chan struct{}, 0)
}

// setConfig sets the (prioritized) user defined config.
func setConfig(m map[string]interface{}) error {
	configLock.Lock()
	defer configLock.Unlock()
	userConfig = m
	resetValidityFlag()

	go pushFullUpdate()
	triggerChange()

	return nil
}

// SetDefaultConfig sets the (fallback) default config.
func SetDefaultConfig(m map[string]interface{}) error {
	configLock.Lock()
	defer configLock.Unlock()
	defaultConfig = m
	resetValidityFlag()

	go pushFullUpdate()
	triggerChange()

	return nil
}

func validateValue(name string, value interface{}) (*Option, error) {
	optionsLock.RLock()
	defer optionsLock.RUnlock()

	option, ok := options[name]
	if !ok {
		return nil, errors.New("config option does not exist")
	}

	switch v := value.(type) {
	case string:
		if option.OptType != OptTypeString {
			return nil, fmt.Errorf("expected type %s for option %s, got type %T", getTypeName(option.OptType), name, v)
		}
		if option.compiledRegex != nil {
			if !option.compiledRegex.MatchString(v) {
				return nil, fmt.Errorf("validation failed: string \"%s\" did not match regex for option %s", v, name)
			}
		}
		return option, nil
	case []string:
		if option.OptType != OptTypeStringArray {
			return nil, fmt.Errorf("expected type %s for option %s, got type %T", getTypeName(option.OptType), name, v)
		}
		if option.compiledRegex != nil {
			for pos, entry := range v {
				if !option.compiledRegex.MatchString(entry) {
					return nil, fmt.Errorf("validation failed: string \"%s\" at index %d did not match regex for option %s", entry, pos, name)
				}
			}
		}
		return option, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		if option.OptType != OptTypeInt {
			return nil, fmt.Errorf("expected type %s for option %s, got type %T", getTypeName(option.OptType), name, v)
		}
		if option.compiledRegex != nil {
			if !option.compiledRegex.MatchString(fmt.Sprintf("%d", v)) {
				return nil, fmt.Errorf("validation failed: number \"%d\" did not match regex for option %s", v, name)
			}
		}
		return option, nil
	case bool:
		if option.OptType != OptTypeBool {
			return nil, fmt.Errorf("expected type %s for option %s, got type %T", getTypeName(option.OptType), name, v)
		}
		return option, nil
	default:
		return nil, fmt.Errorf("invalid option value type: %T", value)
	}
}

// SetConfigOption sets a single value in the (prioritized) user defined config.
func SetConfigOption(name string, value interface{}) error {
	return setConfigOption(name, value, true)
}

func setConfigOption(name string, value interface{}, push bool) error {
	configLock.Lock()
	defer configLock.Unlock()

	var err error

	if value == nil {
		delete(userConfig, name)
	} else {
		var option *Option
		option, err = validateValue(name, value)
		if err == nil {
			userConfig[name] = value
			if push {
				go pushUpdate(option)
			}
		}
	}

	if err == nil {
		resetValidityFlag()
		go saveConfig()
		triggerChange()
	}

	return err
}

// SetDefaultConfigOption sets a single value in the (fallback) default config.
func SetDefaultConfigOption(name string, value interface{}) error {
	return setDefaultConfigOption(name, value, true)
}

func setDefaultConfigOption(name string, value interface{}, push bool) error {
	configLock.Lock()
	defer configLock.Unlock()

	var err error

	if value == nil {
		delete(defaultConfig, name)
	} else {
		var option *Option
		option, err = validateValue(name, value)
		if err == nil {
			defaultConfig[name] = value
			if push {
				go pushUpdate(option)
			}
		}
	}

	if err == nil {
		resetValidityFlag()
		triggerChange()
	}

	return err
}

// findValue find the correct value in the user or default config.
func findValue(name string) (result interface{}) {
	configLock.RLock()
	defer configLock.RUnlock()

	result, ok := userConfig[name]
	if ok {
		return
	}

	result, ok = defaultConfig[name]
	if ok {
		return
	}

	optionsLock.RLock()
	defer optionsLock.RUnlock()

	option, ok := options[name]
	if ok {
		return option.DefaultValue
	}

	log.Errorf("config: request for unregistered option: %s", name)
	return nil
}

// findStringValue validates and returns the value with the given name.
func findStringValue(name string, fallback string) (value string) {
	result := findValue(name)
	if result == nil {
		return fallback
	}
	v, ok := result.(string)
	if ok {
		return v
	}
	return fallback
}

// findStringArrayValue validates and returns the value with the given name.
func findStringArrayValue(name string, fallback []string) (value []string) {
	result := findValue(name)
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

// findIntValue validates and returns the value with the given name.
func findIntValue(name string, fallback int64) (value int64) {
	result := findValue(name)
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

// findBoolValue validates and returns the value with the given name.
func findBoolValue(name string, fallback bool) (value bool) {
	result := findValue(name)
	if result == nil {
		return fallback
	}
	v, ok := result.(bool)
	if ok {
		return v
	}
	return fallback
}
