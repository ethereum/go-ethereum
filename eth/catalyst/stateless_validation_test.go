package catalyst

import (
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

// TestStatelessRejectsForgedAccessList checks that stateless execution validates
// the block access list against re-execution. Stateless execution derives the
// post-state from the received access list, so a payload whose list does not
// match execution must be rejected, the same as engine_newPayloadV5.
func TestStatelessRejectsForgedAccessList(t *testing.T) {
	fork := witnessForks[len(witnessForks)-1]
	if fork.name != "amsterdam" {
		t.Fatalf("unexpected fork %s", fork.name)
	}
	genesis, blocks := forkTestChain(t, 10)
	forkTime := blocks[len(blocks)-2].Time() + 5
	fork.activate(genesis.Config, forkTime)
	genesis.Config.BlobScheduleConfig = params.DefaultBlobSchedule

	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()
	api := newConsensusAPIWithoutHeartbeat(ethservice)
	ethservice.TxPool().Add(blocks[9].Transactions(), true)

	attrs := &engine.PayloadAttributes{
		Timestamp:      blocks[8].Time() + 5,
		Withdrawals:    make([]*types.Withdrawal, 0),
		BeaconRoot:     &witnessBeaconRoot,
		SlotNumber:     &witnessSlotNumber,
		TargetGasLimit: &witnessGasTarget,
	}
	resp, err := fork.fcu(api, engine.ForkchoiceStateV1{HeadBlockHash: blocks[8].Hash()}, attrs)
	if err != nil || resp.PayloadID == nil {
		t.Fatalf("fcu: %v", err)
	}
	envelope, err := api.getPayload(*resp.PayloadID, true, nil, nil)
	if err != nil || envelope.Witness == nil {
		t.Fatalf("getPayload: %v", err)
	}
	requests := requestsOf(envelope)

	// The unmodified payload must validate statelessly and return the honest root.
	ok, err := api.ExecuteStatelessPayloadV5(clearRoots(*envelope.ExecutionPayload), []common.Hash{}, &witnessBeaconRoot, requests, *envelope.Witness)
	if err != nil {
		t.Fatalf("stateless (honest): %v", err)
	}
	if ok.Status != engine.VALID {
		t.Fatalf("honest payload: got %s, want VALID", ok.Status)
	}
	if ok.StateRoot != envelope.ExecutionPayload.StateRoot {
		t.Fatalf("honest payload: stateRoot %x, want %x", ok.StateRoot, envelope.ExecutionPayload.StateRoot)
	}

	// Forge one balance in the access list. Nothing later in the block reads it,
	// so execution is unchanged; only the list disagrees with re-execution.
	var al bal.BlockAccessList
	if err := rlp.DecodeBytes(envelope.ExecutionPayload.BlockAccessList, &al); err != nil {
		t.Fatal(err)
	}
	forged := false
	for i := range al {
		if len(al[i].BalanceChanges) > 0 {
			last := len(al[i].BalanceChanges) - 1
			al[i].BalanceChanges[last].PostBalance = uint256.NewInt(1).Lsh(uint256.NewInt(1), 90)
			forged = true
			break
		}
	}
	if !forged {
		t.Skip("no balance change to forge in this block")
	}
	enc, err := rlp.EncodeToBytes(&al)
	if err != nil {
		t.Fatal(err)
	}
	payload := clearRoots(*envelope.ExecutionPayload)
	payload.BlockAccessList = enc

	// The forged payload must be rejected, not accepted with a forged root.
	res, err := api.ExecuteStatelessPayloadV5(payload, []common.Hash{}, &witnessBeaconRoot, requests, *envelope.Witness)
	if err != nil {
		t.Fatalf("stateless (forged): %v", err)
	}
	if res.Status != engine.INVALID {
		t.Errorf("forged access list: got %s (root %x), want INVALID", res.Status, res.StateRoot)
	}
}

// clearRoots blanks the state and receipts roots, as a stateless payload requires
// (the runner computes and returns them).
func clearRoots(p engine.ExecutableData) engine.ExecutableData {
	p.StateRoot = common.Hash{}
	p.ReceiptsRoot = common.Hash{}
	return p
}

// TestStatelessRejectsForgedRequests checks that stateless execution validates the
// execution requests against re-execution. The requests are not reflected in the
// state or receipt root that stateless execution returns, so a payload whose
// requests disagree with execution must be rejected, as engine_newPayload does.
func TestStatelessRejectsForgedRequests(t *testing.T) {
	for _, fork := range witnessForks {
		if fork.name != "prague" && fork.name != "amsterdam" {
			continue
		}
		t.Run(fork.name, func(t *testing.T) {
			genesis, blocks := forkTestChain(t, 10)
			fork.activate(genesis.Config, blocks[len(blocks)-2].Time()+5)
			genesis.Config.BlobScheduleConfig = params.DefaultBlobSchedule

			n, ethservice := startEthService(t, genesis, blocks[:9])
			defer n.Close()
			api := newConsensusAPIWithoutHeartbeat(ethservice)
			ethservice.TxPool().Add(blocks[9].Transactions(), true)

			attrs := &engine.PayloadAttributes{
				Timestamp:   blocks[8].Time() + 5,
				Withdrawals: make([]*types.Withdrawal, 0),
				BeaconRoot:  &witnessBeaconRoot,
			}
			if fork.slotAndGas {
				attrs.SlotNumber = &witnessSlotNumber
				attrs.TargetGasLimit = &witnessGasTarget
			}
			resp, err := fork.fcu(api, engine.ForkchoiceStateV1{HeadBlockHash: blocks[8].Hash()}, attrs)
			if err != nil || resp.PayloadID == nil {
				t.Fatalf("fcu: %v", err)
			}
			envelope, err := api.getPayload(*resp.PayloadID, true, nil, nil)
			if err != nil || envelope.Witness == nil {
				t.Fatalf("getPayload: %v", err)
			}
			// The unmodified payload must validate statelessly and return the honest roots.
			assertStatelessRoots(t, api, fork, envelope, *envelope.Witness, "honest")

			// Add a withdrawal request (EIP-7002) that execution did not produce: type
			// 0x01 followed by a source address, a validator pubkey and an amount.
			fake := append([]byte{0x01}, make([]byte, 20+48+8)...)
			fake[1] = 0xaa
			forged := *envelope
			forged.Requests = append(append([][]byte{}, envelope.Requests...), fake)
			payload := clearRoots(*envelope.ExecutionPayload)
			forged.ExecutionPayload = &payload

			res, err := fork.stateless(api, &forged, *envelope.Witness)
			if err != nil {
				t.Fatalf("stateless (forged): %v", err)
			}
			if res.Status != engine.INVALID {
				t.Errorf("forged requests: got %s (roots %x %x), want INVALID", res.Status, res.StateRoot, res.ReceiptsRoot)
			}
		})
	}
}
