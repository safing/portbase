package model

import "sync"

type TestModel struct {
	Base
	lock sync.Mutex
}

func (tm *TestModel) Lock() {
	tm.lock.Lock()
}

func (tm *TestModel) Unlock() {
	tm.lock.Unlock()
}
