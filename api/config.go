package api

import (
	"flag"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/log"
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
		Key:             "api/listenAddress",
		Description:     "Define on which IP and port the API should listen on.",
		ExpertiseLevel:  config.ExpertiseLevelDeveloper,
		OptType:         config.OptTypeString,
		DefaultValue:    getDefaultListenAddress(),
		ValidationRegex: "^([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}:[0-9]{1,5}|\\[[:0-9A-Fa-f]+\\]:[0-9]{1,5})$",
		RequiresRestart: true,
	})
	if err != nil {
		return err
	}
	listenAddressConfig = config.GetAsString("api/listenAddress", getDefaultListenAddress())

	return nil
}

// SetDefaultAPIListenAddress sets the default listen address for the API.
func SetDefaultAPIListenAddress(address string) {
	defaultListenAddress = address
}
