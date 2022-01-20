package run

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/safing/portbase/log"
	"github.com/safing/portbase/modules"
)

var (
	printStackOnExit   bool
	enableInputSignals bool

	sigUSR1 = syscall.Signal(0xa) // dummy for windows
)

func init() {
	flag.BoolVar(&printStackOnExit, "print-stack-on-exit", false, "prints the stack before of shutting down")
	flag.BoolVar(&enableInputSignals, "input-signals", false, "emulate signals using stdin")
}

// Run execute a full program lifecycle (including signal handling) based on modules. Just empty-import required packages and do os.Exit(run.Run()).
func Run() int {

	// Start
	err := modules.Start()
	if err != nil {
		if err == modules.ErrCleanExit {
			return 0
		}

		if printStackOnExit {
			printStackTo(os.Stdout, "PRINTING STACK ON EXIT (STARTUP ERROR)")
		}

		_ = modules.Shutdown()
		return modules.GetExitStatusCode()
	}

	// Shutdown
	// catch interrupt for clean shutdown
	signalCh := make(chan os.Signal, 1)
	if enableInputSignals {
		go inputSignals(signalCh)
	}
	signal.Notify(
		signalCh,
		os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		sigUSR1,
	)

signalLoop:
	for {
		select {
		case sig := <-signalCh:
			// only print and continue to wait if SIGUSR1
			if sig == sigUSR1 {
				printStackTo(os.Stderr, "PRINTING STACK ON REQUEST")
				continue signalLoop
			}

			fmt.Println(" <INTERRUPT>")
			log.Warning("main: program was interrupted, shutting down.")

			// catch signals during shutdown
			go func() {
				forceCnt := 5
				for {
					<-signalCh
					forceCnt--
					if forceCnt > 0 {
						fmt.Printf(" <INTERRUPT> again, but already shutting down. %d more to force.\n", forceCnt)
					} else {
						printStackTo(os.Stderr, "PRINTING STACK ON FORCED EXIT")
						os.Exit(1)
					}
				}
			}()

			if printStackOnExit {
				printStackTo(os.Stdout, "PRINTING STACK ON EXIT")
			}

			go func() {
				time.Sleep(3 * time.Minute)
				printStackTo(os.Stderr, "PRINTING STACK - TAKING TOO LONG FOR SHUTDOWN")
				os.Exit(1)
			}()

			_ = modules.Shutdown()
			break signalLoop

		case <-modules.ShuttingDown():
			break signalLoop
		}
	}

	// wait for shutdown to complete, then exit
	return modules.GetExitStatusCode()
}

func inputSignals(signalCh chan os.Signal) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch scanner.Text() {
		case "SIGHUP":
			signalCh <- syscall.SIGHUP
		case "SIGINT":
			signalCh <- syscall.SIGINT
		case "SIGQUIT":
			signalCh <- syscall.SIGQUIT
		case "SIGTERM":
			signalCh <- syscall.SIGTERM
		case "SIGUSR1":
			signalCh <- sigUSR1
		}
	}
}

func printStackTo(writer io.Writer, msg string) {
	_, err := fmt.Fprintf(writer, "===== %s =====\n", msg)
	if err == nil {
		err = pprof.Lookup("goroutine").WriteTo(writer, 1)
	}
	if err != nil {
		log.Errorf("main: failed to write stack trace: %s", err)
	}
}
