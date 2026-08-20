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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

// migrationTestGenesis is the binary-tree genesis with the fork pushed past
// genesis: the chain starts on the merkle trie and migrates at t=48, which at
// twelve-second blocks is the fourth block.
func migrationTestGenesis() *core.Genesis {
	genesis := pbtGenesis()
	config := *genesis.Config
	forkTime := uint64(48)
	config.BinaryTrieTime = &forkTime
	genesis.Config = &config
	return genesis
}

// TestMigrationNodeCrossesTheFork drives the devnet rehearsal end to end: a
// node starts on the merkle trie with the binary tree scheduled, the shadow
// follower replays every block, and at the fork the node builds and imports
// blocks whose headers commit the binary tree - the pre-state coming from the
// shadow's recorded root of the last merkle block.
func TestMigrationNodeCrossesTheFork(t *testing.T) {
	genesis := migrationTestGenesis()
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()

	api := NewConsensusAPI(ethservice)
	chain := ethservice.BlockChain()
	if chain.TrieDB().IsPBT() {
		t.Fatal("migrating node opened on the binary tree")
	}

	parent := chain.CurrentBlock()
	for i := 0; i < 5; i++ {
		// Building on a merkle parent needs the shadow caught up to it - at
		// the boundary its recorded root is the pre-state. The consensus
		// layer's retries hide this in production; the test waits.
		if !genesis.Config.IsBinaryTrie(parent.Number, parent.Time) {
			for start := time.Now(); !chain.ShadowReady(parent.Hash(), parent.Number.Uint64()); {
				if time.Since(start) > 10*time.Second {
					t.Fatalf("block %d: shadow never reached parent %d (%+v)", i+1, parent.Number, chain.MigrationProgress())
				}
				time.Sleep(10 * time.Millisecond)
			}
		}
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
			t.Fatalf("block %d: forkchoice update failed: %v", i+1, err)
		}
		if resp.PayloadStatus.Status != engine.VALID {
			t.Fatalf("block %d: forkchoice update not valid: %v", i+1, resp.PayloadStatus.Status)
		}
		payload, err := api.getPayload(*resp.PayloadID, true, nil, nil)
		if err != nil {
			t.Fatalf("block %d: payload retrieval failed: %v", i+1, err)
		}
		execData := payload.ExecutionPayload
		status, err := api.NewPayloadV5(context.Background(), *execData, []common.Hash{}, &common.Hash{}, []hexutil.Bytes{})
		if err != nil {
			t.Fatalf("block %d: payload import failed: %v", i+1, err)
		}
		if status.Status != engine.VALID {
			t.Fatalf("block %d: imported payload not valid: %v (%s)", i+1, status.Status, derefErr(status.ValidationError))
		}
		fcState = engine.ForkchoiceStateV1{HeadBlockHash: execData.BlockHash}
		if _, err := api.ForkchoiceUpdatedV4(context.Background(), fcState, nil, nil); err != nil {
			t.Fatalf("block %d: setting head failed: %v", i+1, err)
		}
		parent = chain.CurrentBlock()
		if parent.Hash() != execData.BlockHash {
			t.Fatalf("block %d: head is %x, expected %x", i+1, parent.Hash(), execData.BlockHash)
		}
	}
	if head := chain.CurrentBlock().Number.Uint64(); head != 5 {
		t.Fatalf("chain head is %d, want 5", head)
	}

	// The boundary parent's shadow root exists - it fed the activation block -
	// while post-fork blocks get none: execution owns the tree from there.
	boundaryParent := chain.GetHeaderByNumber(3)
	if !chain.ShadowReady(boundaryParent.Hash(), 3) {
		t.Fatal("the boundary parent has no recorded shadow root")
	}
	postFork := chain.GetHeaderByNumber(4)
	if chain.ShadowReady(postFork.Hash(), 4) {
		t.Fatal("a post-fork block got a shadow root recorded")
	}
	if genesis.Config.IsBinaryTrie(boundaryParent.Number, boundaryParent.Time) {
		t.Fatal("block 3 is not a merkle block; the fork moved")
	}
	if !genesis.Config.IsBinaryTrie(postFork.Number, postFork.Time) {
		t.Fatal("block 4 is not a binary-tree block; the fork moved")
	}

	// Both sides of the boundary stay readable: the merkle past through the
	// canonical handle, the binary present through the follower's.
	for _, number := range []uint64{2, 5} {
		header := chain.GetHeaderByNumber(number)
		statedb, err := chain.StateAt(header)
		if err != nil {
			t.Fatalf("state at block %d: %v", number, err)
		}
		if got := statedb.GetBalance(testAddr).ToBig(); got.Cmp(testBalance) != 0 {
			t.Fatalf("balance at block %d = %v, want %v", number, got, testBalance)
		}
	}
}
