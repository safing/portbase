package subsystems

import (
	"strings"

	"github.com/safing/portbase/database"
	_ "github.com/safing/portbase/database/dbmodule" // database module is required
	"github.com/safing/portbase/modules"
)

const (
	configChangeEvent      = "config change"
	subsystemsStatusChange = "status change"
)

var (
	module *modules.Module

	databaseKeySpace string
	db               = database.NewInterface(nil)
)

func init() {
	// enable partial starting
	modules.EnableModuleManagement(handleModuleChanges)

	// register module and enable it for starting
	module = modules.Register("subsystems", prep, start, nil, "config", "database")
	module.Enable()

	// register event for changes in the subsystem
	module.RegisterEvent(subsystemsStatusChange)
}

func prep() error {
	return module.RegisterEventHook("config", configChangeEvent, "control subsystems", handleConfigChanges)
}

func start() error {
	// lock registration
	subsystemsLocked.Set()

	// lock slice and map
	subsystemsLock.Lock()
	// go through all dependencies
	seen := make(map[string]struct{})
	for _, sub := range subsystems {
		// add main module
		sub.Dependencies = append(sub.Dependencies, statusFromModule(sub.module))
		// add dependencies
		sub.addDependencies(sub.module, seen)
	}
	// unlock
	subsystemsLock.Unlock()

	// apply config
	return handleConfigChanges(module.Ctx, nil)
}

func (sub *Subsystem) addDependencies(module *modules.Module, seen map[string]struct{}) {
	for _, module := range module.Dependencies() {
		_, ok := seen[module.Name]
		if !ok {
			// add dependency to modules
			sub.Dependencies = append(sub.Dependencies, statusFromModule(module))
			// mark as seen
			seen[module.Name] = struct{}{}
			// add further dependencies
			sub.addDependencies(module, seen)
		}
	}
}

// SetDatabaseKeySpace sets a key space where subsystem status
func SetDatabaseKeySpace(keySpace string) {
	if databaseKeySpace == "" {
		databaseKeySpace = keySpace

		if !strings.HasSuffix(databaseKeySpace, "/") {
			databaseKeySpace += "/"
		}
	}
}
