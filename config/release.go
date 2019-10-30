// Package config ... (linter fix)
//nolint:dupl
package config

import (
	"fmt"
	"sync/atomic"
)

// Release Level constants
const (
	ReleaseLevelStable       uint8 = 0
	ReleaseLevelBeta         uint8 = 1
	ReleaseLevelExperimental uint8 = 2

	ReleaseLevelNameStable       = "stable"
	ReleaseLevelNameBeta         = "beta"
	ReleaseLevelNameExperimental = "experimental"

	releaseLevelKey = "core/releaseLevel"
)

var (
	releaseLevel *int32
)

func init() {
	var releaseLevelVal int32
	releaseLevel = &releaseLevelVal

	registerReleaseLevelOption()
}

func registerReleaseLevelOption() {
	err := Register(&Option{
		Name:        "Release Level",
		Key:         releaseLevelKey,
		Description: "The Release Level changes which features are available to you. Some beta or experimental features are also available in the stable release channel. Unavailable settings are set to the default value.",

		OptType:        OptTypeString,
		ExpertiseLevel: ExpertiseLevelExpert,
		ReleaseLevel:   ReleaseLevelStable,

		RequiresRestart: false,
		DefaultValue:    ReleaseLevelNameStable,

		ExternalOptType: "string list",
		ValidationRegex: fmt.Sprintf("^(%s|%s|%s)$", ReleaseLevelNameStable, ReleaseLevelNameBeta, ReleaseLevelNameExperimental),
	})
	if err != nil {
		panic(err)
	}
}

func updateReleaseLevel() {
	new := findStringValue(releaseLevelKey, "")
	switch new {
	case ReleaseLevelNameStable:
		atomic.StoreInt32(releaseLevel, int32(ReleaseLevelStable))
	case ReleaseLevelNameBeta:
		atomic.StoreInt32(releaseLevel, int32(ReleaseLevelBeta))
	case ReleaseLevelNameExperimental:
		atomic.StoreInt32(releaseLevel, int32(ReleaseLevelExperimental))
	default:
		atomic.StoreInt32(releaseLevel, int32(ReleaseLevelStable))
	}
}

func getReleaseLevel() uint8 {
	return uint8(atomic.LoadInt32(releaseLevel))
}
