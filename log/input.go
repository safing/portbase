// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package log

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

func log_fastcheck(level severity) bool {
	if fileLevelsActive.IsSet() {
		return true
	}
	if uint32(level) < atomic.LoadUint32(logLevel) {
		return false
	}
	return true
}

func log(level severity, msg string) {

	if !started.IsSet() {
		// resouce intense, but keeps logs before logging started.
		go func(){
			time.Sleep(1 * time.Second)
			log(level, msg)
		}()
		return
	}

	// check if level is enabled
	if !fileLevelsActive.IsSet() && uint32(level) < atomic.LoadUint32(logLevel) {
		return
	}

	// get time
	now := time.Now()

	// get file and line
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = ""
		line = 0
	} else {
		if len(file) > 3 {
			file = file[:len(file)-3]
		} else {
			file = ""
		}
	}

	// check if level is enabled for file or generally
	if fileLevelsActive.IsSet() {
		fileOnly := strings.Split(file, "/")
		sev, ok := fileLevels[fileOnly[len(fileOnly)-1]]
		if ok {
			if level < sev {
				return
			}
		} else {
			if uint32(level) < atomic.LoadUint32(logLevel) {
				return
			}
		}
	}

	// create log object
	log := &logLine{
		msg,
		level,
		now,
		file,
		line,
	}

	// send log to processing
	select {
	case logBuffer <- log:
	default:
		forceEmptyingOfBuffer <- true
		logBuffer <- log
	}

	// wake up writer if necessary
	if logsWaitingFlag.SetToIf(false, true) {
		logsWaiting <- true
	}

}

func Tracef(things ...interface{}) {
	if log_fastcheck(TraceLevel) {
		log(TraceLevel, fmt.Sprintf(things[0].(string), things[1:]...))
	}
}

func Trace(msg string) {
	if log_fastcheck(TraceLevel) {
		log(TraceLevel, msg)
	}
}

func Debugf(things ...interface{}) {
	if log_fastcheck(DebugLevel) {
		log(DebugLevel, fmt.Sprintf(things[0].(string), things[1:]...))
	}
}

func Debug(msg string) {
	if log_fastcheck(DebugLevel) {
		log(DebugLevel, msg)
	}
}

func Infof(things ...interface{}) {
	if log_fastcheck(InfoLevel) {
		log(InfoLevel, fmt.Sprintf(things[0].(string), things[1:]...))
	}
}

func Info(msg string) {
	if log_fastcheck(InfoLevel) {
		log(InfoLevel, msg)
	}
}

func Warningf(things ...interface{}) {
	if log_fastcheck(WarningLevel) {
		log(WarningLevel, fmt.Sprintf(things[0].(string), things[1:]...))
	}
}

func Warning(msg string) {
	if log_fastcheck(WarningLevel) {
		log(WarningLevel, msg)
	}
}

func Errorf(things ...interface{}) {
	if log_fastcheck(ErrorLevel) {
		log(ErrorLevel, fmt.Sprintf(things[0].(string), things[1:]...))
	}
}

func Error(msg string) {
	if log_fastcheck(ErrorLevel) {
		log(ErrorLevel, msg)
	}
}

func Criticalf(things ...interface{}) {
	if log_fastcheck(CriticalLevel) {
		log(CriticalLevel, fmt.Sprintf(things[0].(string), things[1:]...))
	}
}

func Critical(msg string) {
	if log_fastcheck(CriticalLevel) {
		log(CriticalLevel, msg)
	}
}

func Testf(things ...interface{}) {
	fmt.Printf(things[0].(string), things[1:]...)
}

func Test(msg string) {
	fmt.Println(msg)
}
