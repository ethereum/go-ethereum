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

package snapshot

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestCheckDanglingStorage(t *testing.T) {
	t.Run("disk storage", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		rawdb.WriteStorageSnapshot(db, common.HexToHash("0x01"), common.HexToHash("0x02"), []byte{0x01})

		if err := CheckDanglingStorage(db); err == nil {
			t.Fatal("dangling disk storage was not reported")
		}
	})

	t.Run("disk iterator", func(t *testing.T) {
		want := errors.New("test iterator error")
		db := &iteratorErrorDatabase{
			KeyValueStore: rawdb.NewMemoryDatabase(),
			err:           want,
		}
		if err := CheckDanglingStorage(db); !errors.Is(err, want) {
			t.Fatalf("unexpected error: got %v, want %v", err, want)
		}
	})

	t.Run("journal storage", func(t *testing.T) {
		db := rawdb.NewMemoryDatabase()
		account := common.HexToHash("0x01")
		root := common.HexToHash("0x02")
		storage := []journalStorage{{
			Hash: account,
			Keys: []common.Hash{common.HexToHash("0x03")},
			Vals: [][]byte{{0x01}},
		}}
		var journal bytes.Buffer
		for _, entry := range []any{journalCurrentVersion, common.Hash{}, root, []journalAccount{}, storage} {
			if err := rlp.Encode(&journal, entry); err != nil {
				t.Fatal(err)
			}
		}
		rawdb.WriteSnapshotJournal(db, journal.Bytes())

		if err := CheckDanglingStorage(db); err == nil {
			t.Fatal("dangling journal storage was not reported")
		}
	})
}

type iteratorErrorDatabase struct {
	ethdb.KeyValueStore
	err error
}

func (db *iteratorErrorDatabase) NewIterator(prefix, start []byte) ethdb.Iterator {
	return &iteratorError{
		Iterator: db.KeyValueStore.NewIterator(prefix, start),
		err:      db.err,
	}
}

type iteratorError struct {
	ethdb.Iterator
	err error
}

func (it *iteratorError) Error() error {
	return it.err
}
