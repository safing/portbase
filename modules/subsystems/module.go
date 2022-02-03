package subsystems

import (
	"context"
	"flag"
	"fmt"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/modules"
	"github.com/safing/portbase/runtime"
)

const configChangeEvent = "config change"

var (
	// DefaultManager is the default subsystem registry.
	DefaultManager *Manager

	module         *modules.Module
	printGraphFlag bool
)

// Register registers a new subsystem. It's like Manager.Register
// but uses DefaultManager and panics on error.
func Register(id, name, description string, module *modules.Module, configKeySpace string, option *config.Option) {
	err := DefaultManager.Register(id, name, description, module, configKeySpace, option)
	if err != nil {
		panic(err)
	}
}

func init() {
	// The subsystem layer takes over module management. Note that
	// no one must have called EnableModuleManagement. Otherwise
	// the subsystem layer will silently fail managing module
	// dependencies!
	// TODO(ppacher): we SHOULD panic here!
	// TASK(#1431)

	modules.EnableModuleManagement(func(m *modules.Module) {
		if DefaultManager == nil {
			return
		}
		DefaultManager.handleModuleUpdate(m)
	})

	module = modules.Register("subsystems", prep, start, nil, "config", "database", "runtime", "base")
	module.Enable()

	// TODO(ppacher): can we create the default registry during prep phase?
	var err error
	DefaultManager, err = NewManager(runtime.DefaultRegistry)
	if err != nil {
		panic("Failed to create default registry: " + err.Error())
	}

	flag.BoolVar(&printGraphFlag, "print-subsystem-graph", false, "print the subsystem module dependency graph")
}

func prep() error {
	if printGraphFlag {
		DefaultManager.PrintGraph()
		return modules.ErrCleanExit
	}

	// We need to listen for configuration changes so we can
	// start/stop dependend modules in case a subsystem is
	// (de-)activated.
	if err := module.RegisterEventHook(
		"config",
		configChangeEvent,
		"control subsystems",
		func(ctx context.Context, _ interface{}) error {
			err := DefaultManager.CheckConfig(ctx)
			if err != nil {
				module.Error(
					"modulemgmt-failed",
					"A Module failed to start",
					fmt.Sprintf("The subsystem framework failed to start or stop one or more modules.\nError: %s\nCheck logs for more information or try to restart.", err),
				)
				return nil
			}
			module.Resolve("modulemgmt-failed")
			return nil
		},
	); err != nil {
		return fmt.Errorf("register event hook: %w", err)
	}

	return nil
}

func start() error {
	// Registration of subsystems is only allowed during
	// preparation. Make sure any further call to Register()
	// panics.
	if err := DefaultManager.Start(); err != nil {
		return err
	}

	module.StartWorker("initial subsystem configuration", DefaultManager.CheckConfig)

	return nil
}

// PrintGraph prints the subsystem and module graph.
func (mng *Manager) PrintGraph() {
	mng.l.RLock()
	defer mng.l.RUnlock()

	fmt.Println("subsystems dependency graph:")

	// unmark subsystems module
	module.Disable()

	// mark roots
	for _, sub := range mng.subsys {
		sub.module.Enable() // mark as tree root
	}

	for _, sub := range mng.subsys {
		printModuleGraph("", sub.module, true)
	}

	fmt.Println("\nsubsystem module groups:")
	_ = start() // no errors for what we need here
	for _, sub := range mng.subsys {
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
