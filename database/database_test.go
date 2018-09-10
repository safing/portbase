package database

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/Safing/portbase/database/record"
)

type TestRecord struct {
	record.Base
	lock sync.Mutex
	S    string
	I    int
	I8   int8
	I16  int16
	I32  int32
	I64  int64
	UI   uint
	UI8  uint8
	UI16 uint16
	UI32 uint32
	UI64 uint64
	F32  float32
	F64  float64
	B    bool
}

func (tr *TestRecord) Lock() {
}

func (tr *TestRecord) Unlock() {
}

func TestDatabase(t *testing.T) {

	testDir, err := ioutil.TempDir("", "testing-")
	if err != nil {
		t.Fatal(err)
	}

	err = Initialize(testDir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir) // clean up

	err = RegisterDatabase(&RegisteredDatabase{
		Name:        "testing",
		Description: "Unit Test Database",
		StorageType: "badger",
		PrimaryAPI:  "",
	})
	if err != nil {
		t.Fatal(err)
	}

	db := NewInterface(nil)

	new := &TestRecord{}
	new.SetKey("testing:A")
	err = db.Put(new)
	if err != nil {
		t.Fatal(err)
	}

}
