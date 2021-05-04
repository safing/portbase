package notifications

import (
	"github.com/safing/portbase/log"
	"github.com/safing/portbase/modules"
)

// AttachToModule attaches the notification to a module and changes to the
// notification will be reflected on the module failure status.
func (n *Notification) AttachToModule(m *modules.Module) {
	log.Errorf("notifications: attaching %q", n.EventID)

	if m == nil {
		log.Warningf("notifications: cannot remove attached module from notification %s", n.EventID)
		return
	}

	n.lock.Lock()
	defer n.lock.Unlock()

	if n.State != Active {
		log.Warningf("notifications: cannot attach module to inactive notification %s", n.EventID)
		return
	}
	if n.belongsTo != nil {
		log.Warningf("notifications: cannot override attached module for notification %s", n.EventID)
		return
	}

	// Attach module.
	n.belongsTo = m

	// Set module failure status.
	switch n.Type { //nolint:exhaustive
	case Info:
		m.Hint(n.EventID, n.Title, n.Message)
	case Warning:
		m.Warning(n.EventID, n.Title, n.Message)
	case Error:
		m.Error(n.EventID, n.Title, n.Message)
	default:
		log.Warningf("notifications: incompatible type for attaching to module in notification %s", n.EventID)
		m.Error(n.EventID, n.Title, n.Message+" [incompatible notification type]")
	}
}

// resolveModuleFailure removes the notification from the module failure status.
func (n *Notification) resolveModuleFailure() {
	log.Errorf("notifications: resolving %q", n.EventID)

	if n.belongsTo != nil {
		// Resolve failure in attached module.
		n.belongsTo.Resolve(n.EventID)

		// Reset attachment in order to mitigate duplicate failure resolving.
		// Re-attachment is prevented by the state check when attaching.
		n.belongsTo = nil
	}
}

func init() {
	modules.SetFailureUpdateNotifyFunc(mirrorModuleStatus)
}

func mirrorModuleStatus(moduleFailure uint8, id, title, msg string) {
	log.Errorf("notifications: mirroring %d %q %q %q", moduleFailure, id, title, msg)

	// Ignore "resolve all" requests.
	if id == "" {
		return
	}

	// Get notification from storage.
	n, ok := getNotification(id)
	if ok {
		// The notification already exists.

		// Check if we should delete it.
		if moduleFailure == modules.FailureNone {
			n.Delete()
		}

		return
	}

	// A notification for the given ID does not yet exists, create it.
	switch moduleFailure {
	case modules.FailureHint:
		NotifyInfo(id, title, msg)
	case modules.FailureWarning:
		NotifyWarn(id, title, msg)
	case modules.FailureError:
		NotifyError(id, title, msg)
	}
}
