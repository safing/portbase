// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

/*
Package dbutils provides important function for datastore backends without creating an import loop.
*/
package dbutils

import (
	"errors"
	"fmt"

	"github.com/ipfs/go-datastore"

	"github.com/Safing/safing-core/formats/dsd"
	"github.com/Safing/safing-core/formats/varint"
)

type Wrapper struct {
	dbKey  *datastore.Key
	meta   *Meta
	Format uint8
	Data   []byte
}

func NewWrapper(key *datastore.Key, data []byte) (*Wrapper, error) {
	// line crashes with: panic: runtime error: index out of range
	format, _, err := varint.Unpack8(data)
	if err != nil {
		return nil, fmt.Errorf("database: could not get dsd format: %s", err)
	}

	new := &Wrapper{
		Format: format,
		Data:   data,
	}
	new.SetKey(key)

	return new, nil
}

func (w *Wrapper) SetKey(key *datastore.Key) {
	w.dbKey = key
}

func (w *Wrapper) GetKey() *datastore.Key {
	return w.dbKey
}

func (w *Wrapper) FmtKey() string {
	return w.dbKey.String()
}

func DumpModel(uncertain interface{}, storageType uint8) ([]byte, error) {
	wrapped, ok := uncertain.(*Wrapper)
	if ok {
		if storageType != dsd.AUTO && storageType != wrapped.Format {
			return nil, errors.New("could not dump model, wrapped object format mismatch")
		}
		return wrapped.Data, nil
	}

	dumped, err := dsd.Dump(uncertain, storageType)
	if err != nil {
		return nil, err
	}
	return dumped, nil
}
