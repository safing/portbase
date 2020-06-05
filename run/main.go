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
			printStackTo(os.Stdout)
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
				_ = pprof.Lookup("goroutine").WriteTo(os.Stderr, 1)
				continue signalLoop
			}

			fmt.Println(" <INTERRUPT>")
			log.Warning("main: program was interrupted, shutting down.")

			forceCnt := 5
			// catch signals during shutdown
			go func() {
				for {
					<-signalCh
					forceCnt--
					if forceCnt > 0 {
						fmt.Printf(" <INTERRUPT> again, but already shutting down. %d more to force.\n", forceCnt)
					} else {
						fmt.Fprintln(os.Stderr, "===== FORCED EXIT =====")
						printStackTo(os.Stderr)
						os.Exit(1)
					}
				}
			}()

			if printStackOnExit {
				printStackTo(os.Stdout)
			}

			go func() {
				time.Sleep(3 * time.Minute)
				fmt.Fprintln(os.Stderr, "===== TAKING TOO LONG FOR SHUTDOWN =====")
				printStackTo(os.Stderr)
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

func printStackTo(writer io.Writer) {
	fmt.Fprintln(writer, "=== PRINTING TRACES ===")
	fmt.Fprintln(writer, "=== GOROUTINES ===")
	_ = pprof.Lookup("goroutine").WriteTo(writer, 1)
	fmt.Fprintln(writer, "=== BLOCKING ===")
	_ = pprof.Lookup("block").WriteTo(writer, 1)
	fmt.Fprintln(writer, "=== MUTEXES ===")
	_ = pprof.Lookup("mutex").WriteTo(writer, 1)
	fmt.Fprintln(writer, "=== END TRACES ===")
}
