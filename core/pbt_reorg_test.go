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
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/bintrie"
)

// TestPBTReportsRollbackUnsupported pins the chain-level half of how the binary
// tree handles reorgs: it reports itself as unable to roll state back, which is
// what routes every caller onto re-executing the winning branch forward from
// the closest ancestor whose state is still live.
//
// Reverting works by replaying pre-transition account and storage values
// through the trie until the parent root reappears, which is only valid while
// the trie is a pure function of those values. The binary tree also stores
// contract code, and every code chunk is content-addressed, so whether such a
// leaf belongs at the parent root depends on whether any other account held
// the same bytecode - a question no per-account history answers.
//
// `stateRecoverable` returning false is therefore the honest answer, not a
// degradation: `recoverAncestors` and `insertSideChain` both already treat
// rollback as a fast path and fall back to re-execution.
//
// Scope: this pins the reporting and the refusal. That the refusal is
// meaningful - that the same root would otherwise have been accepted - is
// pinned in pathdb.TestPBTRollbackUnsupported against a merkle database. What
// the chain then does about it is pinned by the two tests below: a reorg inside
// the layer window lands correctly, and one past it fails by name.
func TestPBTReportsRollbackUnsupported(t *testing.T) {
	var (
		genesis = pbtSchemeGenesis()
		engine  = beacon.New(ethash.NewFaker())
	)
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if !chain.TrieDB().IsPBT() {
		t.Fatal("chain is not running on the binary tree")
	}
	head := chain.CurrentBlock()
	root := head.Root

	// Live state is still served; only rolling back is out of reach.
	if _, err := chain.StateAt(head); err != nil {
		t.Fatalf("live state is unavailable on the binary tree: %v", err)
	}
	if chain.StateRecoverable(root) {
		t.Fatal("chain reports its state as recoverable; the binary tree cannot revert")
	}
	err = chain.triedb.Recover(root)
	if err == nil {
		t.Fatal("binary tree accepted a rollback request")
	}
	if !strings.Contains(err.Error(), "binary tree") {
		t.Fatalf("rollback refusal does not name its cause: %v", err)
	}
}

// payTo returns a generator that sends value to a fixed recipient, so each
// branch of a fork moves the state somewhere distinguishable.
func payTo(t *testing.T, key *ecdsa.PrivateKey, sender, recipient common.Address, signer types.Signer, amount int64) func(int, *BlockGen) {
	t.Helper()

	return func(i int, gen *BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), recipient, big.NewInt(amount), pbtTestTxGas,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	}
}

// assertFlatMatchesTrie compares the flat store against the tree at a root, in
// both directions including absence, so a store that answered "absent" for
// everything would be caught rather than looking correct and fast.
func assertFlatMatchesTrie(t *testing.T, chain *BlockChain, root common.Hash, addrs ...common.Address) {
	t.Helper()

	flat, err := chain.TrieDB().StateReader(root)
	if err != nil {
		t.Fatalf("flat state is not available at %x: %v", root, err)
	}
	tree, err := bintrie.NewBinaryTrie(root, chain.TrieDB())
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		flatAcct, err := flat.Account(crypto.Keccak256Hash(addr[:]))
		if err != nil {
			t.Fatalf("flat read of %x failed: %v", addr, err)
		}
		treeAcct, err := tree.GetAccount(addr)
		if err != nil {
			t.Fatalf("trie read of %x failed: %v", addr, err)
		}
		if (flatAcct == nil) != (treeAcct == nil) {
			t.Fatalf("account %x: flat has=%t, trie has=%t", addr, flatAcct != nil, treeAcct != nil)
		}
		if flatAcct == nil {
			continue // absent in both, which is an answer
		}
		if flatAcct.Nonce != treeAcct.Nonce || flatAcct.Balance.Cmp(treeAcct.Balance) != 0 {
			t.Fatalf("account %x: flat nonce=%d balance=%v, trie nonce=%d balance=%v",
				addr, flatAcct.Nonce, flatAcct.Balance, treeAcct.Nonce, treeAcct.Balance)
		}
	}
}

// TestPBTReorgInsideWindowKeepsFlatState reorgs between two branches that are
// both still diff layers, and checks the chain lands on the winning root with
// flat state and the tree agreeing afterwards.
//
// The flat store is per-layer, so the data a branch contributed goes away with
// the branch. TestPBTFlatStateFollowsTheBranch pins that one layer down, with
// empty node sets and no trie at all; this is the same claim after real block
// processing, where the two stores can actually disagree.
func TestPBTReorgInsideWindowKeepsFlatState(t *testing.T) {
	var (
		genesis, key, sender, _ = pbtChainGenesis(t)
		engine                  = beacon.New(ethash.NewFaker())
		signer                  = types.LatestSigner(genesis.Config)
		onlyOnA                 = common.Address{0xaa, 0xaa}
		onlyOnB                 = common.Address{0xbb, 0xbb}
		neverPaid               = common.Address{0xcc, 0xcc}
	)
	// Both branches fork at genesis and pay different recipients, so each one
	// leaves a mark the other does not have.
	db, branchA, _ := GenerateChainWithGenesis(genesis, engine, 2, payTo(t, key, sender, onlyOnA, signer, 1000))
	branchB, _ := GenerateChain(genesis.Config, genesis.ToBlock(), engine, db, 3, payTo(t, key, sender, onlyOnB, signer, 5000))

	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if !chain.TrieDB().IsPBT() {
		t.Fatal("chain is not running on the binary tree")
	}
	if _, err := chain.InsertChain(branchA); err != nil {
		t.Fatalf("failed to import the first branch: %v", err)
	}
	if _, err := chain.SetCanonical(branchA[len(branchA)-1]); err != nil {
		t.Fatalf("failed to make the first branch canonical: %v", err)
	}
	// The premise: branch A really did pay its recipient, or the absence
	// asserted after the reorg means nothing.
	statedb, err := chain.State()
	if err != nil {
		t.Fatal(err)
	}
	if statedb.GetBalance(onlyOnA).IsZero() {
		t.Fatal("branch A never paid its recipient; the fixture proves nothing")
	}
	assertFlatMatchesTrie(t, chain, chain.CurrentBlock().Root, sender, onlyOnA, onlyOnB, neverPaid)

	// Reorg onto the competing branch.
	for _, block := range branchB {
		if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
			t.Fatalf("failed to import block %d of the competing branch: %v", block.NumberU64(), err)
		}
	}
	tip := branchB[len(branchB)-1]
	if _, err := chain.SetCanonical(tip); err != nil {
		t.Fatalf("failed to reorg onto the competing branch: %v", err)
	}
	head := chain.CurrentBlock()
	if head.Hash() != tip.Hash() {
		t.Fatalf("head is %x, want %x", head.Hash(), tip.Hash())
	}
	if head.Root != tip.Root() {
		t.Fatalf("head root is %x, want %x", head.Root, tip.Root())
	}
	statedb, err = chain.State()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := statedb.GetBalance(onlyOnB).ToBig(), big.NewInt(5000*int64(len(branchB))); got.Cmp(want) != 0 {
		t.Fatalf("the winning branch's recipient holds %v, want %v", got, want)
	}
	if got := statedb.GetBalance(onlyOnA); !got.IsZero() {
		t.Fatalf("the reorged-out branch's recipient still holds %v", got)
	}
	if got, want := statedb.GetNonce(sender), uint64(len(branchB)); got != want {
		t.Fatalf("sender nonce is %d, want %d", got, want)
	}
	// The values above would also be produced by a read that never consulted
	// flat state, so compare the two stores directly - including the account
	// the losing branch created, which must now be absent in both.
	assertFlatMatchesTrie(t, chain, head.Root, sender, onlyOnA, onlyOnB, neverPaid)
}

// TestPBTReorgPastWindowFailsByName pins the ceiling: a fork point below the
// persisted disk layer cannot be served, and says so.
//
// Re-execution walks back looking for an ancestor whose state is still live. On
// a chain longer than the layer window the disk layer has moved past genesis,
// so no ancestor of the losing branch has one, the walk runs out of blocks, and
// the binary tree reports that rather than blaming a missing parent.
func TestPBTReorgPastWindowFailsByName(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a chain longer than the layer window")
	}
	var (
		genesis, key, sender, _ = pbtChainGenesis(t)
		engine                  = beacon.New(ethash.NewFaker())
		signer                  = types.LatestSigner(genesis.Config)
		// Comfortably past the 128-layer window, so genesis is well below the
		// disk layer by the time the reorg is attempted.
		trunkLength = 200
	)
	// Generate the losing branch first. Once the long trunk has been generated
	// the generator's own layer tree has flattened genesis away, and a branch
	// anchored there would have no state to build on.
	db, losing, _ := GenerateChainWithGenesis(genesis, engine, 2, payTo(t, key, sender, common.Address{0xbb}, signer, 5000))
	trunk, _ := GenerateChain(genesis.Config, genesis.ToBlock(), engine, db, trunkLength, payTo(t, key, sender, common.Address{0xaa}, signer, 1000))

	// Run the chain on its own database rather than the generator's. Reusing
	// the generator's would hand the chain a journal whose disk layer already
	// sits deep in the trunk, so genesis state would be gone before the first
	// block was imported and the trunk would go in as a side chain with no
	// state at all - reaching the ceiling by accident, from the wrong end.
	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if n, err := chain.InsertChain(trunk); err != nil || n != len(trunk) {
		t.Fatalf("imported %d of %d trunk blocks: %v", n, len(trunk), err)
	}
	if _, err := chain.SetCanonical(trunk[len(trunk)-1]); err != nil {
		t.Fatalf("failed to make the trunk canonical: %v", err)
	}
	// The premise: genesis state really is gone, or the walk below would find
	// an ancestor and succeed for an uninteresting reason.
	if chain.HasState(genesis.ToBlock().Root()) {
		t.Fatal("genesis state is still live after a chain longer than the window; the fixture does not reach the ceiling")
	}
	// Now try to reorg to a branch forking at genesis. This can be refused
	// either when the blocks are imported or when the head is moved, since
	// both routes reach the same walk - assert on what it says, not on which
	// call says it.
	var failure error
	for _, block := range losing {
		if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
			failure = err
			break
		}
	}
	if failure == nil {
		_, failure = chain.SetCanonical(losing[len(losing)-1])
	}
	if failure == nil {
		t.Fatal("a reorg past the persisted disk layer was accepted")
	}
	if !strings.Contains(failure.Error(), "cannot rewind past the persisted state") {
		t.Fatalf("reorg past the window failed, but not by name: %v", failure)
	}
}
