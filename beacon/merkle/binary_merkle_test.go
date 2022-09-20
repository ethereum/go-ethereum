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

func TestMergedFormat(t *testing.T) {
	for count := 0; count < 1000; count++ {
		single := NewIndexMapFormat()
		merged := MergedFormat{}
		for {
			if rand.Intn(5) == 0 {
				break
			}
			f := NewIndexMapFormat()
			for {
				if rand.Intn(5) == 0 {
					break
				}
				index := uint64(rand.Intn(255) + 1)
				single.AddLeaf(index, nil)
				f.AddLeaf(index, nil)
			}
			merged = append(merged, f)
		}
		if !formatsEqual(single, merged) {
			t.Errorf("Single and merged formats do not match")
		}
	}
}

func TestIndexMapSubtrees(t *testing.T) {
	for count := 0; count < 1000; count++ {
		single := NewIndexMapFormat()
		withSubtrees := NewIndexMapFormat()
		// put single leaves and subtrees randomly into a single row in order to avoid collisions
		for index := uint64(256); index < 512; index++ {
			switch rand.Intn(100) {
			case 0: // put single leaf at index
				single.AddLeaf(index, nil)
				withSubtrees.AddLeaf(index, nil)
			case 1: // put subtree at index
				subtree := NewIndexMapFormat()
				for {
					subindex := uint64(rand.Intn(255) + 1)
					single.AddLeaf(ChildIndex(index, subindex), nil)
					subtree.AddLeaf(subindex, nil)
					if rand.Intn(5) == 0 { // exit here in order to avoid empty subtrees
						break
					}
				}
				withSubtrees.AddLeaf(index, subtree)
			}
		}
		if !formatsEqual(single, withSubtrees) {
			t.Errorf("Single and subtree formats do not match")
		}
	}
}

func TestRangeFormat(t *testing.T) {
	for count := 0; count < 1000; count++ {
		single := NewIndexMapFormat()
		begin := uint64(rand.Intn(255) + 1)
		nextLevel := uint64(1)
		for nextLevel <= begin {
			nextLevel += nextLevel
		}
		end := begin + uint64(rand.Intn(int(nextLevel-begin)))
		for i := begin; i <= end; i++ {
			single.AddLeaf(i, nil)
		}

		var subFn func(index uint64) ProofFormat
		if rand.Intn(2) == 0 {
			subroot := begin + uint64(rand.Intn(int(end+1-begin)))
			subBegin := uint64(rand.Intn(255) + 1)
			nextLevel = uint64(1)
			for nextLevel <= subBegin {
				nextLevel += nextLevel
			}
			subEnd := subBegin + uint64(rand.Intn(int(nextLevel-subBegin)))
			for i := subBegin; i <= subEnd; i++ {
				single.AddLeaf(ChildIndex(subroot, i), nil)
			}
			subtree := NewRangeFormat(subBegin, subEnd, nil)
			subFn = func(index uint64) ProofFormat {
				if index == subroot {
					return subtree
				}
				return nil
			}
		}

		rangeFormat := NewRangeFormat(begin, end, subFn)
		if !formatsEqual(single, rangeFormat) {
			t.Errorf("Single and range formats do not match")
		}
	}
}

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

func TestMultiProof(t *testing.T) {
	for count := 0; count < 300; count++ {
		failIndex := uint64(128 + rand.Intn(128))
		indexList := make([]uint64, 10)
		for i := range indexList {
			for {
				indexList[i] = uint64(128 + rand.Intn(128))
				if indexList[i]^failIndex > 1 {
					// failIndex should not be available in proofs so it should not be equal to or sibling of a stored index
					break
				}
			}
		}

		readers := make([]ProofReader, len(indexList))
		for i, index := range indexList {
			var mp MultiProof
			mp.Format = NewIndexMapFormat().AddLeaf(index, nil)
			writer := NewMultiProofWriter(mp.Format, &mp.Values, nil)
			testTraverseProof(t, testProofReader, writer, true)
			readers[i] = mp.Reader(nil)
		}

		// create a single multiproof from the merged reader using a subset of indices
		var mp MultiProof
		format := NewIndexMapFormat()
		mpCount := rand.Intn(11) // add a subset of indices to the created multiproof format
		for i := 0; i < mpCount; i++ {
			format.AddLeaf(indexList[i], nil)
		}
		mp.Format = format
		expSuccess := rand.Intn(2) == 0
		if !expSuccess {
			// add an index that should not be available in the merged reader, expect the traversal to fail
			format.AddLeaf(failIndex, nil)
		}
		testTraverseProof(t, MergedReader(readers), NewMultiProofWriter(format, &mp.Values, nil), expSuccess)

		if expSuccess {
			mpwCount := rand.Intn(mpCount + 1) // create writers for a subset of the previously selected indices (available in mp)
			mps := make([]MultiProof, mpwCount)
			writers := make([]ProofWriter, mpwCount)
			for i := range mps {
				mps[i].Format = NewIndexMapFormat().AddLeaf(indexList[i], nil)
				writers[i] = NewMultiProofWriter(mps[i].Format, &mps[i].Values, nil)
			}
			reader := mp.Reader(nil)
			testTraverseProof(t, reader, MergedWriter(writers), true)
			if !reader.Finished() {
				t.Errorf("MultiProofReader not finished")
			}
			// test individual single-value multiproofs
			for i, mp := range mps {
				if valueIndex, ok := ProofFormatIndexMap(mp.Format)[indexList[i]]; !ok || mp.Values[valueIndex] != testMerkleTree[indexList[i]] {
					t.Errorf("Could not find tree index %d in single-value multiproof", indexList[i])
				}
			}
		}
	}
}

func testTraverseProof(t *testing.T, reader ProofReader, writer ProofWriter, expSuccess bool) {
	root, ok := TraverseProof(reader, writer)
	if expSuccess {
		if root != common.Hash(testMerkleTree[1]) {
			t.Errorf("TraverseProof root hash mismatch")
		}
		if !ok {
			t.Errorf("TraverseProof insufficient reader data")
		}
	} else if ok {
		t.Errorf("TraverseProof succeeded (expected to fail)")
	}
}

func formatsEqual(f1, f2 ProofFormat) bool {
	if f1 == nil && f2 == nil {
		return true
	}
	if f1 == nil || f2 == nil {
		return false
	}
	c1l, c1r := f1.Children()
	c2l, c2r := f2.Children()
	return formatsEqual(c1l, c2l) && formatsEqual(c1r, c2r)
}

type testReader byte

var testProofReader = testReader(1)

func (r testReader) Children() (left, right ProofReader) {
	if r >= 128 {
		return nil, nil
	}
	return r * 2, r*2 + 1
}

func (r testReader) ReadNode() (Value, bool) {
	return testMerkleTree[r], true
}

var testMerkleTree [256]Value

func init() {
	hasher := sha256.New()
	for i := byte(255); i >= 1; i-- {
		if i >= 128 {
			testMerkleTree[i][0] = i
		} else {
			hasher.Reset()
			hasher.Write(testMerkleTree[i*2][:])
			hasher.Write(testMerkleTree[i*2+1][:])
			hasher.Sum(testMerkleTree[i][:0])
		}
	}
}
