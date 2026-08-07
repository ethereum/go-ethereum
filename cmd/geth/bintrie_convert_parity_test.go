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
	"encoding/json"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// The conversion parity oracles.
//
// A converter bug is invisible to read-back tests: the tree hashes whatever
// was written and the readers dutifully return it. What pins the converter is
// root equality against writers with independent provenance. Two are used:
//
//   - the state_vectors in trie/bintrie/testdata are allocation-to-root pairs
//     computed by the execution-specs reference, covering the shapes where an
//     embedding makes choices (zero-collapse, delegation, shared code);
//   - the tree's own typed writers, which every executed block goes through,
//     pin the converter against replay on randomized allocations.

// stateVector mirrors the state_vectors entries of eip8297_vectors.json.
type stateVector struct {
	Name     string `json:"name"`
	Accounts []struct {
		Address string            `json:"address"`
		Nonce   uint64            `json:"nonce"`
		Balance *big.Int          `json:"balance"`
		Code    string            `json:"code"`
		Storage map[string]uint64 `json:"storage"`
	} `json:"accounts"`
	Root string `json:"root"`
}

func loadStateVectors(t *testing.T) []stateVector {
	t.Helper()
	blob, err := os.ReadFile("../../trie/bintrie/testdata/eip8297_vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var file struct {
		StateVectors []stateVector `json:"state_vectors"`
	}
	if err := json.Unmarshal(blob, &file); err != nil {
		t.Fatal(err)
	}
	if len(file.StateVectors) == 0 {
		t.Fatal("no state vectors in the file; the exporter did not write them")
	}
	return file.StateVectors
}

// allocOf converts a vector's accounts into a genesis allocation.
func allocOf(t *testing.T, sv stateVector) types.GenesisAlloc {
	t.Helper()
	alloc := make(types.GenesisAlloc, len(sv.Accounts))
	for _, a := range sv.Accounts {
		acct := types.Account{Nonce: a.Nonce, Balance: a.Balance}
		if acct.Balance == nil {
			acct.Balance = new(big.Int)
		}
		if a.Code != "" {
			acct.Code = common.FromHex(a.Code)
		}
		if len(a.Storage) > 0 {
			acct.Storage = make(map[common.Hash]common.Hash, len(a.Storage))
			for slot, val := range a.Storage {
				key, ok := new(big.Int).SetString(slot, 10)
				if !ok {
					t.Fatalf("bad storage key %q", slot)
				}
				var k, v common.Hash
				key.FillBytes(k[:])
				new(big.Int).SetUint64(val).FillBytes(v[:])
				acct.Storage[k] = v
			}
		}
		alloc[common.HexToAddress(a.Address)] = acct
	}
	return alloc
}

// convertAlloc realizes alloc as a merkle-patricia state and runs the
// converter over it, returning the binary root and the database it wrote.
func convertAlloc(t *testing.T, alloc types.GenesisAlloc) (common.Hash, ethdb.Database) {
	t.Helper()
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
		PathDB:    &pathdb.Config{ReadOnly: true},
	})
	defer src.Close()

	binRoot, err := convertState(chaindb, src, root, conversionOptions{})
	if err != nil {
		t.Fatalf("conversion failed: %v", err)
	}
	return binRoot, chaindb
}

// embedAlloc writes alloc through the tree's typed writers - the same calls
// every executed block makes - and returns the root. This is the
// replay-equivalent oracle: converted state and replayed state must commit to
// the same root or a converted node forks from an executing one.
func embedAlloc(t *testing.T, alloc types.GenesisAlloc) common.Hash {
	t.Helper()
	tr, err := bintrie.NewBinaryTrie(types.EmptyBinaryHash, triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.PBTDefaults))
	if err != nil {
		t.Fatal(err)
	}
	for addr, a := range alloc {
		code := a.Code
		acc := &types.StateAccount{
			Nonce:    a.Nonce,
			Balance:  balanceOf(t, a.Balance),
			Root:     types.EmptyRootHash,
			CodeHash: crypto.Keccak256(code),
		}
		var delegation []byte
		if _, ok := types.ParseDelegation(code); ok {
			delegation = code
		}
		if err := tr.UpdateAccount(addr, acc, len(code), delegation); err != nil {
			t.Fatalf("account %x: %v", addr, err)
		}
		if err := tr.UpdateContractCode(addr, common.BytesToHash(acc.CodeHash), code); err != nil {
			t.Fatalf("code for %x: %v", addr, err)
		}
		for k, v := range a.Storage {
			if err := tr.UpdateStorage(addr, k[:], common.TrimLeftZeroes(v[:])); err != nil {
				t.Fatalf("storage %x of %x: %v", k, addr, err)
			}
		}
	}
	return tr.Hash()
}

// balanceOf converts a genesis balance for the typed writers.
func balanceOf(t *testing.T, b *big.Int) *uint256.Int {
	t.Helper()
	if b == nil {
		return new(uint256.Int)
	}
	v, overflow := uint256.FromBig(b)
	if overflow {
		t.Fatalf("balance %v overflows", b)
	}
	return v
}

// assertConvertedTreeClean walks every leaf of the converted tree and checks
// the tree-wide invariant EIP-8297 states: no key holds 32 zero bytes.
func assertConvertedTreeClean(t *testing.T, chaindb ethdb.Database, root common.Hash) {
	t.Helper()
	db := triedb.NewDatabase(chaindb, &triedb.Config{IsPBT: true, PathDB: pathdb.Defaults})
	defer db.Close()
	tr, err := bintrie.NewBinaryTrie(root, db)
	if err != nil {
		t.Fatal(err)
	}
	it, err := tr.NodeIterator(nil)
	if err != nil {
		t.Fatal(err)
	}
	var zero [32]byte
	for it.Next(true) {
		if it.Leaf() && bytes.Equal(it.LeafBlob(), zero[:]) {
			t.Fatalf("leaf %x holds 32 zero bytes, which no key in the state's tree may", it.LeafKey())
		}
	}
	if err := it.Error(); err != nil {
		t.Fatal(err)
	}
}

// TestConvertMatchesReference pins the converter against the execution-specs
// reference: each state vector's allocation, realized as a merkle state and
// converted, must produce the reference-computed binary root.
//
// The vectors are the shapes where the embedding decides anything: all-zero
// code chunks and all-zero basic data (written by no one), EIP-7702
// delegations (a header leaf, not code), shared bytecode (one set of leaves,
// however many holders), and storage on both sides of the slot-64 boundary.
func TestConvertMatchesReference(t *testing.T) {
	for _, sv := range loadStateVectors(t) {
		t.Run(sv.Name, func(t *testing.T) {
			binRoot, chaindb := convertAlloc(t, allocOf(t, sv))
			if want := common.HexToHash(sv.Root); binRoot != want {
				t.Fatalf("converted root %x, reference says %s", binRoot, sv.Root)
			}
			assertConvertedTreeClean(t, chaindb, binRoot)
		})
	}
}

// TestConvertMatchesEmbedding is the randomized differential: a large mixed
// allocation converted from a merkle source must hash identically to the same
// allocation written through the typed writers, which is what replaying the
// equivalent transactions would produce.
//
// Randomization is what reaches the converter-specific hazards the hand-made
// vectors cannot: shared code discovered in whatever order the account-hash
// iteration yields it, and stems assembled from leaves that arrive scattered.
func TestConvertMatchesEmbedding(t *testing.T) {
	rng := rand.New(rand.NewSource(8347))
	alloc := make(types.GenesisAlloc)

	// A pool of code blobs so sharing actually occurs, including a designator
	// and a zero-tailed blob whose trailing chunks must not be written.
	codes := [][]byte{
		nil,
		bytes.Repeat([]byte{0x5b}, 40),
		bytes.Repeat([]byte{0x5b}, 40), // deliberately identical to the previous
		append([]byte{0x60, 0x01, 0x00}, make([]byte, 80)...),
		bytes.Repeat([]byte{0x60, 0x01}, 700), // spans several code stems
		types.AddressToDelegation(common.Address{0xde, 0x1e}),
	}
	for i := 0; i < 60; i++ {
		var addr common.Address
		rng.Read(addr[:])
		acct := types.Account{
			Nonce:   uint64(rng.Intn(3)),
			Balance: big.NewInt(int64(rng.Intn(1 << 30))),
			Code:    codes[rng.Intn(len(codes))],
		}
		if rng.Intn(2) == 0 {
			acct.Storage = make(map[common.Hash]common.Hash)
			for j := 0; j < rng.Intn(6); j++ {
				var k common.Hash
				// Half the slots below the header boundary, half above.
				if rng.Intn(2) == 0 {
					k = common.BigToHash(big.NewInt(int64(rng.Intn(64))))
				} else {
					k = common.BigToHash(big.NewInt(int64(64 + rng.Intn(4096))))
				}
				acct.Storage[k] = common.BigToHash(big.NewInt(int64(rng.Intn(1<<30) + 1)))
			}
		}
		// The genesis writer refuses an account that is entirely absent, so
		// give the codeless, balance-less ones at least a nonce or a slot.
		if acct.Nonce == 0 && acct.Balance.Sign() == 0 && acct.Code == nil && len(acct.Storage) == 0 {
			acct.Nonce = 1
		}
		alloc[addr] = acct
	}

	binRoot, chaindb := convertAlloc(t, alloc)
	if want := embedAlloc(t, alloc); binRoot != want {
		t.Fatalf("converted root %x, the typed writers produce %x", binRoot, want)
	}
	assertConvertedTreeClean(t, chaindb, binRoot)
}
