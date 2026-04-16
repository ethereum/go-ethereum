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

package state

import (
	"sync/atomic"
	"time"
)

// Stats contains all measurements gathered during state execution for
// debugging and metrics purposes.
type Stats struct {
	AccountReads    time.Duration // Account read time
	StorageReads    time.Duration // Storage read time
	CodeReads       time.Duration // Code read time
	AccountHashes   time.Duration // Account trie hash time
	AccountUpdates  time.Duration // Account trie update time
	StorageUpdates  time.Duration // Storage trie update and hash time
	HasherCommits   time.Duration // Trie commit time
	DatabaseCommits time.Duration // Database commit time

	AccountLoaded  int          // Number of accounts retrieved from the database during the state transition
	AccountUpdated int          // Number of accounts updated during the state transition
	AccountDeleted int          // Number of accounts deleted during the state transition
	StorageLoaded  int          // Number of storage slots retrieved from the database during the state transition
	StorageUpdated atomic.Int64 // Number of storage slots updated during the state transition
	StorageDeleted atomic.Int64 // Number of storage slots deleted during the state transition

	// CodeLoadBytes is the total number of bytes read from contract code.
	// This value may be smaller than the actual number of bytes read, since
	// some APIs (e.g. CodeSize) may load the entire code from either the
	// cache or the database when the size is not available in the cache.
	CodeLoaded      int // Number of contract code loaded during the state transition
	CodeLoadBytes   int // Total bytes of resolved code
	CodeUpdated     int // Number of contracts with code changes that persisted
	CodeUpdateBytes int // Total bytes of persisted code written
}

// StateReadTime returns the total time spent on the state read.
func (s *Stats) StateReadTime() time.Duration {
	return s.AccountReads + s.StorageReads + s.CodeReads
}

// StateHashTime returns the total time spent on the state hash.
func (s *Stats) StateHashTime() time.Duration {
	return s.AccountHashes + s.AccountUpdates + s.StorageUpdates
}
