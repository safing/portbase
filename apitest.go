package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Safing/portbase/info"
	"github.com/Safing/portbase/log"
	"github.com/Safing/portbase/modules"
	// include packages here

	_ "github.com/Safing/portbase/api"
	_ "github.com/Safing/portbase/api/testclient"
	"github.com/Safing/portbase/config"
	_ "github.com/Safing/portbase/crypto/random"
	_ "github.com/Safing/portbase/database/dbmodule"
)

// var (
// 	err = debugStartProblems()
// )
//
// func debugStartProblems() error {
// 	go func() {
// 		time.Sleep(1 * time.Second)
// 		fmt.Println("===== TAKING TOO LONG - PRINTING STACK TRACES =====")
// 		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
// 		os.Exit(1)
// 	}()
// 	return nil
// }

func main() {

	// Set Info
	info.Set("Portbase API Development Helper", "0.0.1")

	// Register some dummy config vars

	// Start
	err := modules.Start()
	if err != nil {
		if err == modules.ErrCleanExit {
			os.Exit(0)
		} else {
			modules.Shutdown()
			os.Exit(1)
		}
	}

	// Test config option
	config.Register(&config.Option{
		Name:           "Explode on error",
		Key:            "test/explode_on_error",
		Description:    "Defines how hard we should crash, in case of an error, in Joule.",
		ExpertiseLevel: config.ExpertiseLevelDeveloper,
		OptType:        config.OptTypeInt,
		DefaultValue:   1,
	})
	go func() {
		for {
			for i := 0; i < 1000; i++ {
				time.Sleep(10 * time.Second)
				if i%2 == 0 {
					config.SetConfigOption("test/explode_on_error", i)
				} else {
					config.SetDefaultConfigOption("test/explode_on_error", i)
				}
			}
		}
	}()

	// More test configs
	config.Register(&config.Option{
		Name:           "Activate this",
		Key:            "test/activate",
		Description:    "Activates something.",
		ExpertiseLevel: config.ExpertiseLevelDeveloper,
		OptType:        config.OptTypeBool,
		DefaultValue:   false,
	})
	config.Register(&config.Option{
		Name:           "Cover Name",
		Key:            "test/cover_name",
		Description:    "A cover name to be used for dangerous things.",
		ExpertiseLevel: config.ExpertiseLevelUser,
		OptType:        config.OptTypeString,
		DefaultValue:   "Mr. Smith",
	})
	config.Register(&config.Option{
		Name:            "Operation profile",
		Key:             "test/op_profile",
		Description:     "Set operation profile.",
		ExpertiseLevel:  config.ExpertiseLevelUser,
		OptType:         config.OptTypeString,
		ExternalOptType: "string list",
		DefaultValue:    "normal",
		ValidationRegex: "^(eco|normal|speed)$",
	})
	config.Register(&config.Option{
		Name:            "Block Autonomous Systems",
		Key:             "test/block_as",
		Description:     "Specify Autonomous Systems to be blocked by ASN Number or Organisation Name prefix.",
		ExpertiseLevel:  config.ExpertiseLevelUser,
		OptType:         config.OptTypeStringArray,
		DefaultValue:    []string{},
		ValidationRegex: "^(AS[0-9]{1,10}|[A-Za-z0-9 \\.-_]+)$",
	})
	config.Register(&config.Option{
		Name:            "Favor Countries",
		Key:             "test/fav_countries",
		Description:     "Specify favored Countries. These will be favored if route costs are similar. Specify with 2-Letter County Code, use \"A1\" for Anonymous Proxies and \"A2\" for Satellite Providers. Database used is provided by MaxMind.",
		ExpertiseLevel:  config.ExpertiseLevelUser,
		OptType:         config.OptTypeStringArray,
		ExternalOptType: "country list",
		DefaultValue:    []string{},
		ValidationRegex: "^([A-Z0-9]{2})$",
	})
	config.Register(&config.Option{
		Name:            "TLS Inspection",
		Key:             "test/inspect_tls",
		Description:     "TLS traffic will be inspected to ensure its valid and uses good options.",
		ExpertiseLevel:  config.ExpertiseLevelExpert,
		OptType:         config.OptTypeInt,
		ExternalOptType: "security level",
		DefaultValue:    3,
		ValidationRegex: "^(1|2|3)$",
	})

	// Shutdown
	// catch interrupt for clean shutdown
	signalCh := make(chan os.Signal)
	signal.Notify(
		signalCh,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	select {
	case <-signalCh:
		fmt.Println(" <INTERRUPT>")
		log.Warning("main: program was interrupted, shutting down.")
		modules.Shutdown()
	case <-modules.ShuttingDown():
	}

}
