package subsystems

import (
	"sync"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/log"
	"github.com/safing/portbase/modules"
)

// Subsystem describes a subset of modules that represent a part of a service or program to the user.
type Subsystem struct { //nolint:maligned // not worth the effort
	record.Base
	sync.Mutex

	ID          string
	Name        string
	Description string
	module      *modules.Module

	Modules       []*ModuleStatus
	FailureStatus uint8 // summary: worst status

	ToggleOptionKey string
	toggleOption    *config.Option
	toggleValue     func() bool
	ExpertiseLevel  uint8 // copied from toggleOption
	ReleaseLevel    uint8 // copied from toggleOption

	ConfigKeySpace string
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

// Save saves the Subsystem Status to the database.
func (sub *Subsystem) Save() {
	if databaseKeySpace != "" {
		if !sub.KeyIsSet() {
			sub.SetKey(databaseKeySpace + sub.ID)
		}
		err := db.Put(sub)
		if err != nil {
			log.Errorf("subsystems: could not save subsystem status to database: %s", err)
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
