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
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm/program"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/trie/bintrie"
)

// Reorgs and content-addressed code chunks.
//
// Code lives in the tree, and it is the one thing in it whose leaves are
// shared between accounts: every chunk is keyed by
// CodeChunkStem(codeHash, treeIndex) and shared by every contract with
// identical bytecode, with no account taking part. So a reorg that undoes a
// deployment appears to face a question with no answer in the data - may those
// shared chunks go, or does some other account still need them? - and the
// usual way out is reference counting.
//
// It never has to be answered. Reorgs replace layers rather than reverse them:
// an abandoned branch is removed from the layer tree and cascaded to its
// children (layertree.go), and a diff layer's nodes live in memory only. So
// chunks added by a reverted block go away with the layer, and chunks that
// were persisted belong at or below the disk layer, where the binary tree
// refuses a fork by name. "Was this code already deployed?" is answered by
// whether the ancestor state holds the chunk - no counting, no bookkeeping.
//
// These two tests are the same fixture with one variable: whether the shared
// bytecode already existed on the common ancestor. That variable is the whole
// question, so it is the only difference between them.

// pbtBigCode is a runtime blob of 130 code chunks. The length no longer
// matters for sharing, but it is kept as it was so the deployment still fits
// the gas limits below.
//
// JUMPDEST repeated is not arbitrary: it keeps every byte its own instruction,
// so no PUSH swallows the bytes that follow it.
var pbtBigCode = bytes.Repeat([]byte{0x5b}, 130*31)

// pbtSharedChunk is the chunk these tests follow across the reorg. Chunk 0 is
// the pointed choice: it used to live in the account's own header stem.
const pbtSharedChunk = 0

// pbtCodeDeployGas is the gas limit the deployment transactions carry, and
// pbtCodeBlockGas the block limit that has to hold one.
//
// Deploying pbtBigCode costs ~200 gas per deployed byte on its own, and
// Amsterdam charges state gas on top of that, so both figures are far above
// what the same deployment needed pre-Amsterdam.
const (
	pbtCodeDeployGas = 8_000_000
	pbtCodeBlockGas  = 30_000_000
)

// pbtCodeReorgFixture is a binary-tree genesis with a funded sender and enough
// block gas to deploy pbtBigCode.
func pbtCodeReorgFixture(t *testing.T) (*Genesis, *ecdsa.PrivateKey, common.Address) {
	t.Helper()

	genesis, key, sender, _ := pbtChainGenesis(t)
	genesis.GasLimit = pbtCodeBlockGas
	return genesis, key, sender
}

// deployBigCode returns a generator that deploys pbtBigCode as a contract, and
// the address the contract will land at.
func deployBigCode(t *testing.T, key *ecdsa.PrivateKey, sender common.Address, signer types.Signer, nonce uint64) (func(int, *BlockGen), common.Address) {
	t.Helper()

	// A standard constructor: CODECOPY the blob out of the init code and
	// return it as the runtime code.
	initCode := program.New().ReturnViaCodeCopy(pbtBigCode).Bytes()

	return func(i int, gen *BlockGen) {
		tx, err := types.SignTx(types.NewContractCreation(
			gen.TxNonce(sender), big.NewInt(0), pbtCodeDeployGas,
			new(big.Int).Add(gen.BaseFee(), common.Big1), initCode,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	}, crypto.CreateAddress(sender, nonce)
}

// sendValue returns a generator that moves a little ether, so a branch that
// deploys nothing still changes the state root.
func sendValue(t *testing.T, key *ecdsa.PrivateKey, sender common.Address, signer types.Signer) func(int, *BlockGen) {
	t.Helper()

	return func(i int, gen *BlockGen) {
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), common.Address{0x0f, 0xf1}, big.NewInt(1000), pbtTestTxGas,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	}
}

// codeChunkAt reads a code chunk leaf directly out of the tree at a given
// root. It returns nil when the leaf is absent, which is the whole point.
func codeChunkAt(t *testing.T, chain *BlockChain, root, codeHash common.Hash, chunk uint64) []byte {
	t.Helper()

	tr, err := bintrie.NewBinaryTrie(root, chain.TrieDB())
	if err != nil {
		t.Fatalf("cannot open the tree at %x: %v", root, err)
	}
	// No address takes part: the key is content-addressed, which is what makes
	// the leaf shared.
	value, err := tr.GetStemValue(bintrie.CodeChunkKey(codeHash, chunk))
	if err != nil {
		t.Fatalf("failed to read code chunk %d at %x: %v", chunk, root, err)
	}
	return value
}

// accountAt reads an account out of the tree at a given root, nil if absent.
func accountAt(t *testing.T, chain *BlockChain, root common.Hash, addr common.Address) *types.StateAccount {
	t.Helper()

	tr, err := bintrie.NewBinaryTrie(root, chain.TrieDB())
	if err != nil {
		t.Fatalf("cannot open the tree at %x: %v", root, err)
	}
	acct, err := tr.GetAccount(addr)
	if err != nil {
		t.Fatalf("failed to read account %x at %x: %v", addr, root, err)
	}
	return acct
}

// TestPBTReorgDropsCodeChunks reorgs away the only block that ever deployed a
// large contract, and checks its shared chunks went with it.
//
// Nothing deletes them. The branch that wrote them is dropped from the layer
// tree, and their leaves were never anywhere else.
func TestPBTReorgDropsCodeChunks(t *testing.T) {
	var (
		genesis, key, sender = pbtCodeReorgFixture(t)
		engine               = beacon.New(ethash.NewFaker())
		signer               = types.LatestSigner(genesis.Config)
		codeHash             = crypto.Keccak256Hash(pbtBigCode)
	)
	deploy, contract := deployBigCode(t, key, sender, signer, 0)

	// Both branches fork at genesis. Branch A deploys, branch B does not.
	db, branchA, _ := GenerateChainWithGenesis(genesis, engine, 1, deploy)
	branchB, _ := GenerateChain(genesis.Config, genesis.ToBlock(), engine, db, 1, sendValue(t, key, sender, signer))

	if branchA[0].Root() == branchB[0].Root() {
		t.Fatal("the two branches produced the same state root; the fixture does not fork the state")
	}
	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if !chain.TrieDB().IsPBT() {
		t.Fatal("chain is not running on the binary tree")
	}
	if _, err := chain.InsertChain(branchA); err != nil {
		t.Fatalf("failed to import the deploying branch: %v", err)
	}
	if _, err := chain.SetCanonical(branchA[0]); err != nil {
		t.Fatalf("failed to make the deploying branch canonical: %v", err)
	}
	// The premise: the deployment really did put a leaf in the shared zone.
	// Without this the test below passes for the wrong reason - an absent
	// chunk is also what a mis-derived key looks like.
	if got := codeChunkAt(t, chain, branchA[0].Root(), codeHash, pbtSharedChunk); got == nil {
		t.Fatal("the deploying branch has no shared code chunk; the fixture never exercised the shared zone")
	}
	if acct := accountAt(t, chain, branchA[0].Root(), contract); acct == nil {
		t.Fatal("the contract account is missing on the branch that deployed it")
	}

	// Reorg onto the branch that never deployed anything.
	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), branchB[0], false); err != nil {
		t.Fatalf("failed to import the competing branch: %v", err)
	}
	if _, err := chain.SetCanonical(branchB[0]); err != nil {
		t.Fatalf("failed to reorg onto the competing branch: %v", err)
	}
	head := chain.CurrentBlock()
	if head.Hash() != branchB[0].Hash() {
		t.Fatalf("head is %x, want the competing branch %x", head.Hash(), branchB[0].Hash())
	}
	if head.Root != branchB[0].Root() {
		t.Fatalf("head root is %x, want %x", head.Root, branchB[0].Root())
	}
	if got := codeChunkAt(t, chain, head.Root, codeHash, pbtSharedChunk); got != nil {
		t.Fatalf("the shared code chunk survived a reorg that dropped its only deployment: %x", got)
	}
	if acct := accountAt(t, chain, head.Root, contract); acct != nil {
		t.Fatal("the contract account survived the reorg that dropped its deployment")
	}
}

// TestPBTReorgKeepsSharedCodeChunks is the discriminating case: a reorg drops a
// deployment of bytecode that was *already* deployed by someone else, and the
// shared chunks must stay.
//
// This is the test that fails the moment anything starts deleting chunks on
// revert, and it is the direct answer to "how do we know the code was already
// deployed". Nothing knows. The chunks are in the ancestor state that the
// dropped layer sat on top of, so they are simply still there.
func TestPBTReorgKeepsSharedCodeChunks(t *testing.T) {
	var (
		genesis, key, sender = pbtCodeReorgFixture(t)
		engine               = beacon.New(ethash.NewFaker())
		signer               = types.LatestSigner(genesis.Config)
		codeHash             = crypto.Keccak256Hash(pbtBigCode)
	)
	deployFirst, first := deployBigCode(t, key, sender, signer, 0)
	deploySecond, second := deployBigCode(t, key, sender, signer, 1)

	if first == second {
		t.Fatal("both deployments landed at the same address")
	}
	// A shared trunk deploys the code once. Both branches fork from its tip:
	// A deploys the identical bytecode a second time, B does not.
	db, trunk, _ := GenerateChainWithGenesis(genesis, engine, 1, deployFirst)
	branchA, _ := GenerateChain(genesis.Config, trunk[0], engine, db, 1, deploySecond)
	branchB, _ := GenerateChain(genesis.Config, trunk[0], engine, db, 1, sendValue(t, key, sender, signer))

	if branchA[0].Root() == branchB[0].Root() {
		t.Fatal("the two branches produced the same state root; the fixture does not fork the state")
	}
	// The sharing itself. Identical bytecode now derives an identical key by
	// construction, so what is worth asserting is that the key is
	// content-addressed at all: it lives in the code zone, not in a header.
	if key := bintrie.CodeChunkKey(codeHash, pbtSharedChunk); key[0] != bintrie.CodeZone {
		t.Fatalf("chunk %d is not in the code zone: zone byte %#x", pbtSharedChunk, key[0])
	}
	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if _, err := chain.InsertChain(trunk); err != nil {
		t.Fatalf("failed to import the shared trunk: %v", err)
	}
	if _, err := chain.InsertChain(branchA); err != nil {
		t.Fatalf("failed to import the second deployment: %v", err)
	}
	if _, err := chain.SetCanonical(branchA[0]); err != nil {
		t.Fatalf("failed to make the second deployment canonical: %v", err)
	}
	if got := codeChunkAt(t, chain, branchA[0].Root(), codeHash, pbtSharedChunk); got == nil {
		t.Fatal("no shared code chunk after two deployments; the fixture never exercised the shared zone")
	}
	if acct := accountAt(t, chain, branchA[0].Root(), second); acct == nil {
		t.Fatal("the second contract is missing on the branch that deployed it")
	}

	// Reorg away the second deployment. The first contract still holds this
	// bytecode, so its chunks must survive.
	if _, err := chain.InsertBlockWithoutSetHead(context.Background(), branchB[0], false); err != nil {
		t.Fatalf("failed to import the competing branch: %v", err)
	}
	if _, err := chain.SetCanonical(branchB[0]); err != nil {
		t.Fatalf("failed to reorg onto the competing branch: %v", err)
	}
	head := chain.CurrentBlock()
	if head.Root != branchB[0].Root() {
		t.Fatalf("head root is %x, want %x", head.Root, branchB[0].Root())
	}
	if got := codeChunkAt(t, chain, head.Root, codeHash, pbtSharedChunk); got == nil {
		t.Fatal("the shared chunk was dropped by a reorg, but the first contract still holds that bytecode")
	}
	// The other polarity, in the same test: the reorged-out deployment really
	// is gone. Without this the check above would pass on a tree that never
	// dropped anything at all.
	if acct := accountAt(t, chain, head.Root, second); acct != nil {
		t.Fatal("the reorged-out contract account survived")
	}
	if acct := accountAt(t, chain, head.Root, first); acct == nil {
		t.Fatal("the surviving contract account is gone")
	}
}

// TestPBTReorgCodeAcrossThePersistedLayer pins the delete-or-keep answer when
// the surviving bytecode is below the disk layer rather than in a live diff
// layer, which is the only case the two tests above do not reach.
func TestPBTReorgCodeAcrossThePersistedLayer(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a chain longer than the layer window")
	}
	for _, prior := range []bool{true, false} {
		name := "code already deployed below the disk layer"
		if !prior {
			name = "code first deployed on the reorged-out branch"
		}
		t.Run(name, func(t *testing.T) {
			reorgCodeAcrossPersisted(t, prior)
		})
	}
}

func reorgCodeAcrossPersisted(t *testing.T, prior bool) {
	t.Helper()

	var (
		genesis, key, sender = pbtCodeReorgFixture(t)
		engine               = beacon.New(ethash.NewFaker())
		signer               = types.LatestSigner(genesis.Config)
		codeHash             = crypto.Keccak256Hash(pbtBigCode)
		// 120 + 30 puts the head at 150 and the disk layer at 22, so block 1 is
		// persisted and the fork point at 120 is still live.
		trunkLen, branchLen = 120, 30
	)
	deployFirst, first := deployBigCode(t, key, sender, signer, 0)
	deploySecond, second := deployBigCode(t, key, sender, signer, uint64(trunkLen))
	if first == second {
		t.Fatal("both deployments landed at the same address")
	}
	pay := payTo(t, key, sender, common.Address{0xaa}, signer, 1000)

	db, trunk, _ := GenerateChainWithGenesis(genesis, engine, trunkLen, func(i int, gen *BlockGen) {
		if i == 0 && prior {
			deployFirst(i, gen)
			return
		}
		pay(i, gen)
	})
	tip := trunk[len(trunk)-1]
	branchA, _ := GenerateChain(genesis.Config, tip, engine, db, branchLen, func(i int, gen *BlockGen) {
		if i == 0 {
			deploySecond(i, gen)
			return
		}
		pay(i, gen)
	})
	branchB, _ := GenerateChain(genesis.Config, tip, engine, db, branchLen, payTo(t, key, sender, common.Address{0xbb}, signer, 2000))

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if n, err := chain.InsertChain(trunk); err != nil || n != len(trunk) {
		t.Fatalf("imported %d of %d trunk blocks: %v", n, len(trunk), err)
	}
	// The second deployment is only redundant if the first one already landed.
	if got := codeChunkAt(t, chain, tip.Root(), codeHash, pbtSharedChunk); prior != (got != nil) {
		t.Fatalf("at the trunk tip: chunk present=%v, want %v", got != nil, prior)
	}
	if n, err := chain.InsertChain(branchA); err != nil || n != len(branchA) {
		t.Fatalf("imported %d of %d branch blocks: %v", n, len(branchA), err)
	}
	// Without these the test could pass for the in-memory reason the fixtures
	// above already cover.
	if chain.HasState(trunk[0].Root()) {
		t.Fatal("block 1 is still live, so nothing was persisted")
	}
	if !chain.HasState(tip.Root()) {
		t.Fatal("the fork point is not live, so the reorg cannot be served at all")
	}
	if chain.StateRecoverable(trunk[0].Root()) || chain.StateRecoverable(tip.Root()) {
		t.Fatal("a history route is available; this is not a layer-tree-only result")
	}
	if got := codeChunkAt(t, chain, chain.CurrentBlock().Root, codeHash, pbtSharedChunk); got == nil {
		t.Fatal("the chunk is absent on the branch that deployed it")
	}

	for _, block := range branchB {
		if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
			t.Fatalf("failed to import a branch B block: %v", err)
		}
	}
	if _, err := chain.SetCanonical(branchB[len(branchB)-1]); err != nil {
		t.Fatalf("failed to reorg onto branch B: %v", err)
	}
	head := chain.CurrentBlock()
	if head.Root != branchB[len(branchB)-1].Root() {
		t.Fatalf("head root is %x, want %x", head.Root, branchB[len(branchB)-1].Root())
	}
	if got := codeChunkAt(t, chain, head.Root, codeHash, pbtSharedChunk); prior != (got != nil) {
		t.Fatalf("after the reorg: chunk present=%v, want %v", got != nil, prior)
	}
	if acct := accountAt(t, chain, head.Root, second); acct != nil {
		t.Fatal("the reorged-out contract survived")
	}
	if acct := accountAt(t, chain, head.Root, first); prior != (acct != nil) {
		t.Fatalf("after the reorg: first contract present=%v, want %v", acct != nil, prior)
	}
}

// TestPBTReorgPastThePersistedLayerRefusesWithCode pins that once the block
// which deployed the code has itself been persisted, the reorg that would have
// to unpick it is refused rather than answered.
func TestPBTReorgPastThePersistedLayerRefusesWithCode(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a chain longer than the layer window")
	}
	var (
		genesis, key, sender = pbtCodeReorgFixture(t)
		engine               = beacon.New(ethash.NewFaker())
		signer               = types.LatestSigner(genesis.Config)
		// Branch A must outrun the window by enough that its own first block,
		// the one holding the deployment, ends up below the disk layer.
		trunkLen, branchALen, branchBLen = 120, 130, 30
	)
	deploy, _ := deployBigCode(t, key, sender, signer, uint64(trunkLen))
	pay := payTo(t, key, sender, common.Address{0xaa}, signer, 1000)

	db, trunk, _ := GenerateChainWithGenesis(genesis, engine, trunkLen, pay)
	tip := trunk[len(trunk)-1]
	// Branch B first: generating the long branch A flattens the fork point out of
	// the generator's own layer tree, leaving nothing to build B on.
	branchB, _ := GenerateChain(genesis.Config, tip, engine, db, branchBLen, payTo(t, key, sender, common.Address{0xbb}, signer, 2000))
	branchA, _ := GenerateChain(genesis.Config, tip, engine, db, branchALen, func(i int, gen *BlockGen) {
		if i == 0 {
			deploy(i, gen)
			return
		}
		pay(i, gen)
	})

	chain, err := NewBlockChain(rawdb.NewMemoryDatabase(), genesis, engine, DefaultConfig().WithStateScheme(rawdb.PathScheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if n, err := chain.InsertChain(trunk); err != nil || n != len(trunk) {
		t.Fatalf("imported %d of %d trunk blocks: %v", n, len(trunk), err)
	}
	if n, err := chain.InsertChain(branchA); err != nil || n != len(branchA) {
		t.Fatalf("imported %d of %d branch A blocks: %v", n, len(branchA), err)
	}
	if chain.HasState(tip.Root()) {
		t.Fatal("the fork point is still live; the fixture does not reach the ceiling")
	}
	var failure error
	for _, block := range branchB {
		if _, err := chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
			failure = err
			break
		}
	}
	if failure == nil {
		_, failure = chain.SetCanonical(branchB[len(branchB)-1])
	}
	if failure == nil {
		t.Fatal("a reorg past the persisted deployment was accepted")
	}
	if !strings.Contains(failure.Error(), "cannot rewind past the persisted state") {
		t.Fatalf("refused, but not by name: %v", failure)
	}
}
