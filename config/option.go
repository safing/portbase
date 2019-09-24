package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"github.com/tidwall/sjson"

	"github.com/safing/portbase/database/record"
)

// Various attribute options. Use ExternalOptType for extended types in the frontend.
const (
	OptTypeString      uint8 = 1
	OptTypeStringArray uint8 = 2
	OptTypeInt         uint8 = 3
	OptTypeBool        uint8 = 4

	ExpertiseLevelUser      uint8 = 1
	ExpertiseLevelExpert    uint8 = 2
	ExpertiseLevelDeveloper uint8 = 3

	ReleaseLevelStable       = "stable"
	ReleaseLevelBeta         = "beta"
	ReleaseLevelExperimental = "experimental"
)

func getTypeName(t uint8) string {
	switch t {
	case OptTypeString:
		return "string"
	case OptTypeStringArray:
		return "[]string"
	case OptTypeInt:
		return "int"
	case OptTypeBool:
		return "bool"
	default:
		return "unknown"
	}
}

// Option describes a configuration option.
type Option struct {
	sync.Mutex

	Name        string
	Key         string // in path format: category/sub/key
	Description string

	ReleaseLevel   string
	ExpertiseLevel uint8
	OptType        uint8

	RequiresRestart bool
	DefaultValue    interface{}

	ExternalOptType string
	ValidationRegex string

	activeValue        interface{} // runtime value (loaded from config file or set by user)
	activeDefaultValue interface{} // runtime default value (may be set internally)
	compiledRegex      *regexp.Regexp
}

// Export expors an option to a Record.
func (option *Option) Export() (record.Record, error) {
	option.Lock()
	defer option.Unlock()

	data, err := json.Marshal(option)
	if err != nil {
		return nil, err
	}

	if option.activeValue != nil {
		data, err = sjson.SetBytes(data, "Value", option.activeValue)
		if err != nil {
			return nil, err
		}
	}

	if option.activeDefaultValue != nil {
		data, err = sjson.SetBytes(data, "DefaultValue", option.activeDefaultValue)
		if err != nil {
			return nil, err
		}
	}

	r, err := record.NewWrapper(fmt.Sprintf("config:%s", option.Key), nil, record.JSON, data)
	if err != nil {
		return nil, err
	}
	r.SetMeta(&record.Meta{})

	return r, nil
}

type sortableOptions []*Option

// Len is the number of elements in the collection.
func (opts sortableOptions) Len() int {
	return len(opts)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (opts sortableOptions) Less(i, j int) bool {
	return opts[i].Key < opts[j].Key
}

// Swap swaps the elements with indexes i and j.
func (opts sortableOptions) Swap(i, j int) {
	opts[i], opts[j] = opts[j], opts[i]
}
