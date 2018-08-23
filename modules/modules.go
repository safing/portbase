// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package modules

import (
	"errors"
	"sync"

	"github.com/tevino/abool"
)

var (
	startComplete = abool.NewBool(false)

	modulesLock  sync.Mutex
	modules      = make(map[string]*Module)
	modulesOrder []*Module

	// ErrCleanExit is returned by Start() when the program is interrupted before starting. This can happen for example, when using the "--help" flag.
	ErrCleanExit = errors.New("clean exit requested")
)

// Module represents a module.
type Module struct {
	Name   string
	Active *abool.AtomicBool

	prep     func() error
	start    func() error
	starting bool
	stop     func() error

	dependencies []string
}

// Register registers a new module.
func Register(name string, prep, start, stop func() error, dependencies ...string) *Module {
	newModule := &Module{
		Name:         name,
		Active:       abool.NewBool(false),
		prep:         prep,
		start:        start,
		stop:         stop,
		dependencies: dependencies,
	}
	modulesLock.Lock()
	defer modulesLock.Unlock()
	modulesOrder = append(modulesOrder, newModule)
	modules[name] = newModule
	return newModule
}
