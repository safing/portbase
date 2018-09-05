package model

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Safing/portbase/container"
	"github.com/Safing/portbase/formats/dsd"
	"github.com/Safing/portbase/formats/varint"
)

type Wrapper struct {
	Base
	Format uint8
	Data   []byte
	lock   sync.Mutex
}

func NewRawWrapper(database, key string, data []byte) (*Wrapper, error) {
	version, offset, err := varint.Unpack8(data)
	if version != 1 {
		return nil, fmt.Errorf("incompatible record version: %d", version)
	}

	metaSection, n, err := varint.GetNextBlock(data[offset:])
	if err != nil {
		return nil, fmt.Errorf("could not get meta section: %s", err)
	}
	offset += n

	newMeta := &Meta{}
	_, err = newMeta.GenCodeUnmarshal(metaSection)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal meta section: %s", err)
	}

	format, _, err := varint.Unpack8(data[offset:])
	if err != nil {
		return nil, fmt.Errorf("could not get dsd format: %s", err)
	}

	return &Wrapper{
		Base{
			database,
			key,
			newMeta,
		},
		format,
		data[offset:],
		sync.Mutex{},
	}, nil
}

// NewWrapper returns a new model wrapper for the given data.
func NewWrapper(key string, meta *Meta, data []byte) (*Wrapper, error) {
	format, _, err := varint.Unpack8(data)
	if err != nil {
		return nil, fmt.Errorf("could not get dsd format: %s", err)
	}

	dbName, dbKey := ParseKey(key)

	return &Wrapper{
		Base{
			dbName: dbName,
			dbKey:  dbKey,
			meta:   meta,
		},
		format,
		data,
		sync.Mutex{},
	}, nil
}

// Marshal marshals the object, without the database key or metadata
func (w *Wrapper) Marshal(storageType uint8) ([]byte, error) {
	if storageType != dsd.AUTO && storageType != w.Format {
		return nil, errors.New("could not dump model, wrapped object format mismatch")
	}
	return w.Data, nil
}

// MarshalRecord packs the object, including metadata, into a byte array for saving in a database.
func (w *Wrapper) MarshalRecord() ([]byte, error) {
	// Duplication necessary, as the version from Base would call Base.Marshal instead of Wrapper.Marshal

	if w.Meta() == nil {
		return nil, errors.New("missing meta")
	}

	// version
	c := container.New([]byte{1})

	// meta
	metaSection, err := w.meta.GenCodeMarshal(nil)
	if err != nil {
		return nil, err
	}
	c.AppendAsBlock(metaSection)

	// data
	dataSection, err := w.Marshal(dsd.JSON)
	if err != nil {
		return nil, err
	}
	c.Append(dataSection)

	return c.CompileData(), nil
}

// Lock locks the record.
func (w *Wrapper) Lock() {
	w.lock.Lock()
}

// Unlock unlocks the record.
func (w *Wrapper) Unlock() {
	w.lock.Unlock()
}
