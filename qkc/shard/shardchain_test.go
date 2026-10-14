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

package shard

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	qkcrawdb "github.com/ethereum/go-ethereum/qkc/rawdb"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
)

// noVerifyEngine is the beacon engine without header verification, as the
// beacon engine rejects the QuarkChain extra data.
type noVerifyEngine struct {
	consensus.Engine
}

func (noVerifyEngine) VerifyHeader(consensus.ChainHeaderReader, *types.Header) error {
	return nil
}

func (noVerifyEngine) VerifyHeaders(_ consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	results := make(chan error, len(headers))
	for range headers {
		results <- nil
	}
	return make(chan struct{}), results
}

// testEnv is a shard chain on top of a root genesis block, which is the root
// tip until a test sets another one.
type testEnv struct {
	t      *testing.T
	engine consensus.Engine
	config *params.ChainConfig
	genDb  ethdb.Database // Database to generate the blocks with
	db     ethdb.Database // Database of the chain and the root blocks
	chain  *core.BlockChain
	sc     *ShardChain

	rootGenesis *qkctypes.RootBlock
	genesis     *types.Block
}

func newTestEnv(t *testing.T) *testEnv {
	env := &testEnv{
		t:      t,
		engine: noVerifyEngine{beacon.New(ethash.NewFaker())},
		config: params.MergedTestChainConfig,
		db:     rawdb.NewMemoryDatabase(),
		rootGenesis: qkctypes.NewRootBlockWithHeader(&qkctypes.RootBlockHeader{
			Difficulty:      big.NewInt(1),
			TotalDifficulty: big.NewInt(1),
		}),
	}
	env.addRoot(env.rootGenesis, nil)
	env.setRootTip(env.rootGenesis)

	gspec := &core.Genesis{
		Config:     env.config,
		Difficulty: common.Big0,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		ExtraData:  qkcExtra(env.rootGenesis),
		Alloc:      core.SystemContractAllocs(),
	}
	env.genDb, _, _ = core.GenerateChainWithGenesis(gspec, env.engine, 0, nil)

	chain, err := core.NewBlockChain(env.db, gspec, env.engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	t.Cleanup(chain.Stop)
	env.chain = chain
	env.genesis = chain.Genesis()
	env.sc = NewShardChain(env.db, chain)
	return env
}

// qkcExtra returns the extra data of a shard block referencing the given root
// block.
func qkcExtra(prevRoot *qkctypes.RootBlock) []byte {
	return (&types.QKCExtra{PrevRootBlock: prevRoot.Hash()}).Encode()
}

// newRoot creates a root block on top of parent. Blocks with different seeds
// are different.
func newRoot(parent *qkctypes.RootBlock, seed uint64) *qkctypes.RootBlock {
	return parent.Header().CreateBlockToAppend(nil, nil, nil, &seed, nil)
}

// addRoot stores a root block, along with the latest shard block it confirms
// if any.
func (env *testEnv) addRoot(root *qkctypes.RootBlock, confirmed *types.Block) {
	qkcrawdb.WriteRootBlock(env.db, root)
	if confirmed != nil {
		qkcrawdb.WriteLastConfirmedBlockHash(env.db, root.Hash(), confirmed.Hash())
	}
}

func (env *testEnv) setRootTip(root *qkctypes.RootBlock) {
	qkcrawdb.WriteRootHeadHash(env.db, root.Hash())
}

// makeBlocks creates n shard blocks on top of parent, referencing prevRoot.
// Blocks with different seeds are different.
func (env *testEnv) makeBlocks(parent *types.Block, n int, prevRoot *qkctypes.RootBlock, seed byte) []*types.Block {
	blocks, _ := core.GenerateChain(env.config, parent, env.engine, env.genDb, n, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{seed})
		b.SetExtra(qkcExtra(prevRoot))
	})
	return blocks
}

func (env *testEnv) insert(block *types.Block) {
	env.t.Helper()
	if err := env.sc.InsertShardBlock(block); err != nil {
		env.t.Fatalf("failed to insert block #%d: %v", block.NumberU64(), err)
	}
}

func (env *testEnv) checkHead(want *types.Block) {
	env.t.Helper()
	if head := env.chain.CurrentBlock(); head.Hash() != want.Hash() {
		env.t.Fatalf("head mismatch: have #%d [%x..], want #%d [%x..]",
			head.Number, head.Hash().Bytes()[:4], want.NumberU64(), want.Hash().Bytes()[:4])
	}
}

// The tests below come with diagrams in the style of the ones of ShardChain.
// They draw the root chain above the shard chain, with arrows from each block
// to its parent:
//
//   - r0, r1, ... are root blocks, and * marks the root tip. r1b is another
//     root block at the height of r1.
//   - g is the shard genesis. The other shard blocks are named by fork and
//     height: b2 is the block of fork b at height 2, b[1] in the code.
//   - x[r] is a shard block x referencing the root block r.
//   - A dotted arrow from a root block down to a shard block means the root
//     chain confirms the shard chain up to that block.

// TestInsertShardBlockExtendsHead checks that blocks extending the head become
// the head, and that known blocks are ignored. Fork a is blocks in the code.
//
//	root chain    r0*
//
//	shard chain   g[r0] <-- a1[r0] <-- a2[r0] <-- a3[r0]
//
//	a1, a2 and a3 become the head in turn. Inserting a1 again keeps a3.
func TestInsertShardBlockExtendsHead(t *testing.T) {
	env := newTestEnv(t)
	blocks := env.makeBlocks(env.genesis, 3, env.rootGenesis, 1)
	for _, block := range blocks {
		env.insert(block)
		env.checkHead(block)
	}
	// Known blocks are ignored.
	env.insert(blocks[0])
	env.checkHead(blocks[2])
}

// TestInsertShardBlockUnknownParent checks that a block whose parent is missing
// is rejected. Fork a is blocks in the code.
//
//	root chain    r0*
//
//	shard chain   g[r0] <-- (a1[r0]) <-- a2[r0]
//
//	a1 is not inserted, so inserting a2 fails.
func TestInsertShardBlockUnknownParent(t *testing.T) {
	env := newTestEnv(t)
	blocks := env.makeBlocks(env.genesis, 2, env.rootGenesis, 1)
	if err := env.sc.InsertShardBlock(blocks[1]); !errors.Is(err, consensus.ErrUnknownAncestor) {
		t.Fatalf("error mismatch: have %v, want %v", err, consensus.ErrUnknownAncestor)
	}
}

// TestInsertShardBlockRootChainUninitialized checks that no block is accepted
// before the root tip is known. Fork a is blocks in the code.
//
//	root chain    r0              no root tip
//
//	shard chain   g[r0] <-- a1[r0]
//
//	Inserting a1 fails.
func TestInsertShardBlockRootChainUninitialized(t *testing.T) {
	env := newTestEnv(t)
	qkcrawdb.WriteRootHeadHash(env.db, common.Hash{})
	blocks := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)
	if err := env.sc.InsertShardBlock(blocks[0]); !errors.Is(err, ErrRootChainUninitialized) {
		t.Fatalf("error mismatch: have %v, want %v", err, ErrRootChainUninitialized)
	}
}

// TestInsertShardBlockRootReference checks the root reference rules of
// validateRootReference, on top of the chains below. Each invalid block x is
// rejected, and not inserted.
//
//	              +----- r1b
//	              v
//	root chain    r0 <-- r1 <-- r2* <-- r3
//	                            :       :
//	                            v       v
//	shard chain   g[r0] <------ a1[r1]  ?
//
//	r2 confirms a1, the head. r3 confirms ?, a block missing from the shard
//	chain.
func TestInsertShardBlockRootReference(t *testing.T) {
	env := newTestEnv(t)

	r0 := env.rootGenesis
	r1, r1b := newRoot(r0, 1), newRoot(r0, 2)
	r2 := newRoot(r1, 1)
	r3 := newRoot(r2, 1)
	a1 := env.makeBlocks(env.genesis, 1, r1, 1)[0]
	env.addRoot(r1, nil)
	env.addRoot(r1b, nil)
	env.addRoot(r2, a1)
	env.setRootTip(r2)
	env.insert(a1)
	env.checkHead(a1)

	qkcrawdb.WriteRootBlock(env.db, r3)
	qkcrawdb.WriteLastConfirmedBlockHash(env.db, r3.Hash(), common.Hash{0xff})

	unknown := newRoot(r1, 2)
	tests := []struct {
		name  string
		block *types.Block
		err   error
	}{
		// g[r0] <-- a1[r1] <-- x[u]
		//
		// u, a root block on top of r1, is not stored.
		{"unknown root", env.makeBlocks(a1, 1, unknown, 1)[0], ErrUnknownRootBlock},

		// g[r0] <-- a1[r1] <-- x[r0]
		//
		// r0 is older than r1.
		{"older root", env.makeBlocks(a1, 1, r0, 1)[0], ErrRootNotDescendant},

		// g[r0] <-- a1[r1] <-- x[r1b]
		//
		// r1b is on another root chain than r1.
		{"forked root", env.makeBlocks(a1, 1, r1b, 1)[0], ErrRootNotDescendant},

		// g[r0] <-- a1[r1]
		// ^
		// +-------- x[r2]
		//
		// r2 extends r0, but confirms a1, which x does not extend.
		{"not extending confirmed", env.makeBlocks(env.genesis, 1, r2, 2)[0], ErrNotConfirmedDescendant},

		// g[r0] <-- a1[r1] <-- x[r3]
		//
		// r3 extends r1, but confirms ?, which is missing from the shard chain.
		{"unknown confirmed", env.makeBlocks(a1, 1, r3, 1)[0], ErrUnknownConfirmedBlock},
	}
	for _, test := range tests {
		if err := env.sc.InsertShardBlock(test.block); !errors.Is(err, test.err) {
			t.Errorf("%s: error mismatch: have %v, want %v", test.name, err, test.err)
		}
		if env.chain.HasBlock(test.block.Hash(), test.block.NumberU64()) {
			t.Errorf("%s: invalid block inserted", test.name)
		}
	}
	env.checkHead(a1)

	// g[r0] <-- a1[r1] <-- a2[r2]
	//
	// a2 references r2, which extends r1, and a2 extends a1, which r2 confirms.
	// It is valid, and becomes the head.
	a2 := env.makeBlocks(a1, 1, r2, 1)[0]
	env.insert(a2)
	env.checkHead(a2)
}

// TestShouldUpdateHeadRootNotCanonical checks that a block referencing a root
// block off the root tip chain does not become the head (rule 1). Block a1 is
// block in the code.
//
//	              +----- r1b
//	              v
//	root chain    r0 <-- r1*
//
//	shard chain   g[r0] <-- a1[r1b]
//
//	a1 is valid, as r1b extends r0, so it is inserted. But r1b is not on the
//	chain of the root tip r1, so the head stays g.
func TestShouldUpdateHeadRootNotCanonical(t *testing.T) {
	env := newTestEnv(t)
	r1, r1b := newRoot(env.rootGenesis, 1), newRoot(env.rootGenesis, 2)
	env.addRoot(r1, nil)
	env.addRoot(r1b, nil)
	env.setRootTip(r1)

	block := env.makeBlocks(env.genesis, 1, r1b, 1)[0]
	env.insert(block)
	if !env.chain.HasBlock(block.Hash(), block.NumberU64()) {
		t.Fatal("block not inserted")
	}
	env.checkHead(env.genesis)
}

// TestShouldUpdateHeadLongerFork checks that a fork replaces the head once it
// is higher (rule 4).
//
//	root chain    r0*
//
//	shard chain   g[r0] <-- a1[r0]
//	              ^
//	              +-------- b1[r0] <-- b2[r0]
//
//	a1 becomes the head. b1, as high as a1 and referencing the same root
//	block, does not replace it. b2, higher, does.
func TestShouldUpdateHeadLongerFork(t *testing.T) {
	env := newTestEnv(t)
	a := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)
	b := env.makeBlocks(env.genesis, 2, env.rootGenesis, 2)

	env.insert(a[0])
	env.checkHead(a[0])

	// Same height and same root, the head stays.
	env.insert(b[0])
	env.checkHead(a[0])

	// Higher, the fork becomes the head.
	env.insert(b[1])
	env.checkHead(b[1])
}

// TestShouldUpdateHeadLaterRoot checks that, at the same height, a fork
// referencing a higher root block replaces the head (rule 4).
//
//	root chain    r0 <-- r1*
//
//	shard chain   g[r0] <-- a1[r0]
//	              ^
//	              +-------- b1[r1]
//
//	a1 becomes the head. b1, as high as a1 but referencing r1, higher than
//	r0, replaces it.
func TestShouldUpdateHeadLaterRoot(t *testing.T) {
	env := newTestEnv(t)
	r1 := newRoot(env.rootGenesis, 1)
	env.addRoot(r1, nil)
	env.setRootTip(r1)

	a := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)
	env.insert(a[0])
	env.checkHead(a[0])

	// Same height but a later root, the fork becomes the head.
	b := env.makeBlocks(env.genesis, 1, r1, 2)
	env.insert(b[0])
	env.checkHead(b[0])
}

// TestShouldUpdateHeadConfirmedBarrier checks that a fork not extending the
// block confirmed by the root tip never becomes the head (rule 3).
//
//	root chain    r0 <----- r1*
//	                        :
//	                        v
//	shard chain   g[r0] <-- a1[r0]
//	              ^
//	              +-------- b1[r0] <-- b2[r0] <-- b3[r0]
//
//	a1 becomes the head, then r1, confirming a1, becomes the root tip. None of
//	the blocks of fork b replaces a1, not even b2 and b3, higher than a1.
func TestShouldUpdateHeadConfirmedBarrier(t *testing.T) {
	env := newTestEnv(t)
	a := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)
	env.insert(a[0])

	r1 := newRoot(env.rootGenesis, 1)
	env.addRoot(r1, a[0])
	env.setRootTip(r1)

	b := env.makeBlocks(env.genesis, 3, env.rootGenesis, 2)
	for _, block := range b {
		env.insert(block)
		env.checkHead(a[0])
	}
}
