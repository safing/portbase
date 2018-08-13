// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"fmt"
	"strings"
	"sync"

	"github.com/ipfs/go-datastore"

	"github.com/Safing/safing-core/database/dbutils"
	"github.com/Safing/safing-core/formats/dsd"
)

type Model interface {
	SetKey(*datastore.Key)
	GetKey() *datastore.Key
	FmtKey() string
	// Type() string
	// DefaultNamespace() datastore.Key
	// Create(string) error
	// CreateInLocation(datastore.Key, string) error
	// CreateObject(*datastore.Key, string, Model) error
	// Save() error
	// Delete() error
	// CastError(interface{}, interface{}) error
}

func getTypeName(model interface{}) string {
	full := fmt.Sprintf("%T", model)
	return full[strings.LastIndex(full, ".")+1:]
}

func TypeAssertError(model Model, object interface{}) error {
	return fmt.Errorf("database: could not assert %s to type %T (is type %T)", model.FmtKey(), model, object)
}

// Model Registration

var (
	registeredModels     = make(map[string]func() Model)
	registeredModelsLock sync.RWMutex
)

func RegisterModel(model Model, constructor func() Model) {
	registeredModelsLock.Lock()
	defer registeredModelsLock.Unlock()
	registeredModels[fmt.Sprintf("%T", model)] = constructor
}

func NewModel(model Model) (Model, error) {
	registeredModelsLock.RLock()
	defer registeredModelsLock.RUnlock()
	constructor, ok := registeredModels[fmt.Sprintf("%T", model)]
	if !ok {
		return nil, fmt.Errorf("database: cannot create new %T, not registered", model)
	}
	return constructor(), nil
}

func EnsureModel(uncertain, model Model) (Model, error) {
	wrappedObj, ok := uncertain.(*dbutils.Wrapper)
	if !ok {
		return uncertain, nil
	}
	newModel, err := NewModel(model)
	if err != nil {
		return nil, err
	}
	_, err = dsd.Load(wrappedObj.Data, &newModel)
	if err != nil {
		return nil, fmt.Errorf("database: failed to unwrap %T: %s", model, err)
	}
	newModel.SetKey(wrappedObj.GetKey())
	model = newModel
	return newModel, nil
}

func SilentEnsureModel(uncertain, model Model) Model {
	obj, err := EnsureModel(uncertain, model)
	if err != nil {
		return nil
	}
	return obj
}

func NewMismatchError(got, expected interface{}) error {
	return fmt.Errorf("database: entry (%T) does not match expected model (%T)", got, expected)
}
