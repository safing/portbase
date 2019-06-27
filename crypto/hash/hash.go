// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package hash

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"github.com/safing/portbase/formats/varint"
)

type Hash struct {
	Algorithm Algorithm
	Sum       []byte
}

func FromBytes(bytes []byte) (*Hash, int, error) {
	hash := &Hash{}
	alg, read, err := varint.Unpack8(bytes)
	hash.Algorithm = Algorithm(alg)
	if err != nil {
		return nil, 0, errors.New(fmt.Sprintf("hash: failed to parse: %s", err))
	}
	// TODO: check if length is correct
	hash.Sum = bytes[read:]
	return hash, 0, nil
}

func (h *Hash) Bytes() []byte {
	return append(varint.Pack8(uint8(h.Algorithm)), h.Sum...)
}

func FromSafe64(s string) (*Hash, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("hash: failed to parse: %s", err))
	}
	hash, _, err := FromBytes(bytes)
	return hash, err
}

func (h *Hash) Safe64() string {
	return base64.RawURLEncoding.EncodeToString(h.Bytes())
}

func FromHex(s string) (*Hash, error) {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("hash: failed to parse: %s", err))
	}
	hash, _, err := FromBytes(bytes)
	return hash, err
}

func (h *Hash) Hex() string {
	return hex.EncodeToString(h.Bytes())
}

func (h *Hash) Equal(other *Hash) bool {
	if h.Algorithm != other.Algorithm {
		return false
	}
	return bytes.Equal(h.Sum, other.Sum)
}

func Sum(data []byte, alg Algorithm) *Hash {
	hasher := alg.New()
	hasher.Write(data)
	return &Hash{
		Algorithm: alg,
		Sum:       hasher.Sum(nil),
	}
}

func SumString(data string, alg Algorithm) *Hash {
	hasher := alg.New()
	io.WriteString(hasher, data)
	return &Hash{
		Algorithm: alg,
		Sum:       hasher.Sum(nil),
	}
}

func SumReader(reader io.Reader, alg Algorithm) (*Hash, error) {
	hasher := alg.New()
	_, err := io.Copy(hasher, reader)
	if err != nil {
		return nil, err
	}
	return &Hash{
		Algorithm: alg,
		Sum:       hasher.Sum(nil),
	}, nil
}

func SumAndCompare(data []byte, other Hash) (bool, *Hash) {
	newHash := Sum(data, other.Algorithm)
	return other.Equal(newHash), newHash
}

func SumReaderAndCompare(reader io.Reader, other Hash) (bool, *Hash, error) {
	newHash, err := SumReader(reader, other.Algorithm)
	if err != nil {
		return false, nil, err
	}
	return other.Equal(newHash), newHash, nil
}

func RecommendedAlg(strengthInBits uint16) Algorithm {
	strengthInBytes := uint8(strengthInBits / 8)
	if strengthInBits%8 != 0 {
		strengthInBytes++
	}
	if strengthInBytes == 0 {
		strengthInBytes = uint8(0xFF)
	}
	chosenAlg := orderedByRecommendation[0]
	for _, alg := range orderedByRecommendation {
		strength := alg.SecurityStrength()
		if strength < strengthInBytes {
			break
		}
		chosenAlg = alg
		if strength == strengthInBytes {
			break
		}
	}
	return chosenAlg
}
