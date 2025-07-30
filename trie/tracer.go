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

package trie

import (
	"maps"
	"slices"
)

// opTracer tracks the changes of trie nodes. During the trie operations,
// some nodes can be deleted from the trie, while these deleted nodes
// won't be captured by trie.Hasher or trie.Committer. Thus, these deleted
// nodes won't be removed from the disk at all. opTracer is an auxiliary tool
// used to track all insert and delete operations of trie and capture all
// deleted nodes eventually.
//
// The changed nodes can be mainly divided into two categories: the leaf
// node and intermediate node. The former is inserted/deleted by callers
// while the latter is inserted/deleted in order to follow the rule of trie.
// This tool can track all of them no matter the node is embedded in its
// parent or not, but valueNode is never tracked.
//
// Note opTracer is not thread-safe, callers should be responsible for handling
// the concurrency issues by themselves.
type opTracer struct {
	inserts map[string]struct{}
	deletes map[string]struct{}
}

// newOpTracer initializes the tracer for capturing trie changes.
func newOpTracer() *opTracer {
	return &opTracer{
		inserts: make(map[string]struct{}),
		deletes: make(map[string]struct{}),
	}
}

// onInsert tracks the newly inserted trie node. If it's already
// in the deletion set (resurrected node), then just wipe it from
// the deletion set as it's "untouched".
func (t *opTracer) onInsert(path []byte) {
	if _, present := t.deletes[string(path)]; present {
		delete(t.deletes, string(path))
		return
	}
	t.inserts[string(path)] = struct{}{}
}

// onDelete tracks the newly deleted trie node. If it's already
// in the addition set, then just wipe it from the addition set
// as it's untouched.
func (t *opTracer) onDelete(path []byte) {
	if _, present := t.inserts[string(path)]; present {
		delete(t.inserts, string(path))
		return
	}
	t.deletes[string(path)] = struct{}{}
}

// reset clears the content tracked by tracer.
func (t *opTracer) reset() {
	clear(t.inserts)
	clear(t.deletes)
}

// copy returns a deep copied tracer instance.
func (t *opTracer) copy() *opTracer {
	return &opTracer{
		inserts: maps.Clone(t.inserts),
		deletes: maps.Clone(t.deletes),
	}
}

// deletedList returns a list of node paths which are deleted from the trie.
func (t *opTracer) deletedList() [][]byte {
	paths := make([][]byte, 0, len(t.deletes))
	for path := range t.deletes {
		paths = append(paths, []byte(path))
	}
	return paths
}

// prevalueTracer tracks the original values of resolved trie nodes. Cached trie
// node values are expected to be immutable. A zero-size node value is treated as
// non-existent and should not occur in practice.
//
// Note prevalueTracer is not thread-safe, callers should be responsible for
// handling the concurrency issues by themselves.
type prevalueTracer struct {
	data map[string][]byte
}

// newPrevalueTracer initializes the tracer for capturing resolved trie nodes.
func newPrevalueTracer() *prevalueTracer {
	return &prevalueTracer{
		data: make(map[string][]byte),
	}
}

// put tracks the newly loaded trie node and caches its RLP-encoded
// blob internally. Do not modify the value outside this function,
// as it is not deep-copied.
func (t *prevalueTracer) put(path []byte, val []byte) {
	t.data[string(path)] = val
}

// get returns the cached trie node value. If the node is not found, nil will
// be returned.
func (t *prevalueTracer) get(path []byte) []byte {
	return t.data[string(path)]
}

// hasList returns a list of flags indicating whether the corresponding trie nodes
// specified by the path exist in the trie.
func (t *prevalueTracer) hasList(list [][]byte) []bool {
	exists := make([]bool, 0, len(list))
	for _, path := range list {
		_, ok := t.data[string(path)]
		exists = append(exists, ok)
	}
	return exists
}

// values returns a list of values of the cached trie nodes.
func (t *prevalueTracer) values() [][]byte {
	return slices.Collect(maps.Values(t.data))
}

// reset resets the cached content in the prevalueTracer.
func (t *prevalueTracer) reset() {
	clear(t.data)
}

// copy returns a copied prevalueTracer instance.
func (t *prevalueTracer) copy() *prevalueTracer {
	// Shadow clone is used, as the cached trie node values are immutable
	return &prevalueTracer{
		data: maps.Clone(t.data),
	}
}
