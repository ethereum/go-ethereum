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
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/posw"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	qkcrawdb "github.com/ethereum/go-ethereum/qkc/rawdb"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

// backendClusterClient exposes RPC calls so tests can check payloads, ordering and failures.
type backendClusterClient struct {
	broadcast func(context.Context, uint32, common.Hash, []*types.DepositTx) error
	report    func(context.Context, *types.Header, *uint256.Int) error
}

func (c *backendClusterClient) BroadcastXShardTxList(ctx context.Context, shard uint32, hash common.Hash, list []*types.DepositTx) error {
	return c.broadcast(ctx, shard, hash, list)
}

func (c *backendClusterClient) SendMinorBlockHeaderToMaster(ctx context.Context, header *types.Header, amount *uint256.Int) error {
	return c.report(ctx, header, amount)
}

// TestBackendLifecycle checks PoSW construction, separate root initialization,
// and shard-zero-only stake queries. Stop and failed construction must leave the
// caller's database usable, and Init must not propagate genesis again.
func TestBackendLifecycle(t *testing.T) {
	for _, shardID := range []uint32{0, 1} {
		t.Run(big.NewInt(int64(shardID)).String(), func(t *testing.T) {
			config := *params.MergedTestChainConfig
			config.QuarkChain = params.NewTestQuarkChainConfig(shardID, 0, 1)
			alloc := core.SystemContractAllocs()
			maps.Copy(alloc, core.PoSWStakingContractAllocs(shardID))
			genesis := &core.Genesis{
				GasLimit: 12_000_000, Difficulty: new(big.Int), Mixhash: common.BigToHash(common.Big1),
				ExtraData: (&types.QKCExtra{ShardID: shardID}).Encode(), Alloc: alloc,
			}
			db := rawdb.NewMemoryDatabase()
			t.Cleanup(func() { db.Close() })
			client := &backendClusterClient{
				broadcast: func(context.Context, uint32, common.Hash, []*types.DepositTx) error {
					t.Fatal("Init must not broadcast genesis")
					return nil
				},
				report: func(context.Context, *types.Header, *uint256.Int) error {
					t.Fatal("Init must not report genesis")
					return nil
				},
			}
			s, err := New(&config, genesis, db, nil, client)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { s.Stop() })
			if _, ok := s.engine.(*posw.PoSW); !ok || genesis.Config != nil {
				t.Fatal("New must use PoSW without mutating the supplied genesis")
			}
			if s.RootTip() != (common.Hash{}) {
				t.Fatal("New selected a root before Init")
			}
			if _, err := s.GetUnconfirmedHeaderList(); !errors.Is(err, ErrRootChainUninitialized) {
				t.Fatalf("uninitialized headers: %v", err)
			}
			root := qkctypes.NewRootBlock(&qkctypes.RootBlockHeader{
				Difficulty: *uint256.NewInt(1), TotalDifficulty: *uint256.NewInt(1),
			}, qkctypes.ShardBlockHeaders{s.CurrentBlock()}, nil)
			if err := s.Init(root); err != nil {
				t.Fatal(err)
			}
			if s.RootTip() != root.Hash() {
				t.Fatal("Init did not select the master's root")
			}
			stakes, signer, err := s.GetRootChainStakes(common.Address{}, s.CurrentBlock().Hash())
			if shardID == 0 {
				if err != nil || stakes == nil || !stakes.IsZero() || signer != (common.Address{}) {
					t.Fatalf("root stakes: %v, %v, %v", stakes, signer, err)
				}
				// A missing query header is a lookup failure and must identify its hash.
				missing := common.Hash{0xff}
				stakes, signer, err = s.GetRootChainStakes(common.Address{}, missing)
				if err == nil || !strings.Contains(err.Error(), missing.Hex()) || stakes != nil || signer != (common.Address{}) {
					t.Fatalf("missing shard block header: stakes %v, signer %v, error %v", stakes, signer, err)
				}
			} else if err == nil {
				t.Fatal("nonzero shard served root-chain stakes")
			}
			if err := s.Stop(); err != nil {
				t.Fatal(err)
			}
			if err := s.Stop(); err != nil {
				t.Fatal(err)
			}
			badGenesis := *genesis
			badGenesis.GasLimit++
			var mismatch *core.GenesisMismatchError
			if failed, err := New(&config, &badGenesis, db, nil, client); failed != nil || !errors.As(err, &mismatch) {
				t.Fatalf("mismatched genesis: backend %v, error %v", failed, err)
			}
			if err := db.Put([]byte("caller-owned"), []byte{1}); err != nil {
				t.Fatalf("Stop or failed New closed the borrowed database: %v", err)
			}
			// Reconstruct from the same database; Init remains a separate step.
			reopened, err := New(&config, genesis, db, nil, client)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Stop()
			if err := reopened.Init(root); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// TestBackendBroadcastXShardTxList checks that every other configured shard gets
// its deposits or an empty list, with the source block hash preserved.
func TestBackendBroadcastXShardTxList(t *testing.T) {
	env := newTestEnv(t)
	env.chain.Config().QuarkChain.ShardIDs = []uint32{0, 1, 2}
	// Broadcast reads stored receipts; outbox execution is covered by TestXShardDeposits.
	tx := types.NewTx(&types.LegacyTx{
		To: &params.XShardOutboxAddress, Gas: params.TxGas, GasPrice: big.NewInt(params.GWei),
	})
	receipts := types.Receipts{{
		Status: types.ReceiptStatusSuccessful, CumulativeGasUsed: params.TxGas,
		Logs: []*types.Log{sentMessageLog(1)},
	}}
	block := types.NewBlock(&types.Header{
		ParentHash: env.genesis.Hash(), Number: big.NewInt(1), Time: 1,
		GasLimit: env.genesis.GasLimit(), GasUsed: params.TxGas,
	}, &types.Body{Transactions: []*types.Transaction{tx}}, receipts, trie.NewStackTrie(nil))
	rawdb.WriteBlock(env.db, block)
	rawdb.WriteReceipts(env.db, block.Hash(), block.NumberU64(), receipts)
	to := common.Address{19: 0x0e}
	want := map[uint32][]*types.DepositTx{
		1: {{
			Version: types.DepositTxVersion, SourceBlockHash: block.Hash(), SourceLogIndex: 0,
			From: common.Address{19: 0x0d}, To: &to, Value: uint256.NewInt(0x0a),
			GasPrice: uint256.NewInt(0x0b), Gas: 0x0c, IsMove: true, Data: []byte{1, 2, 3},
		}},
		2: {}, // Configured destinations with no messages still receive a list.
	}
	delivered := make(map[uint32][]*types.DepositTx)
	s := &ShardBackend{chain: env.sc, clusterClient: &backendClusterClient{
		broadcast: func(_ context.Context, shard uint32, hash common.Hash, list []*types.DepositTx) error {
			if hash != block.Hash() {
				t.Fatalf("delivered block %x, want %x", hash, block.Hash())
			}
			delivered[shard] = list
			return nil
		},
	}}
	if err := s.BroadcastXShardTxList(context.Background(), block.Hash()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(delivered, want) {
		t.Fatalf("broadcast deposits: have %v, want %v", delivered, want)
	}
}

// TestBackendAddShardBlockRetry fails either propagation stage after local import.
// Retrying the stored block must resend all lists before reporting to master;
// a broadcast failure must prevent reporting, and coinbase amount is currently zero.
func TestBackendAddShardBlockRetry(t *testing.T) {
	for _, failReport := range []bool{false, true} {
		name := "broadcast failure"
		if failReport {
			name = "master failure"
		}
		t.Run(name, func(t *testing.T) {
			env := newTestEnv(t)
			env.config.QuarkChain.ShardIDs = []uint32{0, 1, 2}
			env.chain.Config().QuarkChain.ShardIDs = []uint32{0, 1, 2}
			block := env.makeBlocks(env.genesis, 1, env.rootGenesis, 1)[0]
			failure := errors.New("delivery failed")
			fail := true
			delivered := make(map[uint32]bool)
			reports := 0
			client := &backendClusterClient{
				broadcast: func(_ context.Context, shard uint32, hash common.Hash, list []*types.DepositTx) error {
					if hash != block.Hash() || (shard != 1 && shard != 2) || len(list) != 0 {
						t.Fatalf("unexpected delivery: shard %d, block %x, list %v", shard, hash, list)
					}
					if fail && !failReport && shard == 1 {
						return failure
					}
					delivered[shard] = true
					return nil
				},
				report: func(_ context.Context, header *types.Header, amount *uint256.Int) error {
					reports++
					if !delivered[1] || !delivered[2] {
						t.Fatal("master notified before all shards received their lists")
					}
					if header.Hash() != block.Hash() || !amount.IsZero() {
						t.Fatalf("reported block %x, coinbase amount %v", header.Hash(), amount)
					}
					if fail && failReport {
						return failure
					}
					return nil
				},
			}
			s := &ShardBackend{chain: env.sc, clusterClient: client}
			if err := s.AddShardBlock(context.Background(), block); !errors.Is(err, failure) {
				t.Fatalf("failed delivery: %v", err)
			}
			if !env.chain.HasBlockAndState(block.Hash(), block.NumberU64()) {
				t.Fatal("fixture must fail propagation after the block is stored")
			}
			if !failReport && reports != 0 {
				t.Fatal("broadcast failure still reported to master")
			}
			fail, reports, delivered = false, 0, make(map[uint32]bool)
			// A fresh wrapper has no in-memory delivery progress to rely on.
			s = &ShardBackend{chain: env.sc, clusterClient: client}
			if err := s.AddShardBlock(context.Background(), block); err != nil {
				t.Fatal(err)
			}
			if len(delivered) != 2 || reports != 1 {
				t.Fatalf("retry deliveries %v, master reports %d", delivered, reports)
			}
		})
	}
}

// failingDepositDB injects a write error before changing the stored deposit list.
type failingDepositDB struct {
	ethdb.KeyValueStore
	err error
}

func (db *failingDepositDB) Put(key, value []byte) error {
	if db.err != nil {
		return db.err
	}
	return db.KeyValueStore.Put(key, value)
}

// TestBackendReceivedDeposits checks write-error propagation and caller retries.
// Stored empty and nonempty lists must survive replacing the ShardChain, while
// an empty received list remains distinguishable from a list not yet received.
func TestBackendReceivedDeposits(t *testing.T) {
	env := newTestEnv(t)
	failure := errors.New("deposit write failed")
	db := &failingDepositDB{KeyValueStore: env.db, err: failure}
	env.sc.db = db
	s := &ShardBackend{chain: env.sc}
	empty, full := common.Hash{1}, common.Hash{2}
	if err := s.AddXshardTxList(empty, nil); !errors.Is(err, failure) {
		t.Fatalf("failed write: %v", err)
	}
	if qkcrawdb.HasXShardDeposits(env.db, empty) {
		t.Fatal("failed write stored an empty list")
	}
	db.err = nil
	if err := s.AddXshardTxList(empty, nil); err != nil {
		t.Fatal(err)
	}
	deposits := []*types.DepositTx{{
		SourceBlockHash: full, SourceLogIndex: 1, To: &common.Address{3},
		Value: uint256.NewInt(4), GasPrice: uint256.NewInt(5), Gas: params.XShardSendGas, IsMove: true, Data: []byte{},
	}}
	if err := s.AddXshardTxList(full, deposits); err != nil {
		t.Fatal(err)
	}
	// The result survives replacing the ShardChain; no in-memory marker is needed.
	s = &ShardBackend{chain: NewShardChain(env.db, env.chain)}
	if !qkcrawdb.HasXShardDeposits(s.chain.db, empty) || qkcrawdb.ReadXShardDeposits(s.chain.db, empty) == nil {
		t.Fatal("received empty list became missing")
	}
	if have := qkcrawdb.ReadXShardDeposits(s.chain.db, full); !reflect.DeepEqual(have, deposits) {
		t.Fatalf("received deposits: have %v, want %v", have, deposits)
	}
	if qkcrawdb.ReadXShardDeposits(s.chain.db, common.Hash{9}) != nil {
		t.Fatal("missing list was reported as received")
	}
}
