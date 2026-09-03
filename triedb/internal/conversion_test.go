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

package internal

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// fakeStorageIterator is a StorageIterator over a fixed list of slots.
type fakeStorageIterator struct {
	count int
	idx   int
}

func (it *fakeStorageIterator) Next() bool {
	if it.idx >= it.count {
		return false
	}
	it.idx++
	return true
}
func (it *fakeStorageIterator) Error() error      { return nil }
func (it *fakeStorageIterator) Hash() common.Hash { return common.BytesToHash([]byte{byte(it.idx)}) }
func (it *fakeStorageIterator) Slot() []byte      { return []byte{byte(it.idx)} }
func (it *fakeStorageIterator) Release()          {}

// TestGenerateTrieRootCancel verifies that GenerateTrieRoot aborts with
// ErrCancelled when the cancel channel is closed.
func TestGenerateTrieRootCancel(t *testing.T) {
	t.Parallel()
	it := &fakeStorageIterator{count: 10_000}
	cancel := make(chan struct{})
	close(cancel)
	_, err := GenerateTrieRoot(nil, "", it, common.HexToHash("0xaa"), StackTrieGenerate, nil, nil, false, cancel)
	if !errors.Is(err, ErrCancelled) {
		t.Fatalf("expected ErrCancelled, got %v", err)
	}
}
