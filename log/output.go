package log

import (
	"fmt"
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
	shutdownWaitGroup.Add(1)
	fmt.Println(fmt.Sprintf("%s%s %s BOF%s", InfoLevel.color(), time.Now().Format(timeFormat), rightArrow, endColor()))
	go writer()
}

func writer() {
	var line *logLine
	var lastLine *logLine
	var duplicates uint64
	defer shutdownWaitGroup.Done()

	for {
		// reset
		line = nil
		lastLine = nil //nolint:ineffassign // only ineffectual in first loop
		duplicates = 0

		// wait until logs need to be processed
		select {
		case <-logsWaiting:
			logsWaitingFlag.UnSet()
		case <-shutdownSignal:
			finalizeWriting()
			return
		}

		// wait for timeslot to log, or when buffer is full
		select {
		case <-writeTrigger:
		case <-forceEmptyingOfBuffer:
		case <-shutdownSignal:
			finalizeWriting()
			return
		}

		// write all the logs!
	writeLoop:
		for {
			select {
			case line = <-logBuffer:

				// look-ahead for deduplication (best effort)
			dedupLoop:
				for {
					// check if there is another line waiting
					select {
					case nextLine := <-logBuffer:
						lastLine = line
						line = nextLine
					default:
						break dedupLoop
					}

					// deduplication
					if !line.Equal(lastLine) {
						// no duplicate
						break dedupLoop
					}

					// duplicate
					duplicates++
				}

				// write actual line
				writeLine(line, duplicates)
				duplicates = 0
			default:
				break writeLoop
			}
		}

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
			fmt.Println(fmt.Sprintf("%s%s %s EOF%s", InfoLevel.color(), time.Now().Format(timeFormat), leftArrow, endColor()))
			return
		}
	}
}
