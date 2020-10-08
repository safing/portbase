package config

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"github.com/tidwall/sjson"

	"github.com/safing/portbase/database/record"
)

// OptionType defines the value type of an option.
type OptionType uint8

// Various attribute options. Use ExternalOptType for extended types in the frontend.
const (
	OptTypeString      OptionType = 1
	OptTypeStringArray OptionType = 2
	OptTypeInt         OptionType = 3
	OptTypeBool        OptionType = 4
)

func getTypeName(t OptionType) string {
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

// PossibleValue defines a value that is possible for
// a configuration setting.
type PossibleValue struct {
	// Name is a human readable name of the option.
	Name string
	// Description is a human readable description of
	// this value.
	Description string
	// Value is the actual value of the option. The type
	// must match the option's value type.
	Value interface{}
}

// Annotations can be attached to configuration options to
// provide hints for user interfaces or other systems working
// or setting configuration options.
// Annotation keys should follow the below format to ensure
// future well-known annotation additions do not conflict
// with vendor/product/package specific annoations.
//
// Format: <vendor/package>:<scope>:<identifier>
type Annotations map[string]interface{}

// Well known annotations defined by this package.
const (
	// DisplayHintAnnotation provides a hint for the user
	// interface on how to render an option.
	// The value of DisplayHintAnnotation is expected to
	// be a string. See DisplayHintXXXX constants below
	// for a list of well-known display hint annotations.
	DisplayHintAnnotation = "safing/portbase:ui:display-hint"
	// DisplayOrderAnnotation provides a hint for the user
	// interface in which order settings should be displayed.
	// The value of DisplayOrderAnnotations is expected to be
	// an number (int).
	DisplayOrderAnnotation = "safing/portbase:ui:order"
	// UnitAnnotations defines the SI unit of an option (if any).
	UnitAnnotation = "safing/portbase:ui:unit"
	// CategoryAnnotations can provide an additional category
	// to each settings. This category can be used by a user
	// interface to group certain options together.
	// User interfaces should treat a CategoryAnnotation, if
	// supported, with higher priority as a DisplayOrderAnnotation.
	CategoryAnnotation = "safing/portbase:ui:category"
	// SubsystemAnnotation can be used to mark an option as part
	// of a module subsystem.
	SubsystemAnnotation = "safing/portbase:module:subsystem"
)

// Values for the DisplayHintAnnotation
const (
	// DisplayHintOneOf is used to mark an option
	// as a "select"-style option. That is, only one of
	// the supported values may be set. This option makes
	// only sense together with the PossibleValues property
	// of Option.
	DisplayHintOneOf = "one-of"
	// DisplayHintOrdered Used to mark a list option as ordered.
	// That is, the order of items is important and a user interface
	// is encouraged to provide the user with re-ordering support
	// (like drag'n'drop).
	DisplayHintOrdered = "ordered"
)

// Option describes a configuration option.
type Option struct {
	sync.Mutex
	// Name holds the name of the configuration options.
	// It should be human readable and is mainly used for
	// presentation purposes.
	// Name is considered immutable after the option has
	// been created.
	Name string
	// Key holds the database path for the option. It should
	// follow the path format `category/sub/key`.
	// Key is considered immutable after the option has
	// been created.
	Key string
	// Description holds a human readable description of the
	// option and what is does. The description should be short.
	// Use the Help property for a longer support text.
	// Description is considered immutable after the option has
	// been created.
	Description string
	// Help may hold a long version of the description providing
	// assistence with the configuration option.
	// Help is considered immutable after the option has
	// been created.
	Help string
	// OptType defines the type of the option.
	// OptType is considered immutable after the option has
	// been created.
	OptType OptionType
	// ExpertiseLevel can be used to set the required expertise
	// level for the option to be displayed to a user.
	// ExpertiseLevel is considered immutable after the option has
	// been created.
	ExpertiseLevel ExpertiseLevel
	// ReleaseLevel is used to mark the stability of the option.
	// ReleaseLevel is considered immutable after the option has
	// been created.
	ReleaseLevel ReleaseLevel
	// RequiresRestart should be set to true if a modification of
	// the options value requires a restart of the whole application
	// to take effect.
	// RequiresRestart is considered immutable after the option has
	// been created.
	RequiresRestart bool
	// DefaultValue holds the default value of the option. Note that
	// this value can be overwritten during runtime (see activeDefaultValue
	// and activeFallbackValue).
	// DefaultValue is considered immutable after the option has
	// been created.
	DefaultValue interface{}
	// ValidationRegex may contain a regular expression used to validate
	// the value of option. If the option type is set to OptTypeStringArray
	// the validation regex is applied to all entries of the string slice.
	// Note that it is recommended to keep the validation regex simple so
	// it can also be used in other languages (mainly JavaScript) to provide
	// a better user-experience by pre-validating the expression.
	// ValidationRegex is considered immutable after the option has
	// been created.
	ValidationRegex string
	// PossibleValues may be set to a slice of values that are allowed
	// for this configuration setting. Note that PossibleValues makes most
	// sense when ExternalOptType is set to HintOneOf
	// PossibleValues is considered immutable after the option has
	// been created.
	PossibleValues []PossibleValue `json:",omitempty"`
	// Annotations adds additional annotations to the configuration options.
	// See documentation of Annotations for more information.
	// Annotations is considered mutable and setting/reading annotation keys
	// must be performed while the option is locked.
	Annotations Annotations

	activeValue         *valueCache // runtime value (loaded from config file or set by user)
	activeDefaultValue  *valueCache // runtime default value (may be set internally)
	activeFallbackValue *valueCache // default value from option registration
	compiledRegex       *regexp.Regexp
}

// AddAnnotation adds the annotation key to option if it's not already set.
func (option *Option) AddAnnotation(key string, value interface{}) {
	option.Lock()
	defer option.Unlock()

	if option.Annotations == nil {
		option.Annotations = make(Annotations)
	}

	if _, ok := option.Annotations[key]; ok {
		return
	}
	option.Annotations[key] = value
}

// SetAnnotation sets the value of the annotation key overwritting an
// existing value if required.
func (option *Option) SetAnnotation(key string, value interface{}) {
	option.Lock()
	defer option.Unlock()

	if option.Annotations == nil {
		option.Annotations = make(Annotations)
	}
	option.Annotations[key] = value
}

// GetAnnotation returns the value of the annotation key
func (option *Option) GetAnnotation(key string) (interface{}, bool) {
	option.Lock()
	defer option.Unlock()

	if option.Annotations == nil {
		return nil, false
	}
	val, ok := option.Annotations[key]
	return val, ok
}

// Export expors an option to a Record.
func (option *Option) Export() (record.Record, error) {
	option.Lock()
	defer option.Unlock()

	return option.export()
}

func (option *Option) export() (record.Record, error) {
	data, err := json.Marshal(option)
	if err != nil {
		return nil, err
	}

	if option.activeValue != nil {
		data, err = sjson.SetBytes(data, "Value", option.activeValue.getData(option))
		if err != nil {
			return nil, err
		}
	}

	if option.activeDefaultValue != nil {
		data, err = sjson.SetBytes(data, "DefaultValue", option.activeDefaultValue.getData(option))
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

type sortByKey []*Option

func (opts sortByKey) Len() int           { return len(opts) }
func (opts sortByKey) Less(i, j int) bool { return opts[i].Key < opts[j].Key }
func (opts sortByKey) Swap(i, j int)      { opts[i], opts[j] = opts[j], opts[i] }
