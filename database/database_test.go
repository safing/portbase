package database

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"runtime/pprof"
	"testing"
	"time"

	q "github.com/Safing/portbase/database/query"
	_ "github.com/Safing/portbase/database/storage/badger"
)

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

	A := NewExample(makeKey(dbName, "A"), "Herbert", 411)
	err = A.Save()
	if err != nil {
		t.Fatal(err)
	}

	B := NewExample(makeKey(dbName, "B"), "Fritz", 347)
	err = B.Save()
	if err != nil {
		t.Fatal(err)
	}

	C := NewExample(makeKey(dbName, "C"), "Norbert", 217)
	err = C.Save()
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

	A1, err := GetExample(makeKey(dbName, "A"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(A, A1) {
		log.Fatalf("A and A1 mismatch, A1: %v", A1)
	}

	query, err := q.New(dbName).Where(
		q.And(
			q.Where("Name", q.EndsWith, "bert"),
			q.Where("Score", q.GreaterThan, 100),
		),
	).Check()
	if err != nil {
		t.Fatal(err)
	}

	it, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}

	cnt := 0
	for _ = range it.Next {
		cnt++
	}
	if it.Error != nil {
		t.Fatal(it.Error)
	}
	if cnt != 2 {
		t.Fatal("expected two records")
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
