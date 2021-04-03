package config

import (
	"errors"
	"fmt"
	"sync"

	"github.com/tevino/abool"
)

var (
	// ErrInvalidJSON is returned by SetConfig and SetDefaultConfig if they receive invalid json.
	ErrInvalidJSON = errors.New("json string invalid")

	// ErrInvalidOptionType is returned by SetConfigOption and SetDefaultConfigOption if given an unsupported option type.
	ErrInvalidOptionType = errors.New("invalid option value type")

	validityFlag     = abool.NewBool(true)
	validityFlagLock sync.RWMutex
)

// getValidityFlag returns a flag that signifies if the configuration has been changed. This flag must not be changed, only read.
func getValidityFlag() *abool.AtomicBool {
	validityFlagLock.RLock()
	defer validityFlagLock.RUnlock()
	return validityFlag
}

// signalChanges marks the configs validtityFlag as dirty and eventually
// triggers a config change event.
func signalChanges() {
	// reset validity flag
	validityFlagLock.Lock()
	validityFlag.SetTo(false)
	validityFlag = abool.NewBool(true)
	validityFlagLock.Unlock()

	module.TriggerEvent(configChangeEvent, nil)
}

// replaceConfig sets the (prioritized) user defined config.
func replaceConfig(newValues map[string]interface{}) error {
	var firstErr error
	var errCnt int

	// RLock the options because we are not adding or removing
	// options from the registration but rather only update the
	// options value which is guarded by the option's lock itself
	optionsLock.RLock()
	defer optionsLock.RUnlock()

	for key, option := range options {
		newValue, ok := newValues[key]

		option.Lock()
		option.activeValue = nil
		if ok {
			valueCache, err := validateValue(option, newValue)
			if err == nil {
				option.activeValue = valueCache
			} else {
				errCnt++
				if firstErr == nil {
					firstErr = err
				}
			}
		}

		handleOptionUpdate(option, true)
		option.Unlock()
	}

	signalChanges()

	if firstErr != nil {
		if errCnt > 0 {
			return fmt.Errorf("encountered %d errors, first was: %s", errCnt, firstErr)
		}
		return firstErr
	}

	return nil
}

// replaceDefaultConfig sets the (fallback) default config.
func replaceDefaultConfig(newValues map[string]interface{}) error {
	var firstErr error
	var errCnt int

	// RLock the options because we are not adding or removing
	// options from the registration but rather only update the
	// options value which is guarded by the option's lock itself
	optionsLock.RLock()
	defer optionsLock.RUnlock()

	for key, option := range options {
		newValue, ok := newValues[key]

		option.Lock()
		option.activeDefaultValue = nil
		if ok {
			valueCache, err := validateValue(option, newValue)
			if err == nil {
				option.activeDefaultValue = valueCache
			} else {
				errCnt++
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		handleOptionUpdate(option, true)
		option.Unlock()
	}

	signalChanges()

	if firstErr != nil {
		if errCnt > 0 {
			return fmt.Errorf("encountered %d errors, first was: %s", errCnt, firstErr)
		}
		return firstErr
	}

	return nil
}

// SetConfigOption sets a single value in the (prioritized) user defined config.
func SetConfigOption(key string, value interface{}) error {
	return setConfigOption(key, value, true)
}

func setConfigOption(key string, value interface{}, push bool) (err error) {
	option, err := GetOption(key)
	if err != nil {
		return err
	}

	option.Lock()
	if value == nil {
		option.activeValue = nil
	} else {
		var valueCache *valueCache
		valueCache, err = validateValue(option, value)
		if err == nil {
			option.activeValue = valueCache
		}
	}

	handleOptionUpdate(option, push)
	option.Unlock()

	if err != nil {
		return err
	}

	// finalize change, activate triggers
	signalChanges()

	return saveConfig()
}

// SetDefaultConfigOption sets a single value in the (fallback) default config.
func SetDefaultConfigOption(key string, value interface{}) error {
	return setDefaultConfigOption(key, value, true)
}

func setDefaultConfigOption(key string, value interface{}, push bool) (err error) {
	option, err := GetOption(key)
	if err != nil {
		return err
	}

	option.Lock()
	if value == nil {
		option.activeDefaultValue = nil
	} else {
		var valueCache *valueCache
		valueCache, err = validateValue(option, value)
		if err == nil {
			option.activeDefaultValue = valueCache
		}
	}

	handleOptionUpdate(option, push)
	option.Unlock()

	if err != nil {
		return err
	}

	// finalize change, activate triggers
	signalChanges()

	// Do not save the configuration, as it only saves the active values, not the
	// active default value.
	return nil
}
