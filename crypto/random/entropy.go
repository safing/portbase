package random

import (
	"encoding/binary"

	"github.com/tevino/abool"

	"github.com/Safing/portbase/config"
	"github.com/Safing/portbase/container"
)

var (
	rngFeeder      = make(chan []byte, 0)
	minFeedEntropy config.IntOption
)

func init() {
	config.Register(&config.Option{
		Name:            "Minimum Feed Entropy",
		Key:             "random.min_feed_entropy",
		Description:     "The minimum amount of entropy before a entropy source is feed to the RNG, in bits.",
		ExpertiseLevel:  config.ExpertiseLevelDeveloper,
		OptType:         config.OptTypeInt,
		DefaultValue:    256,
		ValidationRegex: "^[0-9]{3,5}$",
	})
	minFeedEntropy = config.GetAsInt("random.min_feed_entropy", 256)
}

// The Feeder is used to feed entropy to the RNG.
type Feeder struct {
	input        chan *entropyData
	entropy      int64
	needsEntropy *abool.AtomicBool
	buffer       *container.Container
}

type entropyData struct {
	data    []byte
	entropy int
}

// NewFeeder returns a new entropy Feeder.
func NewFeeder() *Feeder {
	new := &Feeder{
		input:        make(chan *entropyData, 0),
		needsEntropy: abool.NewBool(true),
		buffer:       container.New(),
	}
	go new.run()
	return new
}

// NeedsEntropy returns whether the feeder is currently gathering entropy.
func (f *Feeder) NeedsEntropy() bool {
	return f.needsEntropy.IsSet()
}

// SupplyEntropy supplies entropy to to the Feeder, it will block until the Feeder has read from it.
func (f *Feeder) SupplyEntropy(data []byte, entropy int) {
	f.input <- &entropyData{
		data:    data,
		entropy: entropy,
	}
}

// SupplyEntropyIfNeeded supplies entropy to to the Feeder, but will not block if no entropy is currently needed.
func (f *Feeder) SupplyEntropyIfNeeded(data []byte, entropy int) {
	if f.needsEntropy.IsSet() {
		return
	}

	select {
	case f.input <- &entropyData{
		data:    data,
		entropy: entropy,
	}:
	default:
	}
}

// SupplyEntropyAsInt supplies entropy to to the Feeder, it will block until the Feeder has read from it.
func (f *Feeder) SupplyEntropyAsInt(n int64, entropy int) {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(n))
	f.SupplyEntropy(b, entropy)
}

// SupplyEntropyAsIntIfNeeded supplies entropy to to the Feeder, but will not block if no entropy is currently needed.
func (f *Feeder) SupplyEntropyAsIntIfNeeded(n int64, entropy int) {
	if f.needsEntropy.IsSet() { // avoid allocating a slice if possible
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, uint64(n))
		f.SupplyEntropyIfNeeded(b, entropy)
	}
}

// CloseFeeder stops the feed processing - the responsible goroutine exits.
func (f *Feeder) CloseFeeder() {
	f.input <- nil
}

func (f *Feeder) run() {
	defer f.needsEntropy.UnSet()

	for {
		// gather
		f.needsEntropy.Set()
	gather:
		for {
			select {
			case newEntropy := <-f.input:
				if newEntropy != nil {
					f.buffer.Append(newEntropy.data)
					f.entropy += int64(newEntropy.entropy)
					if f.entropy >= minFeedEntropy() {
						break gather
					}
				}
			case <-shutdownSignal:
				return
			}
		}
		// feed
		f.needsEntropy.UnSet()
		select {
		case rngFeeder <- f.buffer.CompileData():
		case <-shutdownSignal:
			return
		}
		f.buffer = container.New()
	}
}
