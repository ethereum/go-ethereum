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
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// The lifecycle tests: the surroundings a real conversion lives in, where
// the parity oracles pin only what it computes.

// TestConvertMatchesChain converts a state built by real block execution -
// deploys, storage writes and clears, a SetCodeTx delegation - resolved out
// of the journaled diff layers a clean shutdown leaves, against an embedding
// of the dumped post-state.
func TestConvertMatchesChain(t *testing.T) {
	var (
		config = *params.MergedTestChainConfig
		signer = types.LatestSigner(&config)
		engine = beacon.New(ethash.NewFaker())

		key1, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
		key2, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")
		addr1   = crypto.PubkeyToAddress(key1.PublicKey)
		addr2   = crypto.PubkeyToAddress(key2.PublicKey)
		funds   = new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))

		// The constructor writes slots on both sides of the header boundary;
		// the runtime clears slot 1 on any call.
		runtime  = program.New().Sstore(1, 0).Bytes()
		initcode = program.New().
				Sstore(0, 0x11).Sstore(1, 0x22).Sstore(63, 0x33).Sstore(64, 0x44).Sstore(4096, 0x55).
				ReturnViaCodeCopy(runtime).Bytes()

		contractA = crypto.CreateAddress(addr1, 0)
		freshEOA  = common.HexToAddress("0x00000000000000000000000000000000000f0e0a")
		gasPrice  = big.NewInt(10 * params.GWei)
	)
	gspec := &core.Genesis{
		Config: &config,
		Alloc: types.GenesisAlloc{
			addr1: {Balance: funds},
			addr2: {Balance: funds},
		},
	}
	// key2 delegates to contract A.
	auth, err := types.SignSetCode(key2, types.SetCodeAuthorization{
		ChainID: *uint256.MustFromBig(config.ChainID),
		Address: contractA,
		Nonce:   0,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, 2, func(i int, b *core.BlockGen) {
		switch i {
		case 0:
			// Two deploys sharing runtime bytecode, and a fresh EOA.
			b.AddTx(types.MustSignNewTx(key1, signer, &types.LegacyTx{
				Nonce: 0, Gas: 500000, GasPrice: gasPrice, Data: initcode,
			}))
			b.AddTx(types.MustSignNewTx(key1, signer, &types.LegacyTx{
				Nonce: 1, Gas: 500000, GasPrice: gasPrice, Data: initcode,
			}))
			b.AddTx(types.MustSignNewTx(key1, signer, &types.LegacyTx{
				Nonce: 2, Gas: 21000, GasPrice: gasPrice, To: &freshEOA, Value: big.NewInt(1),
			}))
		case 1:
			// Install the delegation, then clear A's slot 1.
			b.AddTx(types.MustSignNewTx(key1, signer, &types.SetCodeTx{
				ChainID:   uint256.MustFromBig(config.ChainID),
				Nonce:     3,
				To:        addr1,
				Gas:       100000,
				GasFeeCap: uint256.MustFromBig(gasPrice),
				GasTipCap: uint256.NewInt(1),
				AuthList:  []types.SetCodeAuthorization{auth},
			}))
			b.AddTx(types.MustSignNewTx(key1, signer, &types.LegacyTx{
				Nonce: 4, Gas: 100000, GasPrice: gasPrice, To: &contractA,
			}))
		}
	})
	chaindb := rawdb.NewMemoryDatabase()
	cfg := core.DefaultConfig()
	cfg.Preimages = true
	cfg.StateScheme = rawdb.PathScheme
	chain, err := core.NewBlockChain(chaindb, gspec, engine, cfg)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	if n, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("block %d not inserted: %v", n, err)
	}
	headRoot := chain.CurrentBlock().Root

	// The live post-state dump is the oracle.
	statedb, err := chain.State()
	if err != nil {
		t.Fatal(err)
	}
	dump, err := statedb.RawDump(&state.DumpConfig{})
	if err != nil {
		t.Fatal(err)
	}
	alloc := make(types.GenesisAlloc, len(dump.Accounts))
	for key, acc := range dump.Accounts {
		addr := common.HexToAddress(key)
		if acc.Address != nil {
			addr = *acc.Address
		}
		balance, ok := new(big.Int).SetString(acc.Balance, 10)
		if !ok {
			t.Fatalf("bad balance %q for %x", acc.Balance, addr)
		}
		entry := types.Account{Nonce: acc.Nonce, Balance: balance, Code: acc.Code}
		if len(acc.Storage) > 0 {
			entry.Storage = make(map[common.Hash]common.Hash, len(acc.Storage))
			for slot, val := range acc.Storage {
				entry.Storage[slot] = common.HexToHash(val)
			}
		}
		alloc[addr] = entry
	}
	// A clean shutdown journals the diff layers holding the head state.
	chain.Stop()

	src := triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.ReadOnly,
	})
	defer src.Close()

	binRoot, err := convertState(chaindb, src, headRoot, conversionOptions{})
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	if want := embedAlloc(t, alloc); binRoot != want {
		t.Fatalf("converted root %x, embedding the executed state produces %x", binRoot, want)
	}
	assertConvertedTreeClean(t, chaindb, binRoot)

	// Read the interesting shapes back as a node would.
	destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
	defer destTriedb.Close()
	converted, err := state.New(binRoot, state.NewPBTDatabase(destTriedb, state.NewCodeDB(chaindb)))
	if err != nil {
		t.Fatal(err)
	}
	if got := converted.GetState(contractA, common.BigToHash(big.NewInt(1))); got != (common.Hash{}) {
		t.Errorf("the cleared slot reads %x, want absence", got)
	}
	if got := converted.GetState(contractA, common.BigToHash(big.NewInt(4096))); got != common.BigToHash(big.NewInt(0x55)) {
		t.Errorf("overflow slot reads %x, want 0x55", got)
	}
	if got := converted.GetCode(addr2); !bytes.Equal(got, types.AddressToDelegation(contractA)) {
		t.Errorf("delegated account's code reads %x, want its designator", got)
	}
	if got := converted.GetNonce(addr2); got != 1 {
		t.Errorf("delegated account's nonce reads %d, want 1", got)
	}
	if got := converted.GetBalance(freshEOA); got.CmpUint64(1) != 0 {
		t.Errorf("fresh EOA balance reads %s, want 1", got)
	}
}

// TestBintrieConvertDiskBacked converts on pebble with a real ancient store:
// verification must not touch the history freezers, and the namespace must
// survive two reopen cycles (the first creates the freezers).
func TestBintrieConvertDiskBacked(t *testing.T) {
	var (
		datadir = t.TempDir()
		kvdir   = filepath.Join(datadir, "kv")
		ancient = filepath.Join(datadir, "ancient")
	)
	openDB := func() ethdb.Database {
		t.Helper()
		pdb, err := pebble.New(kvdir, 0, 0, "", false)
		if err != nil {
			t.Fatalf("failed to open pebble: %v", err)
		}
		db, err := rawdb.Open(pdb, rawdb.OpenOptions{Ancient: ancient})
		if err != nil {
			t.Fatalf("failed to open freezer database: %v", err)
		}
		return db
	}
	chaindb := openDB()
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
		PathDB:    pathdb.ReadOnly,
	})
	binRoot, err := convertState(chaindb, src, root, conversionOptions{tmpDir: datadir})
	if err != nil {
		t.Fatalf("conversion failed on a freezer-backed database: %v", err)
	}
	src.Close()
	chaindb.Close()

	owner := common.HexToAddress("0x1000000000000000000000000000000000000001")
	for cycle := 0; cycle < 2; cycle++ {
		chaindb = openDB()
		destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
		statedb, err := state.New(binRoot, state.NewPBTDatabase(destTriedb, state.NewCodeDB(chaindb)))
		if err != nil {
			t.Fatalf("reopen %d: cannot open the converted state: %v", cycle, err)
		}
		if got := statedb.GetNonce(owner); got != 7 {
			t.Fatalf("reopen %d: nonce reads %d, want 7", cycle, got)
		}
		destTriedb.Close()
		chaindb.Close()
	}

	// A wipe on a freezer-backed database resets the PBT freezers and removes
	// the journal file; re-conversion must reproduce the root.
	chaindb = openDB()
	if err := wipeBinaryTrieState(chaindb, filepath.Join(datadir, "triedb")); err != nil {
		t.Fatalf("wipe failed on a freezer-backed database: %v", err)
	}
	src = triedb.NewDatabase(chaindb, &triedb.Config{
		Preimages: true,
		PathDB:    pathdb.ReadOnly,
	})
	root2, err := convertState(chaindb, src, root, conversionOptions{tmpDir: datadir})
	if err != nil {
		t.Fatalf("re-conversion after wipe failed: %v", err)
	}
	if root2 != binRoot {
		t.Fatalf("re-conversion produced root %x, first run %x", root2, binRoot)
	}
	src.Close()
	chaindb.Close()
}

// TestConvertedBaseAcceptsCommits pins the handoff: the first live commit on
// the state-id-0 base must land, survive a reopen, and read back.
func TestConvertedBaseAcceptsCommits(t *testing.T) {
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
		PathDB:    pathdb.ReadOnly,
	})
	binRoot, err := convertState(chaindb, src, root, conversionOptions{})
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	src.Close()

	// One fresh account through the tree's writers.
	destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
	tr, err := bintrie.NewBinaryTrie(binRoot, destTriedb)
	if err != nil {
		t.Fatalf("cannot open the converted tree: %v", err)
	}
	var (
		newAddr = common.HexToAddress("0x7000000000000000000000000000000000000007")
		acc     = &types.StateAccount{
			Nonce:    1,
			Balance:  uint256.NewInt(5),
			Root:     types.EmptyRootHash,
			CodeHash: types.EmptyCodeHash.Bytes(),
		}
	)
	if err := tr.UpdateAccount(newAddr, acc, 0, nil); err != nil {
		t.Fatal(err)
	}
	newRoot, nodes := tr.Commit(false)

	states := triedb.NewStateSet()
	states.Accounts[crypto.Keccak256Hash(newAddr.Bytes())] = types.SlimAccountRLP(*acc)
	if err := destTriedb.Update(newRoot, binRoot, 1, trienode.NewWithNodeSet(nodes), states); err != nil {
		t.Fatalf("the first live update was refused: %v", err)
	}
	if err := destTriedb.Commit(newRoot, false); err != nil {
		t.Fatalf("the first live commit was refused: %v", err)
	}
	destTriedb.Close()

	// Both the delta and the converted base must read.
	destTriedb = triedb.NewDatabase(chaindb, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
	defer destTriedb.Close()
	statedb, err := state.New(newRoot, state.NewPBTDatabase(destTriedb, state.NewCodeDB(chaindb)))
	if err != nil {
		t.Fatalf("cannot open the committed state: %v", err)
	}
	if got := statedb.GetNonce(newAddr); got != 1 {
		t.Fatalf("the committed account reads nonce %d, want 1", got)
	}
	if got := statedb.GetNonce(common.HexToAddress("0x1000000000000000000000000000000000000001")); got != 7 {
		t.Fatalf("the converted account reads nonce %d after the commit, want 7", got)
	}
}

// TestDeleteSourceLifecycle: after deletion the converted bytes stand alone,
// the merkle trie is gone, and code and preimages survive.
func TestDeleteSourceLifecycle(t *testing.T) {
	t.Run("path scheme", func(t *testing.T) {
		alloc := mixedAlloc(424242)
		chaindb := rawdb.NewMemoryDatabase()
		srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
			Preimages: true,
			PathDB:    pathdb.Defaults,
		})
		gspec := &core.Genesis{
			Config:  params.TestChainConfig,
			BaseFee: big.NewInt(params.InitialBaseFee),
			Alloc:   alloc,
		}
		root := gspec.MustCommit(chaindb, srcTriedb).Root()
		srcTriedb.Close()

		src := triedb.NewDatabase(chaindb, &triedb.Config{
			Preimages: true,
			PathDB:    pathdb.ReadOnly,
		})
		defer src.Close()

		binRoot, err := convertState(chaindb, src, root, conversionOptions{})
		if err != nil {
			t.Fatalf("conversion failed: %v", err)
		}
		if err := deleteMPTData(chaindb, src, root); err != nil {
			t.Fatalf("deletion failed: %v", err)
		}
		// Converted bytes alone must verify.
		pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
		if err := verifyConvertedState(chaindb, binRoot); err != nil {
			t.Fatalf("tree verification fails after source deletion: %v", err)
		}
		if err := verifyFlatState(chaindb, pbtdb, src, binRoot, conversionOptions{}); err != nil {
			t.Fatalf("flat verification fails after source deletion: %v", err)
		}
		// A contract reads through the node's stack.
		var contract common.Address
		var acct types.Account
		for addr, a := range alloc {
			if len(a.Code) > 0 && len(a.Storage) > 0 {
				if _, delegated := types.ParseDelegation(a.Code); !delegated {
					contract, acct = addr, a
					break
				}
			}
		}
		if contract == (common.Address{}) {
			t.Fatal("the fixture holds no contract with code and storage")
		}
		destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
		defer destTriedb.Close()
		statedb, err := state.New(binRoot, state.NewPBTDatabase(destTriedb, state.NewCodeDB(chaindb)))
		if err != nil {
			t.Fatal(err)
		}
		if got := statedb.GetCode(contract); !bytes.Equal(got, acct.Code) {
			t.Fatalf("contract code reads %d bytes, want %d", len(got), len(acct.Code))
		}
		for slot, val := range acct.Storage {
			if got := statedb.GetState(contract, slot); got != val {
				t.Fatalf("slot %x reads %x, want %x", slot, got, val)
			}
			break
		}
		// Merkle gone...
		fresh := triedb.NewDatabase(chaindb, &triedb.Config{PathDB: &pathdb.Config{ReadOnly: true}})
		defer fresh.Close()
		if mpt, err := trie.NewStateTrie(trie.StateTrieID(root), fresh); err == nil {
			it, err := mpt.NodeIterator(nil)
			if err == nil {
				found := false
				for it.Next(true) {
					found = true
					break
				}
				if found || it.Error() == nil {
					t.Fatal("the merkle trie still resolves after deletion")
				}
			}
		}
		// ...code and preimages survive.
		if code := rawdb.ReadCode(chaindb, crypto.Keccak256Hash(acct.Code)); len(code) == 0 {
			t.Fatal("deletion took the contract code with it")
		}
		if pre := rawdb.ReadPreimage(chaindb, crypto.Keccak256Hash(contract.Bytes())); len(pre) == 0 {
			t.Fatal("deletion took the preimages with it")
		}
	})

	t.Run("hash scheme", func(t *testing.T) {
		chaindb := rawdb.NewMemoryDatabase()
		srcTriedb := triedb.NewDatabase(chaindb, &triedb.Config{Preimages: true})
		gspec := &core.Genesis{
			Config:  params.TestChainConfig,
			BaseFee: big.NewInt(params.InitialBaseFee),
			Alloc:   artifactAlloc(),
		}
		root := gspec.MustCommit(chaindb, srcTriedb).Root()

		binRoot, err := convertState(chaindb, srcTriedb, root, conversionOptions{})
		if err != nil {
			t.Fatalf("conversion from a hash-scheme source failed: %v", err)
		}
		if err := deleteMPTData(chaindb, srcTriedb, root); err != nil {
			t.Fatalf("hash-scheme deletion failed: %v", err)
		}
		if node := rawdb.ReadLegacyTrieNode(chaindb, root); len(node) != 0 {
			t.Fatal("the merkle root node survived hash-scheme deletion")
		}
		if err := verifyConvertedState(chaindb, binRoot); err != nil {
			t.Fatalf("tree verification fails after hash-scheme deletion: %v", err)
		}
		srcTriedb.Close()
	})
}
