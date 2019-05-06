// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"time"

	"github.com/Safing/portbase/taskmanager"
)

func writeLine(line *logLine, duplicates uint64) {
	fmt.Println(formatLine(line, duplicates, true))
	// TODO: implement file logging and setting console/file logging
	// TODO: use https://github.com/natefinch/lumberjack
}

func startWriter() {
	shutdownWaitGroup.Add(1)
	fmt.Println(fmt.Sprintf("%s%s %s BOF%s", InfoLevel.color(), time.Now().Format("060102 15:04:05.000"), rightArrow, endColor()))
	go writer()
}

func writer() {
	var line *logLine
	var lastLine *logLine
	var duplicates uint64
	startedTask := false
	defer shutdownWaitGroup.Done()

	for {
		// reset
		line = nil
		lastLine = nil
		duplicates = 0

		// wait until logs need to be processed
		select {
		case <-logsWaiting:
			logsWaitingFlag.UnSet()
		case <-shutdownSignal:
		}

		// wait for timeslot to log, or when buffer is full
		select {
		case <-taskmanager.StartVeryLowPriorityMicroTask():
			startedTask = true
		case <-forceEmptyingOfBuffer:
		case <-shutdownSignal:
			for {
				select {
				case line = <-logBuffer:
					writeLine(line, duplicates)
				case <-time.After(10 * time.Millisecond):
					fmt.Println(fmt.Sprintf("%s%s %s EOF%s", InfoLevel.color(), time.Now().Format("060102 15:04:05.000"), leftArrow, endColor()))
					return
				}
			}
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
						writeLine(lastLine, duplicates)
						duplicates = 0
					} else {
						// duplicate
						duplicates++
					}
				}

				// write actual line
				writeLine(line, duplicates)
				duplicates = 0
			default:
				if startedTask {
					taskmanager.EndMicroTask()
					startedTask = false
				}
				break writeLoop
			}
		}

		// back down a little
		select {
		case <-time.After(10 * time.Millisecond):
		case <-shutdownSignal:
		}

	}
}
