// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"testing"

	datastore "github.com/ipfs/go-datastore"
)

func testQuery(t *testing.T, queryString string, expecting []string) {

	entries, err := EasyQuery(queryString)
	if err != nil {
		t.Errorf("error in query %s: %s", queryString, err)
	}

	totalExcepted := len(expecting)
	total := 0
	fail := false

	keys := datastore.EntryKeys(*entries)

resultLoop:
	for _, key := range keys {
		total++
		for _, expectedName := range expecting {
			if key.Name() == expectedName {
				continue resultLoop
			}
		}
		fail = true
		break
	}

	if !fail && total == totalExcepted {
		return
	}

	t.Errorf("Query %s got %s, expected %s", queryString, keys, expecting)

}

func TestEasyQuery(t *testing.T) {

	// setup test data
	(&(TestingModel{})).CreateInNamespace("EasyQuery", "1")
	(&(TestingModel{})).CreateInNamespace("EasyQuery", "2")
	(&(TestingModel{})).CreateInNamespace("EasyQuery", "3")
	(&(TestingModel{})).CreateInNamespace("EasyQuery/A", "4")
	(&(TestingModel{})).CreateInNamespace("EasyQuery/A/B", "5")
	(&(TestingModel{})).CreateInNamespace("EasyQuery/A/B/C", "6")
	(&(TestingModel{})).CreateInNamespace("EasyQuery/A/B/C/D", "7")

	(&(TestingModel{})).CreateWithTypeName("EasyQuery", "ConfigModel", "X")
	(&(TestingModel{})).CreateWithTypeName("EasyQuery", "ConfigModel", "Y")
	(&(TestingModel{})).CreateWithTypeName("EasyQuery/A", "ConfigModel", "Z")

	testQuery(t, "/Tests/EasyQuery/TestingModel", []string{"1", "2", "3"})
	testQuery(t, "/Tests/EasyQuery/TestingModel:1", []string{"1"})

	testQuery(t, "/Tests/EasyQuery/ConfigModel", []string{"X", "Y"})
	testQuery(t, "/Tests/EasyQuery/ConfigModel:Y", []string{"Y"})

	testQuery(t, "/Tests/EasyQuery/A/", []string{"Z", "4", "5", "6", "7"})
	testQuery(t, "/Tests/EasyQuery/A/B/**", []string{"5", "6"})

}
