package config

import (
	"sync"

	"github.com/tevino/abool"
)

var (
	validityFlag     = abool.NewBool(true)
	validityFlagLock sync.RWMutex

	tableLock sync.RWMutex

	stringTable map[string]string
	intTable    map[string]int
	boolTable   map[string]bool
)

func getValidityFlag() *abool.AtomicBool {
	validityFlagLock.RLock()
	defer validityFlagLock.RUnlock()
	return validityFlag
}

func resetValidityFlag() {
	validityFlagLock.Lock()
	defer validityFlagLock.Unlock()
	validityFlag.SetTo(false)
	validityFlag = abool.NewBool(true)
}

// GetAsString returns a function that returns the wanted string with high performance.
func GetAsString(name string, fallback string) func() string {
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
func GetAsStringArray(name string, fallback []string) func() []string {
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
func GetAsInt(name string, fallback int64) func() int64 {
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
func GetAsBool(name string, fallback bool) func() bool {
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
