package modules

import (
	"context"
	"errors"
	"fmt"

	"github.com/tevino/abool"

	"github.com/safing/portbase/log"
)

type eventHooks struct {
	// hooks holds all registered hooks for the event.
	hooks []*eventHook

	// internal signifies that the event and it's data may not be exposed and may
	// only be propagated internally.
	internal bool
}

type eventHookFn func(context.Context, interface{}) error

type eventHook struct {
	description   string
	hookingModule *Module
	hookFn        eventHookFn
}

// TriggerEvent executes all hook functions registered to the specified event.
func (m *Module) TriggerEvent(event string, data interface{}) {
	if m.OnlineSoon() {
		go m.processEventTrigger(event, data)
	}
}

func (m *Module) processEventTrigger(event string, data interface{}) {
	m.eventHooksLock.RLock()
	defer m.eventHooksLock.RUnlock()

	eventHooks, ok := m.eventHooks[event]
	if !ok {
		log.Warningf(`%s: tried to trigger non-existent event "%s"`, m.Name, event)
		return
	}

	for _, hook := range eventHooks.hooks {
		if hook.hookingModule.OnlineSoon() {
			go m.runEventHook(hook, event, data)
		}
	}

	// Call subscription function, if set.
	if eventSubscriptionFuncReady.IsSet() {
		m.StartWorker("event subscription", func(context.Context) error {
			// Only use data in worker that won't change anymore.
			eventSubscriptionFunc(m.Name, event, eventHooks.internal, data)
			return nil
		})
	}
}

// InjectEvent triggers an event from a foreign module and executes all hook functions registered to that event.
func (m *Module) InjectEvent(sourceEventName, targetModuleName, targetEventName string, data interface{}) error {
	if !m.OnlineSoon() {
		return errors.New("module not yet started")
	}

	if !modulesLocked.IsSet() {
		return errors.New("module system not yet started")
	}

	targetModule, ok := modules[targetModuleName]
	if !ok {
		return fmt.Errorf(`module "%s" does not exist`, targetModuleName)
	}

	targetModule.eventHooksLock.RLock()
	defer targetModule.eventHooksLock.RUnlock()

	targetHooks, ok := targetModule.eventHooks[targetEventName]
	if !ok {
		return fmt.Errorf(`module "%s" has no event named "%s"`, targetModuleName, targetEventName)
	}

	for _, hook := range targetHooks.hooks {
		if hook.hookingModule.OnlineSoon() {
			go m.runEventHook(hook, sourceEventName, data)
		}
	}

	// Call subscription function, if set.
	if eventSubscriptionFuncReady.IsSet() {
		m.StartWorker("event subscription", func(context.Context) error {
			// Only use data in worker that won't change anymore.
			eventSubscriptionFunc(targetModule.Name, targetEventName, targetHooks.internal, data)
			return nil
		})
	}

	return nil
}

func (m *Module) runEventHook(hook *eventHook, event string, data interface{}) {
	// check if source module is ready for handling
	if m.Status() != StatusOnline {
		// target module has not yet fully started, wait until start is complete
		select {
		case <-m.StartCompleted():
			// continue with hook execution
		case <-hook.hookingModule.Stopping():
			return
		case <-m.Stopping():
			return
		}
	}

	// check if destionation module is ready for handling
	if hook.hookingModule.Status() != StatusOnline {
		// target module has not yet fully started, wait until start is complete
		select {
		case <-hook.hookingModule.StartCompleted():
			// continue with hook execution
		case <-hook.hookingModule.Stopping():
			return
		case <-m.Stopping():
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
// The expose argument controls whether these events and the attached data may
// be received by external components via APIs. If not exposed, the database
// record that carries the event and it's data will be marked as secret and as
// a crown jewel. Enforcement is left to the database layer.
func (m *Module) RegisterEvent(event string, expose bool) {
	m.eventHooksLock.Lock()
	defer m.eventHooksLock.Unlock()

	_, ok := m.eventHooks[event]
	if !ok {
		m.eventHooks[event] = &eventHooks{
			hooks:    make([]*eventHook, 0, 1),
			internal: !expose,
		}
	}
}

// RegisterEventHook registers a hook function with (another) modules' event. Whenever a hook is triggered and the receiving module has not yet fully started, hook execution will be delayed until the modules completed starting.
func (m *Module) RegisterEventHook(module string, event string, description string, fn func(context.Context, interface{}) error) error {
	// get target module
	var eventModule *Module
	if module == m.Name {
		eventModule = m
	} else {
		var ok bool
		eventModule, ok = modules[module]
		if !ok {
			return fmt.Errorf(`module "%s" does not exist`, module)
		}
	}

	// get target event
	eventModule.eventHooksLock.Lock()
	defer eventModule.eventHooksLock.Unlock()
	eventHooks, ok := eventModule.eventHooks[event]
	if !ok {
		return fmt.Errorf(`event "%s/%s" does not exist`, eventModule.Name, event)
	}

	// add hook
	eventHooks.hooks = append(eventHooks.hooks, &eventHook{
		description:   description,
		hookingModule: m,
		hookFn:        fn,
	})
	return nil
}

// Subscribe to all events

var (
	eventSubscriptionFunc        func(moduleName, eventName string, internal bool, data interface{})
	eventSubscriptionFuncEnabled = abool.NewBool(false)
	eventSubscriptionFuncReady   = abool.NewBool(false)
)

// SetEventSubscriptionFunc sets a function that is called for every event.
// This enabled the runtime package to expose events.
func SetEventSubscriptionFunc(fn func(moduleName, eventName string, internal bool, data interface{})) bool {
	if eventSubscriptionFuncEnabled.SetToIf(false, true) {
		eventSubscriptionFunc = fn
		eventSubscriptionFuncReady.Set()
		return true
	}
	return false
}
