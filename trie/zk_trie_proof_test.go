// Copyright 2015 The go-ethereum Authors
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

package trie

import (
	"bytes"
	mrand "math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	zkt "github.com/scroll-tech/zktrie/types"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

func init() {
	mrand.Seed(time.Now().Unix())
}

// makeProvers creates Merkle trie provers based on different implementations to
// test all variations.
func makeSMTProvers(mt *ZkTrie) []func(key []byte) *memorydb.Database {
	var provers []func(key []byte) *memorydb.Database

	// Create a direct trie based Merkle prover
	provers = append(provers, func(key []byte) *memorydb.Database {
		word := zkt.NewByte32FromBytesPaddingZero(key)
		k, err := word.Hash()
		if err != nil {
			panic(err)
		}
		proof := memorydb.New()
		err = mt.Prove(common.BytesToHash(k.Bytes()).Bytes(), 0, proof)
		if err != nil {
			panic(err)
		}

		return proof
	})
	return provers
}

func verifyValue(proveVal []byte, vPreimage []byte) bool {
	return bytes.Equal(proveVal, vPreimage)
}

func TestSMTOneElementProof(t *testing.T) {
	tr, _ := NewZkTrie(common.Hash{}, NewZktrieDatabase((memorydb.New())))
	mt := &zkTrieImplTestWrapper{tr.Tree()}
	err := mt.UpdateWord(
		zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("k"), 32)),
		zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("v"), 32)),
	)
	assert.Nil(t, err)
	for i, prover := range makeSMTProvers(tr) {
		keyBytes := bytes.Repeat([]byte("k"), 32)
		proof := prover(keyBytes)
		if proof == nil {
			t.Fatalf("prover %d: nil proof", i)
		}
		if proof.Len() != 2 {
			t.Errorf("prover %d: proof should have 1+1 element (including the magic kv)", i)
		}
		val, err := VerifyProof(common.BytesToHash(mt.Root().Bytes()), keyBytes, proof)
		if err != nil {
			t.Fatalf("prover %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if !verifyValue(val, bytes.Repeat([]byte("v"), 32)) {
			t.Fatalf("prover %d: verified value mismatch: want 'v' get %x", i, val)
		}
	}
}

func TestSMTProof(t *testing.T) {
	mt, vals := randomZktrie(t, 500)
	root := mt.Tree().Root()
	for i, prover := range makeSMTProvers(mt) {
		for _, kv := range vals {
			proof := prover(kv.k)
			if proof == nil {
				t.Fatalf("prover %d: missing key %x while constructing proof", i, kv.k)
			}
			val, err := VerifyProof(common.BytesToHash(root.Bytes()), kv.k, proof)
			if err != nil {
				t.Fatalf("prover %d: failed to verify proof for key %x: %v\nraw proof: %x\n", i, kv.k, err, proof)
			}
			if !verifyValue(val, zkt.NewByte32FromBytesPaddingZero(kv.v)[:]) {
				t.Fatalf("prover %d: verified value mismatch for key %x, want %x, get %x", i, kv.k, kv.v, val)
			}
		}
	}
}

func TestSMTBadProof(t *testing.T) {
	mt, vals := randomZktrie(t, 500)
	root := mt.Tree().Root()
	for i, prover := range makeSMTProvers(mt) {
		for _, kv := range vals {
			proof := prover(kv.k)
			if proof == nil {
				t.Fatalf("prover %d: nil proof", i)
			}
			it := proof.NewIterator(nil, nil)
			for i, d := 0, mrand.Intn(proof.Len()); i <= d; i++ {
				it.Next()
			}
			key := it.Key()
			val, _ := proof.Get(key)
			proof.Delete(key)
			it.Release()

			mutateByte(val)
			proof.Put(crypto.Keccak256(val), val)

			if _, err := VerifyProof(common.BytesToHash(root.Bytes()), kv.k, proof); err == nil {
				t.Fatalf("prover %d: expected proof to fail for key %x", i, kv.k)
			}
		}
	}
}

// Tests that missing keys can also be proven. The test explicitly uses a single
// entry trie and checks for missing keys both before and after the single entry.
func TestSMTMissingKeyProof(t *testing.T) {
	tr, _ := NewZkTrie(common.Hash{}, NewZktrieDatabase((memorydb.New())))
	mt := &zkTrieImplTestWrapper{tr.Tree()}
	err := mt.UpdateWord(
		zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("k"), 32)),
		zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("v"), 32)),
	)
	assert.Nil(t, err)

	prover := makeSMTProvers(tr)[0]

	for i, key := range []string{"a", "j", "l", "z"} {
		keyBytes := bytes.Repeat([]byte(key), 32)
		proof := prover(keyBytes)

		if proof.Len() != 2 {
			t.Errorf("test %d: proof should have 2 element (with magic kv)", i)
		}
		val, err := VerifyProof(common.BytesToHash(mt.Root().Bytes()), keyBytes, proof)
		if err != nil {
			t.Fatalf("test %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if val != nil {
			t.Fatalf("test %d: verified value mismatch: have %x, want nil", i, val)
		}
	}
}

func randomZktrie(t *testing.T, n int) (*ZkTrie, map[string]*kv) {
	tr, err := NewZkTrie(common.Hash{}, NewZktrieDatabase((memorydb.New())))
	if err != nil {
		panic(err)
	}
	mt := &zkTrieImplTestWrapper{tr.Tree()}
	vals := make(map[string]*kv)
	for i := byte(0); i < 100; i++ {

		value := &kv{common.LeftPadBytes([]byte{i}, 32), bytes.Repeat([]byte{i}, 32), false}
		value2 := &kv{common.LeftPadBytes([]byte{i + 10}, 32), bytes.Repeat([]byte{i}, 32), false}

		err = mt.UpdateWord(zkt.NewByte32FromBytesPaddingZero(value.k), zkt.NewByte32FromBytesPaddingZero(value.v))
		assert.Nil(t, err)
		err = mt.UpdateWord(zkt.NewByte32FromBytesPaddingZero(value2.k), zkt.NewByte32FromBytesPaddingZero(value2.v))
		assert.Nil(t, err)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}
	for i := 0; i < n; i++ {
		value := &kv{randBytes(32), randBytes(20), false}
		err = mt.UpdateWord(zkt.NewByte32FromBytesPaddingZero(value.k), zkt.NewByte32FromBytesPaddingZero(value.v))
		assert.Nil(t, err)
		vals[string(value.k)] = value
	}

	return tr, vals
}

// Tests that new "proof trace" feature
func TestProofWithDeletion(t *testing.T) {
	tr, _ := NewZkTrie(common.Hash{}, NewZktrieDatabase((memorydb.New())))
	mt := &zkTrieImplTestWrapper{tr.Tree()}
	key1 := bytes.Repeat([]byte("l"), 32)
	key2 := bytes.Repeat([]byte("m"), 32)
	err := mt.UpdateWord(
		zkt.NewByte32FromBytesPaddingZero(key1),
		zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("v"), 32)),
	)
	assert.NoError(t, err)
	err = mt.UpdateWord(
		zkt.NewByte32FromBytesPaddingZero(key2),
		zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("n"), 32)),
	)
	assert.NoError(t, err)

	proof := memorydb.New()
	s_key1, err := zkt.ToSecureKeyBytes(key1)
	assert.NoError(t, err)

	proofTracer := tr.NewProofTracer()

	err = proofTracer.Prove(s_key1.Bytes(), 0, proof)
	assert.NoError(t, err)
	nd, err := tr.TryGet(key2)
	assert.NoError(t, err)

	s_key2, err := zkt.ToSecureKeyBytes(bytes.Repeat([]byte("x"), 32))
	assert.NoError(t, err)

	err = proofTracer.Prove(s_key2.Bytes(), 0, proof)
	assert.NoError(t, err)
	//assert.Equal(t, len(sibling1), len(delTracer.GetProofs()))

	siblings, err := proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(siblings))

	proofTracer.MarkDeletion(s_key1.Bytes())
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(siblings))
	l := len(siblings[0])
	// a hacking to grep the value part directly from the encoded leaf node,
	// notice the sibling of key `k*32`` is just the leaf of key `m*32`
	assert.Equal(t, siblings[0][l-33:l-1], nd)

	// Marking a key that is currently not hit (but terminated by an empty node)
	// also causes it to be added to the deletion proof
	proofTracer.MarkDeletion(s_key2.Bytes())
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(siblings))

	key3 := bytes.Repeat([]byte("x"), 32)
	err = mt.UpdateWord(
		zkt.NewByte32FromBytesPaddingZero(key3),
		zkt.NewByte32FromBytesPaddingZero(bytes.Repeat([]byte("z"), 32)),
	)
	assert.NoError(t, err)

	proofTracer = tr.NewProofTracer()
	err = proofTracer.Prove(s_key1.Bytes(), 0, proof)
	assert.NoError(t, err)
	err = proofTracer.Prove(s_key2.Bytes(), 0, proof)
	assert.NoError(t, err)

	proofTracer.MarkDeletion(s_key1.Bytes())
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(siblings))

	proofTracer.MarkDeletion(s_key2.Bytes())
	siblings, err = proofTracer.GetDeletionProofs()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(siblings))

	// one of the siblings is just leaf for key2, while
	// another one must be a middle node
	match1 := bytes.Equal(siblings[0][l-33:l-1], nd)
	match2 := bytes.Equal(siblings[1][l-33:l-1], nd)
	assert.True(t, match1 || match2)
	assert.False(t, match1 && match2)
}
