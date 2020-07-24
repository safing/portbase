package config

import (
	"regexp"
	"sync"
)

var (
	optionsLock sync.RWMutex
	options     = make(map[string]*Option)
)

// Register registers a new configuration option.
func Register(option *Option) error {
	if option.Name == "" {
		return newInvalidOptionError("missing .Name", nil)
	}
	if option.Key == "" {
		return newInvalidOptionError("missing .Key", nil)
	}
	if option.Description == "" {
		return newInvalidOptionError("missing .Description", nil)
	}
	if option.OptType == 0 {
		return newInvalidOptionError("missing .OptType", nil)
	}

	var err error

	if option.ValidationRegex != "" {
		option.compiledRegex, err = regexp.Compile(option.ValidationRegex)
		if err != nil {
			return newInvalidOptionError("config: could not compile option.ValidationRegex", err)
		}
	}

	option.activeFallbackValue, err = validateValue(option, option.DefaultValue)
	if err != nil {
		return newInvalidOptionError("config: invalid default value: %s", err)
	}

	optionsLock.Lock()
	defer optionsLock.Unlock()
	options[option.Key] = option

	return nil
}
