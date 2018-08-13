// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

/*
Package database provides a universal interface for interacting with the database.

The Lazy Database

The database system can handle Go structs as well as serialized data by the dsd package.
While data is in transit within the system, it does not know which form it currently has. Only when it reaches its destination, it must ensure that it is either of a certain type or dump it.

Internals

The database system uses the Model interface to transparently handle all types of structs that get saved in the database. Structs include Base struct to fulfill most parts of the Model interface.

Boilerplate Code

Receiving model, using as struct:

  // At some point, declare a pointer to your model.
  // This is only used to identify the model, so you can reuse it safely for this purpose
  var cowModel *Cow // only use this as parameter for database.EnsureModel-like functions

  receivedModel := <- models // chan database.Model
  cow, ok := database.SilentEnsureModel(receivedModel, cowModel).(*Cow)
  if !ok {
    panic("received model does not match expected model")
  }

  // more verbose, in case you need better error handling
  receivedModel := <- models // chan database.Model
  genericModel, err := database.EnsureModel(receivedModel, cowModel)
  if err != nil {
    panic(err)
  }
  cow, ok := genericModel.(*Cow)
  if !ok {
    panic("received model does not match expected model")
  }

Receiving a model, dumping:

  // receivedModel <- chan database.Model
  bytes, err := database.DumpModel(receivedModel, dsd.JSON) // or other dsd format
  if err != nil {
    panic(err)
  }

Model definition:

  // Cow makes moo.
  type Cow struct {
    database.Base
    // Fields...
  }

  var cowModel *Cow // only use this as parameter for database.EnsureModel-like functions

  func init() {
    database.RegisterModel(cowModel, func() database.Model { return new(Cow) })
  }

  // this all you need, but you might find the following code helpful:

  var cowNamespace = datastore.NewKey("/Cow")

  // Create saves Cow with the provided name in the default namespace.
  func (m *Cow) Create(name string) error {
    return m.CreateObject(&cowNamespace, name, m)
  }

  // CreateInNamespace saves Cow with the provided name in the provided namespace.
  func (m *Cow) CreateInNamespace(namespace *datastore.Key, name string) error {
    return m.CreateObject(namespace, name, m)
  }

  // Save saves Cow.
  func (m *Cow) Save() error {
    return m.SaveObject(m)
  }

  // GetCow fetches Cow with the provided name from the default namespace.
  func GetCow(name string) (*Cow, error) {
    return GetCowFromNamespace(&cowNamespace, name)
  }

  // GetCowFromNamespace fetches Cow with the provided name from the provided namespace.
  func GetCowFromNamespace(namespace *datastore.Key, name string) (*Cow, error) {
    object, err := database.GetAndEnsureModel(namespace, name, cowModel)
    if err != nil {
      return nil, err
    }
    model, ok := object.(*Cow)
    if !ok {
      return nil, database.NewMismatchError(object, cowModel)
    }
    return model, nil
  }

*/
package database
