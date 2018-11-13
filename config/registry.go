package config

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
)

var (
	optionsLock sync.RWMutex
	options     = make(map[string]*Option)

	// ErrIncompleteCall is return when RegisterOption is called with empty mandatory values.
	ErrIncompleteCall = errors.New("could not register config option: all fields, except for the validationRegex are mandatory")
)

// Register registers a new configuration option.
func Register(option *Option) error {

	if option.Name == "" ||
		option.Key == "" ||
		option.Description == "" ||
		option.ExpertiseLevel == 0 ||
		option.OptType == 0 {
		return ErrIncompleteCall
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
