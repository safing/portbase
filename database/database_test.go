package database

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/safing/portbase/database/storage"

	q "github.com/safing/portbase/database/query"
	"github.com/safing/portbase/database/record"

	_ "github.com/safing/portbase/database/storage/badger"
	_ "github.com/safing/portbase/database/storage/bbolt"
	_ "github.com/safing/portbase/database/storage/fstree"
	_ "github.com/safing/portbase/database/storage/hashmap"
)

func makeKey(dbName, key string) string {
	return fmt.Sprintf("%s:%s", dbName, key)
}

func testDatabase(t *testing.T, storageType string, testPutMany, testRecordMaintenance bool) { //nolint:gocognit,gocyclo
	t.Run(fmt.Sprintf("TestStorage_%s", storageType), func(t *testing.T) {
		dbName := fmt.Sprintf("testing-%s", storageType)
		fmt.Println(dbName)
		_, err := Register(&Database{
			Name:        dbName,
			Description: fmt.Sprintf("Unit Test Database for %s", storageType),
			StorageType: storageType,
			PrimaryAPI:  "",
		})
		if err != nil {
			t.Fatal(err)
		}
		dbController, err := getController(dbName)
		if err != nil {
			t.Fatal(err)
		}

		// hook
		hook, err := RegisterHook(q.New(dbName).MustBeValid(), &HookBase{})
		if err != nil {
			t.Fatal(err)
		}

		// interface
		db := NewInterface(&Options{
			Local:    true,
			Internal: true,
		})

		// sub
		sub, err := db.Subscribe(q.New(dbName).MustBeValid())
		if err != nil {
			t.Fatal(err)
		}

		A := NewExample(dbName+":A", "Herbert", 411)
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

		cnt := countRecords(t, db, q.New(dbName).Where(
			q.And(
				q.Where("Name", q.EndsWith, "bert"),
				q.Where("Score", q.GreaterThan, 100),
			),
		))
		if cnt != 2 {
			t.Fatalf("expected two records, got %d", cnt)
		}

		// test putmany
		if testPutMany {
			batchPut := db.PutMany(dbName)
			records := []record.Record{A, B, C, nil} // nil is to signify finish
			for _, r := range records {
				err = batchPut(r)
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		// test maintenance
		if testRecordMaintenance {
			now := time.Now().UTC()
			nowUnix := now.Unix()

			// we start with 3 records without expiry
			cnt := countRecords(t, db, q.New(dbName))
			if cnt != 3 {
				t.Fatalf("expected three records, got %d", cnt)
			}
			// delete entry
			A.Meta().Deleted = nowUnix - 61
			err = A.Save()
			if err != nil {
				t.Fatal(err)
			}
			// expire entry
			B.Meta().Expires = nowUnix - 1
			err = B.Save()
			if err != nil {
				t.Fatal(err)
			}

			// one left
			cnt = countRecords(t, db, q.New(dbName))
			if cnt != 1 {
				t.Fatalf("expected one record, got %d", cnt)
			}

			// run maintenance
			err = dbController.MaintainRecordStates(context.TODO(), now.Add(-60*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			// one left
			cnt = countRecords(t, db, q.New(dbName))
			if cnt != 1 {
				t.Fatalf("expected one record, got %d", cnt)
			}

			// check status individually
			_, err = dbController.storage.Get("A")
			if err != storage.ErrNotFound {
				t.Errorf("A should be deleted and purged, err=%s", err)
			}
			B1, err := dbController.storage.Get("B")
			if err != nil {
				t.Fatalf("should exist: %s, original meta: %+v", err, B.Meta())
			}
			if B1.Meta().Deleted == 0 {
				t.Errorf("B should be deleted")
			}

			// delete last entry
			C.Meta().Deleted = nowUnix - 1
			err = C.Save()
			if err != nil {
				t.Fatal(err)
			}

			// run maintenance
			err = dbController.MaintainRecordStates(context.TODO(), now)
			if err != nil {
				t.Fatal(err)
			}

			// check status individually
			B2, err := dbController.storage.Get("B")
			if err == nil {
				t.Errorf("B should be deleted and purged, meta: %+v", B2.Meta())
			} else if err != storage.ErrNotFound {
				t.Errorf("B should be deleted and purged, err=%s", err)
			}
			C2, err := dbController.storage.Get("C")
			if err == nil {
				t.Errorf("C should be deleted and purged, meta: %+v", C2.Meta())
			} else if err != storage.ErrNotFound {
				t.Errorf("C should be deleted and purged, err=%s", err)
			}

			// none left
			cnt = countRecords(t, db, q.New(dbName))
			if cnt != 0 {
				t.Fatalf("expected no records, got %d", cnt)
			}
		}

		err = hook.Cancel()
		if err != nil {
			t.Fatal(err)
		}
		err = sub.Cancel()
		if err != nil {
			t.Fatal(err)
		}

	})
}

func TestDatabaseSystem(t *testing.T) {

	// panic after 10 seconds, to check for locks
	go func() {
		time.Sleep(10 * time.Second)
		fmt.Println("===== TAKING TOO LONG - PRINTING STACK TRACES =====")
		_ = pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
		os.Exit(1)
	}()

	testDir, err := ioutil.TempDir("", "portbase-database-testing-")
	if err != nil {
		t.Fatal(err)
	}

	err = InitializeWithPath(testDir)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(testDir) // clean up

	testDatabase(t, "bbolt", true, true)
	testDatabase(t, "hashmap", true, true)
	testDatabase(t, "fstree", false, false)
	testDatabase(t, "badger", false, false)

	err = MaintainRecordStates(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	err = Maintain(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	err = MaintainThorough(context.TODO())
	if err != nil {
		t.Fatal(err)
	}

	err = Shutdown()
	if err != nil {
		t.Fatal(err)
	}
}

func countRecords(t *testing.T, db *Interface, query *q.Query) int {
	_, err := query.Check()
	if err != nil {
		t.Fatal(err)
	}

	it, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}

	cnt := 0
	for range it.Next {
		cnt++
	}
	if it.Err() != nil {
		t.Fatal(it.Err())
	}
	return cnt
}
