package modules

import "sync/atomic"

// Status holds an exported status summary of the modules system.
type Status struct {
	Modules map[string]*ModuleStatus
	Total   struct {
		Workers         int
		Tasks           int
		MicroTasks      int
		CtrlFuncRunning int
	}
	Config struct {
		MicroTasksThreshhold int
		MediumPriorityDelay  string
		LowPriorityDelay     string
	}
}

// ModuleStatus holds an exported status summary of one module.
type ModuleStatus struct { //nolint:maligned
	Enabled bool

	Status      string
	FailureType string
	FailureID   string
	FailureMsg  string

	Workers         int
	Tasks           int
	MicroTasks      int
	CtrlFuncRunning bool
}

// GetStatus exports status data from the module system.
func GetStatus() *Status {
	// Check if modules have been initialized.
	if modulesLocked.IsNotSet() {
		return nil
	}

	// Create new status.
	status := &Status{
		Modules: make(map[string]*ModuleStatus, len(modules)),
	}

	// Add config.
	status.Config.MicroTasksThreshhold = int(atomic.LoadInt32(microTasksThreshhold))
	status.Config.MediumPriorityDelay = defaultMediumPriorityMaxDelay.String()
	status.Config.LowPriorityDelay = defaultLowPriorityMaxDelay.String()

	// Gather status data.
	for name, module := range modules {
		moduleStatus := &ModuleStatus{
			Enabled:         module.Enabled(),
			Status:          getStatusName(module.Status()),
			Workers:         int(atomic.LoadInt32(module.workerCnt)),
			Tasks:           int(atomic.LoadInt32(module.taskCnt)),
			MicroTasks:      int(atomic.LoadInt32(module.microTaskCnt)),
			CtrlFuncRunning: module.ctrlFuncRunning.IsSet(),
		}

		// Add failure status.
		failureStatus, failureID, failureMsg := module.FailureStatus()
		moduleStatus.FailureType = getFailureStatusName(failureStatus)
		moduleStatus.FailureID = failureID
		moduleStatus.FailureMsg = failureMsg

		// Add to total counts.
		status.Total.Workers += moduleStatus.Workers
		status.Total.Tasks += moduleStatus.Tasks
		status.Total.MicroTasks += moduleStatus.MicroTasks
		if moduleStatus.CtrlFuncRunning {
			status.Total.CtrlFuncRunning++
		}

		// Add to export.
		status.Modules[name] = moduleStatus
	}

	return status
}

func getStatusName(status uint8) string {
	switch status {
	case StatusDead:
		return "dead"
	case StatusPreparing:
		return "preparing"
	case StatusOffline:
		return "offline"
	case StatusStopping:
		return "stopping"
	case StatusStarting:
		return "starting"
	case StatusOnline:
		return "online"
	default:
		return "unknown"
	}
}

func getFailureStatusName(status uint8) string {
	switch status {
	case FailureNone:
		return ""
	case FailureHint:
		return "hint"
	case FailureWarning:
		return "warning"
	case FailureError:
		return "error"
	default:
		return "unknown"
	}
}
