// Package config ... (linter fix)
//nolint:dupl
package config

import (
	"sync/atomic"

	"github.com/tevino/abool"
)

// ExpertiseLevel allows to group settings by user expertise.
// It's useful if complex or technical settings should be hidden
// from the average user while still allowing experts and developers
// to change deep configuration settings.
type ExpertiseLevel uint8

// Expertise Level constants
const (
	ExpertiseLevelUser      ExpertiseLevel = 0
	ExpertiseLevelExpert    ExpertiseLevel = 1
	ExpertiseLevelDeveloper ExpertiseLevel = 2

	ExpertiseLevelNameUser      = "user"
	ExpertiseLevelNameExpert    = "expert"
	ExpertiseLevelNameDeveloper = "developer"

	expertiseLevelKey = "core/expertiseLevel"
)

var (
	expertiseLevelOption     *Option
	expertiseLevel           = new(int32)
	expertiseLevelOptionFlag = abool.New()
)

func init() {
	registerExpertiseLevelOption()
}

func registerExpertiseLevelOption() {
	expertiseLevelOption = &Option{
		Name:           "Expertise Level",
		Key:            expertiseLevelKey,
		Description:    "The Expertise Level controls the perceived complexity. Higher settings will show you more complex settings and information. This might also affect various other things relying on this setting. Modified settings in higher expertise levels stay in effect when switching back. (Unlike the Release Level)",
		OptType:        OptTypeString,
		ExpertiseLevel: ExpertiseLevelUser,
		ReleaseLevel:   ReleaseLevelStable,
		DefaultValue:   ExpertiseLevelNameUser,
		Annotations: Annotations{
			DisplayHintAnnotation: DisplayHintOneOf,
		},
		PossibleValues: []PossibleValue{
			{
				Name:        "Easy",
				Value:       ExpertiseLevelNameUser,
				Description: "Easy application mode by hidding complex settings.",
			},
			{
				Name:        "Expert",
				Value:       ExpertiseLevelNameExpert,
				Description: "Expert application mode. Allows access to almost all configuration options.",
			},
			{
				Name:        "Developer",
				Value:       ExpertiseLevelNameDeveloper,
				Description: "Developer mode. Please be careful!",
			},
		},
	}

	err := Register(expertiseLevelOption)
	if err != nil {
		panic(err)
	}

	expertiseLevelOptionFlag.Set()
}

func updateExpertiseLevel() {
	// check if already registered
	if !expertiseLevelOptionFlag.IsSet() {
		return
	}
	// get value
	value := expertiseLevelOption.activeFallbackValue
	if expertiseLevelOption.activeValue != nil {
		value = expertiseLevelOption.activeValue
	}
	if expertiseLevelOption.activeDefaultValue != nil {
		value = expertiseLevelOption.activeDefaultValue
	}
	// set atomic value
	switch value.stringVal {
	case ExpertiseLevelNameUser:
		atomic.StoreInt32(expertiseLevel, int32(ExpertiseLevelUser))
	case ExpertiseLevelNameExpert:
		atomic.StoreInt32(expertiseLevel, int32(ExpertiseLevelExpert))
	case ExpertiseLevelNameDeveloper:
		atomic.StoreInt32(expertiseLevel, int32(ExpertiseLevelDeveloper))
	default:
		atomic.StoreInt32(expertiseLevel, int32(ExpertiseLevelUser))
	}
}

// GetExpertiseLevel returns the current active expertise level.
func GetExpertiseLevel() uint8 {
	return uint8(atomic.LoadInt32(expertiseLevel))
}
