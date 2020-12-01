package subsystems

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/modules"
	"github.com/safing/portbase/runtime"
	"github.com/tevino/abool"
)

var (
	// ErrManagerStarted is returned when subsystem registration attempt
	// occurs after the manager has been started.
	ErrManagerStarted = errors.New("subsystem manager already started")
	// ErrDuplicateSubsystem is returned when the subsystem to be registered
	// is alreadey known (duplicated subsystem ID).
	ErrDuplicateSubsystem = errors.New("subsystem is already registered")
)

// Manager manages subsystems, provides access via a runtime
// value providers and can takeover module management.
type Manager struct {
	l              sync.RWMutex
	subsys         map[string]*Subsystem
	pushUpdate     runtime.PushFunc
	immutable      *abool.AtomicBool
	debounceUpdate *abool.AtomicBool
	runtime        *runtime.Registry
}

// NewManager returns a new subsystem manager that registers
// itself at rtReg.
func NewManager(rtReg *runtime.Registry) (*Manager, error) {
	mng := &Manager{
		subsys:         make(map[string]*Subsystem),
		immutable:      abool.New(),
		debounceUpdate: abool.New(),
	}

	push, err := rtReg.Register("subsystems/", runtime.SimpleValueGetterFunc(mng.Get))
	if err != nil {
		return nil, err
	}

	mng.pushUpdate = push
	mng.runtime = rtReg

	return mng, nil
}

// Start starts managing subsystems. Note that it's not possible
// to define new subsystems once Start() has been called.
func (mng *Manager) Start() error {
	mng.immutable.Set()

	seen := make(map[string]struct{}, len(mng.subsys))
	configKeyPrefixes := make(map[string]*Subsystem, len(mng.subsys))
	// mark all sub-systems as seen. This prevents sub-systems
	// from being added as a sub-systems dependency in addAndMarkDependencies.
	for _, sub := range mng.subsys {
		seen[sub.module.Name] = struct{}{}
		configKeyPrefixes[sub.ConfigKeySpace] = sub
	}

	// aggregate all modules dependencies (and the subsystem module itself)
	// into the Modules slice. Configuration options form dependent modules
	// will be marked using config.SubsystemAnnotation if not already set.
	for _, sub := range mng.subsys {
		sub.Modules = append(sub.Modules, statusFromModule(sub.module))
		sub.addDependencies(sub.module, seen)
	}

	// Annotate all configuration options with their respective subsystem.
	_ = config.ForEachOption(func(opt *config.Option) error {
		subsys, ok := configKeyPrefixes[opt.Key]
		if !ok {
			return nil
		}

		// Add a new subsystem annotation is it is not already set!
		opt.AddAnnotation(config.SubsystemAnnotation, subsys.ID)

		return nil
	})

	return nil
}

// Get implements runtime.ValueProvider
func (mng *Manager) Get(keyOrPrefix string) ([]record.Record, error) {
	mng.l.RLock()
	defer mng.l.RUnlock()

	dbName := mng.runtime.DatabaseName()
	records := make([]record.Record, 0, len(mng.subsys))
	for _, subsys := range mng.subsys {
		subsys.Lock()
		if !subsys.KeyIsSet() {
			subsys.SetKey(dbName + ":subsystems/" + subsys.ID)
		}
		if strings.HasPrefix(subsys.DatabaseKey(), keyOrPrefix) {
			records = append(records, subsys)
		}
		subsys.Unlock()
	}

	// make sure the order is always the same
	sort.Sort(bySubsystemID(records))

	return records, nil
}

// Register registers a new subsystem. The given option must be a bool option.
// Should be called in init() directly after the modules.Register() function.
// The config option must not yet be registered and will be registered for
// you. Pass a nil option to force enable.
//
// TODO(ppacher): IMHO the subsystem package is not responsible of registering
//                the "toggle option". This would also remove runtime
//                dependency to the config package. Users should either pass
//                the BoolOptionFunc and the expertise/release level directly
//                or just pass the configuration key so those information can
//                be looked up by the registry.
func (mng *Manager) Register(id, name, description string, module *modules.Module, configKeySpace string, option *config.Option) error {
	mng.l.Lock()
	defer mng.l.Unlock()

	if mng.immutable.IsSet() {
		return ErrManagerStarted
	}

	if _, ok := mng.subsys[id]; ok {
		return ErrDuplicateSubsystem
	}

	s := &Subsystem{
		ID:             id,
		Name:           name,
		Description:    description,
		ConfigKeySpace: configKeySpace,
		module:         module,
		toggleOption:   option,
	}

	s.CreateMeta()

	if s.toggleOption != nil {
		s.ToggleOptionKey = s.toggleOption.Key
		s.ExpertiseLevel = s.toggleOption.ExpertiseLevel
		s.ReleaseLevel = s.toggleOption.ReleaseLevel

		if err := config.Register(s.toggleOption); err != nil {
			return fmt.Errorf("failed to register subsystem option: %w", err)
		}

		s.toggleValue = config.GetAsBool(s.ToggleOptionKey, false)
	} else {
		s.toggleValue = func() bool { return true }
	}

	mng.subsys[id] = s

	return nil
}

func (mng *Manager) shouldServeUpdates() bool {
	if !mng.immutable.IsSet() {
		// the manager must be marked as immutable before we
		// are going to handle any module changes.
		return false
	}
	if modules.IsShuttingDown() {
		// we don't care if we are shutting down anyway
		return false
	}
	return true
}

// CheckConfig checks subsystem configuration values and enables
// or disables subsystems and their dependencies as required.
func (mng *Manager) CheckConfig(ctx context.Context) error {
	// DEBUG SNIPPET
	// Slow-start for non-attributable performance issues.
	// You'll need the snippet in the modules too.
	// time.Sleep(11 * time.Second)
	// END DEBUG SNIPPET
	return mng.handleConfigChanges(ctx)
}

func (mng *Manager) handleModuleUpdate(m *modules.Module) {
	if !mng.shouldServeUpdates() {
		return
	}

	// Read lock is fine as the subsystems are write-locked on their own
	mng.l.RLock()
	defer mng.l.RUnlock()

	subsys, ms := mng.findParentSubsystem(m)
	if subsys == nil {
		// the updated module is not handled by any
		// subsystem. We're done here.
		return
	}

	subsys.Lock()
	defer subsys.Unlock()

	updated := compareAndUpdateStatus(m, ms)
	if updated {
		subsys.makeSummary()
	}

	if updated {
		mng.pushUpdate(subsys)
	}
}

func (mng *Manager) handleConfigChanges(_ context.Context) error {
	if !mng.shouldServeUpdates() {
		return nil
	}

	if mng.debounceUpdate.SetToIf(false, true) {
		time.Sleep(100 * time.Millisecond)
		mng.debounceUpdate.UnSet()
	} else {
		return nil
	}

	mng.l.RLock()
	defer mng.l.RUnlock()

	var changed bool
	for _, subsystem := range mng.subsys {
		if subsystem.module.SetEnabled(subsystem.toggleValue()) {
			changed = true
		}
	}
	if !changed {
		return nil
	}

	return modules.ManageModules()
}

func (mng *Manager) findParentSubsystem(m *modules.Module) (*Subsystem, *ModuleStatus) {
	for _, subsys := range mng.subsys {
		for _, ms := range subsys.Modules {
			if ms.Name == m.Name {
				return subsys, ms
			}
		}
	}
	return nil, nil
}

// helper type to sort a slice of []*Subsystem (casted as []record.Record) by
// id. Only use if it's guaranteed that all record.Records are *Subsystem.
// Otherwise Less() will panic.
type bySubsystemID []record.Record

func (sl bySubsystemID) Less(i, j int) bool { return sl[i].(*Subsystem).ID < sl[j].(*Subsystem).ID }
func (sl bySubsystemID) Swap(i, j int)      { sl[i], sl[j] = sl[j], sl[i] }
func (sl bySubsystemID) Len() int           { return len(sl) }
