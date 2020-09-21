package modules

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/safing/portbase/log"
)

type (
	// EventObserverFunc can be registered for one or more event types
	// and will be called with the event payload.
	// Any error returned from the observer function will be logged.
	EventObserverFunc func(context.Context, interface{}) error

	// eventHooks keeps track of registered event subscriptions.
	eventHooks struct {
		sync.RWMutex
		subscriptions map[string][]*subscription
	}

	// subscription defines the subscription to an observable.
	// Any time the observable emits an event the subscriptions
	// callback is called.
	subscription struct {
		// description is a human readable description of the
		// subscription purpose. This is mainly used for logging
		// purposes.
		description string
		// subscriber is a reference to the module that placed
		// the subscription.
		subscriber *Module
		// target holds a reference to the module that is
		// observed by this subscription
		target *Module
		// callback is the function to execute when the observed
		// event occurs.
		callback EventObserverFunc
	}
)

// RegisterEvent registers a new event to allow for registering hooks.
func (m *Module) RegisterEvent(event string) {
	m.events.defineEvent(event)
}

// RegisterEventHook registers a hook function with (another) modules'
// event. Whenever a hook is triggered and the receiving module has not
// yet fully started, hook execution will be delayed until the modules
// completed starting.
func (m *Module) RegisterEventHook(module, event, description string, fn EventObserverFunc) error {
	targetModule := m
	if module != m.Name {
		var ok bool
		// TODO(ppacher): accessing modules[module] here without any
		//                kind of protection seems wrong.... Check with
		//                @dhaavi.
		targetModule, ok = modules[module]
		if !ok {
			return fmt.Errorf(`module "%s" does not exist`, module)
		}
	}

	return targetModule.events.addSubscription(targetModule, m, event, description, fn)
}

// TriggerEvent executes all hook functions registered to the
// specified event.
func (m *Module) TriggerEvent(event string, data interface{}) {
	if m.OnlineSoon() {
		go m.processEventTrigger(event, event, data)
	}
}

// InjectEvent triggers an event from a foreign module and executes
// all hook functions registered to that event.
// Note that sourceEventName is only used for logging purposes while
// targetModuleName and targetEventName must actually exist.
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

	targetModule.processEventTrigger(targetEventName, sourceEventName, data)

	return nil
}

func (m *Module) processEventTrigger(eventID, eventName string, data interface{}) {
	m.events.RLock()
	defer m.events.RUnlock()

	hooks, ok := m.events.subscriptions[eventID]
	if !ok {
		log.Warningf(`%s: tried to trigger non-existent event "%s"`, m.Name, eventID)
		return
	}

	for _, hook := range hooks {
		if hook.subscriber.OnlineSoon() {
			go hook.runEventHook(eventName, data)
		}
	}
}

func (hook *subscription) Name(event string) string {
	return fmt.Sprintf("event hook %s/%s -> %s/%s", hook.target.Name, event, hook.subscriber.Name, hook.description)
}

func waitForModule(ctx context.Context, m *Module) bool {
	select {
	case <-ctx.Done():
		return false
	case <-m.StartCompleted():
		return true
	}
}

func (hook *subscription) runEventHook(event string, data interface{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-hook.subscriber.Stopping():
			cancel()
		case <-hook.target.Stopping():
			cancel()
		case <-ctx.Done():
		}
	}()

	// wait for both modules to become online (or shutdown)
	if !waitForModule(ctx, hook.target) || !waitForModule(ctx, hook.subscriber) {
		return
	}

	err := hook.subscriber.RunWorker(
		hook.Name(event),
		func(ctx context.Context) error {
			return hook.callback(ctx, data)
		},
	)
	if err != nil {
		log.Warningf("%s: failed to execute %s: %s", hook.target.Name, hook.Name(event), err)
	}
}

func (hooks *eventHooks) addSubscription(target, subscriber *Module, event, descr string, fn EventObserverFunc) error {
	hooks.Lock()
	defer hooks.Unlock()

	if hooks.subscriptions == nil {
		return fmt.Errorf("unknown event %q", event)
	}

	if _, ok := hooks.subscriptions[event]; !ok {
		return fmt.Errorf("unknown event %q", event)
	}

	hooks.subscriptions[event] = append(
		hooks.subscriptions[event],
		&subscription{
			description: descr,
			subscriber:  subscriber,
			target:      target,
			callback:    fn,
		},
	)

	return nil
}

func (hooks *eventHooks) defineEvent(event string) {
	hooks.Lock()
	defer hooks.Unlock()

	if hooks.subscriptions == nil {
		hooks.subscriptions = make(map[string][]*subscription)
	}

	if _, ok := hooks.subscriptions[event]; !ok {
		hooks.subscriptions[event] = make([]*subscription, 0, 1)
	}
}
