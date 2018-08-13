// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"testing"

	datastore "github.com/ipfs/go-datastore"
)

type TestingModel struct {
	Base
	Name  string
	Value string
}

var testingModel *TestingModel

func init() {
	RegisterModel(testingModel, func() Model { return new(TestingModel) })
}

func (m *TestingModel) Create(name string) error {
	return m.CreateObject(&Tests, name, m)
}

func (m *TestingModel) CreateInNamespace(namespace string, name string) error {
	testsNamescace := Tests.ChildString(namespace)
	return m.CreateObject(&testsNamescace, name, m)
}

func (m *TestingModel) CreateWithTypeName(namespace string, typeName string, name string) error {
	customNamespace := Tests.ChildString(namespace).ChildString(typeName).Instance(name)

	m.dbKey = &customNamespace
	handleCreateSubscriptions(m)
	return Create(*m.dbKey, m)
}

func (m *TestingModel) Save() error {
	return m.SaveObject(m)
}

func GetTestingModel(name string) (*TestingModel, error) {
	return GetTestingModelFromNamespace(&Tests, name)
}

func GetTestingModelFromNamespace(namespace *datastore.Key, name string) (*TestingModel, error) {
	object, err := GetAndEnsureModel(namespace, name, testingModel)
	if err != nil {
		return nil, err
	}
	model, ok := object.(*TestingModel)
	if !ok {
		return nil, NewMismatchError(object, testingModel)
	}
	return model, nil
}

func TestModel(t *testing.T) {

	// create
	m := TestingModel{
		Name:  "a",
		Value: "b",
	}
	// newKey := datastore.NewKey("/Tests/TestingModel:test")
	// m.dbKey = &newKey
	// err := Put(*m.dbKey, m)
	err := m.Create("")
	if err != nil {
		t.Errorf("database test: could not create object: %s", err)
	}

	// get
	o, err := GetTestingModel(m.dbKey.Name())
	if err != nil {
		t.Errorf("database test: failed to get model: %s (%s)", err, m.dbKey.Name())
	}

	// check fetched object
	if o.Name != "a" || o.Value != "b" {
		t.Errorf("database test: values do not match: got Name=%s and Value=%s", o.Name, o.Value)
	}

	// o, err := Get(*m.dbKey)
	// if err != nil {
	//   t.Errorf("database: could not get object: %s", err)
	// }
	// n, ok := o.(*TestingModel)
	// if !ok {
	//   t.Errorf("database: wrong type, got type %T from %s", o, m.dbKey.String())
	// }

	// save
	o.Value = "c"
	err = o.Save()
	if err != nil {
		t.Errorf("database test: could not save object: %s", err)
	}

	// delete
	err = o.Delete()
	if err != nil {
		t.Errorf("database test: could not delete object: %s", err)
	}

}
