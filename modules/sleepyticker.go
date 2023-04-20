package modules

import "time"

// SleepyTicker is wrapper over time.Ticker that respects the sleep mode of the module.
type SleepyTicker struct {
	ticker         time.Ticker
	module         *Module
	normalDuration time.Duration
	sleepDuration  time.Duration
	sleepMode      bool
}

// newSleepyTicker returns a new SleepyTicker. This is a wrapper of the standard time.Ticker but it respects modules.Module sleep mode. Check https://pkg.go.dev/time#Ticker.
// If sleepDuration is set to 0 ticker will not tick during sleep.
func newSleepyTicker(module *Module, normalDuration time.Duration, sleepDuration time.Duration) *SleepyTicker {
	st := &SleepyTicker{
		ticker:         *time.NewTicker(normalDuration),
		module:         module,
		normalDuration: normalDuration,
		sleepDuration:  sleepDuration,
		sleepMode:      false,
	}

	return st
}

// Wait waits until the module is not in sleep mode and returns time.Ticker.C channel.
func (st *SleepyTicker) Wait() <-chan time.Time {
	sleepModeEnabled := st.module.sleepMode.IsSet()

	// Update Sleep mode
	if sleepModeEnabled != st.sleepMode {
		st.enterSleepMode(sleepModeEnabled)
	}

	// Wait if until sleep mode exits only if sleepDuration is set to 0.
	if sleepModeEnabled && st.sleepDuration == 0 {
		return st.module.WaitIfSleeping()
	}

	return st.ticker.C
}

// Stop turns off a ticker. After Stop, no more ticks will be sent. Stop does not close the channel, to prevent a concurrent goroutine reading from the channel from seeing an erroneous "tick".
func (st *SleepyTicker) Stop() {
	st.ticker.Stop()
}

func (st *SleepyTicker) enterSleepMode(enabled bool) {
	st.sleepMode = enabled
	if enabled {
		if st.sleepDuration > 0 {
			st.ticker.Reset(st.sleepDuration)
		}
	} else {
		st.ticker.Reset(st.normalDuration)
	}
}
