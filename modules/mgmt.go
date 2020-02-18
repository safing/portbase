package modules

import (
	"context"

	"github.com/safing/portbase/log"
	"github.com/tevino/abool"
)

var (
	moduleMgmtEnabled     = abool.NewBool(false)
	modulesChangeNotifyFn func(*Module)
)

// Enable enables the module. Only has an effect if module management is enabled.
func (m *Module) Enable() (changed bool) {
	return m.enabled.SetToIf(false, true)
}

// Disable disables the module. Only has an effect if module management is enabled.
func (m *Module) Disable() (changed bool) {
	return m.enabled.SetToIf(true, false)
}

// SetEnabled sets the module to the desired enabled state. Only has an effect if module management is enabled.
func (m *Module) SetEnabled(enable bool) (changed bool) {
	if enable {
		return m.Enable()
	}
	return m.Disable()
}

// Enabled returns wether or not the module is currently enabled.
func (m *Module) Enabled() bool {
	return m.enabled.IsSet()
}

// EnableModuleManagement enables the module management functionality within modules. The supplied notify function will be called whenever the status of a module changes. The affected module will be in the parameter. You will need to manually enable modules, else nothing will start.
func EnableModuleManagement(changeNotifyFn func(*Module)) {
	if moduleMgmtEnabled.SetToIf(false, true) {
		modulesChangeNotifyFn = changeNotifyFn
	}
}

func (m *Module) notifyOfChange() {
	if moduleMgmtEnabled.IsSet() && modulesChangeNotifyFn != nil {
		m.StartWorker("notify of change", func(ctx context.Context) error {
			modulesChangeNotifyFn(m)
			return nil
		})
	}
}

// ManageModules triggers the module manager to react to recent changes of enabled modules.
func ManageModules() error {
	// check if enabled
	if !moduleMgmtEnabled.IsSet() {
		return nil
	}

	// lock mgmt
	mgmtLock.Lock()
	defer mgmtLock.Unlock()

	log.Info("modules: managing changes")

	// build new dependency tree
	buildEnabledTree()

	// stop unneeded modules
	lastErr := stopModules()
	if lastErr != nil {
		log.Warning(lastErr.Error())
	}

	// start needed modules
	err := startModules()
	if err != nil {
		log.Warning(err.Error())
		lastErr = err
	}

	log.Info("modules: finished managing")
	return lastErr
}

func buildEnabledTree() {
	// reset marked dependencies
	for _, m := range modules {
		m.enabledAsDependency.UnSet()
	}

	// mark dependencies
	for _, m := range modules {
		if m.enabled.IsSet() {
			m.markDependencies()
		}
	}
}

func (m *Module) markDependencies() {
	for _, dep := range m.depModules {
		if dep.enabledAsDependency.SetToIf(false, true) {
			dep.markDependencies()
		}
	}
}
