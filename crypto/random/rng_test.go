package random

import (
	"testing"

	"github.com/Safing/portbase/config"
)

func init() {
	prep()
	Start()
}

func TestRNG(t *testing.T) {
	key := make([]byte, 16)

	config.SetConfigOption("random.rng_cipher", "aes")
	_, err := newCipher(key)
	if err != nil {
		t.Errorf("failed to create aes cipher: %s", err)
	}
	rng.Reseed(key)

	config.SetConfigOption("random.rng_cipher", "serpent")
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
