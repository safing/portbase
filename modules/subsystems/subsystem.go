package subsystems

import (
	"sync"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/modules"
)

// Subsystem describes a subset of modules that represent a part of a
// service or program to the user. Subsystems can be (de-)activated causing
// all related modules to be brought down or up.
type Subsystem struct { //nolint:maligned // not worth the effort
	record.Base
	sync.Mutex
	// ID is a unique identifier for the subsystem.
	ID string
	// Name holds a human readable name of the subsystem.
	Name string
	// Description may holds an optional description of
	// the subsystem's purpose.
	Description string
	// Modules contains all modules that are related to the subsystem.
	// Note that this slice also contains a reference to the subsystem
	// module itself.
	Modules []*ModuleStatus
	// FailureStatus is the worst failure status that is currently
	// set in one of the subsystem's dependencies.
	FailureStatus uint8
	// ToggleOptionKey holds the key of the configuration option
	// that is used to completely enable or disable this subsystem.
	ToggleOptionKey string
	// ExpertiseLevel defines the complexity of the subsystem and is
	// copied from the subsystem's toggleOption.
	ExpertiseLevel config.ExpertiseLevel
	// ReleaseLevel defines the stability of the subsystem and is
	// copied form the subsystem's toggleOption.
	ReleaseLevel config.ReleaseLevel
	// ConfigKeySpace defines the database key prefix that all
	// options that belong to this subsystem have. Note that this
	// value is mainly used to mark all related options with a
	// config.SubsystemAnnotation. Options that are part of
	// this subsystem but don't start with the correct prefix can
	// still be marked by manually setting the appropriate annotation.
	ConfigKeySpace string

	module       *modules.Module
	toggleOption *config.Option
	toggleValue  config.BoolOption
}

// ModuleStatus describes the status of a module.
type ModuleStatus struct {
	Name   string
	module *modules.Module

	// status mgmt
	Enabled bool
	Status  uint8

	// failure status
	FailureStatus uint8
	FailureID     string
	FailureMsg    string
}

func (sub *Subsystem) addDependencies(module *modules.Module, seen map[string]struct{}) {
	for _, module := range module.Dependencies() {
		if _, ok := seen[module.Name]; !ok {
			seen[module.Name] = struct{}{}

			sub.Modules = append(sub.Modules, statusFromModule(module))
			sub.addDependencies(module, seen)
		}
	}
}

func statusFromModule(module *modules.Module) *ModuleStatus {
	status := &ModuleStatus{
		Name:    module.Name,
		module:  module,
		Enabled: module.Enabled() || module.EnabledAsDependency(),
		Status:  module.Status(),
	}
	status.FailureStatus, status.FailureID, status.FailureMsg = module.FailureStatus()

	return status
}

func compareAndUpdateStatus(module *modules.Module, status *ModuleStatus) (changed bool) {
	// check if enabled
	enabled := module.Enabled() || module.EnabledAsDependency()
	if status.Enabled != enabled {
		status.Enabled = enabled
		changed = true
	}

	// check status
	statusLvl := module.Status()
	if status.Status != statusLvl {
		status.Status = statusLvl
		changed = true
	}

	// check failure status
	failureStatus, failureID, failureMsg := module.FailureStatus()
	if status.FailureStatus != failureStatus ||
		status.FailureID != failureID {

		status.FailureStatus = failureStatus
		status.FailureID = failureID
		status.FailureMsg = failureMsg
		changed = true
	}

	return
}

func (sub *Subsystem) makeSummary() {
	// find worst failing module
	sub.FailureStatus = 0
	for _, depStatus := range sub.Modules {
		if depStatus.FailureStatus > sub.FailureStatus {
			sub.FailureStatus = depStatus.FailureStatus
		}
	}
}
