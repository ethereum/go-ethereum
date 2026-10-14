// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// An online self-conversion reads the state of a node that keeps following
// the chain. pathdb's own iterators cannot serve it: they go stale whenever
// the disk layer moves, every block on a synced node. What pathdb leaves in
// the database, though, is always one whole state: it persists a state as one
// atomic batch carrying the state's root and id (triedb/pathdb/buffer.go),
// and on a synced node with a complete flat state nothing else writes the
// merkle flat state. A database iterator reads at the sequence number it was
// opened at, for as long as it lives. So iterators opened together, with no
// flush between the first and the last, scan one persisted state however far
// the chain moves on meanwhile.

// capturePersisted opens, at each range start, a raw iterator over the
// merkle flat accounts and one over the flat storage, all reading the state
// pathdb last persisted, and returns that state's root and a source per
// range. Reading the persisted id and root before and after opening them
// shows that no flush landed in between. Check 2 still proves the scan
// against the root. Release the sources once the scan is done: until then
// they pin what the database held when they opened.
func capturePersisted(db ethdb.Database, ranges []hashRange) (common.Hash, []*persistedSource, error) {
	for range 3 {
		id, root := rawdb.ReadPersistentStateID(db), rawdb.ReadSnapshotRoot(db)
		sources := make([]*persistedSource, len(ranges))
		for i, r := range ranges {
			sources[i] = &persistedSource{
				accts: db.NewIterator(rawdb.SnapshotAccountPrefix, r.start[:]),
				slots: db.NewIterator(rawdb.SnapshotStoragePrefix, r.start[:]),
			}
		}
		if rawdb.ReadPersistentStateID(db) == id && rawdb.ReadSnapshotRoot(db) == root {
			return root, sources, nil
		}
		for _, s := range sources {
			s.release()
		}
	}
	return common.Hash{}, nil, errors.New("the persisted state moved on while every scan iterator opened, three times running")
}

// persistedSource reads one worker range of the persisted merkle flat state
// through the two iterators capturePersisted opened for it, forward only:
// the accounts lead, and each account's storage comes off one shared cursor,
// since an iterator opened later would read a later state.
type persistedSource struct {
	accts, slots ethdb.Iterator
	held         bool // slots holds an entry no account has taken yet
	once         sync.Once
}

func (s *persistedSource) accounts(common.Hash) (pathdb.AccountIterator, error) {
	return &rawAccounts{it: s.accts}, nil
}

func (s *persistedSource) storage(accountHash, _ common.Hash) (pathdb.StorageIterator, error) {
	return &rawStorage{src: s, account: accountHash}, nil
}

// release closes both iterators; later calls do nothing.
func (s *persistedSource) release() {
	s.once.Do(func() {
		s.accts.Release()
		s.slots.Release()
	})
}

// rawAccounts walks flat account entries: "a" ‖ hash -> slim account RLP.
type rawAccounts struct{ it ethdb.Iterator }

func (r *rawAccounts) Next() bool {
	for r.it.Next() {
		if len(r.it.Key()) == len(rawdb.SnapshotAccountPrefix)+common.HashLength {
			return true
		}
	}
	return false
}
func (r *rawAccounts) Error() error { return r.it.Error() }
func (r *rawAccounts) Hash() common.Hash {
	return common.BytesToHash(r.it.Key()[len(rawdb.SnapshotAccountPrefix):])
}
func (r *rawAccounts) Account() []byte { return r.it.Value() }
func (r *rawAccounts) Release()        {} // the source releases the iterator

// rawStorage walks one account's flat storage entries off the source's
// shared cursor: "o" ‖ account hash ‖ slot hash -> slot RLP. It stops at the
// first entry of a later account, which stays held for that account.
type rawStorage struct {
	src     *persistedSource
	account common.Hash
}

func (r *rawStorage) Next() bool {
	s := r.src
	for {
		if !s.held {
			if !s.slots.Next() {
				return false
			}
			s.held = true
		}
		key := s.slots.Key()
		if len(key) != len(rawdb.SnapshotStoragePrefix)+2*common.HashLength {
			s.held = false
			continue
		}
		switch bytes.Compare(key[len(rawdb.SnapshotStoragePrefix):len(rawdb.SnapshotStoragePrefix)+common.HashLength], r.account[:]) {
		case -1:
			// Storage of an account the scan never asked for, which a
			// consistent state does not hold: nothing commits it.
			s.held = false
		case 0:
			s.held = false
			return true
		default:
			return false
		}
	}
}
func (r *rawStorage) Error() error { return r.src.slots.Error() }
func (r *rawStorage) Hash() common.Hash {
	return common.BytesToHash(r.src.slots.Key()[len(rawdb.SnapshotStoragePrefix)+common.HashLength:])
}
func (r *rawStorage) Slot() []byte { return r.src.slots.Value() }
func (r *rawStorage) Release()     {} // the cursor outlives the account
