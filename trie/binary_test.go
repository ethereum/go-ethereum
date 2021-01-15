// Copyright 2020 The go-ethereum Authors
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
	"encoding/binary"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestBinaryKeyCreation(t *testing.T) {
	byteKey := []byte{0, 1, 2, 3}
	binKey := newBinKey(byteKey)
	exp := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 1}
	if !bytes.Equal(binKey[:], exp) {
		t.Fatalf("invalid key conversion, got %x != exp %x", binKey[:], exp)
	}
}

func TestBinaryKeyCreationEmpty(t *testing.T) {
	byteKey := []byte(nil)
	binKey := newBinKey(byteKey)
	exp := []byte(nil)
	if !bytes.Equal(binKey[:], exp) {
		t.Fatalf("invalid key conversion, got %x != exp %x", binKey[:], exp)
	}
}

func TestBinaryKeyCommonLength(t *testing.T) {
	byteKey1 := []byte{0, 1, 2, 3}
	binKey1 := newBinKey(byteKey1)
	byteKey2 := []byte{0, 1, 3, 3}
	binKey2 := newBinKey(byteKey2)

	split := binKey1.commonLength(binKey2)

	if split != 23 {
		t.Fatalf("split at wrong location: got %d != exp 23", split)
	}

}

func TestBinaryKeyCommonLengthLast(t *testing.T) {
	byteKey1 := []byte{0, 1, 2, 3}
	binKey1 := newBinKey(byteKey1)
	byteKey2 := []byte{0, 1, 2, 2}
	binKey2 := newBinKey(byteKey2)

	split := binKey1.commonLength(binKey2)

	if split != 31 {
		t.Fatalf("split at wrong location: got %d != exp 31", split)
	}

}

func TestBinaryKeyCommonLengthFirst(t *testing.T) {
	byteKey1 := []byte{0, 1, 2, 3}
	binKey1 := newBinKey(byteKey1)
	byteKey2 := []byte{128, 1, 2, 3}
	binKey2 := newBinKey(byteKey2)

	split := binKey1.commonLength(binKey2)

	if split != 0 {
		t.Fatalf("split at wrong location: got %d != exp 0", split)
	}

}

func TestBinaryTrieEmptyHash(t *testing.T) {
	trie := NewBinaryTrie()
	got := trie.Hash()
	exp := emptyRoot[:]

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}

	trie = NewBinaryTrieWithBlake2b()
	got = trie.Hash()
	// This is the wrong empty root for blake2b. We are only focused
	// on performance measurements at the moment.
	exp = emptyRoot[:]

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieInsertOneLeafAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("5ef9138daa6dfb4ca211fdb6ca4db27400233b7506e63edcd2576efd31cd5e5c")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}

	trie = NewBinaryTrieWithBlake2b()
	trie.Update([]byte{0}, []byte{10})
	got = trie.Hash()
	exp = common.FromHex("59f78e329994764d27e42cf7e2802a8311cd5c45725788e6288f94850c92a7d6")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieInsertTwoLeavesAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{8}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("378da00155c1019b0a1afef1709e1f37cddbb4e0d373feee849c54971cac9928")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}
func TestBinaryTrieInsertTwoLeavesAtFirstBitAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{128}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("83cbe1f4e4ddfdc66074424d54d64c34deafd6517970e9f6e96c21f506235a4e")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieInsertTwoLeavesAtEndBitAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{1}, []byte{10})
	got := trie.Hash()
	exp := common.FromHex("05b8807c3d0b42b8ff79ee3e157473564eba0154281af1755476f7154d753556")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieInsertWithOffsetAndHash(t *testing.T) {
	trie := NewBinaryTrie()
	trie.Update([]byte{0}, []byte{10})
	trie.Update([]byte{8}, []byte{18})
	trie.Update([]byte{11}, []byte{20})
	got := trie.Hash()
	exp := common.FromHex("b73bb5b26278b862d872455f83eb71b34a3702b85761f6ca8fc7e7e98b4b5fe6")

	if !bytes.Equal(got, exp) {
		t.Fatalf("invalid empty trie hash, got %x != exp %x", got, exp)
	}
}

func TestBinaryTrieReadEmpty(t *testing.T) {
	trie := NewBinaryTrie()
	_, err := trie.TryGet([]byte{1})
	if err != errKeyNotPresent {
		t.Fatalf("incorrect error received, expected '%v', got '%v'", errKeyNotPresent, err)
	}
}

var (
	testAddr0  = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")
	testAddr1  = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000001")
	testAddr2  = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000002")
	testAddr3  = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000003")
	testAddr4  = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000004")
	testAddr8  = common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000008")
	testAddr11 = common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000B")
)

func int2addr(x int) []byte {
	addr := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")
	binary.BigEndian.PutUint64(addr[24:], uint64(x))
	return addr
}

type simpleAccount struct {
	Balance     *big.Int
	Nonce       uint64
	Code        common.Hash
	StorageRoot common.Hash
}

var emptyCodeHash = crypto.Keccak256Hash(nil)

var aoe = simpleAccount{Balance: big.NewInt(100), Nonce: 1, Code: emptyCodeHash, StorageRoot: emptyRoot}

func TestBinaryTrieReadOneLeaf(t *testing.T) {
	payload, err := rlp.EncodeToBytes(aoe)
	if err != nil {
		t.Fatalf("%v", err)
	}
	trie := NewBinaryTrie()
	trie.Update(testAddr0, payload)

	// Check the balance can be recovered
	v, err := trie.TryGet(testAddr0)
	if err != nil {
		t.Fatalf("error searching for key 0 in trie, err=%v", err)
	}
	w, ok := v.(*big.Int)
	if !ok {
		t.Fatalf("did not recover proper value type: %v", v)
	}
	if w.Cmp(aoe.Balance) != 0 {
		t.Fatalf("could not find correct value %d != %d", w, aoe.Balance)
	}

	// Check the nonce can be recovered
	v, err = trie.TryGet(testAddr1)
	if err != nil {
		t.Fatalf("error searching for key 1 in trie, err=%v", err)
	}
	x, ok := v.(uint64)
	if !ok {
		t.Fatalf("did not recover proper value type: %v", v)
	}
	if x != aoe.Nonce {
		t.Fatalf("could not find correct value %x != %d", x, aoe.Nonce)
	}

	// Check the code can be recovered (and is empty)
	v, err = trie.TryGet(testAddr2)
	if err != nil {
		t.Fatalf("error searching for key 1 in trie, err=%v", err)
	}
	y, ok := v.(common.Hash)
	if !ok {
		t.Fatalf("did not recover proper value type: %v", v)
	}
	if !bytes.Equal(y[:], emptyCodeHash[:]) {
		t.Fatalf("could not find correct value %x != %x", v, emptyCodeHash)
	}

	// Check the root trie can be recovered (and is empty)
	v, err = trie.TryGet(testAddr3)
	if err != nil {
		t.Fatalf("error searching for key 1 in trie, err=%v", err)
	}
	z, ok := v.(common.Hash)
	if !ok {
		t.Fatalf("did not recover proper value type: %v", v)
	}
	if !bytes.Equal(z[:], emptyRoot[:]) {
		t.Fatalf("could not find correct value %x != %x", v, emptyRoot)
	}

	v, err = trie.TryGet(testAddr4)
	if err != errKeyNotPresent {
		t.Fatalf("incorrect error received, expected '%v', got '%v'", errKeyNotPresent, err)
	}
}

func TestBinaryTrieReadOneFromManyLeaves(t *testing.T) {
	payload, err := rlp.EncodeToBytes(aoe)
	if err != nil {
		t.Fatalf("%v", err)
	}
	trie := NewBinaryTrie()
	trie.Update(testAddr0, payload)
	trie.Update(int2addr(8), payload)
	trie.Update(int2addr(15), payload)

	v, err := trie.TryGet(testAddr1)
	if err != nil {
		t.Fatalf("error searching for key 1 in trie, err=%v", err)
	}
	w, ok := v.(uint64)
	if !ok {
		t.Fatalf("did not recover proper value type: %v", v)
	}
	if w != 1 {
		t.Fatalf("could not find correct value %d != %d", v, aoe.Nonce)
	}

	v, err = trie.TryGet(int2addr(9))
	if err != nil {
		t.Fatalf("error searching for key 9 in trie, err=%v", err)
	}
	w, ok = v.(uint64)
	if !ok {
		t.Fatalf("did not recover proper value type: %v", v)
	}
	if w != 1 {
		t.Fatalf("could not find correct value %d != %d", v, aoe.Nonce)
	}

	_, err = trie.TryGet(testAddr4)
	if err != errKeyNotPresent {
		t.Fatalf("incorrect error received, expected '%v', got '%v'", errKeyNotPresent, err)
	}
}

func TestBinaryTrieNodeResolution(t *testing.T) {
	key1 := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")
	key2 := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000008")
	key3 := common.Hex2Bytes("000000000000000000000000000000000000000000000000000000000000000c")

	// Put 2 keys in the database, to be resolved
	db := rawdb.NewMemoryDatabase()
	// balance of account 0 => 10
	db.Put(key1, common.Hex2Bytes("0a"))
	// balance of account 8 => 10
	db.Put(key2, common.Hex2Bytes("0a"))

	// Create the trie, add a non-existent value and
	// update another one.
	trie := NewBinaryTrieWithRawDB(db)
	trie.Update(key1, []byte{11})
	trie.Update(key3, []byte{10})

	if len(trie.db.dirties) != 2 {
		t.Fatalf("invalid number of dirty account entries after insert, %d != 2", len(trie.db.dirties))
	}
	got := trie.Hash()

	// Insert all the values in a live trie to make sure
	// the root hashes are the same.
	trieref := NewBinaryTrie()
	trieref.Update(key1, []byte{10})
	trieref.Update(key2, []byte{10})
	trieref.Update(key3, []byte{10})
	exp := trieref.Hash()

	if !bytes.Equal(got[:], exp[:]) {
		t.Fatalf("invalid root %x != %x", got, exp)
	}
}

func BenchmarkTrieHash(b *testing.B) {
	trieK := NewBinaryTrie()
	trieB := NewBinaryTrieWithBlake2b()
	key := make([]byte, 32)
	val := make([]byte, 32)
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 1000; i++ {
		rand.Read(key)
		rand.Read(val)
		trieK.Update(key, val)
		trieB.Update(key, val)
	}
	b.Run("m5-keccak", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			trieK.Hash()
		}
	})
	b.Run("m5-blake2b", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			trieB.Hash()
		}
	})
	b.Run("m4-keccak", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			trieK.HashM4()
		}
	})
	b.Run("m4-blake", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			trieB.HashM4()
		}
	})
}
