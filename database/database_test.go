package database

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime/pprof"
	"sync"
	"testing"
	"time"

	"github.com/Safing/portbase/database/record"
	_ "github.com/Safing/portbase/database/storage/badger"
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

func makeKey(dbName, key string) string {
	return fmt.Sprintf("%s:%s", dbName, key)
}

func testDatabase(t *testing.T, storageType string) {
	dbName := fmt.Sprintf("testing-%s", storageType)
	_, err := Register(&Database{
		Name:        dbName,
		Description: fmt.Sprintf("Unit Test Database for %s", storageType),
		StorageType: storageType,
		PrimaryAPI:  "",
	})
	if err != nil {
		t.Fatal(err)
	}

	db := NewInterface(nil)

	new := &TestRecord{
		S:    "banana",
		I:    42,
		I8:   42,
		I16:  42,
		I32:  42,
		I64:  42,
		UI:   42,
		UI8:  42,
		UI16: 42,
		UI32: 42,
		UI64: 42,
		F32:  42.42,
		F64:  42.42,
		B:    true,
	}
	new.SetMeta(&record.Meta{})
	new.Meta().Update()
	new.SetKey(makeKey(dbName, "A"))
	err = db.Put(new)
	if err != nil {
		t.Fatal(err)
	}

	exists, err := db.Exists(makeKey(dbName, "A"))
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatalf("record %s should exist!", makeKey(dbName, "A"))
	}

	_, err = db.Get(makeKey(dbName, "A"))
	if err != nil {
		t.Fatal(err)
	}
}

func TestDatabaseSystem(t *testing.T) {

	// panic after 10 seconds, to check for locks
	go func() {
		time.Sleep(10 * time.Second)
		fmt.Println("===== TAKING TOO LONG FOR SHUTDOWN - PRINTING STACK TRACES =====")
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		os.Exit(1)
	}()

	testDir, err := ioutil.TempDir("", "testing-")
	if err != nil {
		t.Fatal(err)
	}

	err = Initialize(testDir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir) // clean up

	testDatabase(t, "badger")

	err = MaintainRecordStates()
	if err != nil {
		t.Fatal(err)
	}

	err = Maintain()
	if err != nil {
		t.Fatal(err)
	}

	err = MaintainThorough()
	if err != nil {
		t.Fatal(err)
	}

	err = Shutdown()
	if err != nil {
		t.Fatal(err)
	}

}
