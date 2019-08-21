package log

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Key for context value
type ContextTracerKey struct{}

type ContextTracer struct {
	sync.Mutex
	actions []*Action
}

type Action struct {
	timestamp time.Time
	level     severity
	msg       string
	file      string
	line      int
}

var (
	key       = ContextTracerKey{}
	nilTracer *ContextTracer
)

func AddTracer(ctx context.Context) context.Context {
	if ctx != nil && fastcheckLevel(TraceLevel) {
		_, ok := ctx.Value(key).(*ContextTracer)
		if !ok {
			return context.WithValue(ctx, key, &ContextTracer{})
		}
	}
	return ctx
}

func Tracer(ctx context.Context) *ContextTracer {
	if ctx != nil {
		tracer, ok := ctx.Value(key).(*ContextTracer)
		if ok {
			return tracer
		}
	}
	return nilTracer
}

func (ct *ContextTracer) logTrace(level severity, msg string) {
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

	ct.Lock()
	defer ct.Unlock()
	ct.actions = append(ct.actions, &Action{
		timestamp: time.Now(),
		level:     level,
		msg:       msg,
		file:      file,
		line:      line,
	})
}

func (ct *ContextTracer) Tracef(things ...interface{}) (ok bool) {
	if ct != nil {
		if fastcheckLevel(TraceLevel) {
			ct.logTrace(TraceLevel, fmt.Sprintf(things[0].(string), things[1:]...))
		}
		return true
	}
	return false
}

func (ct *ContextTracer) Trace(msg string) (ok bool) {
	if ct != nil {
		if fastcheckLevel(TraceLevel) {
			ct.logTrace(TraceLevel, msg)
		}
		return true
	}
	return false
}

func (ct *ContextTracer) Warningf(things ...interface{}) (ok bool) {
	if ct != nil {
		if fastcheckLevel(TraceLevel) {
			ct.logTrace(WarningLevel, fmt.Sprintf(things[0].(string), things[1:]...))
		}
		return true
	}
	return false
}

func (ct *ContextTracer) Warning(msg string) (ok bool) {
	if ct != nil {
		if fastcheckLevel(TraceLevel) {
			ct.logTrace(WarningLevel, msg)
		}
		return true
	}
	return false
}

func (ct *ContextTracer) Errorf(things ...interface{}) (ok bool) {
	if ct != nil {
		if fastcheckLevel(TraceLevel) {
			ct.logTrace(ErrorLevel, fmt.Sprintf(things[0].(string), things[1:]...))
		}
		return true
	}
	return false
}

func (ct *ContextTracer) Error(msg string) (ok bool) {
	if ct != nil {
		if fastcheckLevel(TraceLevel) {
			ct.logTrace(ErrorLevel, msg)
		}
		return true
	}
	return false
}

func DebugTrace(ctx context.Context, msg string) (ok bool) {
	tracer, ok := ctx.Value(key).(*ContextTracer)
	if ok && fastcheckLevel(TraceLevel) {
		log(DebugLevel, msg, tracer)
		return true
	}
	log(DebugLevel, msg, nil)
	return false
}

func DebugTracef(ctx context.Context, things ...interface{}) (ok bool) {
	tracer, ok := ctx.Value(key).(*ContextTracer)
	if ok && fastcheckLevel(TraceLevel) {
		log(DebugLevel, fmt.Sprintf(things[0].(string), things[1:]...), tracer)
		return true
	}
	log(DebugLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	return false
}

func InfoTrace(ctx context.Context, msg string) (ok bool) {
	tracer, ok := ctx.Value(key).(*ContextTracer)
	if ok && fastcheckLevel(TraceLevel) {
		log(InfoLevel, msg, tracer)
		return true
	}
	log(InfoLevel, msg, nil)
	return false
}

func InfoTracef(ctx context.Context, things ...interface{}) (ok bool) {
	tracer, ok := ctx.Value(key).(*ContextTracer)
	if ok && fastcheckLevel(TraceLevel) {
		log(InfoLevel, fmt.Sprintf(things[0].(string), things[1:]...), tracer)
		return true
	}
	log(InfoLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	return false
}

func WarningTrace(ctx context.Context, msg string) (ok bool) {
	tracer, ok := ctx.Value(key).(*ContextTracer)
	if ok && fastcheckLevel(TraceLevel) {
		log(WarningLevel, msg, tracer)
		return true
	}
	log(WarningLevel, msg, nil)
	return false
}

func WarningTracef(ctx context.Context, things ...interface{}) (ok bool) {
	tracer, ok := ctx.Value(key).(*ContextTracer)
	if ok && fastcheckLevel(TraceLevel) {
		log(WarningLevel, fmt.Sprintf(things[0].(string), things[1:]...), tracer)
		return true
	}
	log(WarningLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	return false
}

func ErrorTrace(ctx context.Context, msg string) (ok bool) {
	tracer, ok := ctx.Value(key).(*ContextTracer)
	if ok && fastcheckLevel(TraceLevel) {
		log(ErrorLevel, msg, tracer)
		return true
	}
	log(ErrorLevel, msg, nil)
	return false
}

func ErrorTracef(ctx context.Context, things ...interface{}) (ok bool) {
	tracer, ok := ctx.Value(key).(*ContextTracer)
	if ok && fastcheckLevel(TraceLevel) {
		log(ErrorLevel, fmt.Sprintf(things[0].(string), things[1:]...), tracer)
		return true
	}
	log(ErrorLevel, fmt.Sprintf(things[0].(string), things[1:]...), nil)
	return false
}

func fastcheckLevel(level severity) bool {
	return uint32(level) >= atomic.LoadUint32(logLevel)
}
