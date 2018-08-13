// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package hash

import (
	"bytes"
	"testing"
)

var (
	testEmpty = []byte("")
	testFox   = []byte("The quick brown fox jumps over the lazy dog")
)

func testAlgorithm(t *testing.T, alg Algorithm, emptyHex, foxHex string) {

	var err error

	// testEmpty
	hash := Sum(testEmpty, alg)
	if err != nil {
		t.Errorf("test Sum %s (empty): error occured: %s", alg.String(), err)
	}
	if hash.Hex()[2:] != emptyHex {
		t.Errorf("test Sum %s (empty): hex sum mismatch, expected %s, got %s", alg.String(), emptyHex, hash.Hex())
	}

	// testFox
	hash = Sum(testFox, alg)
	if err != nil {
		t.Errorf("test Sum %s (fox): error occured: %s", alg.String(), err)
	}
	if hash.Hex()[2:] != foxHex {
		t.Errorf("test Sum %s (fox): hex sum mismatch, expected %s, got %s", alg.String(), foxHex, hash.Hex())
	}

	// testEmpty
	hash = SumString(string(testEmpty), alg)
	if err != nil {
		t.Errorf("test SumString %s (empty): error occured: %s", alg.String(), err)
	}
	if hash.Hex()[2:] != emptyHex {
		t.Errorf("test SumString %s (empty): hex sum mismatch, expected %s, got %s", alg.String(), emptyHex, hash.Hex())
	}

	// testFox
	hash = SumString(string(testFox), alg)
	if err != nil {
		t.Errorf("test SumString %s (fox): error occured: %s", alg.String(), err)
	}
	if hash.Hex()[2:] != foxHex {
		t.Errorf("test SumString %s (fox): hex sum mismatch, expected %s, got %s", alg.String(), foxHex, hash.Hex())
	}

	// testEmpty
	hash, err = SumReader(bytes.NewReader(testEmpty), alg)
	if err != nil {
		t.Errorf("test SumReader %s (empty): error occured: %s", alg.String(), err)
	}
	if hash.Hex()[2:] != emptyHex {
		t.Errorf("test SumReader %s (empty): hex sum mismatch, expected %s, got %s", alg.String(), emptyHex, hash.Hex())
	}

	// testFox
	hash, err = SumReader(bytes.NewReader(testFox), alg)
	if err != nil {
		t.Errorf("test SumReader %s (fox): error occured: %s", alg.String(), err)
	}
	if hash.Hex()[2:] != foxHex {
		t.Errorf("test SumReader %s (fox): hex sum mismatch, expected %s, got %s", alg.String(), foxHex, hash.Hex())
	}

}

func TestHash(t *testing.T) {
	testAlgorithm(t, SHA2_512,
		"cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
		"07e547d9586f6a73f73fbac0435ed76951218fb7d0c8d788a309d785436bbb642e93a252a954f23912547d1e8a3b5ed6e1bfd7097821233fa0538f3db854fee6",
	)
	testAlgorithm(t, SHA3_512,
		"a69f73cca23a9ac5c8b567dc185a756e97c982164fe25859e0d1dcc1475c80a615b2123af1f5f94c11e3e9402c3ac558f500199d95b6d3e301758586281dcd26",
		"01dedd5de4ef14642445ba5f5b97c15e47b9ad931326e4b0727cd94cefc44fff23f07bf543139939b49128caf436dc1bdee54fcb24023a08d9403f9b4bf0d450",
	)
}
