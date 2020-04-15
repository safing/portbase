package log

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

var (
	schedulingEnabled = false
	writeTrigger      = make(chan struct{})
)

// EnableScheduling enables external scheduling of the logger. This will require to manually trigger writes via TriggerWrite whenevery logs should be written. Please note that full buffers will also trigger writing. Must be called before Start() to have an effect.
func EnableScheduling() {
	if !initializing.IsSet() {
		schedulingEnabled = true
	}
}

// TriggerWriter triggers log output writing.
func TriggerWriter() {
	if started.IsSet() && schedulingEnabled {
		select {
		case writeTrigger <- struct{}{}:
		default:
		}
	}
}

// TriggerWriterChannel returns the channel to trigger log writing. Returned channel will close if EnableScheduling() is not called correctly.
func TriggerWriterChannel() chan struct{} {
	return writeTrigger
}

func writeLine(line *logLine, duplicates uint64) {
	fmt.Println(formatLine(line, duplicates, true))
	// TODO: implement file logging and setting console/file logging
	// TODO: use https://github.com/natefinch/lumberjack
}

func startWriter() {
	fmt.Printf("%s%s %s BOF%s\n", InfoLevel.color(), time.Now().Format(timeFormat), rightArrow, endColor())

	shutdownWaitGroup.Add(1)
	go writerManager()
}

func writerManager() {
	defer shutdownWaitGroup.Done()

	for {
		err := writer()
		if err != nil {
			Errorf("log: writer failed: %s", err)
		} else {
			return
		}
	}
}

func writer() (err error) {
	defer func() {
		// recover from panic
		panicVal := recover()
		if panicVal != nil {
			err = fmt.Errorf("%s", panicVal)

			// write stack to stderr
			fmt.Fprintf(
				os.Stderr,
				`===== Error Report =====
Message: %s
StackTrace:

%s
===== End of Report =====
`,
				err,
				string(debug.Stack()),
			)
		}
	}()

	var currentLine *logLine
	var nextLine *logLine
	var duplicates uint64

	for {
		// reset
		currentLine = nil
		nextLine = nil
		duplicates = 0

		// wait until logs need to be processed
		select {
		case <-logsWaiting: // normal process
			logsWaitingFlag.UnSet()
		case <-forceEmptyingOfBuffer: // log buffer is full!
		case <-shutdownSignal: // shutting down
			finalizeWriting()
			return
		}

		// wait for timeslot to log
		select {
		case <-writeTrigger: // normal process
		case <-forceEmptyingOfBuffer: // log buffer is full!
		case <-shutdownSignal: // shutting down
			finalizeWriting()
			return
		}

		// write all the logs!
	writeLoop:
		for {
			select {
			case nextLine = <-logBuffer:
				// first line we process, just assign to currentLine
				if currentLine == nil {
					currentLine = nextLine
					continue writeLoop
				}

				// we now have currentLine and nextLine

				// if currentLine and nextLine are equal, do not print, just increase counter and continue
				if nextLine.Equal(currentLine) {
					duplicates++
					continue writeLoop
				}

				// if currentLine and line are _not_ equal, output currentLine
				writeLine(currentLine, duplicates)
				// reset duplicate counter
				duplicates = 0
				// set new currentLine
				currentLine = nextLine
			default:
				break writeLoop
			}
		}

		// write final line
		if currentLine != nil {
			writeLine(currentLine, duplicates)
		}
		// reset state
		currentLine = nil //nolint:ineffassign
		nextLine = nil
		duplicates = 0 //nolint:ineffassign

		// back down a little
		select {
		case <-time.After(10 * time.Millisecond):
		case <-shutdownSignal:
			finalizeWriting()
			return
		}

	}
}

func finalizeWriting() {
	for {
		select {
		case line := <-logBuffer:
			writeLine(line, 0)
		case <-time.After(10 * time.Millisecond):
			fmt.Printf("%s%s %s EOF%s\n", InfoLevel.color(), time.Now().Format(timeFormat), leftArrow, endColor())
			return
		}
	}
}
