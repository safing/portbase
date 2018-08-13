// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package modules

import (
	"container/list"
	"os"
	"time"

	"github.com/tevino/abool"
)

var modules *list.List
var addModule chan *Module
var GlobalShutdown chan struct{}
var loggingActive bool

type Module struct {
	Name  string
	Order uint8

	Start         chan struct{}
	Active        *abool.AtomicBool
	startComplete chan struct{}

	Stop         chan struct{}
	Stopped      *abool.AtomicBool
	stopComplete chan struct{}
}

func Register(name string, order uint8) *Module {
	newModule := &Module{
		Name:          name,
		Order:         order,
		Start:         make(chan struct{}),
		Active:        abool.NewBool(true),
		startComplete: make(chan struct{}),

		Stop:         make(chan struct{}),
		Stopped:      abool.NewBool(false),
		stopComplete: make(chan struct{}),
	}
	addModule <- newModule
	return newModule
}

func (module *Module) addToList() {
	if loggingActive {
		logger.Infof("Modules: starting %s", module.Name)
	}
	for e := modules.Back(); e != nil; e = e.Prev() {
		if module.Order > e.Value.(*Module).Order {
			modules.InsertAfter(module, e)
			return
		}
	}
	modules.PushFront(module)
}

func (module *Module) stop() {
	module.Active.UnSet()
	defer module.Stopped.Set()
	for {
		select {
		case module.Stop <- struct{}{}:
		case <-module.stopComplete:
			return
		case <-time.After(1 * time.Second):
			if loggingActive {
				logger.Warningf("Modules: waiting for %s to stop...", module.Name)
			}
		}
	}
}

func (module *Module) StopComplete() {
	if loggingActive {
		logger.Warningf("Modules: stopped %s", module.Name)
	}
	module.stopComplete <- struct{}{}
}

func (module *Module) start() {
	module.Stopped.UnSet()
	defer module.Active.Set()
	for {
		select {
		case module.Start <- struct{}{}:
		case <-module.startComplete:
			return
		}
	}
}

func (module *Module) StartComplete() {
	if loggingActive {
		logger.Infof("Modules: starting %s", module.Name)
	}
	module.startComplete <- struct{}{}
}

func InitiateFullShutdown() {
	close(GlobalShutdown)
}

func fullStop() {
	for e := modules.Back(); e != nil; e = e.Prev() {
		if e.Value.(*Module).Active.IsSet() {
			e.Value.(*Module).stop()
		}
	}
}

func run() {
	select {
	case <-loggerRegistered:
		logger.Info("Modules: starting")
		loggingActive = true
	case <-time.After(1 * time.Second):
	}

	for {
		select {
		case <-GlobalShutdown:
			if loggingActive {
				logger.Warning("Modules: stopping")
			}
			fullStop()
			os.Exit(0)
		case m := <-addModule:
			m.addToList()
			// go m.start()
		}
	}
}

func init() {

	modules = list.New()
	addModule = make(chan *Module, 10)
	GlobalShutdown = make(chan struct{})
	loggerRegistered = make(chan struct{}, 1)
	loggingActive = false

	go run()

}
