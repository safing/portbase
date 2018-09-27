// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"github.com/Safing/portbase/taskmanager"
	"time"
)

func writeLine(line *logLine) {

	fmt.Println(formatLine(line, true))

	// TODO: implement file logging and setting console/file logging
	// TODO: use https://github.com/natefinch/lumberjack

}

func writer() {
	var line *logLine
	startedTask := false
	shutdownWaitGroup.Add(1)
	defer shutdownWaitGroup.Done()

	for {

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
					writeLine(line)
				case <-time.After(10 * time.Millisecond):
					fmt.Println(fmt.Sprintf("%s%s ◀ EOF%s", InfoLevel.color(), time.Now().Format("060102 15:04:05.000"), endColor()))
					return
				}
			}
		}

		// write all the logs!
	writeLoop:
		for {
			select {
			case line = <-logBuffer:
				writeLine(line)
			default:
				if startedTask {
					taskmanager.EndMicroTask()
					startedTask = false
				}
				break writeLoop
			}
		}

	}
}
