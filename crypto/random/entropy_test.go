package random

import (
	"testing"
	"time"
)

func TestFeeder(t *testing.T) {
	// wait for start / first round to complete
	time.Sleep(1 * time.Millisecond)

	f := NewFeeder()

	// go through all functions
	f.NeedsEntropy()
	f.SupplyEntropy([]byte{0}, 0)
	f.SupplyEntropyAsInt(0, 0)
	f.SupplyEntropyIfNeeded([]byte{0}, 0)
	f.SupplyEntropyAsIntIfNeeded(0, 0)

	// fill entropy
	f.SupplyEntropyAsInt(0, 65535)

	// check blocking calls

	waitC := make(chan struct{})
	go func() {
		f.SupplyEntropy([]byte{0}, 0)
		close(waitC)
	}()
	select {
	case <-waitC:
		t.Error("call does not block!")
	case <-time.After(10 * time.Millisecond):
	}

	waitC = make(chan struct{})
	go func() {
		f.SupplyEntropyAsInt(0, 0)
		close(waitC)
	}()
	select {
	case <-waitC:
		t.Error("call does not block!")
	case <-time.After(10 * time.Millisecond):
	}

	// check non-blocking calls

	waitC = make(chan struct{})
	go func() {
		f.SupplyEntropyIfNeeded([]byte{0}, 0)
		close(waitC)
	}()
	select {
	case <-waitC:
	case <-time.After(10 * time.Millisecond):
		t.Error("call blocks!")
	}

	waitC = make(chan struct{})
	go func() {
		f.SupplyEntropyAsIntIfNeeded(0, 0)
		close(waitC)
	}()
	select {
	case <-waitC:
	case <-time.After(10 * time.Millisecond):
		t.Error("call blocks!")
	}

}
