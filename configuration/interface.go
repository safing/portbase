// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package configuration

import (
	"sync"
	"sync/atomic"
)

type Interface struct {
	*Configuration

	LastChange int64
	ConfigLock sync.RWMutex
}

func Get() *Interface {
	lock.RLock()
	defer lock.RUnlock()
	return &Interface{
		Configuration: currentConfig,
		LastChange:    atomic.LoadInt64(lastChange),
	}
}

func (lc *Interface) RLock() {
	lc.ConfigLock.RLock()
}

func (lc *Interface) RUnlock() {
	lc.ConfigLock.RUnlock()
}

func (lc *Interface) Changed() bool {
	lastGlobalChange := atomic.LoadInt64(lastChange)
	if lc.LastChange != lastGlobalChange {
		lc.ConfigLock.Lock()
		lock.RLock()
		lc.Configuration = currentConfig
		lc.LastChange = lastGlobalChange
		lock.RUnlock()
		lc.ConfigLock.Unlock()
		return true
	}
	return false
}

func (lc *Interface) SecurityLevel() int8 {
	return int8(atomic.LoadInt32(securityLevel))
}
