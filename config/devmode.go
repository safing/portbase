package config

import (
	"flag"

	"github.com/safing/portbase/log"
)

// Configuration Keys.
var (
	CfgDevModeKey  = "core/devMode"
	defaultDevMode bool
)

func init() {
	flag.BoolVar(&defaultDevMode, "devmode", false, "enable development mode")
}

func logDevModeOverride() {
	if defaultDevMode {
		log.Warning("config: development mode is enabled by default by the -devmode flag")
	}
}

func registerDevModeOption() error {
	return Register(&Option{
		Name:           "Development Mode",
		Key:            CfgDevModeKey,
		Description:    "In Development Mode, security restrictions are lifted/softened to enable unrestricted access for debugging and testing purposes.",
		OptType:        OptTypeBool,
		ExpertiseLevel: ExpertiseLevelDeveloper,
		ReleaseLevel:   ReleaseLevelStable,
		DefaultValue:   defaultDevMode,
		Annotations: Annotations{
			DisplayOrderAnnotation: 512,
			CategoryAnnotation:     "Development",
		},
	})
}
