// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package hash

import (
	"crypto/sha256"
	"crypto/sha512"
	"hash"

	"golang.org/x/crypto/sha3"
)

type Algorithm uint8

const (
	SHA2_224 Algorithm = 1 + iota
	SHA2_256
	SHA2_512_224
	SHA2_512_256
	SHA2_384
	SHA2_512
	SHA3_224
	SHA3_256
	SHA3_384
	SHA3_512
	BLAKE2S_256
	BLAKE2B_256
	BLAKE2B_384
	BLAKE2B_512
)

var (
	attributes = map[Algorithm][]uint8{
		// block size, output size, security strength - in bytes
		SHA2_224:     []uint8{64, 28, 14},
		SHA2_256:     []uint8{64, 32, 16},
		SHA2_512_224: []uint8{128, 28, 14},
		SHA2_512_256: []uint8{128, 32, 16},
		SHA2_384:     []uint8{128, 48, 24},
		SHA2_512:     []uint8{128, 64, 32},
		SHA3_224:     []uint8{144, 28, 14},
		SHA3_256:     []uint8{136, 32, 16},
		SHA3_384:     []uint8{104, 48, 24},
		SHA3_512:     []uint8{72, 64, 32},
		BLAKE2S_256:  []uint8{64, 32, 16},
		BLAKE2B_256:  []uint8{128, 32, 16},
		BLAKE2B_384:  []uint8{128, 48, 24},
		BLAKE2B_512:  []uint8{128, 64, 32},
	}

	functions = map[Algorithm]func() hash.Hash{
		SHA2_224:     sha256.New224,
		SHA2_256:     sha256.New,
		SHA2_512_224: sha512.New512_224,
		SHA2_512_256: sha512.New512_256,
		SHA2_384:     sha512.New384,
		SHA2_512:     sha512.New,
		SHA3_224:     sha3.New224,
		SHA3_256:     sha3.New256,
		SHA3_384:     sha3.New384,
		SHA3_512:     sha3.New512,
		BLAKE2S_256:  NewBlake2s256,
		BLAKE2B_256:  NewBlake2b256,
		BLAKE2B_384:  NewBlake2b384,
		BLAKE2B_512:  NewBlake2b512,
	}

	// just ordered by strength and establishment, no research conducted yet.
	orderedByRecommendation = []Algorithm{
		SHA3_512,     // {72, 64, 32}
		SHA2_512,     // {128, 64, 32}
		BLAKE2B_512,  // {128, 64, 32}
		SHA3_384,     // {104, 48, 24}
		SHA2_384,     // {128, 48, 24}
		BLAKE2B_384,  // {128, 48, 24}
		SHA3_256,     // {136, 32, 16}
		SHA2_512_256, // {128, 32, 16}
		SHA2_256,     // {64, 32, 16}
		BLAKE2B_256,  // {128, 32, 16}
		BLAKE2S_256,  // {64, 32, 16}
		SHA3_224,     // {144, 28, 14}
		SHA2_512_224, // {128, 28, 14}
		SHA2_224,     // {64, 28, 14}
	}

	// names
	names = map[Algorithm]string{
		SHA2_224:     "SHA2-224",
		SHA2_256:     "SHA2-256",
		SHA2_512_224: "SHA2-512/224",
		SHA2_512_256: "SHA2-512/256",
		SHA2_384:     "SHA2-384",
		SHA2_512:     "SHA2-512",
		SHA3_224:     "SHA3-224",
		SHA3_256:     "SHA3-256",
		SHA3_384:     "SHA3-384",
		SHA3_512:     "SHA3-512",
		BLAKE2S_256:  "Blake2s-256",
		BLAKE2B_256:  "Blake2b-256",
		BLAKE2B_384:  "Blake2b-384",
		BLAKE2B_512:  "Blake2b-512",
	}
)

func (a Algorithm) BlockSize() uint8 {
	att, ok := attributes[a]
	if !ok {
		return 0
	}
	return att[0]
}

func (a Algorithm) Size() uint8 {
	att, ok := attributes[a]
	if !ok {
		return 0
	}
	return att[1]
}

func (a Algorithm) SecurityStrength() uint8 {
	att, ok := attributes[a]
	if !ok {
		return 0
	}
	return att[2]
}

func (a Algorithm) String() string {
	return a.Name()
}

func (a Algorithm) Name() string {
	name, ok := names[a]
	if !ok {
		return ""
	}
	return name
}

func (a Algorithm) New() hash.Hash {
	fn, ok := functions[a]
	if !ok {
		return nil
	}
	return fn()
}
