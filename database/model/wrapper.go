package model

import (
	"errors"
	"fmt"

	"github.com/Safing/safing-core/formats/dsd"
	"github.com/Safing/safing-core/formats/varint"
)

type Wrapper struct {
	dbName string
	dbKey  string
	meta   *Meta
	Format uint8
	Data   []byte
}

func NewWrapper(key string, meta *Meta, data []byte) (*Wrapper, error) {
	format, _, err := varint.Unpack8(data)
	if err != nil {
		return nil, fmt.Errorf("database: could not get dsd format: %s", err)
	}

	new := &Wrapper{
		dbKey:  key,
		meta:   meta,
		Format: format,
		Data:   data,
	}

	return new, nil
}

// Key returns the key of the database record.
func (w *Wrapper) Key() string {
	return w.dbKey
}

// SetKey sets the key on the database record, it should only be called after loading the record. Use MoveTo to save the record with another key.
func (w *Wrapper) SetKey(key string) {
	w.dbKey = key
}

// MoveTo sets a new key for the record and resets all metadata, except for the secret and crownjewel status.
func (w *Wrapper) MoveTo(key string) {
	w.dbKey = key
	w.meta.Reset()
}

// Meta returns the metadata object for this record.
func (w *Wrapper) Meta() *Meta {
	return w.meta
}

// SetMeta sets the metadata on the database record, it should only be called after loading the record. Use MoveTo to save the record with another key.
func (w *Wrapper) SetMeta(meta *Meta) {
	w.meta = meta
}

// Marshal marshals the object, without the database key or metadata
func (w *Wrapper) Marshal(storageType uint8) ([]byte, error) {
	if storageType != dsd.AUTO && storageType != w.Format {
		return nil, errors.New("could not dump model, wrapped object format mismatch")
	}
	return w.Data, nil
}
