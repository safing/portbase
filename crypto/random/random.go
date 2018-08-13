package random

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
)

// just (mostly) a proxy for now, awesome stuff comes later

func Int(randSrc io.Reader, max *big.Int) (n *big.Int, err error) {
	return rand.Int(randSrc, max)
}

func Prime(randSrc io.Reader, bits int) (p *big.Int, err error) {
	return rand.Prime(randSrc, bits)
}

func Read(b []byte) (n int, err error) {
	return rand.Read(b)
}

func Bytes(len int) ([]byte, error) {
	r := make([]byte, len)
	_, err := Read(r)
	if err != nil {
		return nil, fmt.Errorf("failed to get random data: %s", err)
	}
	return r, nil
}

type Reader struct{}

func (r Reader) Read(b []byte) (n int, err error) {
	return Read(b)
}
