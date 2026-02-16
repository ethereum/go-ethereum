package catalyst

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
)

func TestBuildBlockV1(t *testing.T) {
	genesis, blocks := generateMergeChain(5, true)
	n, ethservice := startEthService(t, genesis, blocks)
	defer n.Close()

	parent := ethservice.BlockChain().CurrentBlock()
	attrs := engine.PayloadAttributes{
		Timestamp:             parent.Time + 1,
		Random:                crypto.Keccak256Hash([]byte("test")),
		SuggestedFeeRecipient: parent.Coinbase,
		Withdrawals:           nil,
		BeaconRoot:            nil,
	}

	currentNonce, _ := ethservice.APIBackend.GetPoolNonce(context.Background(), testAddr)
	tx, _ := types.SignTx(types.NewTransaction(currentNonce, testAddr, big.NewInt(1), params.TxGas, big.NewInt(params.InitialBaseFee*2), nil), types.LatestSigner(ethservice.BlockChain().Config()), testKey)

	api := NewTestingAPI(ethservice)

	t.Run("buildOnCurrentHead", func(t *testing.T) {
		envelope, err := api.BuildBlockV1(parent.Hash(), attrs, nil, nil)
		if err != nil {
			t.Fatalf("BuildBlockV1 failed: %v", err)
		}
		if envelope == nil || envelope.ExecutionPayload == nil {
			t.Fatal("expected non-nil envelope and payload")
		}
		payload := envelope.ExecutionPayload
		if payload.ParentHash != parent.Hash() {
			t.Errorf("parent hash mismatch: got %x want %x", payload.ParentHash, parent.Hash())
		}
		if payload.Number != parent.Number.Uint64()+1 {
			t.Errorf("block number mismatch: got %d want %d", payload.Number, parent.Number.Uint64()+1)
		}
		if payload.Timestamp != attrs.Timestamp {
			t.Errorf("timestamp mismatch: got %d want %d", payload.Timestamp, attrs.Timestamp)
		}
		if payload.FeeRecipient != attrs.SuggestedFeeRecipient {
			t.Errorf("fee recipient mismatch: got %x want %x", payload.FeeRecipient, attrs.SuggestedFeeRecipient)
		}
	})

	t.Run("wrongParentHash", func(t *testing.T) {
		wrongParent := common.Hash{0x01}
		_, err := api.BuildBlockV1(wrongParent, attrs, nil, nil)
		if err == nil {
			t.Fatal("expected error when parentHash is not current head")
		}
		if err.Error() != "parentHash is not current head" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("buildEmptyBlock", func(t *testing.T) {
		emptyTxs := []hexutil.Bytes{}
		envelope, err := api.BuildBlockV1(parent.Hash(), attrs, &emptyTxs, nil)
		if err != nil {
			t.Fatalf("BuildBlockV1 with empty txs failed: %v", err)
		}
		if envelope == nil || envelope.ExecutionPayload == nil {
			t.Fatal("expected non-nil envelope and payload")
		}
		if len(envelope.ExecutionPayload.Transactions) != 0 {
			t.Errorf("expected empty block, got %d transactions", len(envelope.ExecutionPayload.Transactions))
		}
	})

	t.Run("buildBlockWithTransactions", func(t *testing.T) {
		enc, _ := tx.MarshalBinary()
		txs := []hexutil.Bytes{enc}
		envelope, err := api.BuildBlockV1(parent.Hash(), attrs, &txs, nil)
		if err != nil {
			t.Fatalf("BuildBlockV1 with transaction failed: %v", err)
		}
		if len(envelope.ExecutionPayload.Transactions) != 1 {
			t.Errorf("expected 1 transaction, got %d", len(envelope.ExecutionPayload.Transactions))
		}
	})

	t.Run("buildBlockWithTransactionsFromTxPool", func(t *testing.T) {
		ethservice.TxPool().Add([]*types.Transaction{tx}, true)
		envelope, err := api.BuildBlockV1(parent.Hash(), attrs, nil, nil)
		if err != nil {
			t.Fatalf("BuildBlockV1 with transaction failed: %v", err)
		}
		if len(envelope.ExecutionPayload.Transactions) != 1 {
			t.Errorf("expected 1 transaction, got %d", len(envelope.ExecutionPayload.Transactions))
		}
	})
}
