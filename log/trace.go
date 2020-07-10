package log

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// tracerKey is the key used for the context key/value storage.
type tracerKey struct{}

// ContextTracer is attached to a context in order bind logs to a context.
type ContextTracer struct {
	m    sync.Mutex
	logs []*logLine
}

// AddTracer adds a ContextTracer to the returned Context. Will return a nil ContextTracer if
// logging level is not set to trace. Will return a nil ContextTracer if one already exists.
// Will return a nil ContextTracer in case of an error. Will return a nil context if nil.
func AddTracer(ctx context.Context) (context.Context, *ContextTracer) {
	if ctx == nil || !fastcheck(TraceLevel) {
		return ctx, nil
	}

	activeLogLevel := atomic.LoadUint32(logLevel)

	// check pkg levels
	if pkgLevelsActive.IsSet() {
		// get file
		_, file, _, ok := runtime.Caller(1)
		if !ok {
			// cannot get file, ignore
			return ctx, nil
		}

		pathSegments := strings.Split(file, "/")
		if len(pathSegments) < 2 {
			// file too short for package levels
			return ctx, nil
		}

		pkgLevelsLock.Lock()
		severity, ok := pkgLevels[pathSegments[len(pathSegments)-2]]
		pkgLevelsLock.Unlock()

		if ok && TraceLevel < severity {
			return ctx, nil
		}
	}

	if uint32(TraceLevel) < activeLogLevel {
		// no package levels set, check against global level
		return ctx, nil
	}

	// check for existing tracer
	_, ok := ctx.Value(tracerKey{}).(*ContextTracer)
	if !ok {
		// add and return new tracer
		tracer := &ContextTracer{}
		return context.WithValue(ctx, tracerKey{}, tracer), tracer
	}
	return ctx, nil
}

// Tracer returns the ContextTracer previously added to ctx or nil.
func Tracer(ctx context.Context) *ContextTracer {
	if ctx == nil {
		return nil
	}
	tracer, _ := ctx.Value(tracerKey{}).(*ContextTracer)
	return tracer
}

// Submit collected logs on the context for further processing/outputting. Does nothing if called on a nil ContextTracer.
func (tracer *ContextTracer) Submit() {
	if tracer == nil {
		return
	}

	if !started.IsSet() {
		// a bit resource intense, but keeps logs before logging started.
		// TODO: create option to disable logging
		go func() {
			<-startedSignal
			tracer.Submit()
		}()
		return
	}

	if len(tracer.logs) == 0 {
		return
	}

	// extract last line as main line
	mainLine := tracer.logs[len(tracer.logs)-1]
	tracer.logs = tracer.logs[:len(tracer.logs)-1]

	// create log object
	log := &logLine{
		msg:       mainLine.msg,
		tracer:    tracer,
		level:     mainLine.level,
		timestamp: mainLine.timestamp,
		file:      mainLine.file,
		line:      mainLine.line,
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

func (tracer *ContextTracer) log(level Severity, msg string) {
	if tracer == nil {
		if fastcheck(level) {
			log(level, msg, nil)
		}
		return
	}

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

	tracer.m.Lock()
	defer tracer.m.Unlock()
	tracer.logs = append(tracer.logs, &logLine{
		timestamp: time.Now(),
		level:     level,
		msg:       msg,
		file:      file,
		line:      line,
	})
}

// Trace is used to log tiny steps. Log traces to context if you can!
func (tracer *ContextTracer) Trace(msg string) {
	tracer.log(TraceLevel, msg)
}

// Tracef is used to log tiny steps. Log traces to context if you can!
func (tracer *ContextTracer) Tracef(format string, things ...interface{}) {
	tracer.Trace(fmt.Sprintf(format, things...))
}

// Debug is used to log minor errors or unexpected events. These occurrences are usually not worth mentioning in itself, but they might hint at a bigger problem.
func (tracer *ContextTracer) Debug(msg string) {
	tracer.log(DebugLevel, msg)
}

// Debugf is used to log minor errors or unexpected events. These occurrences are usually not worth mentioning in itself, but they might hint at a bigger problem.
func (tracer *ContextTracer) Debugf(format string, things ...interface{}) {
	tracer.Debug(fmt.Sprintf(format, things...))
}

// Info is used to log mildly significant events. Should be used to inform about somewhat bigger or user affecting events that happen.
func (tracer *ContextTracer) Info(msg string) {
	tracer.log(InfoLevel, msg)
}

// Infof is used to log mildly significant events. Should be used to inform about somewhat bigger or user affecting events that happen.
func (tracer *ContextTracer) Infof(format string, things ...interface{}) {
	tracer.Info(fmt.Sprintf(format, things...))
}

// Warning is used to log (potentially) bad events, but nothing broke (even a little) and there is no need to panic yet.
func (tracer *ContextTracer) Warning(msg string) {
	tracer.log(WarningLevel, msg)
}

// Warningf is used to log (potentially) bad events, but nothing broke (even a little) and there is no need to panic yet.
func (tracer *ContextTracer) Warningf(format string, things ...interface{}) {
	tracer.Warning(fmt.Sprintf(format, things...))
}

// Error is used to log errors that break or impair functionality. The task/process may have to be aborted and tried again later. The system is still operational. Maybe User/Admin should be informed.
func (tracer *ContextTracer) Error(msg string) {
	tracer.log(ErrorLevel, msg)
}

// Errorf is used to log errors that break or impair functionality. The task/process may have to be aborted and tried again later. The system is still operational.
func (tracer *ContextTracer) Errorf(format string, things ...interface{}) {
	tracer.Error(fmt.Sprintf(format, things...))
}

// Critical is used to log events that completely break the system. Operation connot continue. User/Admin must be informed.
func (tracer *ContextTracer) Critical(msg string) {
	tracer.log(CriticalLevel, msg)
}

// Criticalf is used to log events that completely break the system. Operation connot continue. User/Admin must be informed.
func (tracer *ContextTracer) Criticalf(format string, things ...interface{}) {
	tracer.Critical(fmt.Sprintf(format, things...))
}
