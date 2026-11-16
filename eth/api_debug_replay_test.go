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

package eth

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// replayKey signs the transactions recorded as reverted during building. It is
// funded, but not used by the blocks themselves.
var replayKey, _ = crypto.HexToECDSA("8a1f9a8f95be41cd7ccb6168179afb4504aefe388d1e14474d32c45c72ce7b7a")

// newReplayTestAPI builds an Amsterdam chain with one imported block and
// returns the debug API together with a second block that is not imported,
// to be stored as a bad block by the test.
func newReplayTestAPI(t *testing.T) (*DebugAPI, *types.Block, types.Signer) {
	t.Helper()

	config := *params.MergedTestChainConfig
	config.AmsterdamTime = new(uint64)

	var (
		signer = types.LatestSigner(&config)
		to     = common.HexToAddress("0xdead")
		engine = beacon.New(ethash.NewFaker())
		gspec  = &core.Genesis{
			Config: &config,
			Alloc: types.GenesisAlloc{
				testAddr: {Balance: big.NewInt(params.Ether)},
				crypto.PubkeyToAddress(replayKey.PublicKey): {Balance: big.NewInt(params.Ether)},
				params.BeaconRootsAddress:                   {Nonce: 1, Code: params.BeaconRootsCode},
				params.HistoryStorageAddress:                {Nonce: 1, Code: params.HistoryStorageCode},
				params.WithdrawalQueueAddress:               {Nonce: 1, Code: params.WithdrawalQueueCode},
				params.ConsolidationQueueAddress:            {Nonce: 1, Code: params.ConsolidationQueueCode},
				params.BuilderDepositAddress:                {Nonce: 1, Code: params.BuilderDepositCode},
				params.BuilderExitAddress:                   {Nonce: 1, Code: params.BuilderExitCode},
			},
		}
	)
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, engine, 2, func(i int, b *core.BlockGen) {
		tx, _ := types.SignTx(types.NewTransaction(uint64(i), to, big.NewInt(1), params.TxGas, b.BaseFee(), nil), signer, testKey)
		b.AddTx(tx)
	})
	db := rawdb.NewMemoryDatabase()
	chain, err := core.NewBlockChain(db, gspec, engine, &core.BlockChainConfig{
		TrieCleanLimit: 256,
		TrieDirtyLimit: 256,
		ArchiveMode:    true,
	})
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	t.Cleanup(chain.Stop)
	if _, err := chain.InsertChain(blocks[:1]); err != nil {
		t.Fatalf("failed to insert block: %v", err)
	}
	return NewDebugAPI(&Ethereum{blockchain: chain, chainDb: db}), blocks[1], signer
}

// TestReplayBadBlock checks that a bad block is replayed on both the build and
// the import side, and that a transaction which was reverted during building,
// and fails again on replay, leaves the results matching the block.
func TestReplayBadBlock(t *testing.T) {
	api, block, signer := newReplayTestAPI(t)

	// The nonce is too high, so the transaction fails again on replay.
	reverted, _ := types.SignTx(types.NewTransaction(5, common.Address{}, nil, params.TxGas, block.BaseFee(), nil), signer, replayKey)
	rawdb.WriteBadBlockWithDetails(api.eth.chainDb, block, &rawdb.ExecutionDetail{
		Reason:   "test",
		Reverted: []*rawdb.RevertedTx{{Index: 0, Tx: reverted}},
	})
	out, err := api.ReplayBadBlock(context.Background(), block.Hash())
	if err != nil {
		t.Fatalf("replay failed: %v", err)
	}
	if out["revertedCount"] != 1 {
		t.Fatalf("reverted count %v, want 1", out["revertedCount"])
	}
	for _, field := range []string{"buildStateRoot", "importStateRoot"} {
		if root := out[field].(common.Hash); root != block.Root() {
			t.Errorf("%s %x, want %x", field, root, block.Root())
		}
	}
	for _, field := range []string{"buildBALHash", "importBALHash"} {
		if hash := out[field].(common.Hash); hash != *block.Header().BlockAccessListHash {
			t.Errorf("%s %x, want %x", field, hash, *block.Header().BlockAccessListHash)
		}
	}
}

// TestReplayBadBlockRevertedTxSucceeds checks that the replay reports an error
// instead of crashing when a transaction recorded as reverted during building
// succeeds when it is replayed.
func TestReplayBadBlockRevertedTxSucceeds(t *testing.T) {
	api, block, signer := newReplayTestAPI(t)

	// A valid transfer, it succeeds when replayed.
	reverted, _ := types.SignTx(types.NewTransaction(0, common.Address{}, big.NewInt(1), params.TxGas, block.BaseFee(), nil), signer, replayKey)
	rawdb.WriteBadBlockWithDetails(api.eth.chainDb, block, &rawdb.ExecutionDetail{
		Reason:   "test",
		Reverted: []*rawdb.RevertedTx{{Index: 0, Tx: reverted}},
	})
	_, err := api.ReplayBadBlock(context.Background(), block.Hash())
	if err == nil || !strings.Contains(err.Error(), reverted.Hash().Hex()) {
		t.Fatalf("have error %v, want one naming %v", err, reverted.Hash())
	}
}
