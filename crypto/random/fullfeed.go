package random

import (
	"time"
)

var (
	fullFeedDuration = 100 * time.Millisecond
)

func getFullFeedDuration() time.Duration {

	// full feed every 5x time of reseedAfterSeconds
	secsUntilFullFeed := reseedAfterSeconds() * 5

	// full feed at most once per minute
	if secsUntilFullFeed < 60 {
		secsUntilFullFeed = 60
	}

	return time.Duration(secsUntilFullFeed * int64(time.Second))
}

func fullFeeder() {
	for {

		select {
		case <-time.After(fullFeedDuration):

			rngLock.Lock()
		feedAll:
			for {
				select {
				case data := <-rngFeeder:
					rng.Reseed(data)
				default:
					break feedAll
				}
			}
			rngLock.Unlock()

		case <-shutdownSignal:
			return
		}

		fullFeedDuration = getFullFeedDuration()

	}
}
