package subsystems

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/database"
	_ "github.com/safing/portbase/database/dbmodule" // database module is required
	"github.com/safing/portbase/modules"
)

const (
	configChangeEvent      = "config change"
	subsystemsStatusChange = "status change"
)

var (
	module         *modules.Module
	printGraphFlag bool

	databaseKeySpace string
	db               = database.NewInterface(nil)
)

func init() {
	// The subsystem layer takes over module management. Note that
	// no one must have called EnableModuleManagement. Otherwise
	// the subsystem layer will silently fail managing module
	// dependencies!
	// TODO(ppacher): we SHOULD panic here!
	// TASK(#1431)
	modules.EnableModuleManagement(handleModuleChanges)

	module = modules.Register("subsystems", prep, start, nil, "config", "database", "base")
	module.Enable()

	// register event for changes in the subsystem
	module.RegisterEvent(subsystemsStatusChange)

	flag.BoolVar(&printGraphFlag, "print-subsystem-graph", false, "print the subsystem module dependency graph")
}

func prep() error {
	if printGraphFlag {
		printGraph()
		return modules.ErrCleanExit
	}

	// We need to listen for configuration changes so we can
	// start/stop dependend modules in case a subsystem is
	// (de-)activated.
	if err := module.RegisterEventHook(
		"config",
		configChangeEvent,
		"control subsystems",
		handleConfigChanges,
	); err != nil {
		return fmt.Errorf("register event hook: %w", err)
	}

	return nil
}

func start() error {
	// Registration of subsystems is only allowed during
	// preperation. Make sure any further call to Register()
	// panics.
	subsystemsLocked.Set()

	subsystemsLock.Lock()
	defer subsystemsLock.Unlock()

	seen := make(map[string]struct{}, len(subsystems))
	configKeyPrefixes := make(map[string]*Subsystem, len(subsystems))
	// mark all sub-systems as seen. This prevents sub-systems
	// from being added as a sub-systems dependency in addAndMarkDependencies.
	for _, sub := range subsystems {
		seen[sub.module.Name] = struct{}{}
		configKeyPrefixes[sub.ConfigKeySpace] = sub
	}

	// aggregate all modules dependencies (and the subsystem module itself)
	// into the Modules slice. Configuration options form dependened modules
	// will be marked using config.SubsystemAnnotation if not already set.
	for _, sub := range subsystems {
		sub.Modules = append(sub.Modules, statusFromModule(sub.module))
		sub.addDependencies(sub.module, seen)
	}

	// Annotate all configuration options with their respective subsystem.
	config.ForEachOption(func(opt *config.Option) error {
		subsys, ok := configKeyPrefixes[opt.Key]
		if !ok {
			return nil
		}

		// Add a new subsystem annotation is it is not already set!
		opt.AddAnnotation(config.SubsystemAnnotation, subsys.ID)

		return nil
	})

	// apply config
	module.StartWorker("initial subsystem configuration", func(ctx context.Context) error {
		return handleConfigChanges(module.Ctx, nil)
	})
	return nil
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

// SetDatabaseKeySpace sets a key space where subsystem status
func SetDatabaseKeySpace(keySpace string) {
	if databaseKeySpace == "" {
		databaseKeySpace = keySpace

		if !strings.HasSuffix(databaseKeySpace, "/") {
			databaseKeySpace += "/"
		}
	}
}

func printGraph() {
	fmt.Println("subsystems dependency graph:")

	// unmark subsystems module
	module.Disable()
	// mark roots
	for _, sub := range subsystems {
		sub.module.Enable() // mark as tree root
	}

	for _, sub := range subsystems {
		printModuleGraph("", sub.module, true)
	}

	fmt.Println("\nsubsystem module groups:")
	_ = start() // no errors for what we need here
	for _, sub := range subsystems {
		fmt.Printf("├── %s\n", sub.Name)
		for _, mod := range sub.Modules[1:] {
			fmt.Printf("│   ├── %s\n", mod.Name)
		}
	}
}

func printModuleGraph(prefix string, module *modules.Module, root bool) {
	fmt.Printf("%s├── %s\n", prefix, module.Name)
	if root || !module.Enabled() {
		for _, dep := range module.Dependencies() {
			printModuleGraph(fmt.Sprintf("│   %s", prefix), dep, false)
		}
	}
}
