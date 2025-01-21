package l1

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
)

func TestEventSignatures(t *testing.T) {
	assert.Equal(t, crypto.Keccak256Hash([]byte("CommitBatch(uint256,bytes32)")), ScrollChainABI.Events["CommitBatch"].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("RevertBatch(uint256,bytes32)")), ScrollChainABI.Events["RevertBatch"].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("FinalizeBatch(uint256,bytes32,bytes32,bytes32)")), ScrollChainABI.Events["FinalizeBatch"].ID)
}

func TestUnpackLog(t *testing.T) {
	mockBatchIndex := big.NewInt(123)
	mockBatchHash := crypto.Keccak256Hash([]byte("mockBatch"))
	mockStateRoot := crypto.Keccak256Hash([]byte("mockStateRoot"))
	mockWithdrawRoot := crypto.Keccak256Hash([]byte("mockWithdrawRoot"))

	tests := []struct {
		eventName string
		mockLog   types.Log
		expected  interface{}
		out       interface{}
	}{
		{
			commitBatchEventName,
			types.Log{
				Data:   nil,
				Topics: []common.Hash{ScrollChainABI.Events[commitBatchEventName].ID, common.BigToHash(mockBatchIndex), mockBatchHash},
			},
			&CommitBatchEventUnpacked{
				BatchIndex: mockBatchIndex,
				BatchHash:  mockBatchHash,
			},
			&CommitBatchEventUnpacked{},
		},
		{
			revertBatchEventName,
			types.Log{
				Data:   nil,
				Topics: []common.Hash{ScrollChainABI.Events[revertBatchEventName].ID, common.BigToHash(mockBatchIndex), mockBatchHash},
			},
			&RevertBatchEventUnpacked{
				BatchIndex: mockBatchIndex,
				BatchHash:  mockBatchHash,
			},
			&RevertBatchEventUnpacked{},
		},
		{
			finalizeBatchEventName,
			types.Log{
				Data:   append(mockStateRoot.Bytes(), mockWithdrawRoot.Bytes()...),
				Topics: []common.Hash{ScrollChainABI.Events[finalizeBatchEventName].ID, common.BigToHash(mockBatchIndex), mockBatchHash},
			},
			&FinalizeBatchEventUnpacked{
				BatchIndex:   mockBatchIndex,
				BatchHash:    mockBatchHash,
				StateRoot:    mockStateRoot,
				WithdrawRoot: mockWithdrawRoot,
			},
			&FinalizeBatchEventUnpacked{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.eventName, func(t *testing.T) {
			err := UnpackLog(ScrollChainABI, tt.out, tt.eventName, tt.mockLog)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, tt.out)
		})
	}
}
