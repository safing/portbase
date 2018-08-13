package config

import (
	"errors"
	"sync"

	"github.com/tidwall/gjson"
)

var (
	configLock sync.RWMutex

	userConfig    = ""
	defaultConfig = ""

	// ErrInvalidJSON is returned by SetConfig and SetDefaultConfig if they receive invalid json.
	ErrInvalidJSON = errors.New("json string invalid")
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

// findValue find the correct value in the user or default config
func findValue(name string) (result gjson.Result) {
	configLock.RLock()
	defer configLock.RUnlock()

	result = gjson.Get(userConfig, name)
	if !result.Exists() {
		result = gjson.Get(defaultConfig, name)
	}
	return result
}

// findStringValue validates and return the value with the given name
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

// findIntValue validates and return the value with the given name
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
