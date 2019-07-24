package log

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

func fastcheck(level severity) bool {
	if fileLevelsActive.IsSet() {
		return true
	}
	if uint32(level) < atomic.LoadUint32(logLevel) {
		return false
	}
	return true
}

func log(level severity, msg string, trace *ContextTracer) {

	if !started.IsSet() {
		// a bit resouce intense, but keeps logs before logging started.
		// FIXME: create option to disable logging
		go func() {
			<-startedSignal
			log(level, msg, trace)
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
		if len(fileOnly) < 2 {
			return
		}
		sev, ok := fileLevels[fileOnly[len(fileOnly)-2]]
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
		msg:       msg,
		trace:     trace,
		level:     level,
		timestamp: now,
		file:      file,
		line:      line,
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
	if fastcheck(TraceLevel) {
		log(TraceLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	}
}

func Trace(msg string) {
	if fastcheck(TraceLevel) {
		log(TraceLevel, msg, nil)
	}
}

func Debugf(things ...interface{}) {
	if fastcheck(DebugLevel) {
		log(DebugLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	}
}

func Debug(msg string) {
	if fastcheck(DebugLevel) {
		log(DebugLevel, msg, nil)
	}
}

func Infof(things ...interface{}) {
	if fastcheck(InfoLevel) {
		log(InfoLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	}
}

func Info(msg string) {
	if fastcheck(InfoLevel) {
		log(InfoLevel, msg, nil)
	}
}

func Warningf(things ...interface{}) {
	if fastcheck(WarningLevel) {
		log(WarningLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	}
}

func Warning(msg string) {
	if fastcheck(WarningLevel) {
		log(WarningLevel, msg, nil)
	}
}

func Errorf(things ...interface{}) {
	if fastcheck(ErrorLevel) {
		log(ErrorLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	}
}

func Error(msg string) {
	if fastcheck(ErrorLevel) {
		log(ErrorLevel, msg, nil)
	}
}

func Criticalf(things ...interface{}) {
	if fastcheck(CriticalLevel) {
		log(CriticalLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	}
}

func Critical(msg string) {
	if fastcheck(CriticalLevel) {
		log(CriticalLevel, msg, nil)
	}
}

func Testf(things ...interface{}) {
	fmt.Printf(things[0].(string), things[1:]...)
}

func Test(msg string) {
	fmt.Println(msg)
}
