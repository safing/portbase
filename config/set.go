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

func signalChanges() {
	// refetch and save release level and expertise level
	updateReleaseLevel()
	updateExpertiseLevel()

	// reset validity flag
	validityFlagLock.Lock()
	validityFlag.SetTo(false)
	validityFlag = abool.NewBool(true)
	validityFlagLock.Unlock()

	module.TriggerEvent(configChangeEvent, nil)
}

// setConfig sets the (prioritized) user defined config.
func setConfig(newValues map[string]interface{}) error {
	var firstErr error
	var errCnt int

	optionsLock.Lock()
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
		option.Unlock()
	}
	optionsLock.Unlock()

	signalChanges()
	go pushFullUpdate()

	if firstErr != nil {
		if errCnt > 0 {
			return fmt.Errorf("encountered %d errors, first was: %s", errCnt, firstErr)
		}
		return firstErr
	}

	return nil
}

// SetDefaultConfig sets the (fallback) default config.
func SetDefaultConfig(newValues map[string]interface{}) error {
	var firstErr error
	var errCnt int

	optionsLock.Lock()
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
		option.Unlock()
	}
	optionsLock.Unlock()

	signalChanges()
	go pushFullUpdate()

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
	optionsLock.Lock()
	option, ok := options[key]
	optionsLock.Unlock()
	if !ok {
		return fmt.Errorf("config option %s does not exist", key)
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
	option.Unlock()
	if err != nil {
		return err
	}

	// finalize change, activate triggers
	signalChanges()
	if push {
		go pushUpdate(option)
	}
	return saveConfig()
}

// SetDefaultConfigOption sets a single value in the (fallback) default config.
func SetDefaultConfigOption(key string, value interface{}) error {
	return setDefaultConfigOption(key, value, true)
}

func setDefaultConfigOption(key string, value interface{}, push bool) (err error) {
	optionsLock.Lock()
	option, ok := options[key]
	optionsLock.Unlock()
	if !ok {
		return fmt.Errorf("config option %s does not exist", key)
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
	option.Unlock()
	if err != nil {
		return err
	}

	// finalize change, activate triggers
	signalChanges()
	if push {
		go pushUpdate(option)
	}
	return saveConfig()
}
