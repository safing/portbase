package database

import (
	"github.com/Safing/portbase/database/record"
)

// HookBase implements the Hook interface.
type HookBase struct {
}

// PreGet implements the Hook interface.
func (b *HookBase) PreGet(dbKey string) error {
	return nil
}

// PostGet implements the Hook interface.
func (b *HookBase) PostGet(r record.Record) (record.Record, error) {
	return r, nil
}

// PrePut implements the Hook interface.
func (b *HookBase) PrePut(r record.Record) (record.Record, error) {
	return r, nil
}

// PostPut implements the Hook interface.
func (b *HookBase) PostPut(r record.Record) {
	return
}
