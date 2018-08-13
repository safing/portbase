// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package configuration

import (
	"fmt"
	"testing"
	"time"
)

func TestConfiguration(t *testing.T) {

	config1 := Get()
	fmt.Printf("%v", config1)
	time.Sleep(1 * time.Millisecond)
	config1.Changed()
	time.Sleep(1 * time.Millisecond)
	config1.Save()
	time.Sleep(1 * time.Millisecond)
	config1.Changed()
	time.Sleep(1 * time.Millisecond)

}
