package rng

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"sync"

	"github.com/aead/serpent"
	"github.com/seehuhn/fortuna"

	"github.com/safing/portbase/config"
	"github.com/safing/portbase/modules"
)

var (
	rng             *fortuna.Generator
	rngLock         sync.Mutex
	rngReady        = false
	rngCipherOption config.StringOption

	shutdownSignal = make(chan struct{})
)

func init() {
	modules.Register("random", prep, Start, nil, "base")
}

func prep() error {
	err := config.Register(&config.Option{
		Name:            "RNG Cipher",
		Key:             "random/rng_cipher",
		Description:     "Cipher to use for the Fortuna RNG. Requires restart to take effect.",
		OptType:         config.OptTypeString,
		ExpertiseLevel:  config.ExpertiseLevelDeveloper,
		ReleaseLevel:    config.ReleaseLevelExperimental,
		ExternalOptType: "string list",
		DefaultValue:    "aes",
		ValidationRegex: "^(aes|serpent)$",
	})
	if err != nil {
		return err
	}
	rngCipherOption = config.GetAsString("random/rng_cipher", "aes")

	err = config.Register(&config.Option{
		Name:            "Minimum Feed Entropy",
		Key:             "random/min_feed_entropy",
		Description:     "The minimum amount of entropy before a entropy source is feed to the RNG, in bits.",
		OptType:         config.OptTypeInt,
		ExpertiseLevel:  config.ExpertiseLevelDeveloper,
		ReleaseLevel:    config.ReleaseLevelExperimental,
		DefaultValue:    256,
		ValidationRegex: "^[0-9]{3,5}$",
	})
	if err != nil {
		return err
	}
	minFeedEntropy = config.Concurrent.GetAsInt("random/min_feed_entropy", 256)

	err = config.Register(&config.Option{
		Name:            "Reseed after x seconds",
		Key:             "random/reseed_after_seconds",
		Description:     "Number of seconds until reseed",
		OptType:         config.OptTypeInt,
		ExpertiseLevel:  config.ExpertiseLevelDeveloper,
		ReleaseLevel:    config.ReleaseLevelExperimental,
		DefaultValue:    360, // ten minutes
		ValidationRegex: "^[1-9][0-9]{1,5}$",
	})
	if err != nil {
		return err
	}
	reseedAfterSeconds = config.Concurrent.GetAsInt("random/reseed_after_seconds", 360)

	err = config.Register(&config.Option{
		Name:            "Reseed after x bytes",
		Key:             "random/reseed_after_bytes",
		Description:     "Number of fetched bytes until reseed",
		OptType:         config.OptTypeInt,
		ExpertiseLevel:  config.ExpertiseLevelDeveloper,
		ReleaseLevel:    config.ReleaseLevelExperimental,
		DefaultValue:    1000000, // one megabyte
		ValidationRegex: "^[1-9][0-9]{2,9}$",
	})
	if err != nil {
		return err
	}
	reseedAfterBytes = config.GetAsInt("random/reseed_after_bytes", 1000000)

	return nil
}

func newCipher(key []byte) (cipher.Block, error) {
	cipher := rngCipherOption()
	switch cipher {
	case "aes":
		return aes.NewCipher(key)
	case "serpent":
		return serpent.NewCipher(key)
	default:
		return nil, fmt.Errorf("unknown or unsupported cipher: %s", cipher)
	}
}

// Start starts the RNG. Normally, this should be only called by the portbase/modules package.
func Start() (err error) {
	rngLock.Lock()
	defer rngLock.Unlock()

	rng = fortuna.NewGenerator(newCipher)
	rngReady = true

	// random source: OS
	go osFeeder()

	// random source: goroutine ticks
	go tickFeeder()

	// full feeder
	go fullFeeder()

	return nil
}
