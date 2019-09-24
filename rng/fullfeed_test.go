package rng

import (
	"testing"
)

func TestFullFeeder(t *testing.T) {
	for i := 0; i < 10; i++ {
		go func() {
			rngFeeder <- []byte{0}
		}()
	}
}
