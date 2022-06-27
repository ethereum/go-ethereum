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
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common"
	zkt "github.com/scroll-tech/go-ethereum/core/types/zktrie"
	"github.com/scroll-tech/go-ethereum/ethdb/memorydb"
)

func newEmptyZkTrie() *ZkTrie {
	trie, _ := NewZkTrie(common.Hash{}, NewZktrieDatabase(memorydb.New()))
	return trie
}

// makeTestSecureTrie creates a large enough secure trie for testing.
func makeTestZkTrie() (*ZktrieDatabase, *ZkTrie, map[string][]byte) {
	// Create an empty trie
	triedb := NewZktrieDatabase(memorydb.New())
	trie, _ := NewZkTrie(common.Hash{}, triedb)

	// Fill it with some arbitrary data
	content := make(map[string][]byte)
	for i := byte(0); i < 255; i++ {
		// Map the same data under multiple keys
		key, val := common.LeftPadBytes([]byte{1, i}, 32), bytes.Repeat([]byte{i}, 32)
		content[string(key)] = val
		trie.Update(key, val)

		key, val = common.LeftPadBytes([]byte{2, i}, 32), bytes.Repeat([]byte{i}, 32)
		content[string(key)] = val
		trie.Update(key, val)

		// Add some other data to inflate the trie
		for j := byte(3); j < 13; j++ {
			key, val = common.LeftPadBytes([]byte{j, i}, 32), bytes.Repeat([]byte{j, i}, 16)
			content[string(key)] = val
			trie.Update(key, val)
		}
	}
	trie.Commit(nil)

	// Return the generated trie
	return triedb, trie, content
}

func TestZktrieDelete(t *testing.T) {
	t.Skip("var-len kv not supported")
	trie := newEmptyZkTrie()
	vals := []struct{ k, v string }{
		{"do", "verb"},
		{"ether", "wookiedoo"},
		{"horse", "stallion"},
		{"shaman", "horse"},
		{"doge", "coin"},
		{"ether", ""},
		{"dog", "puppy"},
		{"shaman", ""},
	}
	for _, val := range vals {
		if val.v != "" {
			trie.Update([]byte(val.k), []byte(val.v))
		} else {
			trie.Delete([]byte(val.k))
		}
	}
	hash := trie.Hash()
	exp := common.HexToHash("29b235a58c3c25ab83010c327d5932bcf05324b7d6b1185e650798034783ca9d")
	if hash != exp {
		t.Errorf("expected %x got %x", exp, hash)
	}
}

func TestZktrieGetKey(t *testing.T) {
	trie := newEmptyZkTrie()
	key := []byte("0a1b2c3d4e5f6g7h8i9j0a1b2c3d4e5f")
	value := []byte("9j8i7h6g5f4e3d2c1b0a9j8i7h6g5f4e")
	trie.Update(key, value)

	kPreimage := zkt.NewByte32FromBytesPaddingZero(key)
	kHash, err := kPreimage.Hash()
	assert.Nil(t, err)

	if !bytes.Equal(trie.Get(key), value) {
		t.Errorf("Get did not return bar")
	}
	if k := trie.GetKey(kHash.Bytes()); !bytes.Equal(k, key) {
		t.Errorf("GetKey returned %q, want %q", k, key)
	}
}

func TestZkTrieConcurrency(t *testing.T) {
	// Create an initial trie and copy if for concurrent access
	_, trie, _ := makeTestZkTrie()

	threads := runtime.NumCPU()
	tries := make([]*ZkTrie, threads)
	for i := 0; i < threads; i++ {
		cpy := *trie
		tries[i] = &cpy
	}
	// Start a batch of goroutines interactng with the trie
	pend := new(sync.WaitGroup)
	pend.Add(threads)
	for i := 0; i < threads; i++ {
		go func(index int) {
			defer pend.Done()

			for j := byte(0); j < 255; j++ {
				// Map the same data under multiple keys
				key, val := common.LeftPadBytes([]byte{byte(index), 1, j}, 32), bytes.Repeat([]byte{j}, 32)
				tries[index].Update(key, val)

				key, val = common.LeftPadBytes([]byte{byte(index), 2, j}, 32), bytes.Repeat([]byte{j}, 32)
				tries[index].Update(key, val)

				// Add some other data to inflate the trie
				for k := byte(3); k < 13; k++ {
					key, val = common.LeftPadBytes([]byte{byte(index), k, j}, 32), bytes.Repeat([]byte{k, j}, 16)
					tries[index].Update(key, val)
				}
			}
			tries[index].Commit(nil)
		}(i)
	}
	// Wait for all threads to finish
	pend.Wait()
}
