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

package core

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// pbtChainGenesis is a binary-tree genesis with a funded account whose key is
// known, so generated blocks can carry transactions and therefore change state.
// A chain of empty blocks would prove nothing here: every root would equal the
// genesis root and no read would touch anything new.
func pbtChainGenesis(t *testing.T) (*Genesis, *ecdsa.PrivateKey, common.Address, common.Address) {
	t.Helper()

	key, err := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	if err != nil {
		t.Fatal(err)
	}
	var (
		sender    = crypto.PubkeyToAddress(key.PublicKey)
		recipient = common.Address{0x0f, 0xf1}
		config    = *testPBTChainConfig
	)
	config.Ethash = nil

	return &Genesis{
		Config:     &config,
		Difficulty: big.NewInt(0),
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			sender: {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))},
		},
	}, key, sender, recipient
}

// TestPBTGeneratedChainImportsWithFlatState builds a real multi-block binary-tree
// chain, imports it, and reads the resulting state back.
//
// This is coverage `GenerateChain` used to make impossible. It force-committed
// every block's trie — a hash-scheme requirement, since hashdb keeps nodes in a
// dirty cache until they are flushed by hash — which the path database rejects,
// having already taken the nodes and refusing to commit a root it has persisted.
// So no test in this package could produce a binary-tree chain longer than
// genesis, and the reorg and flat-state claims had to be pinned one layer down
// against synthetic roots.
func TestPBTGeneratedChainImportsWithFlatState(t *testing.T) {
	var (
		genesis, key, sender, recipient = pbtChainGenesis(t)
		engine                          = beacon.New(ethash.NewFaker())
		signer                          = types.LatestSigner(genesis.Config)
		perBlock                        = big.NewInt(1000)
		blockCount                      = 6
	)
	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, blockCount, func(i int, gen *BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), recipient, perBlock, params.TxGas,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	})
	if len(blocks) != blockCount {
		t.Fatalf("generated %d blocks, want %d", len(blocks), blockCount)
	}
	// Generation must have moved the state on, or the import below proves nothing.
	if blocks[len(blocks)-1].Root() == genesis.ToBlock().Root() {
		t.Fatal("the generated chain left the state root unchanged")
	}

	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if !chain.TrieDB().IsPBT() {
		t.Fatal("chain is not running on the binary tree")
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("failed to import the generated chain: %v", err)
	}
	// Post-merge the head is driven by forkchoice, not by insertion, so promote
	// the tip explicitly.
	if _, err := chain.SetCanonical(blocks[len(blocks)-1]); err != nil {
		t.Fatalf("failed to make the imported chain canonical: %v", err)
	}
	if head := chain.CurrentBlock().Number.Uint64(); head != uint64(blockCount) {
		t.Fatalf("chain head is %d, want %d", head, blockCount)
	}

	statedb, err := chain.State()
	if err != nil {
		t.Fatal(err)
	}
	want := new(big.Int).Mul(perBlock, big.NewInt(int64(blockCount)))
	if got := statedb.GetBalance(recipient).ToBig(); got.Cmp(want) != 0 {
		t.Fatalf("recipient balance is %v, want %v", got, want)
	}
	if got := statedb.GetNonce(sender); got != uint64(blockCount) {
		t.Fatalf("sender nonce is %d, want %d", got, blockCount)
	}

	// The values above would also be produced by a state read that never
	// touched flat state, so compare the two stores directly. This is the
	// differential after *real block processing*, as opposed to over a state
	// built by writing to a StateDB - the flat entries here were laid down by
	// the import path, one block at a time.
	// 0xdead is in the list so the comparison covers absence too, not only
	// accounts both stores happen to hold.
	assertFlatMatchesTrie(t, chain, chain.CurrentBlock().Root, sender, recipient, common.Address{0xde, 0xad})
}

// TestPBTGenerateChainStatePreservingBlocks pins the case that actually broke:
// blocks that leave the state root untouched.
//
// This is the trigger, and it is narrower than "no transactions". A block with
// no state delta never reaches the trie database at all — the commit
// short-circuits on an unchanged root — so no layer exists for it, while the
// root itself is already the persisted one. Asking the path database to commit
// that root is the error. Whether the block carried transactions is incidental;
// what matters is whether it moved the state.
//
// Worth pinning separately because the test above uses transactions in every
// block and therefore never exercises this path. Note also that a bare
// binary-tree genesis makes it the *default*: post-merge there is no block
// reward, and the EIP-4788 and EIP-2935 system calls no-op against accounts
// with no code, so an empty block on such a genesis changes nothing at all.
func TestPBTGenerateChainStatePreservingBlocks(t *testing.T) {
	genesis, key, sender, recipient := pbtChainGenesis(t)
	engine := beacon.New(ethash.NewFaker())
	signer := types.LatestSigner(genesis.Config)

	for _, tc := range []struct {
		name  string
		count int
		gen   func(i int, gen *BlockGen)
	}{
		{"all state-preserving", 5, func(i int, gen *BlockGen) {}},
		{"nil generator", 3, nil},
		{"transaction then state-preserving", 2, func(i int, gen *BlockGen) {
			if i != 0 {
				return
			}
			tx, err := types.SignTx(types.NewTransaction(
				gen.TxNonce(sender), recipient, big.NewInt(1), params.TxGas,
				new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
			), signer, key)
			if err != nil {
				t.Fatal(err)
			}
			gen.AddTx(tx)
		}},
		{"state-preserving then transaction", 2, func(i int, gen *BlockGen) {
			if i != 1 {
				return
			}
			tx, err := types.SignTx(types.NewTransaction(
				gen.TxNonce(sender), recipient, big.NewInt(1), params.TxGas,
				new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
			), signer, key)
			if err != nil {
				t.Fatal(err)
			}
			gen.AddTx(tx)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, blocks, _ := GenerateChainWithGenesis(genesis, engine, tc.count, tc.gen)
			if len(blocks) != tc.count {
				t.Fatalf("generated %d blocks, want %d", len(blocks), tc.count)
			}
			// The premise: at least one block must genuinely leave the root
			// where it found it, or this exercises nothing the previous test
			// did not already cover.
			preserving := 0
			prev := genesis.ToBlock().Root()
			for _, block := range blocks {
				if block.Root() == prev {
					preserving++
				}
				prev = block.Root()
			}
			if preserving == 0 {
				t.Fatal("no block preserved the state root; this case does not exercise the path it is named for")
			}
			// The generated states must survive back into a chain built on the
			// same database, which is what callers do with it.
			chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
			if err != nil {
				t.Fatal(err)
			}
			defer chain.Stop()

			if _, err := chain.InsertChain(blocks); err != nil {
				t.Fatalf("failed to import: %v", err)
			}
		})
	}
}

// TestPBTGenerateChainContinues pins the reason the tip is persisted at all:
// callers pass the generator's database back in to extend the chain, which needs
// the last block's state still readable after the first call returned and closed
// its trie database.
func TestPBTGenerateChainContinues(t *testing.T) {
	genesis, key, sender, recipient := pbtChainGenesis(t)
	engine := beacon.New(ethash.NewFaker())
	signer := types.LatestSigner(genesis.Config)

	send := func(gen *BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), recipient, big.NewInt(1), params.TxGas,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	}
	db, first, _ := GenerateChainWithGenesis(genesis, engine, 3, func(i int, gen *BlockGen) { send(gen) })

	// Extend from the tip over the same database. Before the tip was persisted
	// this failed to open the parent's state at all.
	second, _ := GenerateChain(genesis.Config, first[len(first)-1], engine, db, 3, func(i int, gen *BlockGen) { send(gen) })
	if len(second) != 3 {
		t.Fatalf("continued with %d blocks, want 3", len(second))
	}
	if second[0].ParentHash() != first[len(first)-1].Hash() {
		t.Fatal("the continuation does not build on the first chain's tip")
	}
	if second[len(second)-1].Root() == first[len(first)-1].Root() {
		t.Fatal("the continuation left the state root unchanged")
	}
}
