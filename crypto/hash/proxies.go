package hash

import (
	"hash"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
)

func NewBlake2s256() hash.Hash {
	h, _ := blake2s.New256(nil)
	return h
}

func NewBlake2b256() hash.Hash {
	h, _ := blake2b.New256(nil)
	return h
}

func NewBlake2b384() hash.Hash {
	h, _ := blake2b.New384(nil)
	return h
}

func NewBlake2b512() hash.Hash {
	h, _ := blake2b.New512(nil)
	return h
}
