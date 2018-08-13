// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"testing"

	"github.com/Safing/safing-core/formats/dsd"
)

func TestWrapper(t *testing.T) {

	// create Model
	new := &TestingModel{
		Name:  "a",
		Value: "b",
	}
	newTwo := &TestingModel{
		Name:  "c",
		Value: "d",
	}

	// dump
	bytes, err := DumpModel(new, dsd.JSON)
	if err != nil {
		panic(err)
	}
	bytesTwo, err := DumpModel(newTwo, dsd.JSON)
	if err != nil {
		panic(err)
	}

	// wrap
	wrapped, err := NewWrapper(nil, bytes)
	if err != nil {
		panic(err)
	}
	wrappedTwo, err := NewWrapper(nil, bytesTwo)
	if err != nil {
		panic(err)
	}

	// model definition for unwrapping
	var model *TestingModel

	// unwrap
	myModel, ok := SilentEnsureModel(wrapped, model).(*TestingModel)
	if !ok {
		panic("received model does not match expected model")
	}
	if myModel.Name != "a" || myModel.Value != "b" {
		panic("model value mismatch")
	}

	// verbose unwrap
	genericModel, err := EnsureModel(wrappedTwo, model)
	if err != nil {
		panic(err)
	}
	myModelTwo, ok := genericModel.(*TestingModel)
	if !ok {
		panic("received model does not match expected model")
	}
	if myModelTwo.Name != "c" || myModelTwo.Value != "d" {
		panic("model value mismatch")
	}

}
