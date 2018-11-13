package random

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"sync"

	"github.com/aead/serpent"
	"github.com/seehuhn/fortuna"

	"github.com/Safing/portbase/config"
	"github.com/Safing/portbase/modules"
)

var (
	rng             *fortuna.Generator
	rngLock         sync.Mutex
	rngReady        = false
	rngCipherOption config.StringOption

	shutdownSignal = make(chan struct{}, 0)
)

func init() {
	modules.Register("random", prep, Start, stop)

	config.Register(&config.Option{
		Name:            "RNG Cipher",
		Key:             "random/rng_cipher",
		Description:     "Cipher to use for the Fortuna RNG. Requires restart to take effect.",
		ExpertiseLevel:  config.ExpertiseLevelDeveloper,
		OptType:         config.OptTypeString,
		ExternalOptType: "string list",
		DefaultValue:    "aes",
		ValidationRegex: "^(aes|serpent)$",
	})
	rngCipherOption = config.GetAsString("random/rng_cipher", "aes")
}

func prep() error {
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

func stop() error {
	return nil
}
