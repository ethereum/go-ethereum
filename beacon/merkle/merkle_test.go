// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package merkle

import (
	"crypto/sha256"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// testProof returns a random branch for a value at index, and the root it proves.
func testProof(index uint64, value Value) (common.Hash, Values) {
	var branch Values
	for i := index; i > 1; i >>= 1 {
		var sibling Value
		sibling[0], sibling[1] = byte(i), byte(len(branch)+1)
		branch = append(branch, sibling)
		if i&1 == 0 {
			value = sha256.Sum256(append(value[:], sibling[:]...))
		} else {
			value = sha256.Sum256(append(sibling[:], value[:]...))
		}
	}
	return common.Hash(value), branch
}

func TestVerifyNormalizedProof(t *testing.T) {
	value := Value{1, 2, 3}
	root, branch := testProof(812, value) // depth 9
	if err := VerifyProof(root, 812, branch, value); err != nil {
		t.Fatal(err)
	}
	if err := VerifyNormalizedProof(root, 812, branch, value); err != nil {
		t.Fatalf("branch of the index's depth: %v", err)
	}
	normalized := append(make(Values, 2), branch...) // to depth 11
	if err := VerifyNormalizedProof(root, 812, normalized, value); err != nil {
		t.Fatalf("normalized branch: %v", err)
	}
	if err := VerifyProof(root, 812, normalized, value); err == nil {
		t.Fatal("normalized branch accepted by VerifyProof")
	}
	normalized[1][31] = 1
	if err := VerifyNormalizedProof(root, 812, normalized, value); err == nil {
		t.Fatal("non-zero extra item accepted")
	}
	if err := VerifyNormalizedProof(root, 812, branch[1:], value); err == nil {
		t.Fatal("short branch accepted")
	}
	if err := VerifyNormalizedProof(root, 0, branch, value); err == nil {
		t.Fatal("index 0 accepted")
	}
}
