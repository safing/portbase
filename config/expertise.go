// Package config ... (linter fix)
//nolint:dupl
package config

import (
	"fmt"
	"sync/atomic"

	"github.com/tevino/abool"
)

// Expertise Level constants.
const (
	ExpertiseLevelUser      uint8 = 0
	ExpertiseLevelExpert    uint8 = 1
	ExpertiseLevelDeveloper uint8 = 2

	ExpertiseLevelNameUser      = "user"
	ExpertiseLevelNameExpert    = "expert"
	ExpertiseLevelNameDeveloper = "developer"

	expertiseLevelKey = "core/expertiseLevel"
)

var (
	expertiseLevel           *int32
	expertiseLevelOption     *Option
	expertiseLevelOptionFlag = abool.New()
)

func init() {
	var expertiseLevelVal int32
	expertiseLevel = &expertiseLevelVal

	registerExpertiseLevelOption()
}

func registerExpertiseLevelOption() {
	expertiseLevelOption = &Option{
		Name:        "Expertise Level",
		Key:         expertiseLevelKey,
		Description: "The Expertise Level controls the perceived complexity. Higher settings will show you more complex settings and information. This might also affect various other things relying on this setting. Modified settings in higher expertise levels stay in effect when switching back. (Unlike the Release Level)",

		OptType:        OptTypeString,
		ExpertiseLevel: ExpertiseLevelUser,
		ReleaseLevel:   ExpertiseLevelUser,

		RequiresRestart: false,
		DefaultValue:    ExpertiseLevelNameUser,

		ExternalOptType: "string list",
		ValidationRegex: fmt.Sprintf("^(%s|%s|%s)$", ExpertiseLevelNameUser, ExpertiseLevelNameExpert, ExpertiseLevelNameDeveloper),
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
