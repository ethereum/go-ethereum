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
	"bytes"
	"context"
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

// bogotaTestChain returns the fork test chain with Bogota active from the
// block after blocks[8].
func bogotaTestChain(t *testing.T) (*core.Genesis, []*types.Block) {
	genesis, blocks := forkTestChain(t, 10)
	at := blocks[8].Time() + 5
	cfg := genesis.Config
	cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime, cfg.OsakaTime = &at, &at, &at, &at
	cfg.AmsterdamTime, cfg.BogotaTime = &at, &at
	cfg.BlobScheduleConfig = params.DefaultBlobSchedule
	return genesis, blocks
}

// buildBogotaPayload builds a Bogota payload on parent, which only holds the
// inclusion list transactions as the pool is empty.
func buildBogotaPayload(t *testing.T, api *ConsensusAPI, parent *types.Block, list ...[]byte) *engine.ExecutionPayloadEnvelope {
	t.Helper()
	attrs := &engine.PayloadAttributes{
		Timestamp:      parent.Time() + 5,
		Withdrawals:    make([]*types.Withdrawal, 0),
		BeaconRoot:     &witnessBeaconRoot,
		SlotNumber:     &witnessSlotNumber,
		TargetGasLimit: &witnessGasTarget,
		InclusionList:  append([][]byte{}, list...),
	}
	resp, err := api.ForkchoiceUpdatedV5(context.Background(), engine.ForkchoiceStateV1{HeadBlockHash: parent.Hash()}, attrs, nil)
	if err != nil {
		t.Fatalf("forkchoiceUpdated: %v", err)
	}
	if resp.PayloadID == nil {
		t.Fatal("forkchoiceUpdated returned no payload id")
	}
	envelope, err := api.GetPayloadV6(*resp.PayloadID)
	if err != nil {
		t.Fatalf("getPayload: %v", err)
	}
	return envelope
}

func newPayloadV6(api *ConsensusAPI, payload engine.ExecutableData, requests []hexutil.Bytes, list []hexutil.Bytes) (engine.PayloadStatusV2, error) {
	return api.NewPayloadV6(context.Background(), payload, []common.Hash{}, &witnessBeaconRoot, requests, list)
}

// testInclusionList signs a transfer from the test account for an inclusion list.
func testInclusionList(t *testing.T, cfg *params.ChainConfig, nonce uint64) []hexutil.Bytes {
	t.Helper()
	tx, err := types.SignTx(types.NewTransaction(nonce, common.Address{0xaa}, common.Big1, params.TxGas, big.NewInt(params.InitialBaseFee*2), nil), types.LatestSigner(cfg), testKey)
	if err != nil {
		t.Fatal(err)
	}
	enc, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return []hexutil.Bytes{enc}
}

// TestNewPayloadV6InclusionList checks that newPayloadV6 gives the inclusion
// list verdict for valid payloads only.
func TestNewPayloadV6InclusionList(t *testing.T) {
	genesis, blocks := bogotaTestChain(t)
	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()
	api := newConsensusAPIWithoutHeartbeat(ethservice)

	var (
		envelope = buildBogotaPayload(t, api, blocks[8])
		payload  = *envelope.ExecutionPayload
		requests = requestsOf(envelope)

		// The payload is empty, so the test account's next nonce is 9.
		includable = testInclusionList(t, genesis.Config, 9)
		excluded   = testInclusionList(t, genesis.Config, 10)
	)
	tampered := payload
	tampered.StateRoot = common.Hash{0x01}

	check := func(step string, status engine.PayloadStatusV2, wantStatus string, want *bool) {
		t.Helper()
		if status.Status != wantStatus {
			t.Fatalf("%s: status %s, want %s", step, status.Status, wantStatus)
		}
		switch got := status.InclusionListSatisfied; {
		case want == nil && got != nil:
			t.Errorf("%s: inclusionListSatisfied %v, want null", step, *got)
		case want != nil && (got == nil || *got != *want):
			t.Errorf("%s: inclusionListSatisfied %v, want %v", step, got, *want)
		}
	}
	satisfied, unsatisfied := true, false

	status, err := newPayloadV6(api, payload, requests, includable)
	if err != nil {
		t.Fatal(err)
	}
	check("valid payload", status, engine.VALID, &unsatisfied)

	// A payload that doesn't match its hash gets no verdict, and doesn't replace
	// the list kept for the block with that hash.
	status, err = newPayloadV6(api, tampered, requests, excluded)
	if err != nil {
		t.Fatal(err)
	}
	check("invalid payload", status, engine.INVALID, nil)

	resp, err := api.ForkchoiceUpdatedV5(context.Background(), engine.ForkchoiceStateV1{HeadBlockHash: payload.BlockHash}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	check("forkchoice update", resp.PayloadStatus, engine.VALID, &unsatisfied)

	// The block is known now, a new list still gets checked.
	status, err = newPayloadV6(api, payload, requests, excluded)
	if err != nil {
		t.Fatal(err)
	}
	check("known payload, other list", status, engine.VALID, &satisfied)
}

// TestForkchoiceUpdatedV5InclusionList checks that a payload which couldn't be
// validated on arrival gets its verdict from the forkchoice update that later
// finds it valid.
func TestForkchoiceUpdatedV5InclusionList(t *testing.T) {
	genesis, blocks := bogotaTestChain(t)

	// The payload is built on a node with the whole chain...
	n1, eth1 := startEthService(t, genesis, blocks[:9])
	defer n1.Close()
	envelope := buildBogotaPayload(t, newConsensusAPIWithoutHeartbeat(eth1), blocks[8])

	// ...and handed to one that is a block behind.
	n2, eth2 := startEthService(t, genesis, blocks[:8])
	defer n2.Close()
	api := newConsensusAPIWithoutHeartbeat(eth2)

	status, err := newPayloadV6(api, *envelope.ExecutionPayload, requestsOf(envelope), testInclusionList(t, genesis.Config, 9))
	if err != nil {
		t.Fatal(err)
	}
	if status.Status == engine.VALID || status.InclusionListSatisfied != nil {
		t.Fatalf("payload on missing parent: status %s, inclusionListSatisfied %v", status.Status, status.InclusionListSatisfied)
	}
	// Sync brings the parent and the payload in.
	block, err := engine.ExecutableDataToBlock(*envelope.ExecutionPayload, []common.Hash{}, &witnessBeaconRoot, envelope.Requests)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := eth2.BlockChain().InsertChain(types.Blocks{blocks[8], block}); err != nil {
		t.Fatal(err)
	}
	resp, err := api.ForkchoiceUpdatedV5(context.Background(), engine.ForkchoiceStateV1{HeadBlockHash: block.Hash()}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.PayloadStatus.Status != engine.VALID {
		t.Fatalf("forkchoiceUpdated status %s, want VALID", resp.PayloadStatus.Status)
	}
	if got := resp.PayloadStatus.InclusionListSatisfied; got == nil || *got {
		t.Errorf("inclusionListSatisfied %v, want false", got)
	}
}

// TestForkchoiceUpdatedV5BuildsInclusionList checks that a payload built from
// forkchoiceUpdatedV5 attributes satisfies their inclusion list.
func TestForkchoiceUpdatedV5BuildsInclusionList(t *testing.T) {
	genesis, blocks := bogotaTestChain(t)
	n, ethservice := startEthService(t, genesis, blocks[:9])
	defer n.Close()
	api := newConsensusAPIWithoutHeartbeat(ethservice)

	list := testInclusionList(t, genesis.Config, 9)
	envelope := buildBogotaPayload(t, api, blocks[8], list[0])
	status, err := newPayloadV6(api, *envelope.ExecutionPayload, requestsOf(envelope), list)
	if err != nil {
		t.Fatal(err)
	}
	if got := status.InclusionListSatisfied; status.Status != engine.VALID || got == nil || !*got {
		t.Fatalf("status %s, inclusionListSatisfied %v, want VALID and true", status.Status, got)
	}
}

// TestGetInclusionListV1 checks that the inclusion list stays within the size
// limit and leaves out transactions that can't pay the next block's base fee.
func TestGetInclusionListV1(t *testing.T) {
	genesis, _ := generateMergeChain(0, true)
	keys := make([]*ecdsa.PrivateKey, 13)
	for i := range keys {
		keys[i], _ = crypto.GenerateKey()
		genesis.Alloc[crypto.PubkeyToAddress(keys[i].PublicKey)] = types.Account{Balance: big.NewInt(params.Ether)}
	}
	n, ethservice := startEthService(t, genesis, nil)
	defer n.Close()
	api := newConsensusAPIWithoutHeartbeat(ethservice)

	signer := types.LatestSigner(genesis.Config)
	sign := func(key *ecdsa.PrivateKey, feeCap int64) *types.Transaction {
		return types.MustSignNewTx(key, signer, &types.DynamicFeeTx{
			ChainID:   genesis.Config.ChainID,
			GasTipCap: big.NewInt(params.GWei / 1000),
			GasFeeCap: big.NewInt(feeCap),
			Gas:       200_000,
			To:        &common.Address{0xaa},
			Data:      bytes.Repeat([]byte{0xff}, 1000),
		})
	}
	// The oldest transaction can't pay the base fee, the others take up more
	// room together than the list has.
	underpriced := sign(keys[0], params.GWei/100)
	txs := []*types.Transaction{underpriced}
	for _, key := range keys[1:] {
		txs = append(txs, sign(key, 2*params.GWei))
	}
	for i, err := range ethservice.TxPool().Add(txs, true) {
		if err != nil {
			t.Fatalf("tx %d: %v", i, err)
		}
	}
	list, err := api.GetInclusionListV1()
	if err != nil {
		t.Fatal(err)
	}
	var size int
	for _, enc := range list {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(enc); err != nil {
			t.Fatal(err)
		}
		if tx.Hash() == underpriced.Hash() {
			t.Error("underpriced transaction on the list")
		}
		size += len(enc)
	}
	if len(list) == 0 || size > maxInclusionListBytes {
		t.Fatalf("list of %d transactions, %d bytes", len(list), size)
	}
}
