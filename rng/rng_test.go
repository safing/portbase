package rng

import (
	"testing"

	"github.com/safing/portbase/config"
)

func init() {
	err := prep()
	if err != nil {
		panic(err)
	}

	err = Start()
	if err != nil {
		panic(err)
	}
}

func TestRNG(t *testing.T) {
	key := make([]byte, 16)

	err := config.SetConfigOption("random/rng_cipher", "aes")
	if err != nil {
		t.Errorf("failed to set random/rng_cipher config: %s", err)
	}
	_, err = newCipher(key)
	if err != nil {
		t.Errorf("failed to create aes cipher: %s", err)
	}
	rng.Reseed(key)

	err = config.SetConfigOption("random/rng_cipher", "serpent")
	if err != nil {
		t.Errorf("failed to set random/rng_cipher config: %s", err)
	}
	_, err = newCipher(key)
	if err != nil {
		t.Errorf("failed to create serpent cipher: %s", err)
	}
	rng.Reseed(key)

	b := make([]byte, 32)
	_, err = Read(b)
	if err != nil {
		t.Errorf("Read failed: %s", err)
	}
	_, err = Reader.Read(b)
	if err != nil {
		t.Errorf("Read failed: %s", err)
	}

	_, err = Bytes(32)
	if err != nil {
		t.Errorf("Bytes failed: %s", err)
	}
}
