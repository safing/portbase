package api

import (
	"flag"

	"github.com/Safing/portbase/config"
	"github.com/Safing/portbase/log"
)

var (
	listenAddressFlag    string
	listenAddressConfig  config.StringOption
	defaultListenAddress string
)

func init() {
	flag.StringVar(&listenAddressFlag, "api-address", "", "override api listen address")
}

func checkFlags() error {
	if listenAddressFlag != "" {
		log.Warning("api: api/listenAddress config is being overridden by -api-address flag")
	}
	return nil
}

func getListenAddress() string {
	if listenAddressFlag != "" {
		return listenAddressFlag
	}
	return listenAddressConfig()
}

func registerConfig() error {
	err := config.Register(&config.Option{
		Name:            "API Address",
		Key:             "api/listenAddress",
		Description:     "Define on what IP and port the API should listen on. Be careful, changing this may become a security issue.",
		ExpertiseLevel:  config.ExpertiseLevelExpert,
		OptType:         config.OptTypeString,
		DefaultValue:    defaultListenAddress,
		ValidationRegex: "^([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3}:[0-9]{1,5}|\\[[:0-9A-Fa-f]+\\]:[0-9]{1,5})$",
	})
	if err != nil {
		return err
	}
	listenAddressConfig = config.GetAsString("api/listenAddress", defaultListenAddress)

	return nil
}

func SetDefaultAPIListenAddress(address string) {
	if defaultListenAddress == "" {
		defaultListenAddress = address
	}
}
