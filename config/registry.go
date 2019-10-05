package config

import (
	"fmt"
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
		return fmt.Errorf("failed to register option: please set option.Name")
	}
	if option.Key == "" {
		return fmt.Errorf("failed to register option: please set option.Key")
	}
	if option.Description == "" {
		return fmt.Errorf("failed to register option: please set option.Description")
	}
	if option.OptType == 0 {
		return fmt.Errorf("failed to register option: please set option.OptType")
	}

	if option.ValidationRegex != "" {
		var err error
		option.compiledRegex, err = regexp.Compile(option.ValidationRegex)
		if err != nil {
			return fmt.Errorf("config: could not compile option.ValidationRegex: %s", err)
		}
	}

	optionsLock.Lock()
	defer optionsLock.Unlock()
	options[option.Key] = option

	return nil
}
