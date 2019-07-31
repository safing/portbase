package modules

import (
	"errors"
	"fmt"
	"sync"

	"github.com/tevino/abool"
)

var (
	modulesLock sync.Mutex
	modules     = make(map[string]*Module)

	// ErrCleanExit is returned by Start() when the program is interrupted before starting. This can happen for example, when using the "--help" flag.
	ErrCleanExit = errors.New("clean exit requested")
)

// Module represents a module.
type Module struct {
	Name         string
	Prepped      *abool.AtomicBool
	Started      *abool.AtomicBool
	Stopped      *abool.AtomicBool
	inTransition *abool.AtomicBool

	prep  func() error
	start func() error
	stop  func() error

	depNames   []string
	depModules []*Module
	depReverse []*Module
}

func dummyAction() error {
	return nil
}

// Register registers a new module.
func Register(name string, prep, start, stop func() error, dependencies ...string) *Module {
	newModule := &Module{
		Name:         name,
		Prepped:      abool.NewBool(false),
		Started:      abool.NewBool(false),
		Stopped:      abool.NewBool(false),
		inTransition: abool.NewBool(false),
		prep:         prep,
		start:        start,
		stop:         stop,
		depNames:     dependencies,
	}

	// replace nil arguments with dummy action
	if newModule.prep == nil {
		newModule.prep = dummyAction
	}
	if newModule.start == nil {
		newModule.start = dummyAction
	}
	if newModule.stop == nil {
		newModule.stop = dummyAction
	}

	modulesLock.Lock()
	defer modulesLock.Unlock()
	modules[name] = newModule
	return newModule
}

func initDependencies() error {
	for _, m := range modules {
		for _, depName := range m.depNames {

			// get dependency
			depModule, ok := modules[depName]
			if !ok {
				return fmt.Errorf("module %s declares dependency \"%s\", but this module has not been registered", m.Name, depName)
			}

			// link together
			m.depModules = append(m.depModules, depModule)
			depModule.depReverse = append(depModule.depReverse, m)

		}
	}

	return nil
}

// ReadyToPrep returns whether all dependencies are ready for this module to prep.
func (m *Module) ReadyToPrep() bool {
	if m.inTransition.IsSet() || m.Prepped.IsSet() {
		return false
	}

	for _, dep := range m.depModules {
		if !dep.Prepped.IsSet() {
			return false
		}
	}

	return true
}

// ReadyToStart returns whether all dependencies are ready for this module to start.
func (m *Module) ReadyToStart() bool {
	if m.inTransition.IsSet() || m.Started.IsSet() {
		return false
	}

	for _, dep := range m.depModules {
		if !dep.Started.IsSet() {
			return false
		}
	}

	return true
}

// ReadyToStop returns whether all dependencies are ready for this module to stop.
func (m *Module) ReadyToStop() bool {
	if !m.Started.IsSet() || m.inTransition.IsSet() || m.Stopped.IsSet() {
		return false
	}

	for _, revDep := range m.depReverse {
		// not ready if a reverse dependency was started, but not yet stopped
		if revDep.Started.IsSet() && !revDep.Stopped.IsSet() {
			return false
		}
	}

	return true
}
