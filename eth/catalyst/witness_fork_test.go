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
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var (
	witnessBeaconRoot = common.Hash{42}
	witnessSlotNumber = uint64(1)
	witnessGasTarget  = uint64(15_000_000)
)

// witnessFork pins one fork to the engine API versions that serve it. The
// request/response plumbing differs per version, so each fork supplies the three
// calls; everything else about driving a payload is shared by testWitnessRoundtrip.
type witnessFork struct {
	name string

	// activate turns the fork on at the given timestamp. Later forks must also
	// enable the earlier ones.
	activate func(cfg *params.ChainConfig, at uint64)

	// version selects the payload variant the built block is retrieved as.
	version engine.PayloadVersion

	// beaconRoot and slotAndGas mark the optional payload attributes the fork's
	// FCU version requires.
	beaconRoot bool // cancun and later
	slotAndGas bool // amsterdam and later

	fcu        func(*ConsensusAPI, engine.ForkchoiceStateV1, *engine.PayloadAttributes) (engine.ForkChoiceResponse, error)
	newPayload func(*ConsensusAPI, *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error)
	stateless  func(*ConsensusAPI, *engine.ExecutionPayloadEnvelope, hexutil.Bytes) (engine.StatelessPayloadStatusV1, error)
}

// requestsOf returns the envelope's execution requests in the shape the V4/V5
// endpoints expect: non-nil even when the block produced none, since those
// endpoints reject a nil list post-prague.
func requestsOf(envelope *engine.ExecutionPayloadEnvelope) []hexutil.Bytes {
	reqs := make([]hexutil.Bytes, len(envelope.Requests))
	for i, r := range envelope.Requests {
		reqs[i] = r
	}
	return reqs
}

var witnessForks = []witnessFork{
	{
		name:     "shanghai",
		activate: func(cfg *params.ChainConfig, at uint64) { cfg.ShanghaiTime = &at },
		version:  engine.PayloadV2,
		fcu: func(api *ConsensusAPI, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
			return api.ForkchoiceUpdatedWithWitnessV2(context.Background(), state, attrs)
		},
		newPayload: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error) {
			return api.NewPayloadWithWitnessV2(context.Background(), *e.ExecutionPayload)
		},
		stateless: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope, w hexutil.Bytes) (engine.StatelessPayloadStatusV1, error) {
			return api.ExecuteStatelessPayloadV2(*e.ExecutionPayload, w)
		},
	},
	{
		name: "cancun",
		activate: func(cfg *params.ChainConfig, at uint64) {
			cfg.ShanghaiTime, cfg.CancunTime = &at, &at
		},
		version:    engine.PayloadV3,
		beaconRoot: true,
		fcu: func(api *ConsensusAPI, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
			return api.ForkchoiceUpdatedWithWitnessV3(context.Background(), state, attrs)
		},
		newPayload: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error) {
			return api.NewPayloadWithWitnessV3(context.Background(), *e.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot)
		},
		stateless: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope, w hexutil.Bytes) (engine.StatelessPayloadStatusV1, error) {
			return api.ExecuteStatelessPayloadV3(*e.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot, w)
		},
	},
	{
		name: "prague",
		activate: func(cfg *params.ChainConfig, at uint64) {
			cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime = &at, &at, &at
		},
		version:    engine.PayloadV3,
		beaconRoot: true,
		fcu: func(api *ConsensusAPI, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
			return api.ForkchoiceUpdatedWithWitnessV3(context.Background(), state, attrs)
		},
		newPayload: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error) {
			return api.NewPayloadWithWitnessV4(context.Background(), *e.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot, requestsOf(e))
		},
		stateless: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope, w hexutil.Bytes) (engine.StatelessPayloadStatusV1, error) {
			return api.ExecuteStatelessPayloadV4(*e.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot, requestsOf(e), w)
		},
	},
	{
		name: "amsterdam",
		activate: func(cfg *params.ChainConfig, at uint64) {
			cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime = &at, &at, &at
			cfg.OsakaTime, cfg.AmsterdamTime = &at, &at
		},
		version:    engine.PayloadV4,
		beaconRoot: true,
		slotAndGas: true,
		fcu: func(api *ConsensusAPI, state engine.ForkchoiceStateV1, attrs *engine.PayloadAttributes) (engine.ForkChoiceResponse, error) {
			return api.ForkchoiceUpdatedWithWitnessV4(context.Background(), state, attrs, nil)
		},
		newPayload: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope) (engine.PayloadStatusV1, error) {
			return api.NewPayloadWithWitnessV5(context.Background(), *e.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot, requestsOf(e))
		},
		stateless: func(api *ConsensusAPI, e *engine.ExecutionPayloadEnvelope, w hexutil.Bytes) (engine.StatelessPayloadStatusV1, error) {
			return api.ExecuteStatelessPayloadV5(*e.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot, requestsOf(e), w)
		},
	},
}

// TestWitnessAPIsAcrossForks drives every fork through the full witness path:
// forkchoiceUpdatedWithWitness builds a block, newPayloadWithWitness re-executes
// it, and executeStatelessPayload replays it from each of the two witnesses with
// no state of its own.
func TestWitnessAPIsAcrossForks(t *testing.T) {
	for _, fork := range witnessForks {
		t.Run(fork.name, func(t *testing.T) {
			testWitnessRoundtrip(t, fork)
		})
	}
}

func testWitnessRoundtrip(t *testing.T, fork witnessFork) {
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

	// Give the builder something to include, so the witness is not trivially empty.
	ethservice.TxPool().Add(blocks[9].Transactions(), true)

	attrs := &engine.PayloadAttributes{
		Timestamp:   blocks[8].Time() + 5,
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
		HeadBlockHash: blocks[8].Hash(),
	}
	resp, err := fork.fcu(api, state, attrs)
	if err != nil {
		t.Fatalf("forkchoiceUpdatedWithWitness: %v", err)
	}
	if resp.PayloadID == nil {
		t.Fatal("forkchoiceUpdatedWithWitness returned no payload id")
	}
	envelope, err := api.getPayload(*resp.PayloadID, true, nil, nil)
	if err != nil {
		t.Fatalf("getPayload: %v", err)
	}
	if envelope.Witness == nil {
		t.Fatal("built payload carries no witness")
	}
	if have, want := len(envelope.ExecutionPayload.Transactions), blocks[9].Transactions().Len(); have != want {
		t.Fatalf("built %d transactions, want %d", have, want)
	}
	// The witness handed out at build time must replay the block on its own.
	assertStatelessRoots(t, api, fork, envelope, *envelope.Witness, "build")

	// Importing the block must produce a witness of its own, which must replay
	// the block just as well. This is the path a node takes for a block it did
	// not build, so a regression here is invisible to the build-side check above.
	status, err := fork.newPayload(api, envelope)
	if err != nil {
		t.Fatalf("newPayloadWithWitness: %v", err)
	}
	if status.Status != engine.VALID {
		t.Fatalf("newPayloadWithWitness status %q, want %q", status.Status, engine.VALID)
	}
	if status.Witness == nil {
		t.Fatal("imported payload carries no witness")
	}
	assertStatelessRoots(t, api, fork, envelope, *status.Witness, "import")
}

// assertStatelessRoots replays the payload from the given witness with the state
// and receipt roots blanked, so that a stateless run that quietly echoed the
// input back cannot pass, and checks it recomputes the originals.
func assertStatelessRoots(t *testing.T, api *ConsensusAPI, fork witnessFork, envelope *engine.ExecutionPayloadEnvelope, witness hexutil.Bytes, side string) {
	t.Helper()

	wantState := envelope.ExecutionPayload.StateRoot
	wantReceipts := envelope.ExecutionPayload.ReceiptsRoot

	envelope.ExecutionPayload.StateRoot = common.Hash{}
	envelope.ExecutionPayload.ReceiptsRoot = common.Hash{}
	defer func() {
		envelope.ExecutionPayload.StateRoot = wantState
		envelope.ExecutionPayload.ReceiptsRoot = wantReceipts
	}()

	res, err := fork.stateless(api, envelope, witness)
	if err != nil {
		t.Fatalf("executeStatelessPayload (%s witness): %v", side, err)
	}
	if res.Status != engine.VALID {
		t.Fatalf("executeStatelessPayload (%s witness) status %q, want %q", side, res.Status, engine.VALID)
	}
	if res.StateRoot != wantState {
		t.Errorf("stateless state root (%s witness) = %x, want %x", side, res.StateRoot, wantState)
	}
	if res.ReceiptsRoot != wantReceipts {
		t.Errorf("stateless receipt root (%s witness) = %x, want %x", side, res.ReceiptsRoot, wantReceipts)
	}
}

// forkTestChain builds a merged chain of n blocks, each carrying one transaction
// so that the payloads under test are not empty. The genesis comes from
// generateMergeChain, which pre-deploys the system contracts every post-shanghai
// fork calls into.
func forkTestChain(t *testing.T, n int) (*core.Genesis, []*types.Block) {
	t.Helper()

	genesis, _ := generateMergeChain(0, true)
	var nonce uint64
	_, blocks, _ := core.GenerateChainWithGenesis(genesis, beacon.New(ethash.NewFaker()), n, func(i int, g *core.BlockGen) {
		g.OffsetTime(5)
		tx, err := types.SignTx(
			types.NewTransaction(nonce, common.Address{0xaa}, common.Big1, params.TxGas, big.NewInt(params.InitialBaseFee*2), nil),
			types.LatestSigner(genesis.Config), testKey)
		if err != nil {
			t.Fatalf("signing chain tx: %v", err)
		}
		g.AddTx(tx)
		nonce++
	})
	return genesis, blocks
}
