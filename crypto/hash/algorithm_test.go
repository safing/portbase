// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package hash

import "testing"

func TestAttributes(t *testing.T) {

	for alg, att := range attributes {

		name, ok := names[alg]
		if !ok {
			t.Errorf("hash test: name missing for Algorithm ID %d", alg)
		}
		_ = alg.String()

		_, ok = functions[alg]
		if !ok {
			t.Errorf("hash test: function missing for Algorithm %s", name)
		}
		hash := alg.New()

		if len(att) != 3 {
			t.Errorf("hash test: Algorithm %s does not have exactly 3 attributes", name)
		}

		if hash.BlockSize() != int(alg.BlockSize()) {
			t.Errorf("hash test: block size mismatch at Algorithm %s", name)
		}
		if hash.Size() != int(alg.Size()) {
			t.Errorf("hash test: size mismatch at Algorithm %s", name)
		}
		if alg.Size()/2 != alg.SecurityStrength() {
			t.Errorf("hash test: possible strength error at Algorithm %s", name)
		}

	}

	noAlg := Algorithm(255)
	if noAlg.String() != "" {
		t.Error("hash test: invalid Algorithm error")
	}
	if noAlg.BlockSize() != 0 {
		t.Error("hash test: invalid Algorithm error")
	}
	if noAlg.Size() != 0 {
		t.Error("hash test: invalid Algorithm error")
	}
	if noAlg.SecurityStrength() != 0 {
		t.Error("hash test: invalid Algorithm error")
	}
	if noAlg.New() != nil {
		t.Error("hash test: invalid Algorithm error")
	}

}
