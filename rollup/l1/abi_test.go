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
	assert.Equal(t, crypto.Keccak256Hash([]byte("CommitBatch(uint256,bytes32)")), ScrollChainABI.Events[commitBatchEventName].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("RevertBatch(uint256,bytes32)")), ScrollChainABI.Events[revertBatchV0EventName].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("RevertBatch(uint256,uint256)")), ScrollChainABI.Events[revertBatchV7EventName].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("FinalizeBatch(uint256,bytes32,bytes32,bytes32)")), ScrollChainABI.Events[finalizeBatchEventName].ID)
}

func TestMethodSignatures(t *testing.T) {
	assert.Equal(t, crypto.Keccak256Hash([]byte("commitBatch(uint8,bytes,bytes[],bytes)")).Bytes()[:4], ScrollChainABI.Methods[commitBatchMethodName].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("commitBatchWithBlobProof(uint8,bytes,bytes[],bytes,bytes)")).Bytes()[:4], ScrollChainABI.Methods[commitBatchWithBlobProofMethodName].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("commitBatches(uint8,bytes32,bytes32)")).Bytes()[:4], ScrollChainABI.Methods[commitBatchesV7MethodName].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("finalizeBundlePostEuclidV2(bytes,uint256,bytes32,bytes32,bytes)")).Bytes()[:4], ScrollChainABI.Methods[finalizeBundlePostEuclidV2MethodName].ID)
}

func TestUnpackLog(t *testing.T) {
	mockBatchIndex := big.NewInt(123)
	finishMockBatchIndex := big.NewInt(125)
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
			revertBatchV0EventName,
			types.Log{
				Data:   nil,
				Topics: []common.Hash{ScrollChainABI.Events[revertBatchV0EventName].ID, common.BigToHash(mockBatchIndex), mockBatchHash},
			},
			&RevertBatchEventV0Unpacked{
				BatchIndex: mockBatchIndex,
				BatchHash:  mockBatchHash,
			},
			&RevertBatchEventV0Unpacked{},
		},
		{
			revertBatchV7EventName,
			types.Log{
				Data:   nil,
				Topics: []common.Hash{ScrollChainABI.Events[revertBatchV7EventName].ID, common.BigToHash(mockBatchIndex), common.BigToHash(mockBatchIndex)},
			},
			&RevertBatchEventV7Unpacked{
				StartBatchIndex:  mockBatchIndex,
				FinishBatchIndex: mockBatchIndex,
			},
			&RevertBatchEventV7Unpacked{},
		},
		{
			revertBatchV7EventName,
			types.Log{
				Data:   nil,
				Topics: []common.Hash{ScrollChainABI.Events[revertBatchV7EventName].ID, common.BigToHash(mockBatchIndex), common.BigToHash(finishMockBatchIndex)},
			},
			&RevertBatchEventV7Unpacked{
				StartBatchIndex:  mockBatchIndex,
				FinishBatchIndex: finishMockBatchIndex,
			},
			&RevertBatchEventV7Unpacked{},
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
