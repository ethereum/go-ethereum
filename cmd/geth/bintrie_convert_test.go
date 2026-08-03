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
	"math"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

func TestBintrieConvert(t *testing.T) {
	var (
		addr1    = common.HexToAddress("0x1111111111111111111111111111111111111111")
		addr2    = common.HexToAddress("0x2222222222222222222222222222222222222222")
		slotKey1 = common.HexToHash("0x01")
		slotKey2 = common.HexToHash("0x02")
		slotVal1 = common.HexToHash("0xdeadbeef")
		slotVal2 = common.HexToHash("0xcafebabe")
		code     = []byte{0x60, 0x42, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}
	)

	chaindb := rawdb.NewMemoryDatabase()

	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})

	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			addr1: {
				Balance: big.NewInt(1000000),
				Nonce:   5,
			},
			addr2: {
				Balance: big.NewInt(2000000),
				Nonce:   10,
				Code:    code,
				Storage: map[common.Hash]common.Hash{
					slotKey1: slotVal1,
					slotKey2: slotVal2,
				},
			},
		},
	}

	genesisBlock := gspec.MustCommit(chaindb, srcTriedb)
	root := genesisBlock.Root()
	t.Logf("Genesis root: %x", root)
	srcTriedb.Close()

	srcTriedb2 := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    &pathdb.Config{ReadOnly: true},
	})
	defer srcTriedb2.Close()

	destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		IsPBT:  true,
		PathDB: pathdb.Defaults,
	})
	defer destTriedb.Close()

	bt, err := bintrie.NewBinaryTrie(types.EmptyBinaryHash, destTriedb)
	if err != nil {
		t.Fatalf("failed to create binary trie: %v", err)
	}

	currentRoot, err := runConversionLoop(chaindb, srcTriedb2, destTriedb, bt, root, math.MaxUint64)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	t.Logf("Binary trie root: %x", currentRoot)

	bt2, err := bintrie.NewBinaryTrie(currentRoot, destTriedb)
	if err != nil {
		t.Fatalf("failed to reload binary trie: %v", err)
	}

	acc1, err := bt2.GetAccount(addr1)
	if err != nil {
		t.Fatalf("failed to get account1: %v", err)
	}
	if acc1 == nil {
		t.Fatal("account1 not found in binary trie")
	}
	if acc1.Nonce != 5 {
		t.Errorf("account1 nonce: got %d, want 5", acc1.Nonce)
	}
	wantBal1 := uint256.NewInt(1000000)
	if acc1.Balance.Cmp(wantBal1) != 0 {
		t.Errorf("account1 balance: got %s, want %s", acc1.Balance, wantBal1)
	}

	acc2, err := bt2.GetAccount(addr2)
	if err != nil {
		t.Fatalf("failed to get account2: %v", err)
	}
	if acc2 == nil {
		t.Fatal("account2 not found in binary trie")
	}
	if acc2.Nonce != 10 {
		t.Errorf("account2 nonce: got %d, want 10", acc2.Nonce)
	}
	wantBal2 := uint256.NewInt(2000000)
	if acc2.Balance.Cmp(wantBal2) != 0 {
		t.Errorf("account2 balance: got %s, want %s", acc2.Balance, wantBal2)
	}

	treeKey1 := bintrie.StorageSlotKey(addr2, slotKey1[:])
	val1, err := bt2.GetStemValue(treeKey1)
	if err != nil {
		t.Fatalf("failed to get storage slot1: %v", err)
	}
	if len(val1) == 0 {
		t.Fatal("storage slot1 not found")
	}
	got1 := common.BytesToHash(val1)
	if got1 != slotVal1 {
		t.Errorf("storage slot1: got %x, want %x", got1, slotVal1)
	}

	treeKey2 := bintrie.StorageSlotKey(addr2, slotKey2[:])
	val2, err := bt2.GetStemValue(treeKey2)
	if err != nil {
		t.Fatalf("failed to get storage slot2: %v", err)
	}
	if len(val2) == 0 {
		t.Fatal("storage slot2 not found")
	}
	got2 := common.BytesToHash(val2)
	if got2 != slotVal2 {
		t.Errorf("storage slot2: got %x, want %x", got2, slotVal2)
	}

	// Everything above reads the trie directly, which is not how a node reads
	// state. Read the same accounts back the way block processing does.
	assertConvertedStateReadable(t, chaindb, destTriedb, currentRoot, addr1, addr2, slotKey1, slotVal1)
}

// TestPBTDiskIsDetectable pins the fact MakeTrieDatabase's guard rests on: a
// binary tree database can be recognised from disk alone, and a merkle-patricia
// one is never mistaken for it.
//
// The guard refuses to open tree state as merkle, which is what stops the
// snapshot and db commands - all of which pass isPBT=false for want of knowing
// about the tree - from reading a database whose records they cannot decode and
// reporting the absence as emptiness. Both directions matter: a false positive
// would lock those commands out of ordinary merkle databases.
func TestPBTDiskIsDetectable(t *testing.T) {
	merkle := rawdb.NewMemoryDatabase()
	mtdb := triedb.NewDatabase(merkle, &triedb.Config{PathDB: pathdb.Defaults})
	mtdb.Close()
	if rawdb.HasPBTState(merkle) {
		t.Fatal("a merkle-patricia database is marked as holding a binary tree")
	}

	binary := rawdb.NewMemoryDatabase()
	btdb := triedb.NewDatabase(binary, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
	btdb.Close()
	if !rawdb.HasPBTState(binary) {
		t.Fatal("a binary tree database is not detectable from disk; the guard can never fire")
	}
}

// assertConvertedStateReadable reads a converted database through the state
// reader a node actually uses, rather than through the binary trie directly.
//
// The distinction is the whole point. PBTDatabase.StateReader puts the flat
// reader ahead of the trie reader, and multiStateReader returns the first
// answer that comes back without an error - including "this account does not
// exist". So a converted database whose flat state is empty reads as empty
// here while every direct trie assertion above still passes.
func assertConvertedStateReadable(t *testing.T, chaindb ethdb.Database, destTriedb *triedb.Database, root common.Hash, addr1, addr2 common.Address, slotKey, slotVal common.Hash) {
	t.Helper()

	statedb, err := state.New(root, state.NewPBTDatabase(destTriedb, state.NewCodeDB(chaindb)))
	if err != nil {
		t.Fatalf("failed to open the converted state: %v", err)
	}
	if got := statedb.GetNonce(addr1); got != 5 {
		t.Errorf("account1 nonce through the state reader: got %d, want 5", got)
	}
	if got := statedb.GetBalance(addr1).ToBig(); got.Cmp(big.NewInt(1000000)) != 0 {
		t.Errorf("account1 balance through the state reader: got %s, want 1000000", got)
	}
	if got := statedb.GetNonce(addr2); got != 10 {
		t.Errorf("account2 nonce through the state reader: got %d, want 10", got)
	}
	if got := statedb.GetState(addr2, slotKey); got != slotVal {
		t.Errorf("account2 slot through the state reader: got %x, want %x", got, slotVal)
	}
}

func TestBintrieConvertDeleteSource(t *testing.T) {
	addr1 := common.HexToAddress("0x3333333333333333333333333333333333333333")

	chaindb := rawdb.NewMemoryDatabase()

	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})

	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			addr1: {
				Balance: big.NewInt(1000000),
			},
		},
	}

	genesisBlock := gspec.MustCommit(chaindb, srcTriedb)
	root := genesisBlock.Root()
	srcTriedb.Close()

	srcTriedb2 := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    &pathdb.Config{ReadOnly: true},
	})

	destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		IsPBT:  true,
		PathDB: pathdb.Defaults,
	})

	bt, err := bintrie.NewBinaryTrie(types.EmptyBinaryHash, destTriedb)
	if err != nil {
		t.Fatalf("failed to create binary trie: %v", err)
	}

	newRoot, err := runConversionLoop(chaindb, srcTriedb2, destTriedb, bt, root, math.MaxUint64)
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}

	if err := deleteMPTData(chaindb, srcTriedb2, root); err != nil {
		t.Fatalf("deletion failed: %v", err)
	}
	srcTriedb2.Close()

	bt2, err := bintrie.NewBinaryTrie(newRoot, destTriedb)
	if err != nil {
		t.Fatalf("failed to reload binary trie after deletion: %v", err)
	}

	acc, err := bt2.GetAccount(addr1)
	if err != nil {
		t.Fatalf("failed to get account after deletion: %v", err)
	}
	if acc == nil {
		t.Fatal("account not found after MPT deletion")
	}
	wantBal := uint256.NewInt(1000000)
	if acc.Balance.Cmp(wantBal) != 0 {
		t.Errorf("balance after deletion: got %s, want %s", acc.Balance, wantBal)
	}
	destTriedb.Close()
}
