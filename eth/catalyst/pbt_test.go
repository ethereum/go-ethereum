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

package catalyst

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// pbtGenesis builds the genesis a binary-tree devnet runs on: the tree
// committing the state from block zero, with the system contracts pre-deployed
// as block processing expects from Cancun onwards.
//
// It sits on Amsterdam because the tree requires it. An earlier version stopped
// at Osaka, on the grounds that the EIP-7928 block access list hash in the
// header had no field in the executable payload and so could not survive the
// round trip. That gap is closed: ExecutableData carries the access list, and
// the header hash is rebuilt from it.
func pbtGenesis() *core.Genesis {
	u64 := func(v uint64) *uint64 { return &v }
	config := &params.ChainConfig{
		ChainID:                 big.NewInt(1337),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(0),
		ShanghaiTime:            u64(0),
		CancunTime:              u64(0),
		PragueTime:              u64(0),
		OsakaTime:               u64(0),
		AmsterdamTime:           u64(0),
		BinaryTrieTime:          u64(0),
		DepositContractAddress:  params.MainnetChainConfig.DepositContractAddress,
		BlobScheduleConfig: &params.BlobScheduleConfig{
			Cancun: params.DefaultCancunBlobConfig,
			Prague: params.DefaultPragueBlobConfig,
			BPO1:   params.DefaultBPO1BlobConfig,
			BPO2:   params.DefaultBPO2BlobConfig,
		},
	}
	return &core.Genesis{
		Config:     config,
		Difficulty: common.Big0,
		GasLimit:   30_000_000,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		Alloc: types.GenesisAlloc{
			testAddr:                         {Balance: testBalance},
			params.BeaconRootsAddress:        {Nonce: 1, Code: params.BeaconRootsCode, Balance: common.Big0},
			params.HistoryStorageAddress:     {Nonce: 1, Code: params.HistoryStorageCode, Balance: common.Big0},
			params.WithdrawalQueueAddress:    {Nonce: 1, Code: params.WithdrawalQueueCode, Balance: common.Big0},
			params.ConsolidationQueueAddress: {Nonce: 1, Code: params.ConsolidationQueueCode, Balance: common.Big0},
		},
	}
}

func derefErr(s *string) string {
	if s == nil {
		return "<none>"
	}
	return *s
}

// TestPBTNodeProducesAndImportsBlocks drives the engine API against a node
// whose state lives in the EIP-8297 binary tree: the consensus layer asks
// for a payload, the node builds one, and the same node imports it back and
// makes it canonical. This is the devnet path end to end - genesis from an
// allocation, block production, block import - through the same interface a
// consensus client uses.
func TestPBTNodeProducesAndImportsBlocks(t *testing.T) {
	genesis := pbtGenesis()
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	if !ethservice.BlockChain().TrieDB().IsPBT() {
		t.Fatal("node is not running on the binary tree")
	}

	parent := ethservice.BlockChain().CurrentBlock()
	for i := 0; i < 3; i++ {
		// Amsterdam takes the V4 attributes and the V5 payload: the slot number
		// and target gas limit are required, and newPayloadV5 rejects a payload
		// whose block access list is missing.
		slot, targetGasLimit := uint64(i+1), parent.GasLimit
		attrs := &engine.PayloadAttributes{
			Timestamp:             parent.Time + 12,
			Random:                common.Hash{},
			SuggestedFeeRecipient: common.Address{},
			Withdrawals:           []*types.Withdrawal{},
			BeaconRoot:            &common.Hash{},
			SlotNumber:            &slot,
			TargetGasLimit:        &targetGasLimit,
		}
		fcState := engine.ForkchoiceStateV1{HeadBlockHash: parent.Hash()}
		resp, err := api.ForkchoiceUpdatedV4(context.Background(), fcState, attrs, nil)
		if err != nil {
			t.Fatalf("block %d: forkchoice update failed: %v", i, err)
		}
		if resp.PayloadStatus.Status != engine.VALID {
			t.Fatalf("block %d: forkchoice update not valid: %v", i, resp.PayloadStatus.Status)
		}
		payload, err := api.getPayload(*resp.PayloadID, true, nil, nil)
		if err != nil {
			t.Fatalf("block %d: payload retrieval failed: %v", i, err)
		}
		execData := payload.ExecutionPayload
		// The payload commits to the binary tree's root.
		status, err := api.NewPayloadV5(context.Background(), *execData, []common.Hash{}, &common.Hash{}, []hexutil.Bytes{})
		if err != nil {
			t.Fatalf("block %d: payload import failed: %v", i, err)
		}
		if status.Status != engine.VALID {
			t.Fatalf("block %d: imported payload not valid: %v (%s)", i, status.Status, derefErr(status.ValidationError))
		}
		fcState = engine.ForkchoiceStateV1{HeadBlockHash: execData.BlockHash}
		if _, err := api.ForkchoiceUpdatedV4(context.Background(), fcState, nil, nil); err != nil {
			t.Fatalf("block %d: setting head failed: %v", i, err)
		}
		parent = ethservice.BlockChain().CurrentBlock()
		if parent.Hash() != execData.BlockHash {
			t.Fatalf("block %d: head is %x, expected %x", i, parent.Hash(), execData.BlockHash)
		}
	}
	if head := ethservice.BlockChain().CurrentBlock().Number.Uint64(); head != 3 {
		t.Fatalf("chain head is %d, want 3", head)
	}
	// State reads must still work against the tree at the new head.
	statedb, err := ethservice.BlockChain().State()
	if err != nil {
		t.Fatal(err)
	}
	if got := statedb.GetBalance(testAddr).ToBig(); got.Cmp(testBalance) > 0 {
		t.Fatalf("balance grew without transactions: %v", got)
	}
}
