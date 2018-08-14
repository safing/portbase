package config

import (
	"errors"
	"fmt"
	"regexp"
	"sync"
)

// Variable Type IDs for frontend Identification. Values over 100 are free for custom use.
const (
	OptTypeString      uint8 = 1
	OptTypeStringArray uint8 = 2
	OptTypeInt         uint8 = 3
	OptTypeBool        uint8 = 4

	ExpertiseLevelUser      int8 = 1
	ExpertiseLevelExpert    int8 = 2
	ExpertiseLevelDeveloper int8 = 3
)

var (
	optionsLock sync.RWMutex
	options     = make(map[string]*Option)

	// ErrIncompleteCall is return when RegisterOption is called with empty mandatory values.
	ErrIncompleteCall = errors.New("could not register config option: all fields, except for the validationRegex are mandatory")
)

// Option describes a configuration option.
type Option struct {
	Name            string
	Key             string
	Description     string
	ExpertiseLevel  uint8
	OptType         uint8
	DefaultValue    interface{}
	ValidationRegex string
	compiledRegex   *regexp.Regexp
}

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
