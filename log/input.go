package log

import (
	"fmt"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
)

func log(level Severity, tracer *ContextTracer, msg string, args []interface{}) {
	if !fastcheck(level) {
		return
	}

	if !started.IsSet() {
		// a bit resource intense, but keeps logs before logging started.
		// TODO: create option to disable logging
		go func() {
			<-startedSignal
			log(level, tracer, msg, args)
		}()
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
		pathSegments := strings.Split(file, "/")
		if len(pathSegments) < 2 {
			// file too short for package levels
			return
		}
		pkgLevelsLock.Lock()
		severity, ok := pkgLevels[pathSegments[len(pathSegments)-2]]
		pkgLevelsLock.Unlock()
		if ok && level < severity {
			return
		}
	}

	if uint32(level) < atomic.LoadUint32(logLevel) {
		// no package levels set, check against global level
		return
	}

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
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
	forceEmptyingLoop:
		// force empty buffer until we can send to it
		for {
			select {
			case forceEmptyingOfBuffer <- struct{}{}:
			case logBuffer <- log:
				break forceEmptyingLoop
			}
		}
	}

	// wake up writer if necessary
	if logsWaitingFlag.SetToIf(false, true) {
		logsWaiting <- struct{}{}
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
	Tracef(msg)
}

// Tracef is used to log tiny steps. Log traces to context if you can!
func Tracef(format string, things ...interface{}) {
	log(TraceLevel, nil, format, things)
}

// Debug is used to log minor errors or unexpected events. These occurrences are usually not worth mentioning in itself, but they might hint at a bigger problem.
func Debug(msg string) {
	Debugf(msg)
}

// Debugf is used to log minor errors or unexpected events. These occurrences are usually not worth mentioning in itself, but they might hint at a bigger problem.
func Debugf(format string, things ...interface{}) {
	log(DebugLevel, nil, format, things)
}

// Info is used to log mildly significant events. Should be used to inform about somewhat bigger or user affecting events that happen.
func Info(msg string) {
	Infof(msg)
}

// Infof is used to log mildly significant events. Should be used to inform about somewhat bigger or user affecting events that happen.
func Infof(format string, things ...interface{}) {
	log(InfoLevel, nil, format, things)
}

// Warning is used to log (potentially) bad events, but nothing broke (even a little) and there is no need to panic yet.
func Warning(msg string) {
	Warningf(msg)
}

// Warningf is used to log (potentially) bad events, but nothing broke (even a little) and there is no need to panic yet.
func Warningf(format string, things ...interface{}) {
	log(WarningLevel, nil, format, things)
}

// Error is used to log errors that break or impair functionality. The task/process may have to be aborted and tried again later. The system is still operational. Maybe User/Admin should be informed.
func Error(msg string) {
	Errorf(msg)
}

// Errorf is used to log errors that break or impair functionality. The task/process may have to be aborted and tried again later. The system is still operational.
func Errorf(format string, things ...interface{}) {
	log(ErrorLevel, nil, format, things)
}

// Critical is used to log events that completely break the system. Operation connot continue. User/Admin must be informed.
func Critical(msg string) {
	Criticalf(msg)
}

// Criticalf is used to log events that completely break the system. Operation connot continue. User/Admin must be informed.
func Criticalf(format string, things ...interface{}) {
	log(CriticalLevel, nil, format, things)
}
