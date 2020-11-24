package api

import (
	"flag"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/log"
)

// Config Keys
const (
	CfgDefaultListenAddressKey = "core/listenAddress"
)

var (
	listenAddressFlag    string
	listenAddressConfig  config.StringOption
	defaultListenAddress string
)

func init() {
	flag.StringVar(&listenAddressFlag, "api-address", "", "override api listen address")
}

func logFlagOverrides() {
	if listenAddressFlag != "" {
		log.Warning("api: api/listenAddress default config is being overridden by -api-address flag")
	}
}

func getDefaultListenAddress() string {
	// check if overridden
	if listenAddressFlag != "" {
		return listenAddressFlag
	}
	// return internal default
	return defaultListenAddress
}

func registerConfig() error {
	err := config.Register(&config.Option{
		Name:            "API Address",
		Key:             CfgDefaultListenAddressKey,
		Description:     "Defines the IP address and port for the internal API.",
		OptType:         config.OptTypeString,
		ExpertiseLevel:  config.ExpertiseLevelDeveloper,
		ReleaseLevel:    config.ReleaseLevelStable,
		DefaultValue:    getDefaultListenAddress(),
		ValidationRegex: "^([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}:[0-9]{1,5}|\\[[:0-9A-Fa-f]+\\]:[0-9]{1,5})$",
		RequiresRestart: true,
		Annotations: config.Annotations{
			config.DisplayOrderAnnotation: 513,
			config.CategoryAnnotation:     "Development",
		},
	})
	if err != nil {
		return err
	}
	listenAddressConfig = config.GetAsString(CfgDefaultListenAddressKey, getDefaultListenAddress())

	return nil
}

// SetDefaultAPIListenAddress sets the default listen address for the API.
func SetDefaultAPIListenAddress(address string) {

	defaultListenAddress = address
}
