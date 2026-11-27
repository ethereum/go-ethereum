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

package triedb

import (
	"bytes"
	"fmt"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/triedb/internal"
)

type iteratorFactory func(prefix, start []byte) ethdb.Iterator

// rangeIterators walks the account and storage snapshots in the same direction.
type rangeIterators struct {
	newIterator iteratorFactory
	reverse     bool
	acct        *internal.HoldableIterator
	stor        *internal.HoldableIterator
}

func newRangeIterators(newIterator iteratorFactory, reverse bool, start common.Hash) *rangeIterators {
	r := &rangeIterators{
		newIterator: newIterator,
		reverse:     reverse,
	}
	r.open(start)
	return r
}

func (r *rangeIterators) open(start common.Hash) {
	var storageStart []byte
	if r.reverse {
		// Include every slot of the first owner in the descending scan.
		storageStart = slices.Concat(start[:], common.MaxHash[:])
	} else {
		// Include every slot of the first owner in the ascending scan.
		storageStart = start[:]
	}
	r.acct = openFlatIterator(r.newIterator, rawdb.SnapshotAccountPrefix, start[:], common.HashLength)
	r.stor = openFlatIterator(r.newIterator, rawdb.SnapshotStoragePrefix, storageStart, 2*common.HashLength)
}

func (r *rangeIterators) compare(a, b []byte) int {
	cmp := bytes.Compare(a, b)
	if r.reverse {
		return -cmp
	}
	return cmp
}

// resumeAfterStorage skips the owner handled by the storage shards. Start
// both scans at the adjacent hash in scan order, or finish if it wraps.
func (r *rangeIterators) resumeAfterStorage(owner common.Hash) {
	if r.reverse {
		for i := len(owner) - 1; i >= 0; i-- {
			owner[i]--
			if owner[i] != 0xff {
				r.open(owner)
				return
			}
		}
	} else {
		for i := len(owner) - 1; i >= 0; i-- {
			owner[i]++
			if owner[i] != 0 {
				r.open(owner)
				return
			}
		}
	}
	r.acct = emptyFlatIterator()
	r.stor = emptyFlatIterator()
}

func (r *rangeIterators) release() {
	if r.acct != nil {
		r.acct.Release()
	}
	if r.stor != nil {
		r.stor.Release()
	}
	r.acct, r.stor = nil, nil
}

// flushIfFull renews the read views after a write so compaction can reclaim
// obsolete tables even if a partition takes a long time to finish.
func (r *rangeIterators) flushIfFull(batch *generationBatch, where string) error {
	if !batch.full() {
		return nil
	}
	if err := batch.flush(); err != nil {
		return fmt.Errorf("flush batch (%s): %w", where, err)
	}
	var err error
	r.acct, err = reopenFlatIterator(r.newIterator, r.acct, rawdb.SnapshotAccountPrefix, common.HashLength)
	if err != nil {
		return fmt.Errorf("reopen account iterator: %w", err)
	}
	r.stor, err = reopenFlatIterator(r.newIterator, r.stor, rawdb.SnapshotStoragePrefix, 2*common.HashLength)
	if err != nil {
		return fmt.Errorf("reopen storage iterator: %w", err)
	}
	return nil
}

func openFlatIterator(newIterator iteratorFactory, prefix, start []byte, suffixLen int) *internal.HoldableIterator {
	it := newIterator(prefix, start)
	return internal.NewHoldableIterator(rawdb.NewKeyLengthIterator(it, len(prefix)+suffixLen))
}

// reopenFlatIterator replaces the iterator, positioning it so the next call
// to Next returns the first unprocessed entry in the same direction.
func reopenFlatIterator(newIterator iteratorFactory, old *internal.HoldableIterator, prefix []byte, suffixLen int) (*internal.HoldableIterator, error) {
	if !old.Next() {
		err := old.Error()
		old.Release()
		return emptyFlatIterator(), err
	}
	// The key slice becomes invalid when the iterator is released.
	next := bytes.Clone(old.Key()[len(prefix):])
	old.Release()
	return openFlatIterator(newIterator, prefix, next, suffixLen), nil
}

func emptyFlatIterator() *internal.HoldableIterator {
	return internal.NewHoldableIterator(memorydb.New().NewIterator(nil, nil))
}
