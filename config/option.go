package config

import (
	"fmt"
	"regexp"
	"encoding/json"

	"github.com/tidwall/sjson"

	"github.com/safing/portbase/database/record"
)

// Variable Type IDs for frontend Identification. Use ExternalOptType for extended types in the frontend.
const (
	OptTypeString      uint8 = 1
	OptTypeStringArray uint8 = 2
	OptTypeInt         uint8 = 3
	OptTypeBool        uint8 = 4

	ExpertiseLevelUser      uint8 = 1
	ExpertiseLevelExpert    uint8 = 2
	ExpertiseLevelDeveloper uint8 = 3
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
	Name            string
	Key             string // category/sub/key
	Description     string
	ExpertiseLevel  uint8
	OptType         uint8
	DefaultValue    interface{}
	ExternalOptType string
	ValidationRegex string
	compiledRegex   *regexp.Regexp
}

// Export expors an option to a Record.
func (opt *Option) Export() (record.Record, error) {
	data, err := json.Marshal(opt)
	if err != nil {
		return nil, err
	}

	configLock.RLock()
	defer configLock.RUnlock()

	userValue, ok := userConfig[opt.Key]
	if ok {
		data, err = sjson.SetBytes(data, "Value", userValue)
		if err != nil {
			return nil, err
		}
	}

	defaultValue, ok := defaultConfig[opt.Key]
	if ok {
		data, err = sjson.SetBytes(data, "DefaultValue", defaultValue)
		if err != nil {
			return nil, err
		}
	}

	r, err := record.NewWrapper(fmt.Sprintf("config:%s", opt.Key), nil, record.JSON, data)
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
