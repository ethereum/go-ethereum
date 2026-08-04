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
	"bytes"
	"context"
	"maps"
	"math/big"
	"slices"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/rlp"
)

// The binary tree does not support everything the merkle-patricia trie does.
// Each unsupported operation is guarded in its own package, which makes the set
// of them hard to see and easy to regress one at a time. These tests cover the
// ones that live in this package; the index below is the whole contract, so it
// can be read without hunting for the guards.
//
// What is being pinned is not "this is unimplemented" but "this refuses, and
// says why". Silently returning a wrong answer is the failure mode that
// matters: a stateless run that rebuilds the wrong kind of database still
// produces a root, and a dump that gives up still returns bytes.
//
//	Operation                     Refused at                       Pinned by
//	---------------------------------------------------------------------------
//	witness statistics            core/blockchain.go triedbConfig  this file
//	historic state                core/blockchain_reader.go        core/pbt_scheme_test.go
//	hash-scheme trie database     core/blockchain.go triedbConfig  core/pbt_scheme_test.go
//	account dumping               core/state/dump.go               core/state/pbt_capabilities_test.go
//	pathdb rollback (Recover)     triedb/pathdb/database.go        triedb/pathdb/pbt_rollback_test.go
//	state sync / AdoptSyncedState triedb/pathdb/database.go        triedb/pathdb/pbt_rollback_test.go
//	opening a tree datadir as MPT cmd/utils MakeTrieDatabase       cmd/geth/bintrie_convert_test.go
//	debug_storageRangeAt          eth/api_debug.go                 not pinned - guard read, no test
//
// Anything on this list failing a spec fixture is a known gap rather than a
// conformance defect, which is the distinction that makes the reference
// integration readable.
//
// Stateless execution used to head that list. It is supported now, and the
// tests below pin the opposite property: that a block re-executes from its
// witness alone, and that a witness missing anything is refused instead of
// answered around.

// pbtWitnessFixture builds a one-block binary tree chain and returns the chain,
// the block, and the execution witness gathered while processing it.
func pbtWitnessFixture(t *testing.T) (*BlockChain, *types.Block, *stateless.Witness) {
	t.Helper()

	genesis, key, sender, recipient := pbtChainGenesis(t)
	engine := beacon.New(ethash.NewFaker())
	signer := types.LatestSigner(genesis.Config)

	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, 1, func(i int, gen *BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), recipient, big.NewInt(1000), pbtTestTxGas,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	})
	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(chain.Stop)

	parent := chain.GetHeaderByNumber(0)
	res, err := chain.ProcessBlock(context.Background(), parent.Root, blocks[0], ExecuteConfig{MakeWitness: true})
	if err != nil {
		t.Fatalf("processing the block to gather a witness: %v", err)
	}
	witness := res.Witness()
	if witness == nil {
		t.Fatal("no witness was gathered")
	}
	if len(witness.Nodes) == 0 {
		t.Fatal("the binary tree witness holds no nodes; it cannot reconstruct anything")
	}
	return chain, blocks[0], witness
}

// TestPBTStatelessExecution pins that a binary tree block can be re-executed
// from nothing but its witness.
//
// The tree is addressed differently from the merkle one - a group record folds
// at a depth stored inside it, so it is named by path rather than by the hash
// of its own bytes - which is why the witness keeps its paths and is rebuilt
// into a path database rather than a hash one.
func TestPBTStatelessExecution(t *testing.T) {
	chain, block, witness := pbtWitnessFixture(t)
	if !chain.Config().IsPBT() {
		t.Fatal("the fixture is not a binary tree configuration; this proves nothing")
	}
	// The stateless runner expects to compute these itself.
	header := types.CopyHeader(block.Header())
	header.Root, header.ReceiptHash = common.Hash{}, common.Hash{}
	task := types.NewBlockWithHeader(header).WithBody(*block.Body())

	stateRoot, receiptRoot, err := ExecuteStateless(context.Background(), chain.Config(), vm.Config{}, task, witness)
	if err != nil {
		t.Fatalf("stateless execution of a binary tree block failed: %v", err)
	}
	if stateRoot != block.Root() {
		t.Fatalf("stateless state root mismatch: got %x, want %x", stateRoot, block.Root())
	}
	if receiptRoot != block.ReceiptHash() {
		t.Fatalf("stateless receipt root mismatch: got %x, want %x", receiptRoot, block.ReceiptHash())
	}
}

// TestPBTStatelessRejectsIncompleteWitness pins that a witness missing a node
// the block touches is refused, rather than answered around.
//
// This is the failure that has to be loud. A missing node is latched into the
// state database's error and the read is served as an absent account or a zero
// slot, so execution runs to completion and returns a root computed over state
// that was never there. Nothing downstream re-checks that error, so without an
// explicit look the caller is handed a plausible wrong answer.
func TestPBTStatelessRejectsIncompleteWitness(t *testing.T) {
	chain, block, witness := pbtWitnessFixture(t)

	header := types.CopyHeader(block.Header())
	header.Root, header.ReceiptHash = common.Hash{}, common.Hash{}
	task := types.NewBlockWithHeader(header).WithBody(*block.Body())

	// Drop nodes one at a time; every one of them is reachable during the
	// re-execution, so each omission has to be caught.
	paths := slices.Sorted(maps.Keys(witness.Nodes))
	for _, path := range paths {
		holed := witness.Copy()
		delete(holed.Nodes, path)

		root, _, err := ExecuteStateless(context.Background(), chain.Config(), vm.Config{}, task, holed)
		if err == nil {
			t.Fatalf("witness without the node at path %x still executed, returning root %x", path, root)
		}
		if root != (common.Hash{}) {
			t.Fatalf("a rejected witness still produced a root: %x", root)
		}
	}
}

// TestPBTWitnessSurvivesEncoding pins that a binary tree witness survives the
// consensus encoding, which is the form debug_executionWitness hands out and
// the form anything off-process would receive.
//
// The paths are the part at risk. A merkle witness is a bag of blobs and needs
// no keys, so the key array went unused and unwritten; a binary witness is
// meaningless without it, and a decoder that dropped or misaligned it would
// produce a witness that still looks well-formed and rebuilds into the wrong
// tree.
func TestPBTWitnessSurvivesEncoding(t *testing.T) {
	chain, block, witness := pbtWitnessFixture(t)

	var buf bytes.Buffer
	if err := witness.EncodeRLP(&buf); err != nil {
		t.Fatalf("encoding a binary tree witness: %v", err)
	}
	decoded := new(stateless.Witness)
	if err := rlp.DecodeBytes(buf.Bytes(), decoded); err != nil {
		t.Fatalf("decoding a binary tree witness: %v", err)
	}
	if len(decoded.Nodes) != len(witness.Nodes) {
		t.Fatalf("round trip changed the node count: %d -> %d", len(witness.Nodes), len(decoded.Nodes))
	}
	for path, want := range witness.Nodes {
		got, ok := decoded.Nodes[path]
		if !ok {
			t.Fatalf("round trip lost the node at path %x", path)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("path %x: node changed across the round trip", path)
		}
	}
	// The real assertion: the decoded witness still reconstructs the state.
	// Matching maps would not catch paths that survive but no longer address
	// what they did.
	header := types.CopyHeader(block.Header())
	header.Root, header.ReceiptHash = common.Hash{}, common.Hash{}
	task := types.NewBlockWithHeader(header).WithBody(*block.Body())

	stateRoot, _, err := ExecuteStateless(context.Background(), chain.Config(), vm.Config{}, task, decoded)
	if err != nil {
		t.Fatalf("a round-tripped witness failed to execute: %v", err)
	}
	if stateRoot != block.Root() {
		t.Fatalf("round-tripped witness produced root %x, want %x", stateRoot, block.Root())
	}
}

// TestPBTRefusesWitnessStats pins that witness statistics are refused rather
// than collected wrongly.
//
// WitnessStats reads a node's path as a nibble string and its depth as that
// string's length, then buckets into a fixed sixteen levels. A binary path is
// a two-byte bit count followed by packed bits: the depth is wrong from the
// very first node, and once a walk passes 113 bits the length exceeds sixteen
// and indexes off the end of the histogram. The flag also force-enables
// stateless self-validation, so it is reachable without asking for it.
func TestPBTRefusesWitnessStats(t *testing.T) {
	genesis, _, _, _ := pbtChainGenesis(t)
	engine := beacon.New(ethash.NewFaker())
	db, _, _ := GenerateChainWithGenesis(genesis, engine, 1, func(i int, gen *BlockGen) {})

	options := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	options.EnableWitnessStats = true

	chain, err := NewBlockChain(db, genesis, engine, options)
	if err == nil {
		chain.Stop()
		t.Fatal("a binary tree chain opened with witness statistics enabled")
	}
	if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("the refusal does not name its cause: %v", err)
	}

	// The control: the same flag on a merkle chain is accepted, so the refusal
	// is known to be about the tree rather than about the flag.
	plain := *genesis
	cfg := *genesis.Config
	cfg.PBT = false
	plain.Config = &cfg
	mdb, _, _ := GenerateChainWithGenesis(&plain, engine, 1, func(i int, gen *BlockGen) {})
	mchain, err := NewBlockChain(mdb, &plain, engine, options)
	if err != nil {
		t.Fatalf("witness statistics were refused on a merkle chain: %v", err)
	}
	mchain.Stop()
}

// TestPBTStatelessSelfValidationOnImport drives the whole thing through
// block import, which is the way a node actually reaches it.
//
// StatelessSelfValidation is an ordinary flag, and witness building is gated
// only on IsByzantium, so a binary tree node can turn this on and this is the
// path it takes. The import succeeding is the assertion: blockchain.go
// re-executes the block against nothing but the witness and compares both
// roots itself, so a witness that failed to reconstruct the state would show
// up here as a root mismatch rather than as a passing import.
func TestPBTStatelessSelfValidationOnImport(t *testing.T) {
	genesis, key, sender, recipient := pbtChainGenesis(t)
	engine := beacon.New(ethash.NewFaker())
	signer := types.LatestSigner(genesis.Config)

	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, 1, func(i int, gen *BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), recipient, big.NewInt(1000), pbtTestTxGas,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	})

	options := DefaultConfig().WithStateScheme(rawdb.PathScheme)
	options.StatelessSelfValidation = true

	chain, err := NewBlockChain(db, genesis, engine, options)
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if _, err = chain.InsertChain(blocks); err != nil {
		t.Fatalf("a binary tree block failed to import under stateless self-validation: %v", err)
	}
	// The import succeeding is the assertion. blockchain.go re-executes the
	// block against nothing but the witness and compares both roots itself, so
	// a witness that did not reconstruct the state would surface here as a
	// root mismatch rather than as a passing import.
	if head := chain.CurrentBlock().Number.Uint64(); head != 1 {
		t.Fatalf("chain head is %d, want 1", head)
	}
}
