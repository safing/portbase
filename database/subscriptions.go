// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Safing/safing-core/modules"
	"github.com/Safing/safing-core/taskmanager"

	"github.com/ipfs/go-datastore"
	"github.com/tevino/abool"
)

var subscriptionModule *modules.Module
var subscriptions []*Subscription
var subLock sync.Mutex

var databaseUpdate chan Model
var databaseCreate chan Model
var databaseDelete chan *datastore.Key

var workIsWaiting chan *struct{}
var workIsWaitingFlag *abool.AtomicBool
var forceProcessing chan *struct{}

type Subscription struct {
	typeAndLocation map[string]bool
	exactObject     map[string]bool
	children        map[string]uint8
	Created         chan Model
	Updated         chan Model
	Deleted         chan *datastore.Key
}

func NewSubscription() *Subscription {
	subLock.Lock()
	defer subLock.Unlock()
	sub := &Subscription{
		typeAndLocation: make(map[string]bool),
		exactObject:     make(map[string]bool),
		children:        make(map[string]uint8),
		Created:         make(chan Model, 128),
		Updated:         make(chan Model, 128),
		Deleted:         make(chan *datastore.Key, 128),
	}
	subscriptions = append(subscriptions, sub)
	return sub
}

func (sub *Subscription) Subscribe(subKey string) {
	subLock.Lock()
	defer subLock.Unlock()

	namespaces := strings.Split(subKey, "/")[1:]
	lastSpace := ""
	if len(namespaces) != 0 {
		lastSpace = namespaces[len(namespaces)-1]
	}

	switch {
	case lastSpace == "":
		// save key without leading "/"
		// save with depth 255 to get all
		sub.children[strings.Trim(subKey, "/")] = 0xFF
	case strings.HasPrefix(lastSpace, "*"):
		// save key without leading or trailing "/" or "*"
		// save full wanted depth - this makes comparison easier
		sub.children[strings.Trim(subKey, "/*")] = uint8(len(lastSpace) + len(namespaces) - 1)
	case strings.Contains(lastSpace, ":"):
		sub.exactObject[subKey] = true
	default:
		sub.typeAndLocation[subKey] = true
	}
}

func (sub *Subscription) Unsubscribe(subKey string) {
	subLock.Lock()
	defer subLock.Unlock()

	namespaces := strings.Split(subKey, "/")[1:]
	lastSpace := ""
	if len(namespaces) != 0 {
		lastSpace = namespaces[len(namespaces)-1]
	}

	switch {
	case lastSpace == "":
		delete(sub.children, strings.Trim(subKey, "/"))
	case strings.HasPrefix(lastSpace, "*"):
		delete(sub.children, strings.Trim(subKey, "/*"))
	case strings.Contains(lastSpace, ":"):
		delete(sub.exactObject, subKey)
	default:
		delete(sub.typeAndLocation, subKey)
	}
}

func (sub *Subscription) Destroy() {
	subLock.Lock()
	defer subLock.Unlock()

	for k, v := range subscriptions {
		if v.Created == sub.Created {
			defer func() {
				subscriptions = append(subscriptions[:k], subscriptions[k+1:]...)
			}()
			close(sub.Created)
			close(sub.Updated)
			close(sub.Deleted)
			return
		}
	}
}

func (sub *Subscription) Subscriptions() *[]string {
	subStrings := make([]string, 0)
	for subString := range sub.exactObject {
		subStrings = append(subStrings, subString)
	}
	for subString := range sub.typeAndLocation {
		subStrings = append(subStrings, subString)
	}
	for subString, depth := range sub.children {
		if depth == 0xFF {
			subStrings = append(subStrings, fmt.Sprintf("/%s/", subString))
		} else {
			subStrings = append(subStrings, fmt.Sprintf("/%s/%s", subString, strings.Repeat("*", int(depth)-len(strings.Split(subString, "/")))))
		}
	}
	return &subStrings
}

func (sub *Subscription) String() string {
	return fmt.Sprintf("<Subscription [%s]>", strings.Join(*sub.Subscriptions(), " "))
}

func (sub *Subscription) send(key *datastore.Key, model Model, created bool) {
	if model == nil {
		sub.Deleted <- key
	} else if created {
		sub.Created <- model
	} else {
		sub.Updated <- model
	}
}

func process(key *datastore.Key, model Model, created bool) {
	subLock.Lock()
	defer subLock.Unlock()

	stringRep := key.String()
	// "/Comedy/MontyPython/Actor:JohnCleese"
	typeAndLocation := key.Path().String()
	// "/Comedy/MontyPython/Actor"
	namespaces := key.Namespaces()
	// ["Comedy", "MontyPython", "Actor:JohnCleese"]
	depth := uint8(len(namespaces))
	// 3

subscriptionLoop:
	for _, sub := range subscriptions {
		if _, ok := sub.exactObject[stringRep]; ok {
			sub.send(key, model, created)
			continue subscriptionLoop
		}
		if _, ok := sub.typeAndLocation[typeAndLocation]; ok {
			sub.send(key, model, created)
			continue subscriptionLoop
		}
		for i := 0; i < len(namespaces); i++ {
			if subscribedDepth, ok := sub.children[strings.Join(namespaces[:i], "/")]; ok {
				if subscribedDepth >= depth {
					sub.send(key, model, created)
					continue subscriptionLoop
				}
			}
		}
	}

}

func init() {
	subscriptionModule = modules.Register("Database:Subscriptions", 128)
	subscriptions = make([]*Subscription, 0)
	subLock = sync.Mutex{}

	databaseUpdate = make(chan Model, 32)
	databaseCreate = make(chan Model, 32)
	databaseDelete = make(chan *datastore.Key, 32)

	workIsWaiting = make(chan *struct{}, 0)
	workIsWaitingFlag = abool.NewBool(false)
	forceProcessing = make(chan *struct{}, 0)

	go run()
}

func run() {
	for {
		select {
		case <-subscriptionModule.Stop:
			subscriptionModule.StopComplete()
			return
		case <-workIsWaiting:
			work()
		}
	}
}

func work() {
	defer workIsWaitingFlag.UnSet()

	// wait
	select {
	case <-taskmanager.StartMediumPriorityMicroTask():
		defer taskmanager.EndMicroTask()
	case <-forceProcessing:
	}

	// work
	for {
		select {
		case model := <-databaseCreate:
			process(model.GetKey(), model, true)
		case model := <-databaseUpdate:
			process(model.GetKey(), model, false)
		case key := <-databaseDelete:
			process(key, nil, false)
		default:
			return
		}
	}
}

func handleCreateSubscriptions(model Model) {
	select {
	case databaseCreate <- model:
	default:
		forceProcessing <- nil
		databaseCreate <- model
	}
	if workIsWaitingFlag.SetToIf(false, true) {
		workIsWaiting <- nil
	}
}

func handleUpdateSubscriptions(model Model) {
	select {
	case databaseUpdate <- model:
	default:
		forceProcessing <- nil
		databaseUpdate <- model
	}
	if workIsWaitingFlag.SetToIf(false, true) {
		workIsWaiting <- nil
	}
}

func handleDeleteSubscriptions(key *datastore.Key) {
	select {
	case databaseDelete <- key:
	default:
		forceProcessing <- nil
		databaseDelete <- key
	}
	if workIsWaitingFlag.SetToIf(false, true) {
		workIsWaiting <- nil
	}
}
