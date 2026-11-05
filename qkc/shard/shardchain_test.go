// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package shard

import (
	"context"
	"errors"
	"maps"
	"math/big"
	"strings"
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
	"github.com/holiman/uint256"
)

// noVerifyEngine skips header verification, as the beacon engine rejects the
// QuarkChain extra data. It hides consensus.SealVerifier, so imports of blocks
// built with the PoSW engine skip seal verification too.
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
	config := *params.NewTestQuarkChainConfig(10001, 10001, 10002)
	env := &testEnv{
		t:      t,
		engine: noVerifyEngine{beacon.New(ethash.NewFaker())},
		config: &config,
		db:     rawdb.NewMemoryDatabase(),
	}

	gspec := &core.Genesis{
		Config:     env.config,
		Difficulty: common.Big0,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		ExtraData:  (&types.QKCExtra{ChainID: big.NewInt(10001)}).Encode(),
		Alloc:      core.SystemContractAllocs(),
	}
	maps.Copy(gspec.Alloc, core.PoSWStakingContractAllocs(&config))
	env.genDb, _, _ = core.GenerateChainWithGenesis(gspec, env.engine, 0, nil)

	chain, err := core.NewBlockChain(env.db, gspec, env.engine, nil)
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	t.Cleanup(func() { env.chain.Stop() })
	env.chain = chain
	env.genesis = chain.Genesis()
	// G0 has an empty previous-root hash. Only after deriving G0 can R0
	// commit to its header, avoiding a circular hash dependency. Init must
	// store both R0 and its confirmation; the fixture does not pre-seed them.
	env.rootGenesis = qkctypes.NewRootBlock(&qkctypes.RootBlockHeader{
		Difficulty:      *uint256.NewInt(1),
		TotalDifficulty: *uint256.NewInt(1),
	}, qkctypes.ShardBlockHeaders{env.genesis.Header()}, nil)
	env.sc = NewShardChain(env.db, chain, nil)
	if err := env.sc.Init(env.rootGenesis); err != nil {
		t.Fatalf("failed to initialize shard chain: %v", err)
	}
	return env
}

// qkcExtra returns the extra data of a shard block referencing the given root
// block.
func qkcExtra(prevRoot *qkctypes.RootBlock) []byte {
	return (&types.QKCExtra{ChainID: big.NewInt(10001), PrevRootBlock: prevRoot.Hash()}).Encode()
}

// newRoot creates a root block on top of parent, carrying the given shard block
// headers. Blocks with different seeds are different.
func newRoot(parent *qkctypes.RootBlock, seed uint64, headers ...*types.Header) *qkctypes.RootBlock {
	block := parent.Header().CreateBlockToAppend(nil, nil, nil, &seed, nil)
	return qkctypes.NewRootBlock(block.Header(), headers, nil)
}

// makeShardHeaders creates a linked sequence of headers for a shard. Each
// header references the corresponding root block in prevRoots.
func makeShardHeaders(chainID int64, startBlockNumber int64, prevRoots ...*qkctypes.RootBlock) []*types.Header {
	headers := make([]*types.Header, len(prevRoots))
	for i, prevRoot := range prevRoots {
		headers[i] = &types.Header{
			ParentHash: common.Hash{},
			Number:     big.NewInt(startBlockNumber + int64(i)),
			Difficulty: common.Big0,
			Extra:      (&types.QKCExtra{ChainID: big.NewInt(chainID), PrevRootBlock: prevRoot.Hash()}).Encode(),
		}
		if i > 0 {
			headers[i].ParentHash = headers[i-1].Hash()
		}
	}
	return headers
}

// addRoot stores a root block with its non-nil confirmed checkpoint. A root
// without new local headers still inherits its parent's confirmed block.
func (env *testEnv) addRoot(root *qkctypes.RootBlock, confirmed *types.Block) {
	qkcrawdb.WriteRootBlock(env.db, root)
	qkcrawdb.WriteLastConfirmedBlockHash(env.db, root.Hash(), confirmed.Hash())
}

func (env *testEnv) setRootTip(root *qkctypes.RootBlock) {
	env.sc.rootTip = root.Header()
}

// insertRoot inserts a root block, and checks whether it became the root tip.
func (env *testEnv) insertRoot(root *qkctypes.RootBlock, wantSwitched bool) {
	env.t.Helper()
	switched, err := env.sc.InsertRootBlock(root)
	if err != nil {
		env.t.Fatalf("failed to insert root block #%d: %v", root.Number(), err)
	}
	if switched != wantSwitched {
		env.t.Fatalf("root block #%d switch mismatch: have %v, want %v", root.Number(), switched, wantSwitched)
	}
}

func (env *testEnv) checkRootTip(want *qkctypes.RootBlock) {
	env.t.Helper()
	if have := env.sc.rootTip; have == nil || have.Hash() != want.Hash() {
		env.t.Fatalf("root tip mismatch: have %v, want #%d [%x..]", have, want.Number(), want.Hash().Bytes()[:4])
	}
}

func (env *testEnv) checkLastConfirmed(root *qkctypes.RootBlock, want *types.Block) {
	env.t.Helper()
	if have := qkcrawdb.ReadLastConfirmedBlockHash(env.db, root.Hash()); have != want.Hash() {
		env.t.Fatalf("root block #%d last confirmed mismatch: have [%x..], want #%d [%x..]",
			root.Number(), have.Bytes()[:4], want.NumberU64(), want.Hash().Bytes()[:4])
	}
}

// makeBlocks creates n shard blocks on top of parent, referencing prevRoot.
// Blocks with different seeds are different.
func (env *testEnv) makeBlocks(parent *types.Block, n int, prevRoot *qkctypes.RootBlock, seed byte) []*types.Block {
	env.t.Helper()
	// Generate against the supplied root even when a test has not imported it.
	view := &testXShardChain{
		roots:    make(map[common.Hash]*qkctypes.RootBlock),
		deposits: make(map[common.Hash][]*types.DepositTx),
	}
	for root := prevRoot; root != nil; root = env.sc.RootBlock(root.ParentHash()) {
		view.roots[root.Hash()] = root
		for _, header := range root.ShardBlockHeaders() {
			view.deposits[header.Hash()] = env.sc.XShardDeposits(header.Hash())
		}
	}
	inbox := NewXShardInbox(env.config, view)
	blocks, _ := core.GenerateChain(env.config, parent, env.engine, env.genDb, n, func(i int, b *core.BlockGen) {
		b.SetCoinbase(common.Address{seed})
		extra := &types.QKCExtra{
			ChainID:        env.config.ChainID,
			PrevRootBlock:  prevRoot.Hash(),
			XShardGasLimit: b.Gas() / params.XShardGasLimitDivisor,
		}
		deposits, err := inbox.Deposits(b.PrevBlock(i-1).Header(), prevRoot.Hash())
		if err != nil {
			env.t.Fatal(err)
		}
		extra.Cursor = deposits.Cursor()
		for {
			deposit, err := deposits.Next()
			if err != nil {
				env.t.Fatal(err)
			}
			if deposit == nil {
				extra.Cursor = deposits.Cursor()
				break
			}
			if deposit.Gas > extra.XShardGasLimit-extra.XShardGasUsed {
				break
			}
			gas := b.Gas()
			b.AddTx(types.NewTx(deposit))
			extra.XShardGasUsed += gas - b.Gas()
			extra.Cursor = deposits.Cursor()
		}
		b.SetExtra(extra.Encode())
	})
	return blocks
}

func (env *testEnv) insert(block *types.Block) {
	env.t.Helper()
	if err := env.sc.InsertShardBlock(block); err != nil {
		env.t.Fatalf("failed to insert block #%d: %v", block.NumberU64(), err)
	}
}

// reopenChain replaces a stopped blockchain while keeping the shard's root store.
func (env *testEnv) reopenChain() {
	env.t.Helper()
	chain, err := core.NewBlockChain(env.db, nil, env.engine, nil)
	if err != nil {
		env.t.Fatalf("reopen blockchain: %v", err)
	}
	env.chain, env.sc.chain = chain, chain
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
//   - g is the shard genesis, with an empty previous-root hash. Other blocks
//     are named by fork and height: b2 is fork b at height 2, b[1] in the code.
//   - x[r] is a shard block x referencing the root block r.
//   - A dotted arrow from a root block down to a shard block means the root
//     chain confirms the shard chain up to that block.

// TestShardChainInitGenesis bootstraps from a root that is not in the local
// root store. G0 has an empty previous-root hash; R0 commits to G0 afterwards.
// Only the local genesis is compared: the remote header's height/hash are left
// to that shard. Bootstrap must support the first shard and root imports.
//
//	shard genesis G0[empty] --> hash(G0) --> R0 body commitment --> hash(R0)
//	first shard block A1[R0] extends G0 without a circular hash dependency.
func TestShardChainInitGenesis(t *testing.T) {
	env := newTestEnv(t)
	rootDB := rawdb.NewMemoryDatabase()
	t.Cleanup(func() { rootDB.Close() })
	remote := makeShardHeaders(10002, 99, env.rootGenesis)[0]
	qkcrawdb.WriteXShardDeposits(rootDB, remote.Hash(), nil)
	root := qkctypes.NewRootBlock(env.rootGenesis.Header(), qkctypes.ShardBlockHeaders{remote, env.genesis.Header()}, nil)
	env.sc = NewShardChain(rootDB, env.chain, nil)
	if err := env.sc.Init(root); err != nil {
		t.Fatalf("bootstrap genesis: %v", err)
	}
	stored := qkcrawdb.ReadRootBlock(rootDB, root.Hash())
	if stored == nil || len(stored.ShardBlockHeaders()) != 2 {
		t.Fatal("Init did not persist the complete genesis root")
	}
	if hash := qkcrawdb.ReadLastConfirmedBlockHash(rootDB, root.Hash()); hash != env.genesis.Hash() {
		t.Fatalf("genesis confirmation mismatch: %x", hash)
	}
	env.checkRootTip(root)
	env.checkHead(env.genesis)
	if err := env.sc.Init(root); !errors.Is(err, ErrRootChainAlreadyInitialized) {
		t.Fatalf("repeat genesis bootstrap: %v", err)
	}

	// The first block can see R0 even though its parent G0 sees EmptyHash.
	a1 := env.makeBlocks(env.genesis, 1, root, 1)[0]
	env.insert(a1)
	r1 := newRoot(root, 1, a1.Header())
	env.insertRoot(r1, true)
	if hash := qkcrawdb.ReadLastConfirmedBlockHash(rootDB, r1.Hash()); hash != a1.Hash() {
		t.Fatalf("first root confirmation mismatch: %x", hash)
	}
	env.checkRootTip(r1)
	env.checkHead(a1)
}

// TestShardChainInitGenesisValidation rejects malformed local genesis entries
// before storing any root data or publishing the root tip.
// All roots have a correct body commitment except the explicit commitment case.
func TestShardChainInitGenesisValidation(t *testing.T) {
	env := newTestEnv(t)
	rootDB := rawdb.NewMemoryDatabase()
	t.Cleanup(func() { rootDB.Close() })
	env.sc = NewShardChain(rootDB, env.chain, nil)
	for _, name := range []string{"missing local", "duplicate local", "non-genesis local", "hash mismatch", "nonempty previous root", "body commitment", "root parent"} {
		t.Run(name, func(t *testing.T) {
			header := env.genesis.Header()
			headers := qkctypes.ShardBlockHeaders{header}
			rootHeader := env.rootGenesis.Header()
			switch name {
			case "missing local":
				headers = makeShardHeaders(10002, 0, env.rootGenesis)
			case "duplicate local":
				headers = append(headers, header)
			case "non-genesis local":
				header.Number = big.NewInt(1)
			case "hash mismatch":
				header.Coinbase = common.Address{1}
			case "nonempty previous root":
				header.Extra = qkcExtra(env.rootGenesis)
			case "root parent":
				rootHeader.ParentHash = common.Hash{1}
			}
			root := qkctypes.NewRootBlock(rootHeader, headers, nil)
			if name == "body commitment" {
				root = root.WithBody(nil, nil)
			}
			if err := env.sc.Init(root); !errors.Is(err, ErrInvalidRootBlock) {
				t.Fatalf("invalid genesis root: %v", err)
			}
			if env.sc.rootTip != nil {
				t.Fatal("invalid genesis published initialization state")
			}
			if qkcrawdb.HasRootBlock(rootDB, root.Hash()) || qkcrawdb.ReadLastConfirmedBlockHash(rootDB, root.Hash()) != (common.Hash{}) {
				t.Fatal("invalid genesis persisted root data")
			}
			if env.chain.CurrentBlock().Hash() != env.genesis.Hash() {
				t.Fatal("invalid genesis changed the shard head")
			}
		})
	}
	// Validation failures leave the same instance available for a valid Init.
	if err := env.sc.Init(env.rootGenesis); err != nil {
		t.Fatalf("retry with valid genesis: %v", err)
	}
}

// TestShardChainInitGenesisRetry checks both sides of the storage/head boundary.
// Root and confirmation commit together. A failed head switch may leave both
// stored, but rootTip must remain unset until the head switch succeeds.
func TestShardChainInitGenesisRetry(t *testing.T) {
	for _, name := range []string{"root batch", "head switch", "stopped genesis"} {
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t)
			stopChain := name != "root batch"
			head := env.genesis
			if name != "stopped genesis" {
				head = env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)[0]
				env.insert(head)
			}
			rootDB := rawdb.NewMemoryDatabase()
			t.Cleanup(func() { rootDB.Close() })
			writeErr := errors.New("genesis batch failed")
			store := &failingRootDB{KeyValueStore: rootDB, err: writeErr}
			if !stopChain {
				store.failAt = 1
			} else {
				env.chain.Stop()
			}
			env.sc = NewShardChain(store, env.chain, nil)
			err := env.sc.Init(env.rootGenesis)
			if err == nil || (!stopChain && !errors.Is(err, writeErr)) {
				t.Fatalf("expected initialization failure: %v", err)
			}
			if env.sc.rootTip != nil {
				t.Fatal("failed genesis initialization published state")
			}
			if qkcrawdb.HasRootBlock(rootDB, env.rootGenesis.Hash()) != stopChain {
				t.Fatal("unexpected root persistence after initialization failure")
			}
			want := common.Hash{}
			if stopChain {
				want = env.genesis.Hash()
			}
			if hash := qkcrawdb.ReadLastConfirmedBlockHash(rootDB, env.rootGenesis.Hash()); hash != want {
				t.Fatalf("partial genesis checkpoint: %x, want %x", hash, want)
			}
			env.checkHead(head)
			if stopChain {
				env.reopenChain()
			}
			if err := env.sc.Init(env.rootGenesis); err != nil {
				t.Fatalf("retry genesis initialization: %v", err)
			}
			env.checkRootTip(env.rootGenesis)
			env.checkHead(env.genesis)
		})
	}
}

// TestShardChainInitHead checks that initialization restores exactly the stored
// confirmation, rather than retaining compatible but unconfirmed descendants.
//
//	root chain    r0 <-- r1*
//
//	shard chain   g     <-- a1[r0] <-- a2[r0]   previous head
//	              ^
//	              +-------- b1[r0]
//
// Before Init, the canonical chain is g -> a1 -> a2, with a2 as its head.
// b1 is a stored, noncanonical block on another fork extending g. In the code,
// a1 and a2 are a[0] and a[1]; b1 is the separate block generated from genesis.
// Each subtest uses a different last-confirmed checkpoint for r1:
//
//   - a1, a canonical ancestor: rewind from a2 to a1, dropping the compatible
//     but unconfirmed descendant a2.
//   - b1, a noncanonical fork: switch from fork a to fork b, making b1 the head.
//   - a2, the current head: keep a2; only initialize the root tip to r1.
//   - g, an explicit genesis confirmation: rewind from a2 to g. A missing
//     confirmation is rejected, as covered by TestShardChainInitMissingConfirmation.
//
// Import after recovery is covered by TestShardChainInitMissingConfirmation.
func TestShardChainInitHead(t *testing.T) {
	for _, target := range []string{"canonical confirmation", "fork confirmation", "current head", "genesis confirmation"} {
		t.Run(target, func(t *testing.T) {
			env := newTestEnv(t)
			a := env.makeBlocks(env.genesis, 2, env.rootGenesis, 1)
			b1 := env.makeBlocks(env.genesis, 1, env.rootGenesis, 2)[0]
			for _, block := range append(a, b1) {
				env.insert(block)
			}
			env.checkHead(a[1])

			var confirmed *types.Block
			var headers []*types.Header
			switch target {
			case "canonical confirmation":
				confirmed, headers = a[0], []*types.Header{a[0].Header()}
			case "fork confirmation":
				confirmed, headers = b1, []*types.Header{b1.Header()}
			case "current head":
				confirmed, headers = a[1], []*types.Header{a[0].Header(), a[1].Header()}
			case "genesis confirmation":
				confirmed = env.genesis
			}
			root := newRoot(env.rootGenesis, 1, headers...)
			env.addRoot(root, confirmed)

			// Simulate reconnect/restart with the same local DB and blockchain.
			// Initialization must use the stored checkpoint, not this input body.
			env.sc = NewShardChain(env.db, env.chain, nil)
			input := root.WithBody(nil, []byte("master tracking data"))
			if err := env.sc.Init(input); err != nil {
				t.Fatalf("initialize: %v", err)
			}
			env.checkHead(confirmed)
			env.checkRootTip(root)
		})
	}
}

// TestShardChainInitUnknownRoot rejects nil or unknown checkpoints without
// storing the input or changing either head. Retry is covered separately below.
func TestShardChainInitUnknownRoot(t *testing.T) {
	for _, nilInput := range []bool{false, true} {
		name := "unknown"
		if nilInput {
			name = "nil"
		}
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t)
			env.sc = NewShardChain(env.db, env.chain, nil)
			root := newRoot(env.rootGenesis, 1)
			input := root
			if nilInput {
				input = nil
			}
			err := env.sc.Init(input)
			if !errors.Is(err, ErrUnknownRootBlock) {
				t.Fatalf("unknown checkpoint: %v", err)
			}
			if !nilInput && !errors.Is(err, ErrClusterStateInconsistency) {
				t.Fatalf("missing cluster state inconsistency: %v", err)
			}
			if env.sc.rootTip != nil || qkcrawdb.HasRootBlock(env.db, root.Hash()) {
				t.Fatal("unknown checkpoint initialized the chain or was persisted")
			}
			env.checkHead(env.genesis)
		})
	}
}

// TestShardChainInitMissingConfirmation checks that a stored root without a
// confirmation is not a usable checkpoint. Init must not silently rewind to
// genesis, persist a fabricated confirmation, or enable either import path.
func TestShardChainInitMissingConfirmation(t *testing.T) {
	env := newTestEnv(t)
	blocks := env.makeBlocks(env.genesis, 2, env.rootGenesis, 1)
	for _, block := range blocks {
		env.insert(block)
	}
	root := newRoot(env.rootGenesis, 1, blocks[0].Header())
	qkcrawdb.WriteRootBlock(env.db, root) // The confirmation record is missing.
	env.sc = NewShardChain(env.db, env.chain, nil)
	if err := env.sc.Init(root); !errors.Is(err, ErrInvalidRootBlock) || errors.Is(err, ErrUnknownConfirmedBlock) || errors.Is(err, ErrShardStateInconsistency) {
		t.Fatalf("missing confirmation: %v", err)
	}
	if env.sc.rootTip != nil {
		t.Fatal("missing confirmation initialized the chain")
	}
	env.checkHead(blocks[1])
	if hash := qkcrawdb.ReadLastConfirmedBlockHash(env.db, root.Hash()); hash != (common.Hash{}) {
		t.Fatalf("Init fabricated a confirmation: %x", hash)
	}
	next := env.makeBlocks(blocks[0], 1, root, 2)[0]
	if err := env.sc.InsertShardBlock(next); !errors.Is(err, ErrRootChainUninitialized) {
		t.Fatalf("shard import after failed initialization: %v", err)
	}
	if switched, err := env.sc.InsertRootBlock(newRoot(root, 1)); switched || !errors.Is(err, ErrRootChainUninitialized) {
		t.Fatalf("root import after failed initialization: switched %v, error %v", switched, err)
	}

	// The sync/bootstrap layer supplies the real checkpoint. The same shard
	// instance can retry and must recover to a1, not to genesis or the old a2.
	qkcrawdb.WriteLastConfirmedBlockHash(env.db, root.Hash(), blocks[0].Hash())
	if err := env.sc.Init(root); err != nil {
		t.Fatalf("retry initialization: %v", err)
	}
	env.checkHead(blocks[0])
	env.checkRootTip(root)
	env.insert(next)
	env.insertRoot(newRoot(root, 1), true)
	env.checkHead(next)
}

// TestShardChainInitMissingConfirmedBlock checks partial local recovery: a root
// can be present while its confirmed header or body has not been restored yet.
// Init must remain retryable, and must not close the underlying blockchain.
func TestShardChainInitMissingConfirmedBlock(t *testing.T) {
	for _, headerOnly := range []bool{false, true} {
		name := "missing header"
		if headerOnly {
			name = "missing body"
		}
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t)
			block := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)[0]
			root := newRoot(env.rootGenesis, 1, block.Header())
			env.addRoot(root, block)
			if headerOnly {
				rawdb.WriteHeader(env.db, block.Header())
			}
			env.sc = NewShardChain(env.db, env.chain, nil)
			want := ErrUnknownConfirmedBlock
			if headerOnly {
				want = consensus.ErrUnknownAncestor
			}
			err := env.sc.Init(root)
			if !errors.Is(err, ErrShardStateInconsistency) || !errors.Is(err, want) || errors.Is(err, ErrInvalidRootBlock) {
				t.Fatalf("missing confirmed data: have %v, want %v", err, want)
			}
			if env.sc.rootTip != nil {
				t.Fatal("failed initialization published root tip")
			}
			env.checkHead(env.genesis)

			// Simulate the separate sync layer restoring and executing the
			// confirmed block. Core import must still work after the failure.
			if _, err := env.chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
				t.Fatalf("restore confirmed block: %v", err)
			}
			if err := env.sc.Init(root); err != nil {
				t.Fatalf("retry initialization: %v", err)
			}
			env.checkHead(block)
			env.checkRootTip(root)
		})
	}
}

// TestShardChainInitHeadFailure checks that a SetCanonical error does not mark
// initialization complete. An externally stopped blockchain must be reopened by
// the service before retrying; Init itself never stops a healthy blockchain.
func TestShardChainInitHeadFailure(t *testing.T) {
	env := newTestEnv(t)
	block := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)[0]
	env.insert(block)
	root := newRoot(env.rootGenesis, 1) // The stored genesis confirmation rewinds to g.
	env.addRoot(root, env.genesis)
	env.sc = NewShardChain(env.db, env.chain, nil)
	env.chain.Stop()
	if err := env.sc.Init(root); err == nil {
		t.Fatal("initialization succeeded despite stopped blockchain")
	}
	if env.sc.rootTip != nil {
		t.Fatal("failed head switch published root tip")
	}
	env.checkHead(block)

	env.reopenChain()
	if err := env.sc.Init(root); err != nil {
		t.Fatalf("retry initialization: %v", err)
	}
	env.checkHead(env.genesis)
	env.checkRootTip(root)
}

// TestShardChainInitConcurrent checks that two master notifications cannot both
// pass the uninitialized check: the entire initialization runs under c.mu.
func TestShardChainInitConcurrent(t *testing.T) {
	env := newTestEnv(t)
	env.sc = NewShardChain(env.db, env.chain, nil)
	root := env.rootGenesis
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			results <- env.sc.Init(root)
		}()
	}
	close(start)
	var successes, repeats int
	for range 2 {
		switch err := <-results; {
		case err == nil:
			successes++
		case errors.Is(err, ErrRootChainAlreadyInitialized):
			repeats++
		default:
			t.Fatalf("concurrent initialization: %v", err)
		}
	}
	if successes != 1 || repeats != 1 {
		t.Fatalf("initialization results: successes %d, repeats %d", successes, repeats)
	}
	env.checkRootTip(root)
	env.checkHead(env.genesis)
}

// TestInsertShardBlockExtendsHead checks that blocks extending the head become
// the head, and that known blocks are ignored. Fork a is blocks in the code.
//
//	root chain    r0*
//
//	shard chain   g     <-- a1[r0] <-- a2[r0] <-- a3[r0]
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
//	shard chain   g     <-- (a1[r0]) <-- a2[r0]
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
//	shard chain   g     <-- a1[r0]
//
//	Inserting a1 fails.
func TestInsertShardBlockRootChainUninitialized(t *testing.T) {
	env := newTestEnv(t)
	env.sc.rootTip = nil
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
//	shard chain   g     <------ a1[r1]  ?
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
	env.addRoot(r1, env.genesis)
	env.addRoot(r1b, env.genesis)
	env.addRoot(r2, a1)
	env.setRootTip(r2)
	env.insert(a1)
	env.checkHead(a1)

	qkcrawdb.WriteRootBlock(env.db, r3)
	qkcrawdb.WriteLastConfirmedBlockHash(env.db, r3.Hash(), common.Hash{0xff})

	unknown := newRoot(r1, 2)
	// Build on the valid root first, then corrupt the reference for rejection tests.
	valid := env.makeBlocks(a1, 1, r1, 1)[0]
	withRoot := func(root *qkctypes.RootBlock) *types.Block {
		header := valid.Header()
		extra, _ := header.QKCExtra()
		extra.PrevRootBlock = root.Hash()
		header.Extra = extra.Encode()
		return valid.WithSeal(header)
	}
	tests := []struct {
		name              string
		block             *types.Block
		err               error
		inconsistent      bool
		shardInconsistent bool
	}{
		// g <-- a1[r1] <-- x[u]
		//
		// u, a root block on top of r1, is not stored.
		{"unknown root", env.makeBlocks(a1, 1, unknown, 1)[0], ErrUnknownRootBlock, false, false},

		// g <-- a1[r1] <-- x[r0]
		//
		// r0 is older than r1.
		{"older root", withRoot(r0), ErrRootNotDescendant, false, false},

		// g <-- a1[r1] <-- x[r1b]
		//
		// r1b is on another root chain than r1.
		{"forked root", withRoot(r1b), ErrRootNotDescendant, false, false},

		// g <-- a1[r1]
		// ^
		// +-------- x[r2]
		//
		// r2 extends r0, but confirms a1, which x does not extend.
		{"not extending confirmed", env.makeBlocks(env.genesis, 1, r2, 2)[0], ErrNotConfirmedDescendant, false, false},

		// g <-- a1[r1] <-- x[r3]
		//
		// r3 extends r1, but confirms ?, which is missing from the shard chain.
		{"unknown confirmed", env.makeBlocks(a1, 1, r3, 1)[0], ErrUnknownConfirmedBlock, false, true},
	}
	for _, test := range tests {
		err := env.sc.InsertShardBlock(test.block)
		if !errors.Is(err, test.err) {
			t.Errorf("%s: error mismatch: have %v, want %v", test.name, err, test.err)
		}
		if have := errors.Is(err, ErrClusterStateInconsistency); have != test.inconsistent {
			t.Errorf("%s: cluster inconsistency mismatch: have %v, want %v", test.name, have, test.inconsistent)
		}
		if have := errors.Is(err, ErrShardStateInconsistency); have != test.shardInconsistent {
			t.Errorf("%s: shard inconsistency mismatch: have %v, want %v", test.name, have, test.shardInconsistent)
		}
		if test.err == ErrRootNotDescendant {
			// Bypassing ShardChain must still reject invalid inbox ancestry.
			if _, err := env.chain.InsertChain(types.Blocks{test.block}); !errors.Is(err, ErrRootNotDescendant) {
				t.Errorf("%s: direct InsertChain error %v, want %v", test.name, err, ErrRootNotDescendant)
			}
		}
		if env.chain.HasBlock(test.block.Hash(), test.block.NumberU64()) {
			t.Errorf("%s: invalid block inserted", test.name)
		}
	}
	env.checkHead(a1)

	// g <-- a1[r1] <-- a2[r2]
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
//	shard chain   g     <-- a1[r1b]
//
//	a1 is valid, as r1b extends r0, so it is inserted. But r1b is not on the
//	chain of the root tip r1, so the head stays g.
func TestShouldUpdateHeadRootNotCanonical(t *testing.T) {
	env := newTestEnv(t)
	r1, r1b := newRoot(env.rootGenesis, 1), newRoot(env.rootGenesis, 2)
	env.addRoot(r1, env.genesis)
	env.addRoot(r1b, env.genesis)
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
//	shard chain   g     <-- a1[r0]
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
//	shard chain   g     <-- a1[r0]
//	              ^
//	              +-------- b1[r1]
//
//	a1 becomes the head. b1, as high as a1 but referencing r1, higher than
//	r0, replaces it.
func TestShouldUpdateHeadLaterRoot(t *testing.T) {
	env := newTestEnv(t)
	r1 := newRoot(env.rootGenesis, 1)
	env.addRoot(r1, env.genesis)
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
//	shard chain   g     <-- a1[r0]
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

// TestInsertRootBlockConfirms checks that root blocks extending the root tip
// become the root tip, record the latest shard block they confirm, and keep
// the head, and that known root blocks are ignored. Fork a is a in the code.
//
//	root chain    r0 <----- r1 <------ r2 <-- r3*
//	                        :          :
//	                        v          v
//	shard chain   g     <-- a1[r0] <-- a2[r1] <-- a3[r1]
//
//	r1 confirms a1, r2 confirms a2, and r3 adds no local headers, so its last
//	confirmed block is a2 too. Inserting r1 again changes nothing.
func TestInsertRootBlockConfirms(t *testing.T) {
	env := newTestEnv(t)
	a := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)
	env.insert(a[0])

	r1 := newRoot(env.rootGenesis, 1, a[0].Header())
	env.insertRoot(r1, true)
	env.checkRootTip(r1)
	env.checkLastConfirmed(r1, a[0])
	env.checkHead(a[0])

	// Known root blocks are ignored.
	env.insertRoot(r1, false)
	env.checkRootTip(r1)

	a = append(a, env.makeBlocks(a[0], 2, r1, 1)...)
	env.insert(a[1])
	env.insert(a[2])

	r2 := newRoot(r1, 1, a[1].Header())
	env.insertRoot(r2, true)
	env.checkRootTip(r2)
	env.checkLastConfirmed(r2, a[1])
	env.checkHead(a[2])

	r3 := newRoot(r2, 1)
	env.insertRoot(r3, true)
	env.checkRootTip(r3)
	env.checkLastConfirmed(r3, a[1])
	env.checkHead(a[2])
}

// TestInsertRootBlockTotalDifficulty checks that a root block replaces the root
// tip only with a higher total difficulty, and is stored either way.
//
//	              +----- r1b, r1c*
//	              v
//	root chain    r0 <-- r1
//
//	r1 becomes the root tip. r1b, as heavy as r1, does not replace it, but
//	r1c, heavier, does.
func TestInsertRootBlockTotalDifficulty(t *testing.T) {
	env := newTestEnv(t)
	r1, r1b := newRoot(env.rootGenesis, 1), newRoot(env.rootGenesis, 2)
	env.insertRoot(r1, true)
	env.checkRootTip(r1)
	env.checkLastConfirmed(r1, env.genesis)

	// Same total difficulty, the root tip stays. An empty body still stores
	// the inherited confirmation, including for a root on another fork.
	env.insertRoot(r1b, false)
	env.checkRootTip(r1)
	env.checkLastConfirmed(r1b, env.genesis)
	if !qkcrawdb.HasRootBlock(env.db, r1b.Hash()) {
		t.Fatal("root block not stored")
	}
	// Higher total difficulty, the root block becomes the root tip.
	seed := uint64(3)
	r1c := env.rootGenesis.Header().CreateBlockToAppend(nil, uint256.NewInt(2), nil, &seed, nil)
	env.insertRoot(r1c, true)
	env.checkRootTip(r1c)
	env.checkLastConfirmed(r1c, env.genesis)
	env.checkHead(env.genesis)
}

// TestInsertRootBlockValidation checks the rules of validateRootBlock, on top
// of the chains below. Each invalid root block x is rejected, and not stored.
// This diagram shows the local shard 0 chain and root fork; a root block
// carries at most one block of the local shard.
//
//	              +----- r1b
//	              v
//	root chain    r0 <----- r1*
//	                        :
//	                        v
//	shard chain   g     <-- a1[r0] <-- a2[r0] <-- a3[r0]
//	                        ^
//	                        +-------- f2[r1b]
//
//	r1 confirms a1. The diagram shows only local shard 0. Separately, the
//	test passes remote headers to validateRootBlock to check cross-shard deposits.
//	The target is shard 0, shard 1 is in its network config, and r0 is the root
//	genesis containing the local shard genesis. Under these conditions:
//	  1. b1[r0] has deposits, so validation accepts it.
//	  2. b2[r0] has no deposits, so validation fails: deposits are required at r0.
//	  3. b3[r1] has deposits, so validation accepts it; b4[r1] has none, so
//	     validation fails because deposits are required at r1 too.
//	  4. t1[r1] is from shard 7, outside the network config, but has deposits;
//	     validation fails because shard 7 does not send deposits to shard 0.
func TestInsertRootBlockValidation(t *testing.T) {
	env := newTestEnv(t)
	r0 := env.rootGenesis
	blocks := env.makeBlocks(env.genesis, 3, r0, 1)
	for _, block := range blocks {
		env.insert(block)
	}
	r1, r1b := newRoot(r0, 1, blocks[0].Header()), newRoot(r0, 2)
	env.insertRoot(r1, true)
	env.insertRoot(r1b, false)
	aForkB2 := env.makeBlocks(blocks[0], 1, r1b, 2)[0]
	env.insert(aForkB2)

	bHeaders := makeShardHeaders(10002, 1, r0, r0, r1, r1)
	b1, b2, b3, b4 := bHeaders[0], bHeaders[1], bHeaders[2], bHeaders[3]
	t1 := makeShardHeaders(10008, 1, r1)[0]
	qkcrawdb.WriteXShardDeposits(env.db, b1.Hash(), nil)
	qkcrawdb.WriteXShardDeposits(env.db, b3.Hash(), nil)
	qkcrawdb.WriteXShardDeposits(env.db, t1.Hash(), nil)
	b1Extra, err := b1.QKCExtra()
	if err != nil {
		t.Fatal(err)
	}
	if err := env.sc.validateXShardDeposits(b1, b1Extra, env.config); err != nil {
		t.Fatalf("block referencing the genesis root with an empty deposit list: %v", err)
	}
	env.chain.Config().QuarkChain.MaxBlocksPerRootBlock = 1

	badVersion := newRoot(r1, 1).Header()
	badVersion.Version = 1
	tests := []struct {
		name         string
		root         *qkctypes.RootBlock
		err          error
		inconsistent bool
	}{
		// Rule 1.
		{"unknown parent", newRoot(newRoot(r1, 2), 1), ErrUnknownRootBlock, false},
		{"genesis height", qkctypes.NewRootBlockWithHeader(&qkctypes.RootBlockHeader{ParentHash: r0.Hash()}), ErrRootNotAfterGenesis, true},
		{"version", qkctypes.NewRootBlockWithHeader(badVersion), ErrRootBlockVersion, false},

		// Rule 2: x carries a2 of a fork missing from the shard chain, then
		// a2 and a3, more than one block of the shard.
		{"unknown block", newRoot(r1, 1, env.makeBlocks(blocks[0], 1, r1, 3)[0].Header()), ErrUnknownConfirmedBlock, true},
		{"too many blocks", newRoot(r1, 1, blocks[1].Header(), blocks[2].Header()), ErrTooManyConfirmedBlocks, false},

		// Rule 2: deposits are required for b2 at r0 and b4 at r1,
		// and unexpected for t1 because its shard is outside the network.
		{"missing deposits", newRoot(r1, 1, b4), ErrMissingXShardDeposits, true},
		{"missing deposits at genesis root", newRoot(r1, 1, b2), ErrMissingXShardDeposits, true},
		{"deposits from unknown shard", newRoot(r1, 1, t1), ErrUnexpectedXShardDeposits, true},

		// Rule 3: x carries a3, which is not a child of a1, then b2, which
		// is, but references r1b, not an ancestor of x.
		{"gap", newRoot(r1, 1, blocks[2].Header()), ErrConfirmedNotContiguous, true},
		{"regression to genesis", newRoot(r1, 1, env.genesis.Header()), ErrConfirmedNotContiguous, true},
		{"confirmed root fork", newRoot(r1, 1, aForkB2.Header()), ErrConfirmedRootNotAncestor, true},
	}
	for _, test := range tests {
		_, err := env.sc.InsertRootBlock(test.root)
		if !errors.Is(err, test.err) {
			t.Errorf("%s: error mismatch: have %v, want %v", test.name, err, test.err)
		}
		if have := errors.Is(err, ErrClusterStateInconsistency); have != test.inconsistent {
			t.Errorf("%s: cluster inconsistency mismatch: have %v, want %v", test.name, have, test.inconsistent)
		}
		if test.inconsistent && err != nil && !strings.Contains(err.Error(), ErrClusterStateInconsistency.Error()) {
			t.Errorf("%s: cluster inconsistency missing from log message: %v", test.name, err)
		}
		if errors.Is(err, ErrShardStateInconsistency) {
			t.Errorf("%s: input validation error incorrectly marked as shard inconsistency: %v", test.name, err)
		}
		if qkcrawdb.HasRootBlock(env.db, test.root.Hash()) {
			t.Errorf("%s: invalid root block stored", test.name)
		}
	}
	env.checkRootTip(r1)
	env.checkHead(blocks[2])

	// r2 carries a2, b1, b2 and b3. All remote blocks have deposit lists;
	// b1 and b2 reference r0, while b3 references r1.
	qkcrawdb.WriteXShardDeposits(env.db, b2.Hash(), nil)
	r2 := newRoot(r1, 1, blocks[1].Header(), b1, b2, b3)
	env.insertRoot(r2, true)
	env.checkRootTip(r2)
	env.checkLastConfirmed(r2, blocks[1])
	env.checkHead(blocks[2])
}

func TestValidateXShardDeposits(t *testing.T) {
	r0 := qkctypes.NewRootBlock(&qkctypes.RootBlockHeader{}, nil, nil)
	r1 := newRoot(r0, 1)
	unknown := newRoot(r0, 2)
	config := params.NewTestQuarkChainConfig(10001, 10001, 10002)
	tests := []struct {
		name    string
		chainID int64
		root    *qkctypes.RootBlock
		stored  bool
		err     error
	}{
		{"unknown root without deposits", 10002, unknown, false, ErrUnknownRootBlock},
		{"unknown root with deposits", 10002, unknown, true, ErrUnknownRootBlock},
		{"genesis root without deposits", 10002, r0, false, ErrMissingXShardDeposits},
		{"genesis root with empty deposits", 10002, r0, true, nil},
		{"later root without deposits", 10002, r1, false, ErrMissingXShardDeposits},
		{"later root with empty deposits", 10002, r1, true, nil},
		{"unknown shard without deposits", 10008, r1, false, nil},
		{"unknown shard with deposits", 10008, r1, true, ErrUnexpectedXShardDeposits},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := rawdb.NewMemoryDatabase()
			t.Cleanup(func() { db.Close() })
			qkcrawdb.WriteRootBlock(db, r0)
			qkcrawdb.WriteRootBlock(db, r1)
			block := makeShardHeaders(test.chainID, 1, test.root)[0]
			extra, err := block.QKCExtra()
			if err != nil {
				t.Fatal(err)
			}
			if test.stored {
				qkcrawdb.WriteXShardDeposits(db, block.Hash(), nil)
			}
			chain := &ShardChain{db: db}
			err = chain.validateXShardDeposits(block, extra, config)
			if !errors.Is(err, test.err) {
				t.Fatalf("error mismatch: have %v, want %v", err, test.err)
			}
			if errors.Is(err, ErrClusterStateInconsistency) != (test.err != nil) {
				t.Fatalf("cluster inconsistency classification mismatch: %v", err)
			}
			if test.err == ErrUnknownRootBlock && !strings.Contains(err.Error(), test.root.Hash().Hex()[2:]) {
				t.Fatalf("missing root hash in error: %v", err)
			}
		})
	}
}

// The cluster must make local shard blocks available before a root confirms
// them. Missing blocks violate this guarantee, rather than local stored state.
func TestInsertRootBlockMissingLocalBlock(t *testing.T) {
	env := newTestEnv(t)
	missing := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)[0]
	root := newRoot(env.rootGenesis, 1, missing.Header())
	switched, err := env.sc.InsertRootBlock(root)
	if switched || !errors.Is(err, ErrClusterStateInconsistency) || !errors.Is(err, ErrUnknownConfirmedBlock) || errors.Is(err, ErrShardStateInconsistency) {
		t.Fatalf("missing confirmed local block: switched %v, error %v", switched, err)
	}
	if !strings.Contains(err.Error(), ErrClusterStateInconsistency.Error()) {
		t.Fatalf("cluster inconsistency missing from log message: %v", err)
	}
	if qkcrawdb.HasRootBlock(env.db, root.Hash()) {
		t.Fatal("root stored despite missing confirmed local block")
	}
	env.checkRootTip(env.rootGenesis)
	env.checkHead(env.genesis)
}

// TestInsertRootBlockConfirmsFork checks that the head moves to the latest
// block confirmed by the new root tip when it is on a fork of the head (case 1
// of setHeadForRootTip).
//
//	root chain    r0 <----- r1 <------ r2*
//	                        :          :
//	                        v          v
//	shard chain   g     <-- a1[r0] <-- b2[r0] <-- b3[r0]
//	                        ^
//	                        +-------- a2[r0] <-- a3[r0]
//
//	a3 is the head. r1 confirms a1 and keeps a3. r2 confirms b2, so
//	the head moves to b2, not to b3.
func TestInsertRootBlockConfirmsFork(t *testing.T) {
	env := newTestEnv(t)
	a := env.makeBlocks(env.genesis, 3, env.rootGenesis, 1)
	b := env.makeBlocks(a[0], 2, env.rootGenesis, 2)
	for _, block := range append(a, b...) {
		env.insert(block)
	}
	env.checkHead(a[2])

	r1 := newRoot(env.rootGenesis, 1, a[0].Header())
	env.insertRoot(r1, true)
	env.checkHead(a[2])

	r2 := newRoot(r1, 1, b[0].Header())
	env.insertRoot(r2, true)
	env.checkRootTip(r2)
	env.checkHead(b[0])

	// Re-inserting the known root tip does not trigger another switch.
	env.insertRoot(r2, false)
	env.checkRootTip(r2)
	env.checkHead(b[0])
}

// Missing local blocks for recorded confirmations are shard inconsistencies,
// not cluster check failures. Fork choice must preserve the category and cause.
func TestInsertShardBlockForkChoiceError(t *testing.T) {
	env := newTestEnv(t)
	blocks := env.makeBlocks(env.genesis, 2, env.rootGenesis, 1)
	for _, block := range blocks {
		env.insert(block)
	}
	root := newRoot(env.rootGenesis, 1)
	env.insertRoot(root, true)
	qkcrawdb.WriteLastConfirmedBlockHash(env.db, root.Hash(), common.Hash{0xff})

	fork := env.makeBlocks(env.genesis, 1, env.rootGenesis, 2)[0]
	err := env.sc.InsertShardBlock(fork)
	if !errors.Is(err, ErrShardStateInconsistency) || !errors.Is(err, ErrUnknownConfirmedBlock) || errors.Is(err, ErrClusterStateInconsistency) {
		t.Fatalf("missing fork-choice confirmation error mismatch: %v", err)
	}
	if !strings.Contains(err.Error(), ErrShardStateInconsistency.Error()) {
		t.Fatalf("shard inconsistency missing from log message: %v", err)
	}
	if env.chain.HasBlock(fork.Hash(), fork.NumberU64()) {
		t.Fatal("block inserted despite fork-choice inconsistency")
	}
	// Known roots are ignored without rereading their stored confirmation.
	env.insertRoot(root, false)
	env.checkRootTip(root)
	env.checkHead(blocks[1])
}

// TestShardChainMissingConfirmation checks the stored-data contract after
// initialization. Every path that reads a confirmation rejects a missing one;
// this is distinct from a recorded confirmation whose local block is missing.
func TestShardChainMissingConfirmation(t *testing.T) {
	for _, flow := range []string{"root reference", "fork choice", "root parent"} {
		t.Run(flow, func(t *testing.T) {
			env := newTestEnv(t)
			blocks := env.makeBlocks(env.genesis, 2, env.rootGenesis, 1)
			for _, block := range blocks {
				env.insert(block)
			}
			root := newRoot(env.rootGenesis, 1)
			env.insertRoot(root, true)
			qkcrawdb.WriteLastConfirmedBlockHash(env.db, root.Hash(), common.Hash{})

			var err error
			switch flow {
			case "root reference", "fork choice":
				parent, prevRoot := blocks[1], root
				if flow == "fork choice" {
					parent, prevRoot = env.genesis, env.rootGenesis
				}
				block := env.makeBlocks(parent, 1, prevRoot, 2)[0]
				err = env.sc.InsertShardBlock(block)
				if env.chain.HasBlock(block.Hash(), block.NumberU64()) {
					t.Fatal("block imported despite missing confirmation")
				}
			case "root parent":
				nextRoot := newRoot(root, 1)
				_, err = env.sc.InsertRootBlock(nextRoot)
				if qkcrawdb.HasRootBlock(env.db, nextRoot.Hash()) {
					t.Fatal("root stored despite missing parent confirmation")
				}
			}
			if !errors.Is(err, ErrInvalidRootBlock) || errors.Is(err, ErrUnknownConfirmedBlock) || errors.Is(err, ErrShardStateInconsistency) || errors.Is(err, ErrClusterStateInconsistency) {
				t.Fatalf("missing stored confirmation: %v", err)
			}
			env.checkRootTip(root)
			env.checkHead(blocks[1])
		})
	}
}

// Missing stored references are shard inconsistencies, not pyquarkchain check
// failures, and must not carry the cluster inconsistency marker.
func TestInsertShardBlockStoredReferenceError(t *testing.T) {
	env := newTestEnv(t)
	root := newRoot(env.rootGenesis, 1)
	env.addRoot(root, env.genesis)
	parent := env.makeBlocks(env.genesis, 1, root, 1)[0]
	env.insert(parent)
	block := env.makeBlocks(parent, 1, root, 1)[0]
	header := block.Header()
	extra, _ := header.QKCExtra()
	extra.PrevRootBlock = env.rootGenesis.Hash()
	header.Extra = extra.Encode()
	block = block.WithSeal(header)

	// Simulate loss of root data while keeping the shard blockchain intact.
	rootDB := rawdb.NewMemoryDatabase()
	t.Cleanup(func() { rootDB.Close() })
	qkcrawdb.WriteRootBlock(rootDB, env.rootGenesis)
	qkcrawdb.WriteLastConfirmedBlockHash(rootDB, env.rootGenesis.Hash(), env.genesis.Hash())
	env.sc.db = rootDB
	if err := env.sc.InsertShardBlock(block); !errors.Is(err, ErrShardStateInconsistency) || !errors.Is(err, ErrUnknownRootBlock) || errors.Is(err, ErrClusterStateInconsistency) {
		t.Fatalf("missing stored parent reference error mismatch: %v", err)
	}
	if env.chain.HasBlock(block.Hash(), block.NumberU64()) {
		t.Fatal("block inserted despite missing stored parent reference")
	}
}

// At equal shard height, fork choice reads the current head's root reference.
// That reference was checked on import, so losing it is a shard inconsistency.
func TestInsertShardBlockCurrentRootError(t *testing.T) {
	env := newTestEnv(t)
	root := newRoot(env.rootGenesis, 1)
	env.addRoot(root, env.genesis)
	env.setRootTip(root)
	head := env.makeBlocks(env.genesis, 1, root, 1)[0]
	env.insert(head)

	rootDB := rawdb.NewMemoryDatabase()
	t.Cleanup(func() { rootDB.Close() })
	qkcrawdb.WriteRootBlock(rootDB, env.rootGenesis)
	qkcrawdb.WriteLastConfirmedBlockHash(rootDB, env.rootGenesis.Hash(), env.genesis.Hash())
	env.sc.db = rootDB
	env.setRootTip(env.rootGenesis)
	fork := env.makeBlocks(env.genesis, 1, env.rootGenesis, 2)[0]
	if err := env.sc.InsertShardBlock(fork); !errors.Is(err, ErrShardStateInconsistency) || !errors.Is(err, ErrUnknownRootBlock) || errors.Is(err, ErrClusterStateInconsistency) {
		t.Fatalf("missing current head root reference error mismatch: %v", err)
	}
	if env.chain.HasBlock(fork.Hash(), fork.NumberU64()) {
		t.Fatal("block inserted despite missing current head root reference")
	}
	env.checkHead(head)
}

// The descendant helpers return false when ancestry is unavailable.
func TestShardChainMissingAncestry(t *testing.T) {
	t.Run("root", func(t *testing.T) {
		env := newTestEnv(t)
		parent := newRoot(env.rootGenesis, 1)
		root := newRoot(parent, 1)
		env.addRoot(root, env.genesis) // The stored root's parent is missing.
		if env.sc.isRootDescendant(root.Header(), env.rootGenesis.Header()) {
			t.Fatal("root reported as descendant despite missing ancestry")
		}
	})
	t.Run("shard", func(t *testing.T) {
		env := newTestEnv(t)
		blocks := env.makeBlocks(env.genesis, 2, env.rootGenesis, 1)
		rawdb.WriteBlock(env.db, blocks[1]) // The stored block's parent is missing.
		if env.sc.isShardDescendant(blocks[1].Header(), env.genesis.Header()) {
			t.Fatal("shard block reported as descendant despite missing ancestry")
		}
	})
}

// TestShardChainUnconfirmedHeaders checks that queries return the earliest
// pending canonical headers in ascending order under each configured limit.
// A root may inherit its confirmation; genesis and fully confirmed blocks must
// never appear in the pending list.
func TestShardChainUnconfirmedHeaders(t *testing.T) {
	env := newTestEnv(t)
	if headers, err := env.sc.GetUnconfirmedHeaderList(); err != nil || len(headers) != 0 {
		t.Fatalf("genesis pending headers: %v, %v", headers, err)
	}
	blocks := env.makeBlocks(env.genesis, 5, env.rootGenesis, 1)
	for _, block := range blocks {
		env.insert(block)
	}
	root := newRoot(env.rootGenesis, 1, blocks[0].Header(), blocks[1].Header())
	env.insertRoot(root, true)
	// A root with no new local headers retains its parent's confirmation.
	root = newRoot(root, 1)
	env.insertRoot(root, true)
	t.Run("parent hash traversal", func(t *testing.T) {
		limit := env.chain.Config().QuarkChain.MaxBlocksPerRootBlock
		env.chain.Config().QuarkChain.MaxBlocksPerRootBlock = 2
		defer func() { env.chain.Config().QuarkChain.MaxBlocksPerRootBlock = limit }()
		// Only the endpoint uses its height index; an interior index must not
		// redirect the parent-hash traversal onto a stored fork.
		fork := env.makeBlocks(blocks[1], 1, env.rootGenesis, 2)[0]
		rawdb.WriteBlock(env.db, fork)
		rawdb.WriteCanonicalHash(env.db, fork.Hash(), fork.NumberU64())
		defer rawdb.WriteCanonicalHash(env.db, blocks[2].Hash(), blocks[2].NumberU64())
		headers, err := env.sc.GetUnconfirmedHeaderList()
		if err != nil || len(headers) != 2 || headers[0].Hash() != blocks[2].Hash() {
			t.Fatalf("headers must follow the head's parent hashes: %v, error %v", headers, err)
		}
		// A fork cannot become a valid confirmation just because its height index matches.
		qkcrawdb.WriteLastConfirmedBlockHash(env.db, root.Hash(), fork.Hash())
		defer qkcrawdb.WriteLastConfirmedBlockHash(env.db, root.Hash(), blocks[1].Hash())
		if headers, err := env.sc.GetUnconfirmedHeaderList(); !errors.Is(err, ErrShardStateInconsistency) || headers != nil {
			t.Fatalf("confirmation off the head's ancestry: %v, error %v", headers, err)
		}
	})
	t.Run("missing canonical endpoint", func(t *testing.T) {
		// The limited range ends at height four; a missing canonical endpoint
		// is a local inconsistency rather than an empty result or a fallback fork.
		limit := env.chain.Config().QuarkChain.MaxBlocksPerRootBlock
		env.chain.Config().QuarkChain.MaxBlocksPerRootBlock = 2
		defer func() { env.chain.Config().QuarkChain.MaxBlocksPerRootBlock = limit }()
		rawdb.DeleteCanonicalHash(env.db, blocks[3].NumberU64())
		defer rawdb.WriteCanonicalHash(env.db, blocks[3].Hash(), blocks[3].NumberU64())
		if headers, err := env.sc.GetUnconfirmedHeaderList(); !errors.Is(err, ErrShardStateInconsistency) || headers != nil {
			t.Fatalf("missing canonical endpoint: %v, error %v", headers, err)
		}
	})
	for _, limit := range []uint64{1, 2, 3, 9} {
		env.chain.Config().QuarkChain.MaxBlocksPerRootBlock = limit
		headers, err := env.sc.GetUnconfirmedHeaderList()
		if err != nil {
			t.Fatal(err)
		}
		count := 3
		if limit < uint64(count) {
			count = int(limit)
		}
		if len(headers) != count {
			t.Fatalf("limit %d: have %d headers, want %d", limit, len(headers), count)
		}
		for i, header := range headers {
			if header.Hash() != blocks[i+2].Hash() {
				t.Fatalf("limit %d: header %d is not the earliest unconfirmed block", limit, i)
			}
		}
	}
	root = newRoot(root, 1, blocks[2].Header(), blocks[3].Header(), blocks[4].Header())
	env.insertRoot(root, true)
	if headers, err := env.sc.GetUnconfirmedHeaderList(); err != nil || len(headers) != 0 {
		t.Fatalf("confirmed headers still pending: %v, %v", headers, err)
	}
}

// failingRootDB fails one batch commit without changing the underlying data.
// Only the shard's root-block store uses this wrapper, not the shard blockchain.
type failingRootDB struct {
	ethdb.KeyValueStore
	failAt int
	err    error
}

func (db *failingRootDB) NewBatch() ethdb.Batch {
	return &failingRootBatch{Batch: db.KeyValueStore.NewBatch(), db: db}
}

type failingRootBatch struct {
	ethdb.Batch
	db *failingRootDB
}

func (batch *failingRootBatch) Write() error {
	batch.db.failAt--
	if batch.db.failAt == 0 {
		return batch.db.err
	}
	return batch.Batch.Write()
}

// TestInsertRootBlockFailureRecovery checks the head-first import ordering.
// Head failures leave the root unknown; batch failures leave the head advanced
// but the old root tip selected. A failed root can be re-imported once writes
// succeed. Restart recovery first restores a locally stored checkpoint with Init.
func TestInsertRootBlockFailureRecovery(t *testing.T) {
	for _, test := range []struct {
		name      string
		failAt    int
		stopChain bool
		restart   bool
	}{
		{name: "root storage reimport", failAt: 1},
		{name: "root storage restart", failAt: 1, restart: true},
		{name: "shard head restart", stopChain: true, restart: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			env := newTestEnv(t)
			a := env.makeBlocks(env.genesis, 3, env.rootGenesis, 1)
			b := env.makeBlocks(a[0], 1, env.rootGenesis, 2)
			for _, block := range append(a, b...) {
				env.insert(block)
			}
			r1 := newRoot(env.rootGenesis, 1, a[0].Header())
			env.insertRoot(r1, true)
			r2 := newRoot(r1, 1, b[0].Header())

			writeErr := errors.New("root batch write failed")
			env.sc.db = &failingRootDB{KeyValueStore: env.db, failAt: test.failAt, err: writeErr}
			if test.stopChain {
				env.chain.Stop()
			}
			switched, err := env.sc.InsertRootBlock(r2)
			if err == nil || switched {
				t.Fatalf("failed insertion: switched %v, error %v", switched, err)
			}
			if test.failAt != 0 && !errors.Is(err, writeErr) {
				t.Fatalf("error mismatch: have %v, want %v", err, writeErr)
			}
			env.checkRootTip(r1)
			if qkcrawdb.HasRootBlock(env.db, r2.Hash()) {
				t.Fatal("root stored despite failed insertion")
			}
			if hash := qkcrawdb.ReadLastConfirmedBlockHash(env.db, r2.Hash()); hash != (common.Hash{}) {
				t.Fatalf("confirmation stored despite failed root storage: %x", hash)
			}
			if test.stopChain {
				env.checkHead(a[2])
			} else {
				env.checkHead(b[0])
			}

			if test.restart {
				env.chain.Stop()
				env.reopenChain()
				env.sc = NewShardChain(env.sc.db, env.chain, nil)
				// r2 was never stored, so recovery uses the valid r1 checkpoint.
				if err := env.sc.Init(r1); err != nil {
					t.Fatalf("failed to initialize restarted shard chain: %v", err)
				}
				env.checkRootTip(r1)
				env.checkHead(a[0])
			}
			env.insertRoot(r2, true)
			env.checkRootTip(r2)
			env.checkLastConfirmed(r2, b[0])
			env.checkHead(b[0])
			env.insertRoot(r2, false)
		})
	}
}

// TestInsertRootBlockConfirmsAboveHead checks that the head moves forward to
// the latest block confirmed by the new root tip when it extends the head (case
// 1 of setHeadForRootTip).
//
//	              +----- r1b <-- r2b*
//	              v              :
//	root chain    r0 <-- r1      :
//	                             v
//	shard chain   g     <------- a1[r1b]
//
//	r1 is the root tip, so a1, referencing r1b, does not become the head.
//	r2b, heavier than r1, confirms a1, so the head moves to a1.
func TestInsertRootBlockConfirmsAboveHead(t *testing.T) {
	env := newTestEnv(t)
	r1, r1b := newRoot(env.rootGenesis, 1), newRoot(env.rootGenesis, 2)
	env.insertRoot(r1, true)
	env.insertRoot(r1b, false)

	a := env.makeBlocks(env.genesis, 1, r1b, 1)
	env.insert(a[0])
	env.checkHead(env.genesis)

	r2b := newRoot(r1b, 1, a[0].Header())
	env.insertRoot(r2b, true)
	env.checkRootTip(r2b)
	env.checkHead(a[0])
}

// TestInsertRootBlockRewindsHead checks that the head moves back to the latest
// block referencing the new root tip or one of its ancestors (case 2 of
// setHeadForRootTip). Fork a is a in the code.
//
//	                        +----- r2b
//	                        v
//	root chain    r0 <----- r1 <-- r2 <-- r3*
//	                        :
//	                        v
//	shard chain   g     <-- a1[r0] <-- a2[r1] <-- a3[r2b] <-- a4[r2b]
//
//	r1 confirms a1 after the genesis baseline, and so do its descendants. r2b becomes the root
//	tip, and a2 to a4 the head in turn. r2, as heavy as r2b, does not
//	replace it, but r3 does, so the head moves back from a4 to a2.
func TestInsertRootBlockRewindsHead(t *testing.T) {
	env := newTestEnv(t)
	a := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)
	env.insert(a[0])

	r1 := newRoot(env.rootGenesis, 1, a[0].Header())
	r2, r2b := newRoot(r1, 1), newRoot(r1, 2)
	env.insertRoot(r1, true)
	env.insertRoot(r2b, true)

	a = append(a, env.makeBlocks(a[0], 1, r1, 1)...)
	a = append(a, env.makeBlocks(a[1], 2, r2b, 1)...)
	for _, block := range a[1:] {
		env.insert(block)
	}
	env.checkHead(a[3])

	env.insertRoot(r2, false)
	env.checkHead(a[3])

	r3 := newRoot(r2, 1)
	env.insertRoot(r3, true)
	env.checkRootTip(r3)
	env.checkLastConfirmed(r3, a[0])
	env.checkHead(a[1])
}
