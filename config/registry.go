package config

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
)

var (
	optionsLock sync.RWMutex
	options     = make(map[string]*Option)
)

// ForEachOption calls fn for each defined option. If fn returns
// and error the iteration is stopped and the error is returned.
// Note that ForEachOption does not guarantee a stable order of
// iteration between multiple calles. ForEachOption does NOT lock
// opt when calling fn.
func ForEachOption(fn func(opt *Option) error) error {
	optionsLock.Lock()
	defer optionsLock.Unlock()

	for _, opt := range options {
		if err := fn(opt); err != nil {
			return err
		}
	}
	return nil
}

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

	if option.ValidationRegex == "" && option.PossibleValues != nil {
		values := make([]string, len(option.PossibleValues))
		for idx, val := range option.PossibleValues {
			values[idx] = fmt.Sprintf("%v", val.Value)
		}
		option.ValidationRegex = fmt.Sprintf("^(%s)$", strings.Join(values, "|"))
	}

	var err error
	if option.ValidationRegex != "" {
		option.compiledRegex, err = regexp.Compile(option.ValidationRegex)
		if err != nil {
			return fmt.Errorf("config: could not compile option.ValidationRegex: %s", err)
		}
	}

	option.activeFallbackValue, err = validateValue(option, option.DefaultValue)
	if err != nil {
		return fmt.Errorf("config: invalid default value: %s", err)
	}

	optionsLock.Lock()
	defer optionsLock.Unlock()
	options[option.Key] = option

	return nil
}
