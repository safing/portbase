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
				time.Sleep(1 * time.Second)
				if i%2 == 0 {
					config.SetConfigOption("test/explode_on_error", i)
				} else {
					config.SetDefaultConfigOption("test/explode_on_error", i)
				}
			}
		}
	}()

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
