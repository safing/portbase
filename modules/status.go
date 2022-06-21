package modules

import (
	"context"

	"github.com/tevino/abool"
)

// Module Status Values.
const (
	StatusDead      uint8 = 0 // not prepared, not started
	StatusPreparing uint8 = 1
	StatusOffline   uint8 = 2 // prepared, not started
	StatusStopping  uint8 = 3
	StatusStarting  uint8 = 4
	StatusOnline    uint8 = 5 // online and running
)

// Module Failure Status Values.
const (
	FailureNone    uint8 = 0
	FailureHint    uint8 = 1
	FailureWarning uint8 = 2
	FailureError   uint8 = 3
)

// Ready Stati.
const (
	statusWaiting uint8 = iota
	statusReady
	statusNothingToDo
)

var (
	failureUpdateNotifyFunc        func(moduleFailure uint8, id, title, msg string)
	failureUpdateNotifyFuncEnabled = abool.NewBool(false)
	failureUpdateNotifyFuncReady   = abool.NewBool(false)
)

// SetFailureUpdateNotifyFunc sets a function that is called on every change
// of a module's failure status.
func SetFailureUpdateNotifyFunc(fn func(moduleFailure uint8, id, title, msg string)) bool {
	if failureUpdateNotifyFuncEnabled.SetToIf(false, true) {
		failureUpdateNotifyFunc = fn
		failureUpdateNotifyFuncReady.Set()
		return true
	}
	return false
}

// Online returns whether the module is online.
func (m *Module) Online() bool {
	return m.Status() == StatusOnline
}

// OnlineSoon returns whether the module is or is about to be online.
func (m *Module) OnlineSoon() bool {
	if moduleMgmtEnabled.IsSet() &&
		!m.enabled.IsSet() &&
		!m.enabledAsDependency.IsSet() {
		return false
	}
	return !m.stopFlag.IsSet()
}

// Status returns the current module status.
func (m *Module) Status() uint8 {
	m.RLock()
	defer m.RUnlock()

	return m.status
}

// FailureStatus returns the current failure status, ID and message.
func (m *Module) FailureStatus() (failureStatus uint8, failureID, failureMsg string) {
	m.RLock()
	defer m.RUnlock()

	return m.failureStatus, m.failureID, m.failureMsg
}

// Hint sets failure status to hint. This is a somewhat special failure status,
// as the module is believed to be working correctly, but there is an important
// module specific information to convey. The supplied failureID is for
// improved automatic handling within connected systems, the failureMsg is for
// humans.
// The given ID must be unique for the given title and message. A call to
// Hint(), Warning() or Error() with the same ID as the existing one will be
// ignored.
func (m *Module) Hint(id, title, msg string) {
	m.setFailure(FailureHint, id, title, msg, true)
}

// Warning sets failure status to warning. The supplied failureID is for
// improved automatic handling within connected systems, the failureMsg is for
// humans.
// The given ID must be unique for the given title and message. A call to
// Hint(), Warning() or Error() with the same ID as the existing one will be
// ignored.
func (m *Module) Warning(id, title, msg string) {
	m.setFailure(FailureWarning, id, title, msg, true)
}

// Error sets failure status to error. The supplied failureID is for improved
// automatic handling within connected systems, the failureMsg is for humans.
// The given ID must be unique for the given title and message. A call to
// Hint(), Warning() or Error() with the same ID as the existing one will be
// ignored.
func (m *Module) Error(id, title, msg string) {
	m.setFailure(FailureError, id, title, msg, true)
}

func (m *Module) setFailure(status uint8, id, title, msg string, lockModule bool) {
	var resolveFailureID string
	func() {
		if lockModule {
			m.Lock()
			defer m.Unlock()
		}

		// Ignore calls with the same ID.
		if id == m.failureID {
			return
		}

		// Copy data for failure status update worker.
		resolveFailureID = m.failureID

		// Set new failure status.
		m.failureStatus = status
		m.failureID = id
		m.failureTitle = title
		m.failureMsg = msg
	}()

	// Notify of module change.
	m.notifyOfChange()

	// Propagate failure status.
	if failureUpdateNotifyFuncReady.IsSet() {
		_ = m.RunWorker("failure status updater", func(context.Context) error {
			// Only use data in worker that won't change anymore.

			// Resolve previous failure state if available.
			if resolveFailureID != "" {
				failureUpdateNotifyFunc(FailureNone, resolveFailureID, "", "")
			}

			// Notify of new failure state.
			failureUpdateNotifyFunc(status, id, title, msg)

			return nil
		})
	}
}

// Resolve removes the failure state from the module if the given failureID matches the current failure ID. If the given failureID is an empty string, Resolve removes any failure state.
func (m *Module) Resolve(failureID string) {
	m.Lock()
	defer m.Unlock()

	// Check if resolving is necessary.
	if failureID != "" && failureID != m.failureID {
		// Return immediately if not resolving any (`""`) or if the failure ID
		// does not match.
		return
	}

	// Copy data for failure status update worker.
	resolveFailureID := m.failureID

	// Set failure status on module.
	m.failureStatus = FailureNone
	m.failureID = ""
	m.failureTitle = ""
	m.failureMsg = ""

	// Notify of module change.
	m.notifyOfChange()

	// Propagate failure status.
	if failureUpdateNotifyFuncReady.IsSet() {
		m.StartWorker("failure status updater", func(context.Context) error {
			// Only use data in worker that won't change anymore.
			failureUpdateNotifyFunc(FailureNone, resolveFailureID, "", "")
			return nil
		})
	}
}

// readyToPrep returns whether all dependencies are ready for this module to prep.
func (m *Module) readyToPrep() uint8 {
	// check if valid state for prepping
	if m.Status() != StatusDead {
		return statusNothingToDo
	}

	for _, dep := range m.depModules {
		if dep.Status() < StatusOffline {
			return statusWaiting
		}
	}

	return statusReady
}

// readyToStart returns whether all dependencies are ready for this module to start.
func (m *Module) readyToStart() uint8 {
	// check if start is wanted
	if moduleMgmtEnabled.IsSet() {
		if !m.enabled.IsSet() && !m.enabledAsDependency.IsSet() {
			return statusNothingToDo
		}
	}

	// check if valid state for starting
	if m.Status() != StatusOffline {
		return statusNothingToDo
	}

	// check if all dependencies are ready
	for _, dep := range m.depModules {
		if dep.Status() < StatusOnline {
			return statusWaiting
		}
	}

	return statusReady
}

// readyToStop returns whether all dependencies are ready for this module to stop.
func (m *Module) readyToStop() uint8 {
	// check if stop is wanted
	if moduleMgmtEnabled.IsSet() && !shutdownFlag.IsSet() {
		if m.enabled.IsSet() || m.enabledAsDependency.IsSet() {
			return statusNothingToDo
		}
	}

	// check if valid state for stopping
	if m.Status() != StatusOnline {
		return statusNothingToDo
	}

	for _, revDep := range m.depReverse {
		// not ready if a reverse dependency was started, but not yet stopped
		if revDep.Status() > StatusOffline {
			return statusWaiting
		}
	}

	return statusReady
}
