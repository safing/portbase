package rng

import (
	"crypto/rand"
	"time"

	"github.com/safing/portbase/log"
)

func osFeeder() {
	feeder := NewFeeder()
	for {

		// get feed entropy
		minEntropyBytes := int(minFeedEntropy())/8 + 1
		if minEntropyBytes < 32 {
			minEntropyBytes = 64
		}

		// get entropy
		osEntropy := make([]byte, minEntropyBytes)
		n, err := rand.Read(osEntropy)
		if err != nil {
			log.Errorf("could not read entropy from os: %s", err)
			time.Sleep(10 * time.Second)
		}
		if n != minEntropyBytes {
			log.Errorf("could not read enough entropy from os: got only %d bytes instead of %d", n, minEntropyBytes)
			time.Sleep(10 * time.Second)
		}

		// feed
		feeder.SupplyEntropy(osEntropy, minEntropyBytes*8)
	}
}
