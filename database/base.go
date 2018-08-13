// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"errors"
	"strings"

	"github.com/Safing/safing-core/database/dbutils"

	"github.com/ipfs/go-datastore"
	uuid "github.com/satori/go.uuid"
)

type Base struct {
	dbKey *datastore.Key
	meta  *dbutils.Meta
}

func (m *Base) SetKey(key *datastore.Key) {
	m.dbKey = key
}

func (m *Base) GetKey() *datastore.Key {
	return m.dbKey
}

func (m *Base) FmtKey() string {
	return m.dbKey.String()
}

func (m *Base) Meta() *dbutils.Meta {
	return m.meta
}

func (m *Base) CreateObject(namespace *datastore.Key, name string, model Model) error {
	var newKey datastore.Key
	if name == "" {
		newKey = NewInstance(namespace.ChildString(getTypeName(model)), strings.Replace(uuid.NewV4().String(), "-", "", -1))
	} else {
		newKey = NewInstance(namespace.ChildString(getTypeName(model)), name)
	}
	m.dbKey = &newKey
	return Create(*m.dbKey, model)
}

func (m *Base) SaveObject(model Model) error {
	if m.dbKey == nil {
		return errors.New("cannot save new object, use Create() instead")
	}
	return Update(*m.dbKey, model)
}

func (m *Base) Delete() error {
	if m.dbKey == nil {
		return errors.New("cannot delete object unsaved object")
	}
	return Delete(*m.dbKey)
}

func NewInstance(k datastore.Key, s string) datastore.Key {
	return datastore.NewKey(k.String() + ":" + s)
}
