package utils

import (
	"sync"

	"github.com/tevino/abool"
)

// FlagController is a simple system to broadcast a flag value.
type FlagController struct {
	flag   *abool.AtomicBool
	signal chan struct{}
	lock   sync.Mutex
}

// Flag receives changes from its FlagController.
// A Flag must only be used in one goroutine and is not concurrency safe,
// but fast.
type Flag struct {
	flag       *abool.AtomicBool
	signal     chan struct{}
	controller *FlagController
}

// NewFlagController returns a new FlagController.
// In the initial state, the flag is not set and the singal does not trigger.
func NewFlagController() *FlagController {
	return &FlagController{
		flag:   abool.New(),
		signal: make(chan struct{}),
		lock:   sync.Mutex{},
	}
}

// NewFlag returns a new Flag controlled by this controller.
// In the initial state, the flag is set and the singal triggers.
// You can call Refresh immediately to get the current state from the
// controller.
func (cfc *FlagController) NewFlag() *Flag {
	newFlag := &Flag{
		flag:       abool.NewBool(true),
		signal:     make(chan struct{}),
		controller: cfc,
	}
	close(newFlag.signal)
	return newFlag
}

// NotifyAndReset notifies all flags of this controller and resets the
// controller state.
func (cfc *FlagController) NotifyAndReset() {
	cfc.lock.Lock()
	defer cfc.lock.Unlock()

	// Notify all flags of the change.
	cfc.flag.Set()
	close(cfc.signal)

	// Reset
	cfc.flag = abool.New()
	cfc.signal = make(chan struct{})
}

// Signal returns a channel that waits for the flag to be set. This does not
// reset the Flag itself, you'll need to call Refresh for that.
func (cf *Flag) Signal() <-chan struct{} {
	return cf.signal
}

// IsSet returns whether the flag was set since the last Refresh.
// This does not reset the Flag itself, you'll need to call Refresh for that.
func (cf *Flag) IsSet() bool {
	return cf.flag.IsSet()
}

// Refresh fetches the current state from the controller.
func (cf *Flag) Refresh() {
	cf.controller.lock.Lock()
	defer cf.controller.lock.Unlock()

	// Copy current flag and signal from the controller.
	cf.flag = cf.controller.flag
	cf.signal = cf.controller.signal
}
