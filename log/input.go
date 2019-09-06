package log

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

func log(level Severity, msg string, tracer *ContextTracer) {

	if !started.IsSet() {
		// a bit resouce intense, but keeps logs before logging started.
		// FIXME: create option to disable logging
		go func() {
			<-startedSignal
			log(level, msg, tracer)
		}()
		return
	}

	// check if level is enabled
	if !pkgLevelsActive.IsSet() && uint32(level) < atomic.LoadUint32(logLevel) {
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
	if pkgLevelsActive.IsSet() {
		fileOnly := strings.Split(file, "/")
		if len(fileOnly) < 2 {
			return
		}
		sev, ok := pkgLevels[fileOnly[len(fileOnly)-2]]
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
		tracer:    tracer,
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

func fastcheck(level Severity) bool {
	if pkgLevelsActive.IsSet() {
		return true
	}
	if uint32(level) >= atomic.LoadUint32(logLevel) {
		return true
	}
	return false
}

// Trace is used to log tiny steps. Log traces to context if you can!
func Trace(msg string) {
	if fastcheck(TraceLevel) {
		log(TraceLevel, msg, nil)
	}
}

// Tracef is used to log tiny steps. Log traces to context if you can!
func Tracef(format string, things ...interface{}) {
	if fastcheck(TraceLevel) {
		log(TraceLevel, fmt.Sprintf(format, things...), nil)
	}
}

// Debug is used to log minor errors or unexpected events. These occurences are usually not worth mentioning in itself, but they might hint at a bigger problem.
func Debug(msg string) {
	if fastcheck(DebugLevel) {
		log(DebugLevel, msg, nil)
	}
}

// Debugf is used to log minor errors or unexpected events. These occurences are usually not worth mentioning in itself, but they might hint at a bigger problem.
func Debugf(format string, things ...interface{}) {
	if fastcheck(DebugLevel) {
		log(DebugLevel, fmt.Sprintf(format, things...), nil)
	}
}

// Info is used to log mildly significant events. Should be used to inform about somewhat bigger or user affecting events that happen.
func Info(msg string) {
	if fastcheck(InfoLevel) {
		log(InfoLevel, msg, nil)
	}
}

// Infof is used to log mildly significant events. Should be used to inform about somewhat bigger or user affecting events that happen.
func Infof(format string, things ...interface{}) {
	if fastcheck(InfoLevel) {
		log(InfoLevel, fmt.Sprintf(format, things...), nil)
	}
}

// Warning is used to log (potentially) bad events, but nothing broke (even a little) and there is no need to panic yet.
func Warning(msg string) {
	if fastcheck(WarningLevel) {
		log(WarningLevel, msg, nil)
	}
}

// Warningf is used to log (potentially) bad events, but nothing broke (even a little) and there is no need to panic yet.
func Warningf(format string, things ...interface{}) {
	if fastcheck(WarningLevel) {
		log(WarningLevel, fmt.Sprintf(format, things...), nil)
	}
}

// Error is used to log errors that break or impair functionality. The task/process may have to be aborted and tried again later. The system is still operational. Maybe User/Admin should be informed.
func Error(msg string) {
	if fastcheck(ErrorLevel) {
		log(ErrorLevel, msg, nil)
	}
}

// Errorf is used to log errors that break or impair functionality. The task/process may have to be aborted and tried again later. The system is still operational.
func Errorf(format string, things ...interface{}) {
	if fastcheck(ErrorLevel) {
		log(ErrorLevel, fmt.Sprintf(format, things...), nil)
	}
}

// Critical is used to log events that completely break the system. Operation connot continue. User/Admin must be informed.
func Critical(msg string) {
	if fastcheck(CriticalLevel) {
		log(CriticalLevel, msg, nil)
	}
}

// Criticalf is used to log events that completely break the system. Operation connot continue. User/Admin must be informed.
func Criticalf(format string, things ...interface{}) {
	if fastcheck(CriticalLevel) {
		log(CriticalLevel, fmt.Sprintf(format, things...), nil)
	}
}
