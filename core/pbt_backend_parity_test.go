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

// backendRun is what running the same chain on one commitment produced.
type backendRun struct {
	root      common.Hash
	balance   *big.Int
	nonce     uint64
	blockRoot common.Hash
}

// runParityChain builds and imports an identical chain on the requested
// commitment and reports the observable results.
//
// Each arm uses the node scheme its generator wrote: the binary tree is
// path-only, and GenerateChainWithGenesis commits a merkle-patricia genesis
// under the hash scheme. That does not weaken the comparison - the scheme
// decides how nodes are keyed on disk, not what the trie hashes to, so a
// merkle-patricia root is the same either way.
func runParityChain(t *testing.T, pbt bool, forkTweak func(*params.ChainConfig)) backendRun {
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
	if pbt {
		config.BinaryTrieTime = u64(0)
	} else {
		config.BinaryTrieTime = nil
	}
	if forkTweak != nil {
		forkTweak(&config)
	}
	if err := config.CheckConfigForkOrder(); err != nil {
		t.Fatalf("configuration rejected (pbt=%v): %v", pbt, err)
	}
	genesis := &Genesis{
		Config:     &config,
		Difficulty: big.NewInt(0),
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			sender: {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))},
		},
	}

	var (
		engine     = beacon.New(ethash.NewFaker())
		signer     = types.LatestSigner(genesis.Config)
		perBlock   = big.NewInt(1000)
		blockCount = 4
	)
	db, blocks, _ := GenerateChainWithGenesis(genesis, engine, blockCount, func(i int, gen *BlockGen) {
		// Amsterdam prices a plain transfer well above the 21000 intrinsic
		// figure, so the limit is generous on purpose: this test is about the
		// commitment, and a gas-starved transaction would silently make both
		// arms agree on an empty result.
		tx, err := types.SignTx(types.NewTransaction(
			gen.TxNonce(sender), recipient, perBlock, 1_000_000,
			new(big.Int).Add(gen.BaseFee(), common.Big1), nil,
		), signer, key)
		if err != nil {
			t.Fatal(err)
		}
		gen.AddTx(tx)
	})
	if len(blocks) != blockCount {
		t.Fatalf("generated %d blocks, want %d (pbt=%v)", len(blocks), blockCount, pbt)
	}

	scheme := rawdb.HashScheme
	if pbt {
		scheme = rawdb.PathScheme
	}
	chain, err := NewBlockChain(db, genesis, engine, DefaultConfig().WithStateScheme(scheme))
	if err != nil {
		t.Fatal(err)
	}
	defer chain.Stop()

	if got := chain.TrieDB().IsPBT(); got != pbt {
		t.Fatalf("chain runs on binary tree = %v, want %v", got, pbt)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatalf("failed to import the chain (pbt=%v): %v", pbt, err)
	}
	if _, err := chain.SetCanonical(blocks[len(blocks)-1]); err != nil {
		t.Fatalf("failed to make the chain canonical (pbt=%v): %v", pbt, err)
	}
	statedb, err := chain.State()
	if err != nil {
		t.Fatal(err)
	}
	// The transactions have to have landed, or the comparison below is between
	// two identical no-ops and proves nothing.
	balance := statedb.GetBalance(recipient).ToBig()
	want := new(big.Int).Mul(perBlock, big.NewInt(int64(blockCount)))
	if balance.Cmp(want) != 0 {
		t.Fatalf("recipient balance is %v, want %v (pbt=%v): the transactions never landed", balance, want, pbt)
	}
	return backendRun{
		root:      chain.CurrentBlock().Root,
		balance:   balance,
		nonce:     statedb.GetNonce(sender),
		blockRoot: blocks[len(blocks)-1].Root(),
	}
}

// TestBothCommitmentsRunTheSameChain is the check that the binary tree is
// optional.
//
// EIP-8297 is a commitment change, so Amsterdam has to work on either backend:
// a node choosing the merkle-patricia trie must execute the same transactions
// to the same balances and nonces, and only the state root - the thing the
// commitment decides - may differ.
//
// Nothing pinned this before. Every binary-tree test ran the tree, and every
// merkle-patricia test ran a different scenario, so "the tree is opt-in" rested
// on the two never being compared.
func TestBothCommitmentsRunTheSameChain(t *testing.T) {
	tree := runParityChain(t, true, nil)
	mpt := runParityChain(t, false, nil)

	if tree.balance.Cmp(mpt.balance) != 0 {
		t.Fatalf("recipient balance differs by commitment: tree %v, trie %v", tree.balance, mpt.balance)
	}
	if tree.nonce != mpt.nonce {
		t.Fatalf("sender nonce differs by commitment: tree %d, trie %d", tree.nonce, mpt.nonce)
	}
	// The roots must differ: identical roots would mean one arm did not use the
	// commitment it was asked for.
	if tree.root == mpt.root {
		t.Fatalf("both commitments produced root %x; the backends are not distinct", tree.root)
	}
	if tree.root != tree.blockRoot {
		t.Fatalf("binary tree head root %x does not match the generated block %x", tree.root, tree.blockRoot)
	}
	if mpt.root != mpt.blockRoot {
		t.Fatalf("merkle-patricia head root %x does not match the generated block %x", mpt.root, mpt.blockRoot)
	}
}

// TestBothCommitmentsRunOnBogota repeats the check one fork later.
//
// The tree is defined from Amsterdam onwards and is absent from both Rules and
// the fork-ordering list, so it should compose with a later fork without anyone
// teaching it about that fork. Bogota is the first chance to test that claim,
// and no Bogota chain was exercised anywhere in the tree before this.
func TestBothCommitmentsRunOnBogota(t *testing.T) {
	bogota := func(c *params.ChainConfig) { c.BogotaTime = u64(0) }

	tree := runParityChain(t, true, bogota)
	mpt := runParityChain(t, false, bogota)

	if tree.balance.Cmp(mpt.balance) != 0 {
		t.Fatalf("recipient balance differs by commitment on bogota: tree %v, trie %v", tree.balance, mpt.balance)
	}
	if tree.nonce != mpt.nonce {
		t.Fatalf("sender nonce differs by commitment on bogota: tree %d, trie %d", tree.nonce, mpt.nonce)
	}
	if tree.root == mpt.root {
		t.Fatalf("both commitments produced root %x on bogota; the backends are not distinct", tree.root)
	}
}
