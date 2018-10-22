package config

import "sync"

type safe struct{}

var (
	// Concurrent makes concurrency safe get methods available.
	Concurrent = &safe{}
)

// GetAsString returns a function that returns the wanted string with high performance.
func (cs *safe) GetAsString(name string, fallback string) StringOption {
	valid := getValidityFlag()
	value := findStringValue(name, fallback)
	var lock sync.Mutex
	return func() string {
		lock.Lock()
		defer lock.Unlock()
		if !valid.IsSet() {
			valid = getValidityFlag()
			value = findStringValue(name, fallback)
		}
		return value
	}
}

// GetAsStringArray returns a function that returns the wanted string with high performance.
func (cs *safe) GetAsStringArray(name string, fallback []string) StringArrayOption {
	valid := getValidityFlag()
	value := findStringArrayValue(name, fallback)
	var lock sync.Mutex
	return func() []string {
		lock.Lock()
		defer lock.Unlock()
		if !valid.IsSet() {
			valid = getValidityFlag()
			value = findStringArrayValue(name, fallback)
		}
		return value
	}
}

// GetAsInt returns a function that returns the wanted int with high performance.
func (cs *safe) GetAsInt(name string, fallback int64) IntOption {
	valid := getValidityFlag()
	value := findIntValue(name, fallback)
	var lock sync.Mutex
	return func() int64 {
		lock.Lock()
		defer lock.Unlock()
		if !valid.IsSet() {
			valid = getValidityFlag()
			value = findIntValue(name, fallback)
		}
		return value
	}
}

// GetAsBool returns a function that returns the wanted int with high performance.
func (cs *safe) GetAsBool(name string, fallback bool) BoolOption {
	valid := getValidityFlag()
	value := findBoolValue(name, fallback)
	var lock sync.Mutex
	return func() bool {
		lock.Lock()
		defer lock.Unlock()
		if !valid.IsSet() {
			valid = getValidityFlag()
			value = findBoolValue(name, fallback)
		}
		return value
	}
}
