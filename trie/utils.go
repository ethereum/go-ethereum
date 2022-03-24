// Copyright 2021 The go-ethereum Authors
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

import "sync"

// tracer tracks the changes of trie nodes and captures the origin value of the
// modified nodes. The life cycle of tracer corresponds to the trie commit operation
// and should be reset after each commit.
type tracer struct {
	insert map[string]struct{}
	delete map[string]struct{}
	origin map[string][]byte
	lock   sync.RWMutex
}

// newTracer initializes the tracer for capturing trie changes.
func newTracer() *tracer {
	return &tracer{
		insert: make(map[string]struct{}),
		delete: make(map[string]struct{}),
		origin: make(map[string][]byte),
	}
}

// onRead tracks the newly loaded trie node and caches the rlp-encoded blob internally.
func (t *tracer) onRead(key []byte, val []byte) {
	// Don't panic on uninitialized tracer, it's possible in testing.
	if t == nil {
		return
	}
	t.lock.Lock()
	defer t.lock.Unlock()

	t.origin[string(key)] = val
}

// onInsert tracks the newly inserted trie node. If it's already in the deletion set
// (resurrected node), then just wipe it from the deletion set as the "untouched".
func (t *tracer) onInsert(key []byte) {
	// Don't panic on uninitialized tracer, it's possible in testing.
	if t == nil {
		return
	}
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, present := t.delete[string(key)]; present {
		delete(t.delete, string(key))
		return
	}
	t.insert[string(key)] = struct{}{}
}

// onDelete tracks the newly deleted trie node. If it's already in the addition set,
// then just wipe it from the addition set as the "untouched".
func (t *tracer) onDelete(key []byte) {
	// Don't panic on uninitialized tracer, it's possible in testing.
	if t == nil {
		return
	}
	t.lock.Lock()
	defer t.lock.Unlock()

	if _, present := t.insert[string(key)]; present {
		delete(t.insert, string(key))
		return
	}
	t.delete[string(key)] = struct{}{}
}

// insertList returns the tracked inserted trie nodes in list.
func (t *tracer) insertList() [][]byte {
	// Don't panic on uninitialized tracer, it's possible in testing.
	if t == nil {
		return nil
	}
	t.lock.RLock()
	defer t.lock.RUnlock()

	var ret [][]byte
	for key := range t.insert {
		ret = append(ret, []byte(key))
	}
	return ret
}

// deleteList returns the tracked deleted trie nodes in list.
func (t *tracer) deleteList() [][]byte {
	// Don't panic on uninitialized tracer, it's possible in testing.
	if t == nil {
		return nil
	}
	t.lock.RLock()
	defer t.lock.RUnlock()

	var ret [][]byte
	for key := range t.delete {
		ret = append(ret, []byte(key))
	}
	return ret
}

// getPrev returns the cached original value of the specified node.
func (t *tracer) getPrev(key []byte) []byte {
	// Don't panic on uninitialized tracer, it's possible in testing.
	if t == nil {
		return nil
	}
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.origin[string(key)]
}

// reset cleans out the cached content.
func (t *tracer) reset() {
	// Don't panic on uninitialized tracer, it's possible in testing.
	if t == nil {
		return
	}
	t.lock.Lock()
	defer t.lock.Unlock()

	t.insert = make(map[string]struct{})
	t.delete = make(map[string]struct{})
	t.origin = make(map[string][]byte)
}

// copy returns a deep-copied tracer.
func (t *tracer) copy() *tracer {
	// Don't panic on uninitialized tracer, it's possible in testing.
	if t == nil {
		return nil
	}
	t.lock.RLock()
	defer t.lock.RUnlock()

	var (
		insert = make(map[string]struct{})
		delete = make(map[string]struct{})
		origin = make(map[string][]byte)
	)
	for key := range t.insert {
		insert[key] = struct{}{}
	}
	for key := range t.delete {
		delete[key] = struct{}{}
	}
	for key, val := range t.origin {
		origin[key] = val
	}
	return &tracer{
		insert: insert,
		delete: delete,
		origin: origin,
	}
}
