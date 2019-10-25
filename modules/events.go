package modules

import (
	"context"
	"fmt"

	"github.com/safing/portbase/log"
)

type eventHookFn func(context.Context, interface{}) error

type eventHook struct {
	description   string
	hookingModule *Module
	hookFn        eventHookFn
}

// TriggerEvent executes all hook functions registered to the specified event.
func (m *Module) TriggerEvent(event string, data interface{}) {
	go m.processEventTrigger(event, data)
}

func (m *Module) processEventTrigger(event string, data interface{}) {
	m.eventHooksLock.RLock()
	defer m.eventHooksLock.RUnlock()

	hooks, ok := m.eventHooks[event]
	if !ok {
		log.Warningf(`%s: tried to trigger non-existent event "%s"`, m.Name, event)
		return
	}

	for _, hook := range hooks {
		if !hook.hookingModule.ShutdownInProgress() {
			go m.runEventHook(hook, event, data)
		}
	}
}

func (m *Module) runEventHook(hook *eventHook, event string, data interface{}) {
	if !hook.hookingModule.Started.IsSet() {
		// target module has not yet fully started, wait until start is complete
		select {
		case <-startCompleteSignal:
		case <-shutdownSignal:
			return
		}
	}

	err := hook.hookingModule.RunWorker(
		fmt.Sprintf("event hook %s/%s -> %s/%s", m.Name, event, hook.hookingModule.Name, hook.description),
		func(ctx context.Context) error {
			return hook.hookFn(ctx, data)
		},
	)
	if err != nil {
		log.Warningf("%s: failed to execute event hook %s/%s -> %s/%s: %s", hook.hookingModule.Name, m.Name, event, hook.hookingModule.Name, hook.description, err)
	}
}

// RegisterEvent registers a new event to allow for registering hooks.
func (m *Module) RegisterEvent(event string) {
	m.eventHooksLock.Lock()
	defer m.eventHooksLock.Unlock()

	_, ok := m.eventHooks[event]
	if !ok {
		m.eventHooks[event] = make([]*eventHook, 0, 1)
	}
}

// RegisterEventHook registers a hook function with (another) modules' event. Whenever a hook is triggered and the receiving module has not yet fully started, hook execution will be delayed until all modules completed starting.
func (m *Module) RegisterEventHook(module string, event string, description string, fn func(context.Context, interface{}) error) error {
	// get target module
	var eventModule *Module
	if module == m.Name {
		eventModule = m
	} else {
		var ok bool
		modulesLock.RLock()
		eventModule, ok = modules[module]
		modulesLock.RUnlock()
		if !ok {
			return fmt.Errorf(`module "%s" does not exist`, module)
		}
	}

	// get target event
	eventModule.eventHooksLock.Lock()
	defer eventModule.eventHooksLock.Unlock()
	hooks, ok := eventModule.eventHooks[event]
	if !ok {
		return fmt.Errorf(`event "%s/%s" does not exist`, eventModule.Name, event)
	}

	// add hook
	eventModule.eventHooks[event] = append(hooks, &eventHook{
		description:   description,
		hookingModule: m,
		hookFn:        fn,
	})
	return nil
}
