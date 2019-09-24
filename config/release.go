package config

import (
	"fmt"
	"sync"
)

const (
	releaseLevelKey = "core/release_level"
)

var (
	releaseLevel     = ReleaseLevelStable
	releaseLevelLock sync.Mutex
)

func init() {
	registerReleaseLevelOption()
}

func registerReleaseLevelOption() {
	err := Register(&Option{
		Name:        "Release Selection",
		Key:         releaseLevelKey,
		Description: "Select maturity level of features that should be available",

		OptType:        OptTypeString,
		ExpertiseLevel: ExpertiseLevelExpert,
		ReleaseLevel:   ReleaseLevelStable,

		RequiresRestart: false,
		DefaultValue:    ReleaseLevelStable,

		ExternalOptType: "string list",
		ValidationRegex: fmt.Sprintf("^(%s|%s|%s)$", ReleaseLevelStable, ReleaseLevelBeta, ReleaseLevelExperimental),
	})
	if err != nil {
		panic(err)
	}
}

func updateReleaseLevel() {
	new := findStringValue(releaseLevelKey, "")
	releaseLevelLock.Lock()
	if new == "" {
		releaseLevel = ReleaseLevelStable
	} else {
		releaseLevel = new
	}
	releaseLevelLock.Unlock()
}

func getReleaseLevel() string {
	releaseLevelLock.Lock()
	defer releaseLevelLock.Unlock()
	return releaseLevel
}
