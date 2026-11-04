// Copyright 2026 The quarkchain Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/posw"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/legacypool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
	qkcrawdb "github.com/ethereum/go-ethereum/qkc/rawdb"
)

type miningBackend struct {
	chain *core.BlockChain
	pool  *txpool.TxPool
}

func (b miningBackend) BlockChain() *core.BlockChain { return b.chain }
func (b miningBackend) TxPool() *txpool.TxPool       { return b.pool }

func newMiningEnv(t *testing.T, shardID uint32) (*testEnv, *txpool.TxPool, *miner.Miner) {
	roots, _, _, _, _, shard0, shard1 := testRootChain()
	config := *params.MergedTestChainConfig
	config.QuarkChain = params.NewTestQuarkChainConfig(shardID, 0, 1, 2)
	env := &testEnv{t: t, db: rawdb.NewMemoryDatabase(), config: &config, rootGenesis: roots[0]}
	incoming := shard1
	if shardID == 0 {
		incoming = shard0
	}
	for hash, deposits := range incoming.deposits {
		qkcrawdb.WriteXShardDeposits(env.db, hash, deposits)
	}
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	alloc := core.SystemContractAllocs()
	maps.Copy(alloc, core.PoSWStakingContractAllocs(shardID))
	alloc[crypto.PubkeyToAddress(key.PublicKey)] = types.Account{Balance: big.NewInt(params.Ether)}
	genesis := &core.Genesis{
		Config: &config, GasLimit: 54000, Alloc: alloc,
		ExtraData: (&types.QKCExtra{ShardID: shardID, Cursor: pos(1, 0, 0)}).Encode(),
	}
	// Build with the PoSW engine, which prepares the difficulty, without
	// verifying the test seals.
	engine, err := posw.New(config.QuarkChain, genesis.ToBlock().Header())
	if err != nil {
		t.Fatal(err)
	}
	env.engine = noVerifyEngine{engine}
	chain, err := core.NewBlockChain(env.db, genesis, env.engine, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(chain.Stop)
	env.chain, env.genesis = chain, chain.Genesis()
	poolConfig := legacypool.DefaultConfig
	poolConfig.Journal = ""
	pool, err := txpool.New(poolConfig.PriceLimit, chain, []txpool.SubPool{legacypool.New(poolConfig, chain)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	minerConfig := miner.DefaultConfig
	minerConfig.GasCeil = genesis.GasLimit
	builder := miner.New(miningBackend{chain, pool}, minerConfig, env.engine)
	env.sc = NewShardChain(env.db, chain, builder)
	for _, root := range roots {
		env.addRoot(root, env.genesis)
	}
	if err := env.sc.Init(roots[len(roots)-1]); err != nil {
		t.Fatal(err)
	}
	return env, pool, builder
}

func TestBuildShardBlock(t *testing.T) {
	for _, shardID := range []uint32{0, 1} {
		t.Run(string(rune('0'+shardID)), func(t *testing.T) {
			env, pool, _ := newMiningEnv(t, shardID)
			key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
			userTx := types.MustSignNewTx(key, types.LatestSigner(env.config), &types.DynamicFeeTx{
				ChainID: env.config.ChainID, Gas: params.TxGas, To: &common.Address{0xee},
				GasFeeCap: big.NewInt(2 * params.GWei), GasTipCap: big.NewInt(params.GWei),
			})
			if err := pool.Add([]*types.Transaction{userTx}, true)[0]; err != nil {
				t.Fatal(err)
			}
			root := env.sc.rootTip.Hash()
			args := miner.QKCBlockArgs{Timestamp: 1, FeeRecipient: common.Address{0xcc}}
			block, err := env.sc.BuildBlock(context.Background(), args)
			if err != nil {
				t.Fatal(err)
			}
			env.checkHead(env.genesis)
			extra, _ := block.Header().QKCExtra()
			cursor := pos(4, 0, 0)
			if shardID == 0 {
				cursor = pos(2, 0, 1)
			}
			if extra.PrevRootBlock != root || extra.ShardID != shardID || extra.Cursor != cursor || extra.XShardGasUsed != 27000 {
				t.Fatalf("unexpected extra: %+v", extra)
			}
			if txs := block.Transactions(); len(txs) != 4 || txs[3].Hash() != userTx.Hash() {
				t.Fatalf("expected three deposits followed by the pool transaction, got %d transactions", len(txs))
			}
			if shardID == 0 {
				root1 := env.sc.RootBlockByNumber(root, 1)
				if block.Transactions()[0].Hash() != types.NewTx(rootCoinbaseDeposit(root1)).Hash() {
					t.Fatal("root reward not first")
				}
			}
			env.insert(block)
			env.checkHead(block)
			// Resume the real cursor from the imported parent, without replaying deposits.
			args.Timestamp, args.NoTxs = 2, true
			next, err := env.sc.BuildBlock(context.Background(), args)
			if err != nil {
				t.Fatal(err)
			}
			nextExtra, _ := next.Header().QKCExtra()
			count := 0
			if shardID == 0 {
				count = 2
			}
			if next.ParentHash() != block.Hash() || nextExtra.Cursor != pos(4, 0, 0) || len(next.Transactions()) != count {
				t.Fatalf("unexpected resumed block: cursor %+v, %d transactions", nextExtra.Cursor, len(next.Transactions()))
			}
			env.insert(next)
			env.checkHead(next)
		})
	}
}

func TestBuildShardBlockRootReferences(t *testing.T) {
	env, _, _ := newMiningEnv(t, 1)
	ctx := context.Background()
	root3 := env.sc.rootTip.Hash()
	root1 := env.sc.RootBlockByNumber(root3, 1)
	args := miner.QKCBlockArgs{Timestamp: 1, PrevRootBlock: root1.Hash(), NoTxs: true}
	a, err := env.sc.BuildBlock(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	env.insert(a)
	args.Parent, args.Timestamp = env.genesis.Hash(), 2
	b, err := env.sc.BuildBlock(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	env.insert(b)
	env.checkHead(a)
	fork := newRoot(env.rootGenesis, 99)
	env.addRoot(fork, env.genesis)
	qkcrawdb.WriteLastConfirmedBlockHash(env.db, root3, a.Hash())
	for _, tt := range []struct {
		name         string
		parent, root common.Hash
		want         error
	}{
		{"unknown parent", common.Hash{0xff}, root3, consensus.ErrUnknownAncestor},
		{"unknown root", a.Hash(), common.Hash{0xff}, ErrUnknownRootBlock},
		{"root goes backward", a.Hash(), env.rootGenesis.Hash(), ErrRootNotDescendant},
		{"divergent root", a.Hash(), fork.Hash(), ErrRootNotDescendant},
		{"fork before confirmed block", b.Hash(), root3, ErrNotConfirmedDescendant},
	} {
		t.Run(tt.name, func(t *testing.T) {
			args.Parent, args.PrevRootBlock, args.Timestamp = tt.parent, tt.root, 3
			if _, err := env.sc.BuildBlock(ctx, args); !errors.Is(err, tt.want) {
				t.Fatalf("error %v, want %v", err, tt.want)
			}
		})
	}
	args.Parent, args.PrevRootBlock, args.Timestamp = a.Hash(), root3, 3
	next, err := env.sc.BuildBlock(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	env.insert(next)
	env.checkHead(next)
}

func TestBuildShardBlockBuilderSettings(t *testing.T) {
	env, pool, builder := newMiningEnv(t, 1)
	key, _ := crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	userTx := types.MustSignNewTx(key, types.LatestSigner(env.config), &types.DynamicFeeTx{
		ChainID: env.config.ChainID, Gas: params.TxGas, To: &common.Address{0xee},
		GasFeeCap: big.NewInt(2 * params.GWei), GasTipCap: big.NewInt(params.GWei),
	})
	if err := pool.Add([]*types.Transaction{userTx}, true)[0]; err != nil {
		t.Fatal(err)
	}
	// Rebuild on the same parent so only the live builder settings change.
	for _, tt := range []struct {
		name    string
		gasCeil uint64
		tip     int64
		wantTxs int
	}{
		{"initial", 54000, params.GWei / 1000, 4},
		{"higher gas ceiling and tip", 108000, 2 * params.GWei, 3},
		{"restored", 54000, params.GWei / 1000, 4},
	} {
		t.Run(tt.name, func(t *testing.T) {
			builder.SetGasCeil(tt.gasCeil)
			if err := builder.SetGasTip(big.NewInt(tt.tip)); err != nil {
				t.Fatal(err)
			}
			block, err := env.sc.BuildBlock(context.Background(), miner.QKCBlockArgs{Timestamp: 1})
			if err != nil {
				t.Fatal(err)
			}
			// The PoSW engine fixes the gas limit to genesis, whatever the ceiling.
			if want := env.genesis.GasLimit(); block.GasLimit() != want {
				t.Fatalf("gas limit %d, want %d", block.GasLimit(), want)
			}
			txs := block.Transactions()
			if len(txs) != tt.wantTxs {
				t.Fatalf("transaction count %d, want %d", len(txs), tt.wantTxs)
			}
			for _, tx := range txs[:3] {
				if tx.Type() != types.DepositTxType {
					t.Fatal("builder tip filter excluded mandatory deposits")
				}
			}
			if tt.wantTxs == 4 && txs[3].Hash() != userTx.Hash() {
				t.Fatal("missing pool transaction")
			}
		})
	}
	env.checkHead(env.genesis)
}

func TestBuildShardBlockMissingDeposits(t *testing.T) {
	env, _, _ := newMiningEnv(t, 1)
	source := sourceBlock(0, 99)
	root := newRoot(env.sc.RootBlock(env.sc.rootTip.Hash()), 99).WithBody([]*types.Header{source}, nil)
	env.addRoot(root, env.genesis)
	env.setRootTip(root)
	args := miner.QKCBlockArgs{Timestamp: 1, NoTxs: true}
	// The old three deposits exhaust the budget, but the validator still probes
	// the next source list to distinguish backlog from EOF.
	if _, err := env.sc.BuildBlock(context.Background(), args); !errors.Is(err, ErrXShardDepositsMissing) {
		t.Fatalf("error %v, want missing deposits", err)
	}
	qkcrawdb.WriteXShardDeposits(env.db, source.Hash(), nil)
	block, err := env.sc.BuildBlock(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	extra, _ := block.Header().QKCExtra()
	if extra.Cursor != pos(5, 0, 0) {
		t.Fatalf("cursor %+v, want EOF at 5", extra.Cursor)
	}
	env.insert(block)
}

func TestBuildShardBlockInputs(t *testing.T) {
	env, _, builder := newMiningEnv(t, 1)
	args := miner.QKCBlockArgs{Timestamp: 1}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := env.sc.BuildBlock(ctx, args); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	withoutBuilder := NewShardChain(env.db, env.chain, nil)
	if _, err := withoutBuilder.BuildBlock(context.Background(), args); err == nil {
		t.Fatal("accepted missing builder")
	}
	withoutPool := miner.New(miningBackend{chain: env.chain}, miner.DefaultConfig, env.engine)
	withoutPoolChain := NewShardChain(env.db, env.chain, withoutPool)
	if err := withoutPoolChain.Init(env.sc.RootBlock(env.sc.rootTip.Hash())); err != nil {
		t.Fatal(err)
	}
	if _, err := withoutPoolChain.BuildBlock(context.Background(), args); err == nil {
		t.Fatal("accepted missing pool")
	}
	args.NoTxs = true
	if block, err := withoutPoolChain.BuildBlock(context.Background(), args); err != nil {
		t.Fatalf("NoTxs without pool: %v", err)
	} else if len(block.Transactions()) != 3 {
		t.Fatalf("NoTxs omitted deposits: got %d transactions, want 3", len(block.Transactions()))
	}
	args.NoTxs = false
	args.Timestamp = 0
	if _, err := env.sc.BuildBlock(context.Background(), args); err == nil {
		t.Fatal("accepted invalid timestamp")
	}
	args.Timestamp = 1
	env.sc = NewShardChain(env.db, env.chain, builder)
	for _, root := range []common.Hash{{}, env.rootGenesis.Hash()} {
		args.PrevRootBlock = root
		if _, err := env.sc.BuildBlock(context.Background(), args); !errors.Is(err, ErrRootChainUninitialized) {
			t.Fatalf("uninitialized root tip with previous root %x: %v", root, err)
		}
	}
}
