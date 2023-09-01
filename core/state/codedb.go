// Copyright 2023 The go-ethereum Authors
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
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

const (
	// Number of codehash->size associations to keep.
	codeSizeCacheSize = 100000

	// Cache size granted for caching clean code.
	codeCacheSize = 64 * 1024 * 1024
)

var errMismatchedLength = errors.New("provided lists have different lengths")

// CodeDB is an implementation of the CodeStore interface, designed for providing
// efficient read and write functionalities for contract code.
type CodeDB struct {
	db    ethdb.KeyValueStore
	size  *lru.Cache[common.Hash, int]
	cache *lru.SizeConstrainedCache[common.Hash, []byte]
}

// NewCodeDB returns a codeDB instance with given database.
func NewCodeDB(db ethdb.KeyValueStore) *CodeDB {
	return &CodeDB{
		db:    db,
		size:  lru.NewCache[common.Hash, int](codeSizeCacheSize),
		cache: lru.NewSizeConstrainedCache[common.Hash, []byte](codeCacheSize),
	}
}

// ReadCode implements CodeReader, retrieving a particular contract's code
// with given contract address and code hash.
func (db *CodeDB) ReadCode(address common.Address, codeHash common.Hash) ([]byte, error) {
	code, _ := db.cache.Get(codeHash)
	if len(code) > 0 {
		return code, nil
	}
	code = rawdb.ReadCode(db.db, codeHash)
	if len(code) > 0 {
		db.cache.Add(codeHash, code)
		db.size.Add(codeHash, len(code))
		return code, nil
	}
	return nil, errors.New("not found")
}

// ReadCodeSize implements CodeReader, retrieving a particular contracts code's size
// with given contract address and code hash.
func (db *CodeDB) ReadCodeSize(addr common.Address, codeHash common.Hash) (int, error) {
	if cached, ok := db.size.Get(codeHash); ok {
		return cached, nil
	}
	code, err := db.ReadCode(addr, codeHash)
	if err != nil {
		return 0, err
	}
	return len(code), nil
}

// WriteCodes implements CodeWriter, writing the provided a list of contract codes
// into database.
func (db *CodeDB) WriteCodes(addresses []common.Address, hashes []common.Hash, codes [][]byte) error {
	if len(addresses) != len(hashes) {
		return errMismatchedLength
	}
	if len(addresses) != len(codes) {
		return errMismatchedLength
	}
	batch := db.db.NewBatch()
	for i := 0; i < len(addresses); i++ {
		rawdb.WriteCode(batch, hashes[i], codes[i])
	}
	return batch.Write()
}

// ReadCodeWithPrefix retrieves a particular contract's code. If the code can't
// be found in the cache, then check the existence with **new** db scheme.
func (db *CodeDB) ReadCodeWithPrefix(address common.Address, codeHash common.Hash) ([]byte, error) {
	code, _ := db.cache.Get(codeHash)
	if len(code) > 0 {
		return code, nil
	}
	code = rawdb.ReadCodeWithPrefix(db.db, codeHash)
	if len(code) > 0 {
		db.cache.Add(codeHash, code)
		db.size.Add(codeHash, len(code))
		return code, nil
	}
	return nil, errors.New("not found")
}
