// Copyright 2024 The go-ethereum Authors
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

package snap

import (
	"bytes"
	"math/rand"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/trie"
)

type replayer struct {
	paths    []string      // sort in fifo order
	hashes   []common.Hash // empty for deletion
	unknowns int           // counter for unknown write
}

func newBatchReplay() *replayer {
	return &replayer{}
}

func (r *replayer) decode(key []byte, value []byte) {
	account := rawdb.IsAccountTrieNode(key)
	storage := rawdb.IsStorageTrieNode(key)
	if !account && !storage {
		r.unknowns += 1
		return
	}
	var path []byte
	if account {
		_, path = rawdb.ResolveAccountTrieNodeKey(key)
	} else {
		_, owner, inner := rawdb.ResolveStorageTrieNode(key)
		path = append(owner.Bytes(), inner...)
	}
	r.paths = append(r.paths, string(path))

	if len(value) == 0 {
		r.hashes = append(r.hashes, common.Hash{})
	} else {
		r.hashes = append(r.hashes, crypto.Keccak256Hash(value))
	}
}

// updates returns a set of effective mutations. Multiple mutations targeting
// the same node path will be merged in FIFO order.
func (r *replayer) modifies() map[string]common.Hash {
	set := make(map[string]common.Hash)
	for i, path := range r.paths {
		set[path] = r.hashes[i]
	}
	return set
}

// updates returns the number of updates.
func (r *replayer) updates() int {
	var count int
	for _, hash := range r.modifies() {
		if hash == (common.Hash{}) {
			continue
		}
		count++
	}
	return count
}

// Put inserts the given value into the key-value data store.
func (r *replayer) Put(key []byte, value []byte) error {
	r.decode(key, value)
	return nil
}

// Delete removes the key from the key-value data store.
func (r *replayer) Delete(key []byte) error {
	r.decode(key, nil)
	return nil
}

func byteToHex(str []byte) []byte {
	l := len(str) * 2
	var nibbles = make([]byte, l)
	for i, b := range str {
		nibbles[i*2] = b / 16
		nibbles[i*2+1] = b % 16
	}
	return nibbles
}

// innerNodes returns the internal nodes narrowed by two boundaries along with
// the leftmost and rightmost sub-trie roots.
func innerNodes(first, last []byte, includeLeft, includeRight bool, nodes map[string]common.Hash, t *testing.T) (map[string]common.Hash, []byte, []byte) {
	var (
		leftRoot  []byte
		rightRoot []byte
		firstHex  = byteToHex(first)
		lastHex   = byteToHex(last)
		inner     = make(map[string]common.Hash)
	)
	for path, hash := range nodes {
		if hash == (common.Hash{}) {
			t.Fatalf("Unexpected deletion, %v", []byte(path))
		}
		// Filter out the siblings on the left side or the left boundary nodes.
		if !includeLeft && (bytes.Compare(firstHex, []byte(path)) > 0 || bytes.HasPrefix(firstHex, []byte(path))) {
			continue
		}
		// Filter out the siblings on the right side or the right boundary nodes.
		if !includeRight && (bytes.Compare(lastHex, []byte(path)) < 0 || bytes.HasPrefix(lastHex, []byte(path))) {
			continue
		}
		inner[path] = hash

		// Track the path of the leftmost sub trie root
		if leftRoot == nil || bytes.Compare(leftRoot, []byte(path)) > 0 {
			leftRoot = []byte(path)
		}
		// Track the path of the rightmost sub trie root
		if rightRoot == nil ||
			(bytes.Compare(rightRoot, []byte(path)) < 0) ||
			(bytes.Compare(rightRoot, []byte(path)) > 0 && bytes.HasPrefix(rightRoot, []byte(path))) {
			rightRoot = []byte(path)
		}
	}
	return inner, leftRoot, rightRoot
}

func buildPartial(owner common.Hash, db ethdb.KeyValueReader, batch ethdb.Batch, entries []*kv, first, last int) *replayer {
	tr := newPathTrie(owner, first != 0, db, batch)
	for i := first; i <= last; i++ {
		tr.update(entries[i].k, entries[i].v)
	}
	tr.commit(last == len(entries)-1)

	replay := newBatchReplay()
	batch.Replay(replay)

	return replay
}

// TestPartialGentree verifies if the trie constructed with partial states can
// generate consistent trie nodes that match those of the full trie.
func TestPartialGentree(t *testing.T) {
	for round := 0; round < 100; round++ {
		var (
			n       = rand.Intn(1024) + 10
			entries []*kv
		)
		for i := 0; i < n; i++ {
			var val []byte
			if rand.Intn(3) == 0 {
				val = testrand.Bytes(3)
			} else {
				val = testrand.Bytes(32)
			}
			entries = append(entries, &kv{
				k: testrand.Bytes(32),
				v: val,
			})
		}
		slices.SortFunc(entries, (*kv).cmp)

		nodes := make(map[string]common.Hash)
		tr := trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
			nodes[string(path)] = hash
		})
		for i := 0; i < len(entries); i++ {
			tr.Update(entries[i].k, entries[i].v)
		}
		tr.Hash()

		check := func(first, last int) {
			var (
				db    = rawdb.NewMemoryDatabase()
				batch = db.NewBatch()
			)
			// Build the partial tree with specific boundaries
			r := buildPartial(common.Hash{}, db, batch, entries, first, last)
			if r.unknowns > 0 {
				t.Fatalf("Unknown database write: %d", r.unknowns)
			}

			// Ensure all the internal nodes are produced
			var (
				set         = r.modifies()
				inner, _, _ = innerNodes(entries[first].k, entries[last].k, first == 0, last == len(entries)-1, nodes, t)
			)
			for path, hash := range inner {
				if _, ok := set[path]; !ok {
					t.Fatalf("Missing nodes %v", []byte(path))
				}
				if hash != set[path] {
					t.Fatalf("Inconsistent node, want %x, got: %x", hash, set[path])
				}
			}
			if r.updates() != len(inner) {
				t.Fatalf("Unexpected node write detected, want: %d, got: %d", len(inner), r.updates())
			}
		}
		for j := 0; j < 100; j++ {
			var (
				first int
				last  int
			)
			for {
				first = rand.Intn(len(entries))
				last = rand.Intn(len(entries))
				if first <= last {
					break
				}
			}
			check(first, last)
		}
		var cases = []struct {
			first int
			last  int
		}{
			{0, len(entries) - 1},                // full
			{1, len(entries) - 1},                // no left
			{2, len(entries) - 1},                // no left
			{2, len(entries) - 2},                // no left and right
			{2, len(entries) - 2},                // no left and right
			{len(entries) / 2, len(entries) / 2}, // single
			{0, 0},                               // single first
			{len(entries) - 1, len(entries) - 1}, // single last
		}
		for _, c := range cases {
			check(c.first, c.last)
		}
	}
}

// TestGentreeDanglingClearing tests if the dangling nodes falling within the
// path space of constructed tree can be correctly removed.
func TestGentreeDanglingClearing(t *testing.T) {
	for round := 0; round < 100; round++ {
		var (
			n       = rand.Intn(1024) + 10
			entries []*kv
		)
		for i := 0; i < n; i++ {
			var val []byte
			if rand.Intn(3) == 0 {
				val = testrand.Bytes(3)
			} else {
				val = testrand.Bytes(32)
			}
			entries = append(entries, &kv{
				k: testrand.Bytes(32),
				v: val,
			})
		}
		slices.SortFunc(entries, (*kv).cmp)

		nodes := make(map[string]common.Hash)
		tr := trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
			nodes[string(path)] = hash
		})
		for i := 0; i < len(entries); i++ {
			tr.Update(entries[i].k, entries[i].v)
		}
		tr.Hash()

		check := func(first, last int) {
			var (
				db    = rawdb.NewMemoryDatabase()
				batch = db.NewBatch()
			)
			// Write the junk nodes as the dangling
			var injects []string
			for path := range nodes {
				for i := 0; i < len(path); i++ {
					_, ok := nodes[path[:i]]
					if ok {
						continue
					}
					injects = append(injects, path[:i])
				}
			}
			if len(injects) == 0 {
				return
			}
			for _, path := range injects {
				rawdb.WriteAccountTrieNode(db, []byte(path), testrand.Bytes(32))
			}

			// Build the partial tree with specific range
			replay := buildPartial(common.Hash{}, db, batch, entries, first, last)
			if replay.unknowns > 0 {
				t.Fatalf("Unknown database write: %d", replay.unknowns)
			}
			set := replay.modifies()

			// Make sure the injected junks falling within the path space of
			// committed trie nodes are correctly deleted.
			_, leftRoot, rightRoot := innerNodes(entries[first].k, entries[last].k, first == 0, last == len(entries)-1, nodes, t)
			for _, path := range injects {
				if bytes.Compare([]byte(path), leftRoot) < 0 && !bytes.HasPrefix(leftRoot, []byte(path)) {
					continue
				}
				if bytes.Compare([]byte(path), rightRoot) > 0 {
					continue
				}
				if hash, ok := set[path]; !ok || hash != (common.Hash{}) {
					t.Fatalf("Missing delete, %v", []byte(path))
				}
			}
		}
		for j := 0; j < 100; j++ {
			var (
				first int
				last  int
			)
			for {
				first = rand.Intn(len(entries))
				last = rand.Intn(len(entries))
				if first <= last {
					break
				}
			}
			check(first, last)
		}
		var cases = []struct {
			first int
			last  int
		}{
			{0, len(entries) - 1},                // full
			{1, len(entries) - 1},                // no left
			{2, len(entries) - 1},                // no left
			{2, len(entries) - 2},                // no left and right
			{2, len(entries) - 2},                // no left and right
			{len(entries) / 2, len(entries) / 2}, // single
			{0, 0},                               // single first
			{len(entries) - 1, len(entries) - 1}, // single last
		}
		for _, c := range cases {
			check(c.first, c.last)
		}
	}
}

// TestFlushPartialTree tests the gentrie can produce complete inner trie nodes
// even with lots of batch flushes.
func TestFlushPartialTree(t *testing.T) {
	var entries []*kv
	for i := 0; i < 1024; i++ {
		var val []byte
		if rand.Intn(3) == 0 {
			val = testrand.Bytes(3)
		} else {
			val = testrand.Bytes(32)
		}
		entries = append(entries, &kv{
			k: testrand.Bytes(32),
			v: val,
		})
	}
	slices.SortFunc(entries, (*kv).cmp)

	nodes := make(map[string]common.Hash)
	tr := trie.NewStackTrie(func(path []byte, hash common.Hash, blob []byte) {
		nodes[string(path)] = hash
	})
	for i := 0; i < len(entries); i++ {
		tr.Update(entries[i].k, entries[i].v)
	}
	tr.Hash()

	var cases = []struct {
		first int
		last  int
	}{
		{0, len(entries) - 1},                // full
		{1, len(entries) - 1},                // no left
		{10, len(entries) - 1},               // no left
		{10, len(entries) - 2},               // no left and right
		{10, len(entries) - 10},              // no left and right
		{11, 11},                             // single
		{0, 0},                               // single first
		{len(entries) - 1, len(entries) - 1}, // single last
	}
	for _, c := range cases {
		var (
			db       = rawdb.NewMemoryDatabase()
			batch    = db.NewBatch()
			combined = db.NewBatch()
		)
		inner, _, _ := innerNodes(entries[c.first].k, entries[c.last].k, c.first == 0, c.last == len(entries)-1, nodes, t)

		tr := newPathTrie(common.Hash{}, c.first != 0, db, batch)
		for i := c.first; i <= c.last; i++ {
			tr.update(entries[i].k, entries[i].v)
			if rand.Intn(2) == 0 {
				tr.commit(false)

				batch.Replay(combined)
				batch.Write()
				batch.Reset()
			}
		}
		tr.commit(c.last == len(entries)-1)

		batch.Replay(combined)
		batch.Write()
		batch.Reset()

		r := newBatchReplay()
		combined.Replay(r)

		// Ensure all the internal nodes are produced
		set := r.modifies()
		for path, hash := range inner {
			if _, ok := set[path]; !ok {
				t.Fatalf("Missing nodes %v", []byte(path))
			}
			if hash != set[path] {
				t.Fatalf("Inconsistent node, want %x, got: %x", hash, set[path])
			}
		}
		if r.updates() != len(inner) {
			t.Fatalf("Unexpected node write detected, want: %d, got: %d", len(inner), r.updates())
		}
	}
}

// TestBoundSplit ensures two consecutive trie chunks are not overlapped with
// each other.
func TestBoundSplit(t *testing.T) {
	var entries []*kv
	for i := 0; i < 1024; i++ {
		var val []byte
		if rand.Intn(3) == 0 {
			val = testrand.Bytes(3)
		} else {
			val = testrand.Bytes(32)
		}
		entries = append(entries, &kv{
			k: testrand.Bytes(32),
			v: val,
		})
	}
	slices.SortFunc(entries, (*kv).cmp)

	for j := 0; j < 100; j++ {
		var (
			next int
			last int
			db   = rawdb.NewMemoryDatabase()

			lastRightRoot []byte
		)
		for {
			if next == len(entries) {
				break
			}
			last = rand.Intn(len(entries)-next) + next

			r := buildPartial(common.Hash{}, db, db.NewBatch(), entries, next, last)
			set := r.modifies()

			// Skip if the chunk is zero-size
			if r.updates() == 0 {
				next = last + 1
				continue
			}

			// Ensure the updates in two consecutive chunks are not overlapped.
			// The only overlapping part should be deletion.
			if lastRightRoot != nil && len(set) > 0 {
				// Derive the path of left-most node in this chunk
				var leftRoot []byte
				for path, hash := range r.modifies() {
					if hash == (common.Hash{}) {
						t.Fatalf("Unexpected deletion %v", []byte(path))
					}
					if leftRoot == nil || bytes.Compare(leftRoot, []byte(path)) > 0 {
						leftRoot = []byte(path)
					}
				}
				if bytes.HasPrefix(lastRightRoot, leftRoot) || bytes.HasPrefix(leftRoot, lastRightRoot) {
					t.Fatalf("Two chunks are not correctly separated, lastRight: %v, left: %v", lastRightRoot, leftRoot)
				}
			}

			// Track the updates as the last chunk
			var rightRoot []byte
			for path := range set {
				if rightRoot == nil ||
					(bytes.Compare(rightRoot, []byte(path)) < 0) ||
					(bytes.Compare(rightRoot, []byte(path)) > 0 && bytes.HasPrefix(rightRoot, []byte(path))) {
					rightRoot = []byte(path)
				}
			}
			lastRightRoot = rightRoot
			next = last + 1
		}
	}
}

// TestTinyPartialTree tests if the partial tree is too tiny(has less than two
// states), then nothing should be committed.
func TestTinyPartialTree(t *testing.T) {
	var entries []*kv
	for i := 0; i < 1024; i++ {
		var val []byte
		if rand.Intn(3) == 0 {
			val = testrand.Bytes(3)
		} else {
			val = testrand.Bytes(32)
		}
		entries = append(entries, &kv{
			k: testrand.Bytes(32),
			v: val,
		})
	}
	slices.SortFunc(entries, (*kv).cmp)

	for i := 0; i < len(entries); i++ {
		next := i
		last := i + 1
		if last >= len(entries) {
			last = len(entries) - 1
		}
		db := rawdb.NewMemoryDatabase()
		r := buildPartial(common.Hash{}, db, db.NewBatch(), entries, next, last)

		if next != 0 && last != len(entries)-1 {
			if r.updates() != 0 {
				t.Fatalf("Unexpected data writes, got: %d", r.updates())
			}
		}
	}
}
