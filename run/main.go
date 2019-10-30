package run

import (
	"bufio"
	"flag"
	"fmt"
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

		_ = modules.Shutdown()
		return modules.GetExitStatusCode()
	}

	// Shutdown
	// catch interrupt for clean shutdown
	signalCh := make(chan os.Signal)
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
				_ = pprof.Lookup("goroutine").WriteTo(os.Stderr, 2)
				continue signalLoop
			}

			fmt.Println(" <INTERRUPT>")
			log.Warning("main: program was interrupted, shutting down.")

			// catch signals during shutdown
			go func() {
				for {
					<-signalCh
					fmt.Println(" <INTERRUPT> again, but already shutting down")
				}
			}()

			if printStackOnExit {
				fmt.Println("=== PRINTING TRACES ===")
				fmt.Println("=== GOROUTINES ===")
				_ = pprof.Lookup("goroutine").WriteTo(os.Stdout, 2)
				fmt.Println("=== BLOCKING ===")
				_ = pprof.Lookup("block").WriteTo(os.Stdout, 2)
				fmt.Println("=== MUTEXES ===")
				_ = pprof.Lookup("mutex").WriteTo(os.Stdout, 2)
				fmt.Println("=== END TRACES ===")
			}

			go func() {
				time.Sleep(60 * time.Second)
				fmt.Fprintln(os.Stderr, "===== TAKING TOO LONG FOR SHUTDOWN - PRINTING STACK TRACES =====")
				_ = pprof.Lookup("goroutine").WriteTo(os.Stderr, 2)
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
