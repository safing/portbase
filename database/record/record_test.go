package model

import "sync"

type TestRecord struct {
	Base
	lock sync.Mutex
}

func (tm *TestRecord) Lock() {
	tm.lock.Lock()
}

func (tm *TestRecord) Unlock() {
	tm.lock.Unlock()
}
