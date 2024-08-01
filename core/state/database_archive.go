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

package state

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// stateReader wraps a pathdb archive reader.
type archiveReader struct {
	reader *pathdb.ArchiveReader
}

// Account implements Reader, retrieving the account specified by the address.
//
// An error will be returned if the associated snapshot is already stale or
// the requested account is not yet covered by the snapshot.
//
// The returned account might be nil if it's not existent.
func (r *archiveReader) Account(addr common.Address) (*types.StateAccount, error) {
	account, err := r.reader.Account(addr)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, nil
	}
	acct := &types.StateAccount{
		Nonce:    account.Nonce,
		Balance:  account.Balance,
		CodeHash: account.CodeHash,
		Root:     common.BytesToHash(account.Root),
	}
	if len(acct.CodeHash) == 0 {
		acct.CodeHash = types.EmptyCodeHash.Bytes()
	}
	if acct.Root == (common.Hash{}) {
		acct.Root = types.EmptyRootHash
	}
	return acct, nil
}

// Storage implements Reader, retrieving the storage slot specified by the
// address and slot key.
//
// An error will be returned if the associated snapshot is already stale or
// the requested storage slot is not yet covered by the snapshot.
//
// The returned storage slot might be empty if it's not existent.
func (r *archiveReader) Storage(addr common.Address, key common.Hash) (common.Hash, error) {
	blob, err := r.reader.Storage(addr, key)
	if err != nil {
		return common.Hash{}, err
	}
	if len(blob) == 0 {
		return common.Hash{}, nil
	}
	_, content, _, err := rlp.Split(blob)
	if err != nil {
		return common.Hash{}, err
	}
	var slot common.Hash
	slot.SetBytes(content)
	return slot, nil
}

// Stats returns the statistics of the reader, specifically detailing the time
// spent on account reading and storage reading.
func (r *archiveReader) Stats() (time.Duration, time.Duration) { return 0, 0 }

// Copy implements Reader, returning a deep-copied archive readerr.
func (r *archiveReader) Copy() Reader {
	return &archiveReader{reader: r.reader}
}

// ArchiveDB is the implementation of Database interface, with the ability to
// access historical state.
type ArchiveDB struct {
	Database
	triedb *triedb.Database
}

// NewArchiveDatabase creates an archive database.
func NewArchiveDatabase(db Database) *ArchiveDB {
	return &ArchiveDB{
		Database: db,
		triedb:   db.TrieDB(),
	}
}

// Reader implements Database interface, returning a reader of the specific state.
func (db *ArchiveDB) Reader(stateRoot common.Hash) (Reader, error) {
	// Short circuit if the requested state is available in live database
	r, err := db.Database.Reader(stateRoot)
	if err == nil {
		return r, nil
	}
	// Construct the archive reader then
	hr, err := db.triedb.HistoricReader(stateRoot)
	if err != nil {
		return nil, err
	}
	return &archiveReader{reader: hr}, nil
}
