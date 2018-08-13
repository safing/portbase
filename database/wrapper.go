// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"github.com/ipfs/go-datastore"

	"github.com/Safing/safing-core/database/dbutils"
)

func NewWrapper(key *datastore.Key, data []byte) (*dbutils.Wrapper, error) {
	return dbutils.NewWrapper(key, data)
}

func DumpModel(uncertain interface{}, storageType uint8) ([]byte, error) {
	return dbutils.DumpModel(uncertain, storageType)
}
