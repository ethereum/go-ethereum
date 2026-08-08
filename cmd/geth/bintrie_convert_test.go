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
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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

	currentRoot, err := convertState(chaindb, srcTriedb2, root, conversionOptions{})
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	t.Logf("Binary trie root: %x", currentRoot)

	if err := verifyConvertedState(chaindb, currentRoot); err != nil {
		t.Fatalf("verification failed: %v", err)
	}

	// The conversion finalized the namespace on disk; a database opened over
	// it afterwards - the way a converted node starts - must pick the root up
	// from there.
	destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		IsPBT:  true,
		PathDB: pathdb.Defaults,
	})
	defer destTriedb.Close()

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
	// Absence is an answer too, and on this path an authoritative one: a
	// flat-store miss is "does not exist", with the trie never consulted.
	absent := common.HexToAddress("0x00000000000000000000000000000000deadbeef")
	if got := statedb.GetBalance(absent); !got.IsZero() {
		t.Errorf("absent account has balance %s through the state reader", got)
	}
	if got := statedb.GetNonce(absent); got != 0 {
		t.Errorf("absent account has nonce %d through the state reader", got)
	}
	if got := statedb.GetState(addr1, common.HexToHash("0x77")); got != (common.Hash{}) {
		t.Errorf("absent slot reads %x through the state reader", got)
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

	newRoot, err := convertState(chaindb, srcTriedb2, root, conversionOptions{})
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if err := verifyConvertedState(chaindb, newRoot); err != nil {
		t.Fatalf("verification failed, which must gate deletion: %v", err)
	}

	if err := deleteMPTData(chaindb, srcTriedb2, root); err != nil {
		t.Fatalf("deletion failed: %v", err)
	}
	srcTriedb2.Close()

	destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		IsPBT:  true,
		PathDB: pathdb.Defaults,
	})

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

// TestConvertRefusesDirtyNamespace pins the crash story. Conversion writes
// flat state and trie nodes first and the attestation marker last, so a run
// that dies leaves keys in the namespace but no marker. Nothing may treat
// that debris as either a completed conversion or a fresh database: the
// converter must refuse to run over it - whatever shape it has, which depends
// on where the run stopped - and only --force, which wipes the namespace,
// clears the way. The equivalent refusal at database-open time is pinned in
// triedb/pathdb (TestAttestFlatState).
func TestConvertRefusesDirtyNamespace(t *testing.T) {
	addr1 := common.HexToAddress("0x4444444444444444444444444444444444444444")

	chaindb := rawdb.NewMemoryDatabase()
	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})
	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			addr1: {Balance: big.NewInt(1000000), Nonce: 1},
		},
	}
	genesis := gspec.MustCommit(chaindb, srcTriedb)
	root := genesis.Root()
	srcTriedb.Close()

	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    &pathdb.Config{ReadOnly: true},
	})
	defer src.Close()

	// Scan-phase debris: a single flat-state record, no trie nodes, no root,
	// no marker. This is the shape a marker probe cannot see.
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
	rawdb.WriteAccountSnapshot(pbtdb, common.Hash{0x01}, []byte{0x01})
	if dirty, err := hasBinaryTrieState(chaindb); err != nil {
		t.Fatalf("namespace probe failed: %v", err)
	} else if !dirty {
		t.Fatal("conversion debris is invisible to the namespace probe")
	}
	if _, err := convertState(chaindb, src, root, conversionOptions{}); err == nil {
		t.Fatal("conversion ran over the debris of a previous run")
	}

	// --force wipes the namespace, after which conversion must run and leave
	// nothing of the debris behind.
	if err := wipeBinaryTrieState(chaindb); err != nil {
		t.Fatalf("wipe failed: %v", err)
	}
	binRoot, err := convertState(chaindb, src, root, conversionOptions{})
	if err != nil {
		t.Fatalf("conversion failed after wipe: %v", err)
	}
	if err := verifyConvertedState(chaindb, binRoot); err != nil {
		t.Fatalf("verification failed after wipe: %v", err)
	}
	if got := rawdb.ReadAccountSnapshot(pbtdb, common.Hash{0x01}); len(got) != 0 {
		t.Fatal("the wipe left the debris record in place")
	}
	// The wipe must not have touched chain data, which shares the namespace
	// prefix: a block body key is a binary tree key to any bare prefix sweep.
	if !rawdb.HasBody(chaindb, genesis.Hash(), 0) {
		t.Fatal("the wipe deleted the genesis block body along with the tree")
	}

	// A completed conversion is itself a dirty namespace: re-running without
	// --force must refuse rather than overwrite.
	if _, err := convertState(chaindb, src, root, conversionOptions{}); err == nil {
		t.Fatal("conversion ran over a completed conversion")
	}
}

// TestConvertVerifiers pins that the two verification passes actually catch
// what they exist to catch: a tree whose disk records cannot rebuild the
// root, and a flat store that no longer re-derives it. Each tamper must turn
// verification red, and undoing it green again.
func TestConvertVerifiers(t *testing.T) {
	chaindb := rawdb.NewMemoryDatabase()
	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})
	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc:   artifactAlloc(),
	}
	root := gspec.MustCommit(chaindb, srcTriedb).Root()
	srcTriedb.Close()

	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    &pathdb.Config{ReadOnly: true},
	})
	defer src.Close()

	binRoot, err := convertState(chaindb, src, root, conversionOptions{})
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))

	// tamper grabs the first record of a family and removes it, returning
	// the undo.
	tamper := func(prefix []byte) (key, value []byte) {
		t.Helper()
		it := pbtdb.NewIterator(prefix, nil)
		defer it.Release()
		if !it.Next() {
			t.Fatalf("no records under family %x to tamper with", prefix)
		}
		key, value = common.CopyBytes(it.Key()), common.CopyBytes(it.Value())
		if err := pbtdb.Delete(key); err != nil {
			t.Fatal(err)
		}
		return key, value
	}
	restore := func(key, value []byte) {
		t.Helper()
		if err := pbtdb.Put(key, value); err != nil {
			t.Fatal(err)
		}
	}

	// A missing tree record must fail the tree walk.
	key, value := tamper(rawdb.TrieNodeAccountPrefix)
	if err := verifyConvertedState(chaindb, binRoot); err == nil {
		t.Fatal("a missing tree record survived tree verification")
	}
	restore(key, value)
	if err := verifyConvertedState(chaindb, binRoot); err != nil {
		t.Fatalf("restored tree fails verification: %v", err)
	}

	// A missing flat account must fail the flat re-derivation - and must NOT
	// fail the tree walk, which is exactly why the flat pass exists.
	key, value = tamper(rawdb.SnapshotAccountPrefix)
	if err := verifyConvertedState(chaindb, binRoot); err != nil {
		t.Fatalf("a flat-only gap failed the tree walk: %v", err)
	}
	if err := verifyFlatState(chaindb, pbtdb, src, binRoot, conversionOptions{}); err == nil {
		t.Fatal("a missing flat account survived flat-state verification")
	}
	restore(key, value)
	if err := verifyFlatState(chaindb, pbtdb, src, binRoot, conversionOptions{}); err != nil {
		t.Fatalf("restored flat state fails verification: %v", err)
	}

	// A corrupt flat slot value must fail the flat re-derivation.
	key, value = tamper(rawdb.SnapshotStoragePrefix)
	restore(key, []byte{0x01}) // rlp("") of a different value
	if err := verifyFlatState(chaindb, pbtdb, src, binRoot, conversionOptions{}); err == nil {
		t.Fatal("a corrupt flat slot survived flat-state verification")
	}
	restore(key, value)
	if err := verifyFlatState(chaindb, pbtdb, src, binRoot, conversionOptions{}); err != nil {
		t.Fatalf("restored flat state fails verification: %v", err)
	}
}

// TestWipeRestoresVirginNamespace pins two properties of --force. The wipe
// must return the namespace to exactly its pre-conversion key set - pinning
// rawdb.PBTKeyFamilies against drift, since any family the list misses would
// survive as invisible debris - and a re-conversion over the wiped namespace
// must reproduce the identical root and byte-identical artifacts.
func TestWipeRestoresVirginNamespace(t *testing.T) {
	chaindb := rawdb.NewMemoryDatabase()
	srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.Defaults,
	})
	gspec := &core.Genesis{
		Config:  params.TestChainConfig,
		BaseFee: big.NewInt(params.InitialBaseFee),
		Alloc:   artifactAlloc(),
	}
	root := gspec.MustCommit(chaindb, srcTriedb).Root()
	srcTriedb.Close()

	// Everything under the namespace prefix before conversion is chain data
	// (the genesis body) that a wipe must not touch.
	prefixKeys := func() map[string]struct{} {
		keys := make(map[string]struct{})
		it := chaindb.NewIterator(rawdb.PBTPrefix, nil)
		defer it.Release()
		for it.Next() {
			keys[string(it.Key())] = struct{}{}
		}
		return keys
	}
	before := prefixKeys()

	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    &pathdb.Config{ReadOnly: true},
	})
	defer src.Close()

	convert := func(dir string) (common.Hash, []byte, []byte) {
		t.Helper()
		binRoot, err := convertState(chaindb, src, root, conversionOptions{
			tmpDir:       dir,
			snapshotPath: filepath.Join(dir, "snapshot.bin"),
			preimagePath: filepath.Join(dir, "preimages.bin"),
		})
		if err != nil {
			t.Fatalf("conversion failed: %v", err)
		}
		snap, err := os.ReadFile(filepath.Join(dir, "snapshot.bin"))
		if err != nil {
			t.Fatal(err)
		}
		pre, err := os.ReadFile(filepath.Join(dir, "preimages.bin"))
		if err != nil {
			t.Fatal(err)
		}
		return binRoot, snap, pre
	}
	root1, snap1, pre1 := convert(t.TempDir())

	if err := wipeBinaryTrieState(chaindb); err != nil {
		t.Fatalf("wipe failed: %v", err)
	}
	after := prefixKeys()
	if len(after) != len(before) {
		t.Fatalf("wipe left %d keys under the namespace prefix, started with %d", len(after), len(before))
	}
	for key := range after {
		if _, ok := before[key]; !ok {
			t.Fatalf("wipe left a converted key behind: %x", key)
		}
	}

	root2, snap2, pre2 := convert(t.TempDir())
	if root1 != root2 {
		t.Fatalf("re-conversion produced root %x, first run %x", root2, root1)
	}
	if !bytes.Equal(snap1, snap2) {
		t.Fatal("re-conversion produced a different snapshot artifact")
	}
	if !bytes.Equal(pre1, pre2) {
		t.Fatal("re-conversion produced a different preimage file")
	}
}

// TestConvertCorruptPreimageRefused pins the integrity of the one input the
// converter cannot derive: the preimage store. It is a bare hash-to-value
// table, so a corrupt entry would otherwise become a wrong-stem tree that
// every downstream check confirms - the verifiers re-derive from the same
// value - and artifacts whose digests no honest producer reproduces. The
// conversion must refuse instead, and refuse before anything survives: no
// artifact files, no attestation.
func TestConvertCorruptPreimageRefused(t *testing.T) {
	newFixture := func(t *testing.T) (ethdb.Database, *triedb.Database, common.Hash) {
		t.Helper()
		chaindb := rawdb.NewMemoryDatabase()
		srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
			Preimages: true,
			PathDB:    pathdb.Defaults,
		})
		gspec := &core.Genesis{
			Config:  params.TestChainConfig,
			BaseFee: big.NewInt(params.InitialBaseFee),
			Alloc:   artifactAlloc(),
		}
		root := gspec.MustCommit(chaindb, srcTriedb).Root()
		srcTriedb.Close()
		src := triedb.NewDatabase(chaindb, &triedb.Config{
			Preimages: true,
			PathDB:    &pathdb.Config{ReadOnly: true},
		})
		t.Cleanup(func() { src.Close() })
		return chaindb, src, root
	}
	convert := func(t *testing.T, chaindb ethdb.Database, src *triedb.Database, root common.Hash) (error, string, string) {
		t.Helper()
		dir := t.TempDir()
		snapPath, prePath := filepath.Join(dir, "snapshot.bin"), filepath.Join(dir, "preimages.bin")
		_, err := convertState(chaindb, src, root, conversionOptions{
			tmpDir:       dir,
			snapshotPath: snapPath,
			preimagePath: prePath,
		})
		return err, snapPath, prePath
	}
	t.Run("wrong account preimage", func(t *testing.T) {
		chaindb, src, root := newFixture(t)
		// A well-formed 20-byte value that is not the preimage of its key.
		addr := common.HexToAddress("0x1000000000000000000000000000000000000001")
		rawdb.WritePreimages(chaindb, map[common.Hash][]byte{
			crypto.Keccak256Hash(addr.Bytes()): common.HexToAddress("0x9999999999999999999999999999999999999999").Bytes(),
		})
		err, snapPath, prePath := convert(t, chaindb, src, root)
		if err == nil || !strings.Contains(err.Error(), "corrupt preimage") {
			t.Fatalf("a corrupt account preimage converted; err = %v", err)
		}
		for _, path := range []string{snapPath, prePath} {
			if _, statErr := os.Stat(path); statErr == nil {
				t.Fatalf("a refused conversion left the artifact %s behind", path)
			}
		}
		if rawdb.HasPBTState(chaindb) {
			t.Fatal("a refused conversion attested its namespace")
		}
	})

	t.Run("wrong-length slot preimage", func(t *testing.T) {
		chaindb, src, root := newFixture(t)
		// 33 bytes: over the word size, which used to reach the key deriver
		// and panic there rather than error.
		slot := common.BigToHash(big.NewInt(1))
		rawdb.WritePreimages(chaindb, map[common.Hash][]byte{
			crypto.Keccak256Hash(slot[:]): make([]byte, 33),
		})
		err, snapPath, _ := convert(t, chaindb, src, root)
		if err == nil || !strings.Contains(err.Error(), "corrupt preimage") {
			t.Fatalf("a wrong-length slot preimage converted; err = %v", err)
		}
		if _, statErr := os.Stat(snapPath); statErr == nil {
			t.Fatal("a refused conversion left the snapshot artifact behind")
		}
		if rawdb.HasPBTState(chaindb) {
			t.Fatal("a refused conversion attested its namespace")
		}
	})
}
