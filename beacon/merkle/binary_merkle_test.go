// Copyright 2022 The go-ethereum Authors
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
	"math/bits"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/minio/sha256-simd"
)

func TestSingleProof(t *testing.T) {
	for index := uint64(1); index < 256; index++ {
		proof := make(Values, 63-bits.LeadingZeros64(index))
		writer := NewCallbackWriter(NewIndexMapFormat().AddLeaf(index, nil), func(i uint64, v Value) {
			shift := bits.LeadingZeros64(i) - bits.LeadingZeros64(index)
			if i^(index>>shift) == 1 {
				proof[shift] = v
			}
		})
		testTraverseProof(t, testProofReader, writer, true)
		root, ok := VerifySingleProof(proof, index, testMerkleTree[index])
		if root != common.Hash(testMerkleTree[1]) {
			t.Errorf("VerifySingleProof root hash mismatch (index = %d)", index)
		}
		if !ok {
			t.Errorf("VerifySingleProof length invalid (index = %d)", index)
		}
	}
}
