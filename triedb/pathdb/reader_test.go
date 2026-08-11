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

package pathdb_test

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// TestStandaloneCorruptAccountPanic verifies that a corrupt account blob stored
// in the flat state surfaces an error from StateReader.Account instead of
// panicking the process.
//
// See https://github.com/ethereum/go-ethereum/issues/35472: both (*reader).Account
// and (*HistoricalStateReader).Account documented that "an error will be returned
// if the read operation exits abnormally", but the decode sites called panic(err)
// on a corrupt account blob.
func TestStandaloneCorruptAccountPanic(t *testing.T) {
	disk, err := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{Ancient: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	cfg := &pathdb.Config{NoAsyncFlush: true}
	db := pathdb.New(disk, cfg, false)
	defer db.Close()
	defer disk.Close()

	addr := common.HexToAddress("0x1000")
	addrHash := crypto.Keccak256Hash(addr.Bytes())
	accountRLP := types.SlimAccountRLP(types.StateAccount{Balance: new(uint256.Int)})

	tr, err := trie.New(trie.StateTrieID(types.EmptyRootHash), db)
	if err != nil {
		t.Fatal(err)
	}
	tr.Update(addrHash.Bytes(), accountRLP)
	root, set := tr.Commit(false)

	merged := trienode.NewMergedNodeSet()
	merged.Merge(set)
	states := pathdb.NewStateSetWithOrigin(map[common.Hash][]byte{addrHash: accountRLP}, nil, nil, nil, false)
	if err := db.Update(root, types.EmptyRootHash, 0, merged, states); err != nil {
		t.Fatal(err)
	}
	if err := db.Commit(root, false); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 1000 && !db.SnapshotCompleted(); i++ {
		time.Sleep(time.Millisecond)
	}
	// One garbage byte in the flat account entry is enough.
	if err := disk.Put(append(rawdb.SnapshotAccountPrefix, addrHash.Bytes()...), []byte{0x01}); err != nil {
		t.Fatal(err)
	}
	r, err := db.StateReader(root)
	if err != nil {
		t.Fatal(err)
	}
	// Previously panicked on the corrupt blob; now the read must surface an
	// error and return a nil account.
	acct, err := r.Account(addrHash)
	if err == nil {
		t.Fatalf("expected error for corrupt account entry, got account=%v err=nil", acct)
	}
	if acct != nil {
		t.Fatalf("expected nil account for corrupt entry, got account=%v", acct)
	}
	t.Logf("account=%v err=%v", acct, err)
}
