package model

import "time"

// Meta holds
type Meta struct {
	created   int64
	modified  int64
	expires   int64
	deleted   int64
	secret    bool // secrets must not be sent to the UI, only synced between nodes
	cronjewel bool // crownjewels must never leave the instance, but may be read by the UI
}

// SetAbsoluteExpiry sets an absolute expiry time (in seconds), that is not affected when the record is updated.
func (m *Meta) SetAbsoluteExpiry(seconds int64) {
	m.expires = seconds
	m.deleted = 0
}

// SetRelativateExpiry sets a relative expiry time (ie. TTL in seconds) that is automatically updated whenever the record is updated/saved.
func (m *Meta) SetRelativateExpiry(seconds int64) {
	if seconds >= 0 {
		m.deleted = -seconds
	}
}

// GetAbsoluteExpiry returns the absolute expiry time.
func (m *Meta) GetAbsoluteExpiry() int64 {
	return m.expires
}

// GetRelativeExpiry returns the current relative expiry time - ie. seconds until expiry.
func (m *Meta) GetRelativeExpiry() int64 {
	if m.deleted < 0 {
		return -m.deleted
	}

	abs := m.expires - time.Now().Unix()
	if abs < 0 {
		return 0
	}
	return abs
}

// MakeCrownJewel marks the database records as a crownjewel, meaning that it will not be sent/synced to other devices.
func (m *Meta) MakeCrownJewel() {
	m.cronjewel = true
}

// MakeSecret sets the database record as secret, meaning that it may only be used internally, and not by interfacing processes, such as the UI.
func (m *Meta) MakeSecret() {
	m.secret = true
}

// Update updates the internal meta states and should be called before writing the record to the database.
func (m *Meta) Update() {
	now := time.Now().Unix()
	m.modified = now
	if m.created == 0 {
		m.created = now
	}
	if m.deleted < 0 {
		m.expires = now - m.deleted
	}
}

// Reset resets all metadata, except for the secret and crownjewel status.
func (m *Meta) Reset() {
	m.created = 0
	m.modified = 0
	m.expires = 0
	m.deleted = 0
}

// CheckScope checks whether the current database record exists for the given scope.
func (m *Meta) CheckScope(now int64, local, internal bool) (recordExists bool) {
	switch {
	case m.deleted > 0:
		return false
	case m.expires < now:
		return false
	case !local && m.cronjewel:
		return false
	case !internal && m.secret:
		return false
	default:
		return true
	}
}
