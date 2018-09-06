// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

// import (
// 	"strconv"
// 	"strings"
// 	"sync"
// 	"testing"
// )
//
// var subTestWg sync.WaitGroup
//
// func waitForSubs(t *testing.T, sub *Subscription, highest int) {
// 	defer subTestWg.Done()
// 	expecting := 1
// 	var subbedModel Model
// forLoop:
// 	for {
// 		select {
// 		case subbedModel = <-sub.Created:
// 		case subbedModel = <-sub.Updated:
// 		}
// 		t.Logf("got model from subscription: %s", subbedModel.GetKey().String())
// 		if !strings.HasPrefix(subbedModel.GetKey().Name(), "sub") {
// 			// not a model that we use for testing, other tests might be interfering
// 			continue forLoop
// 		}
// 		number, err := strconv.Atoi(strings.TrimPrefix(subbedModel.GetKey().Name(), "sub"))
// 		if err != nil || number != expecting {
// 			t.Errorf("test subscription: got unexpected model %s, expected sub%d", subbedModel.GetKey().String(), expecting)
// 			continue forLoop
// 		}
// 		if number == highest {
// 			return
// 		}
// 		expecting++
// 	}
// }
//
// func TestSubscriptions(t *testing.T) {
//
// 	// create subscription
// 	sub := NewSubscription()
//
// 	// FIRST TEST
//
// 	subTestWg.Add(1)
// 	go waitForSubs(t, sub, 3)
// 	sub.Subscribe("/Tests/")
// 	t.Log(sub.String())
//
// 	(&(TestingModel{})).CreateInNamespace("", "sub1")
// 	(&(TestingModel{})).CreateInNamespace("A", "sub2")
// 	(&(TestingModel{})).CreateInNamespace("A/B/C/D/E", "sub3")
//
// 	subTestWg.Wait()
//
// 	// SECOND TEST
//
// 	subTestWg.Add(1)
// 	go waitForSubs(t, sub, 3)
// 	sub.Unsubscribe("/Tests/")
// 	sub.Subscribe("/Tests/A/****")
// 	t.Log(sub.String())
//
// 	(&(TestingModel{})).CreateInNamespace("", "subX")
// 	(&(TestingModel{})).CreateInNamespace("A", "sub1")
// 	(&(TestingModel{})).CreateInNamespace("A/B/C/D", "sub2")
// 	(&(TestingModel{})).CreateInNamespace("A/B/C/D/E", "subX")
// 	(&(TestingModel{})).CreateInNamespace("A", "sub3")
//
// 	subTestWg.Wait()
//
// 	// THIRD TEST
//
// 	subTestWg.Add(1)
// 	go waitForSubs(t, sub, 3)
// 	sub.Unsubscribe("/Tests/A/****")
// 	sub.Subscribe("/Tests/TestingModel:sub1")
// 	sub.Subscribe("/Tests/TestingModel:sub1/TestingModel")
// 	t.Log(sub.String())
//
// 	(&(TestingModel{})).CreateInNamespace("", "sub1")
// 	(&(TestingModel{})).CreateInNamespace("", "subX")
// 	(&(TestingModel{})).CreateInNamespace("TestingModel:sub1", "sub2")
// 	(&(TestingModel{})).CreateInNamespace("TestingModel:sub1/A", "subX")
// 	(&(TestingModel{})).CreateInNamespace("TestingModel:sub1", "sub3")
//
// 	subTestWg.Wait()
//
// 	// FINAL STUFF
//
// 	model := &TestingModel{}
// 	model.CreateInNamespace("Invalid", "subX")
// 	model.Save()
//
// 	sub.Destroy()
//
// 	// time.Sleep(1 * time.Second)
// 	// pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
//
// }
