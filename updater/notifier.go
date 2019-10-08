package updater

import (
	"github.com/tevino/abool"
)

type notifier struct {
	upgradeAvailable *abool.AtomicBool
	notifyChannel    chan struct{}
}

func newNotifier() *notifier {
	return &notifier{
		upgradeAvailable: abool.NewBool(false),
		notifyChannel:    make(chan struct{}),
	}
}

func (n *notifier) markAsUpgradeable() {
	if n.upgradeAvailable.SetToIf(false, true) {
		close(n.notifyChannel)
	}
}

// UpgradeAvailable returns whether an upgrade is available for this file.
func (file *File) UpgradeAvailable() bool {
	return file.notifier.upgradeAvailable.IsSet()
}

// WaitForAvailableUpgrade blocks (selectable) until an upgrade for this file is available.
func (file *File) WaitForAvailableUpgrade() <-chan struct{} {
	return file.notifier.notifyChannel
}

// registry wide change notifications

func (reg *ResourceRegistry) notifyOfChanges() {
	if !reg.notifyHooksEnabled.IsSet() {
		return
	}

	reg.RLock()
	defer reg.RUnlock()

	for _, hook := range reg.notifyHooks {
		go hook()
	}
}

// RegisterNotifyHook registers a function that is called (as a goroutine) every time the resource registry changes.
func (reg *ResourceRegistry) RegisterNotifyHook(fn func()) {
	reg.Lock()
	defer reg.Unlock()

	reg.notifyHooks = append(reg.notifyHooks, fn)
}
