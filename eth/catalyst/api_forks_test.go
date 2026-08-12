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

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// engineFork pins one fork to the engine API versions that serve it.
type engineFork struct {
	name string

	// activate turns the fork on at the given timestamp, along with the earlier
	// ones it builds upon.
	activate func(cfg *params.ChainConfig, at uint64)

	// beaconRoot and slotAndGas mark the optional payload attributes the fork's
	// FCU version requires.
	beaconRoot bool // cancun and later
	slotAndGas bool // amsterdam and later

	fcu        func(*ConsensusAPI, engine.ForkchoiceStateV1, *engine.PayloadAttributes) (engine.ForkChoiceResponse, error)
	getPayload func(*ConsensusAPI, engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error)
	newPayload func(*ConsensusAPI, *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error)
}

func fcuV3(api *ConsensusAPI, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
	return api.ForkchoiceUpdatedV3(context.Background(), state, attrs)
}

func newPayloadV4(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error) {
	return api.NewPayloadV4(context.Background(), *e.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot, requestsOf(e))
}

var engineForks = []engineFork{
	{
		name:     "shanghai",
		activate: func(cfg *params.ChainConfig, at uint64) { cfg.ShanghaiTime = &at },
		fcu: func(api *ConsensusAPI, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
			return api.ForkchoiceUpdatedV2(context.Background(), state, attrs)
		},
		getPayload: func(api *ConsensusAPI, id engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
			return api.GetPayloadV2(id)
		},
		newPayload: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error) {
			return api.NewPayloadV2(context.Background(), *e.ExecutionPayload)
		},
	},
	{
		name: "cancun",
		activate: func(cfg *params.ChainConfig, at uint64) {
			cfg.ShanghaiTime, cfg.CancunTime = &at, &at
		},
		beaconRoot: true,
		fcu:        fcuV3,
		getPayload: func(api *ConsensusAPI, id engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
			return api.GetPayloadV3(id)
		},
		newPayload: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error) {
			return api.NewPayloadV3(context.Background(), *e.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot)
		},
	},
	{
		name: "prague",
		activate: func(cfg *params.ChainConfig, at uint64) {
			cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime = &at, &at, &at
		},
		beaconRoot: true,
		fcu:        fcuV3,
		getPayload: func(api *ConsensusAPI, id engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
			return api.GetPayloadV4(id)
		},
		newPayload: newPayloadV4,
	},
	{
		// Osaka reuses prague's fcu and newPayload but is the only fork that
		// getPayloadV5 admits, so it is not redundant with prague here the way
		// it is for the witness endpoints.
		name: "osaka",
		activate: func(cfg *params.ChainConfig, at uint64) {
			cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime = &at, &at, &at
			cfg.OsakaTime = &at
		},
		beaconRoot: true,
		fcu:        fcuV3,
		getPayload: func(api *ConsensusAPI, id engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
			return api.GetPayloadV5(id)
		},
		newPayload: newPayloadV4,
	},
	{
		name: "amsterdam",
		activate: func(cfg *params.ChainConfig, at uint64) {
			cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime = &at, &at, &at
			cfg.OsakaTime, cfg.AmsterdamTime = &at, &at
		},
		beaconRoot: true,
		slotAndGas: true,
		fcu: func(api *ConsensusAPI, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
			return api.ForkchoiceUpdatedV4(context.Background(), state, attrs, nil)
		},
		getPayload: func(api *ConsensusAPI, id engine.PayloadID) (*engine.ExecutionPayloadEnvelope, error) {
			return api.GetPayloadV6(id)
		},
		newPayload: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error) {
			return api.NewPayloadV5(context.Background(), *e.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot, requestsOf(e))
		},
	},
}

// TestEngineAPIAcrossForks drives every fork through the block lifecycle the
// consensus client performs: forkchoiceUpdated starts a build, getPayload
// collects it, newPayload imports it, and a second forkchoiceUpdated makes it
// the head.
func TestEngineAPIAcrossForks(t *testing.T) {
	for _, fork := range engineForks {
		t.Run(fork.name, func(t *testing.T) {
			testEngineRoundtrip(t, fork)
		})
	}
}

func testEngineRoundtrip(t *testing.T, fork engineFork) {
	genesis, blocks := forkTestChain(t, 10)

	// Activate the fork just past the chain we preload, so the block we build is
	// the first one under it. Only the config is touched here: it is not part of
	// the genesis hash, so the already-generated blocks still chain onto it.
	forkTime := blocks[len(blocks)-2].Time() + 5
	fork.activate(genesis.Config, forkTime)
	genesis.Config.BlobScheduleConfig = params.DefaultBlobSchedule

	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()

	api := newConsensusAPIWithoutHeartbeat(ethservice)

	// The pool is primed so the builder has work, but the block's contents are
	// deliberately not asserted: the versioned getPayload resolves to whatever
	// the builder had ready and stops it, so the transaction count is a race.
	// Block contents under each fork are covered by TestWitnessAPIsAcrossForks,
	// which resolves the full payload.
	ethservice.TxPool().Add(blocks[9].Transactions(), true)

	parent := blocks[8]
	attrs := &engine.PayloadAttributes{
		Timestamp:   parent.Time() + 5,
		Withdrawals: make([]*types.Withdrawal, 0),
	}
	if fork.beaconRoot {
		attrs.BeaconRoot = &witnessBeaconRoot
	}
	if fork.slotAndGas {
		attrs.SlotNumber = &witnessSlotNumber
		attrs.TargetGasLimit = &witnessGasTarget
	}
	state := engine.ForkchoiceStateV1{
		HeadBlockHash: parent.Hash(),
	}
	resp, err := fork.fcu(api, state, attrs)
	if err != nil {
		t.Fatalf("forkchoiceUpdated: %v", err)
	}
	if resp.PayloadStatus.Status != engine.VALID {
		t.Fatalf("forkchoiceUpdated status %q, want %q", resp.PayloadStatus.Status, engine.VALID)
	}
	if resp.PayloadID == nil {
		t.Fatal("forkchoiceUpdated returned no payload id")
	}
	envelope, err := fork.getPayload(api, *resp.PayloadID)
	if err != nil {
		t.Fatalf("getPayload: %v", err)
	}
	payload := envelope.ExecutionPayload
	if payload.ParentHash != parent.Hash() {
		t.Fatalf("built on %x, want %x", payload.ParentHash, parent.Hash())
	}
	// Import the block the way a node that did not build it would.
	status, err := fork.newPayload(api, envelope)
	if err != nil {
		t.Fatalf("newPayload: %v", err)
	}
	if status.Status != engine.VALID {
		t.Fatalf("newPayload status %q, want %q (%v)", status.Status, engine.VALID, status.ValidationError)
	}
	if status.LatestValidHash == nil || *status.LatestValidHash != payload.BlockHash {
		t.Fatalf("newPayload latest valid hash = %v, want %x", status.LatestValidHash, payload.BlockHash)
	}
	// Importing alone must not move the head; only the forkchoice call does.
	if head := ethservice.BlockChain().CurrentBlock().Hash(); head != parent.Hash() {
		t.Fatalf("head moved on newPayload alone: %x, want %x", head, parent.Hash())
	}
	// Now adopt it, and check it really became canonical.
	resp, err = fork.fcu(api, engine.ForkchoiceStateV1{HeadBlockHash: payload.BlockHash}, nil)
	if err != nil {
		t.Fatalf("forkchoiceUpdated (adopt): %v", err)
	}
	if resp.PayloadStatus.Status != engine.VALID {
		t.Fatalf("forkchoiceUpdated (adopt) status %q, want %q", resp.PayloadStatus.Status, engine.VALID)
	}
	head := ethservice.BlockChain().CurrentBlock()
	if head.Hash() != payload.BlockHash {
		t.Fatalf("head = %x, want %x", head.Hash(), payload.BlockHash)
	}
	if head.Number.Uint64() != parent.NumberU64()+1 {
		t.Fatalf("head number = %d, want %d", head.Number.Uint64(), parent.NumberU64()+1)
	}
}
