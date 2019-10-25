package modules

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

var (
	errorReportingChannel chan *ModuleError
	reportToStdErr        = true
	reportingLock         sync.RWMutex
)

// ModuleError wraps a panic, error or message into an error that can be reported.
type ModuleError struct {
	Message string

	ModuleName string
	TaskName   string
	TaskType   string // one of "worker", "task", "microtask" or custom
	Severity   string // one of "info", "error", "panic" or custom

	PanicValue interface{}
	StackTrace string
}

// NewInfoMessage creates a new, reportable, info message (including a stack trace).
func (m *Module) NewInfoMessage(message string) *ModuleError {
	return &ModuleError{
		Message:    message,
		ModuleName: m.Name,
		Severity:   "info",
		StackTrace: string(debug.Stack()),
	}
}

// NewErrorMessage creates a new, reportable, error message (including a stack trace).
func (m *Module) NewErrorMessage(taskName string, err error) *ModuleError {
	return &ModuleError{
		Message:    err.Error(),
		ModuleName: m.Name,
		Severity:   "error",
		StackTrace: string(debug.Stack()),
	}
}

// NewPanicError creates a new, reportable, panic error message (including a stack trace).
func (m *Module) NewPanicError(taskName, taskType string, panicValue interface{}) *ModuleError {
	me := &ModuleError{
		Message:    fmt.Sprintf("panic: %s", panicValue),
		ModuleName: m.Name,
		TaskName:   taskName,
		TaskType:   taskType,
		Severity:   "panic",
		PanicValue: panicValue,
		StackTrace: string(debug.Stack()),
	}
	me.Message = me.Error()
	return me
}

// Error returns the string representation of the error.
func (me *ModuleError) Error() string {
	return me.Message
}

// Report reports the error through the configured reporting channel.
func (me *ModuleError) Report() {
	reportingLock.RLock()
	defer reportingLock.RUnlock()

	if errorReportingChannel != nil {
		select {
		case errorReportingChannel <- me:
		default:
		}
	}

	if reportToStdErr {
		// default to writing to stderr
		fmt.Fprintf(
			os.Stderr,
			`===== Error Report =====
Message: %s
Timestamp: %s
ModuleName: %s
TaskName: %s
TaskType: %s
Severity: %s
PanicValue: %s
StackTrace:

%s
===== End of Report =====
`,
			me.Message,
			time.Now(),
			me.ModuleName,
			me.TaskName,
			me.TaskType,
			me.Severity,
			me.PanicValue,
			me.StackTrace,
		)
	}
}

// IsPanic returns whether the given error is a wrapped panic by the modules package and additionally returns it, if true.
func IsPanic(err error) (bool, *ModuleError) {
	switch val := err.(type) {
	case *ModuleError:
		return true, val
	default:
		return false, nil
	}
}

// SetErrorReportingChannel sets the channel to report module errors through. By default only panics are reported, all other errors need to be manually wrapped into a *ModuleError and reported.
func SetErrorReportingChannel(reportingChannel chan *ModuleError) {
	reportingLock.Lock()
	defer reportingLock.Unlock()

	errorReportingChannel = reportingChannel
}

// SetStdErrReporting controls error reporting to stderr.
func SetStdErrReporting(on bool) {
	reportingLock.Lock()
	defer reportingLock.Unlock()

	reportToStdErr = on
}
