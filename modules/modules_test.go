// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package modules

import (
	"fmt"
	"time"
)

func newTestModule(name string, order uint8) {

	fmt.Printf("up %s\n", name)
	module := Register("TestModule", order)

	go func() {
		<-module.Stop
		fmt.Printf("down %s\n", name)
		module.StopComplete()
	}()

}

func Example() {

	// wait for logger registration timeout
	time.Sleep(1010 * time.Millisecond)

	newTestModule("1", 1)
	newTestModule("4", 4)
	newTestModule("3", 3)
	newTestModule("2", 2)
	newTestModule("5", 5)

	InitiateFullShutdown()

	time.Sleep(10 * time.Millisecond)

	// Output:
	// up 1
	// up 4
	// up 3
	// up 2
	// up 5
	// down 5
	// down 4
	// down 3
	// down 2
	// down 1

}
