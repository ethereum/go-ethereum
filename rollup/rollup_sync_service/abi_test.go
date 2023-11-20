package rollup_sync_service

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/crypto"
)

func TestEventSignatures(t *testing.T) {
	scrollChainABI, err := scrollChainMetaData.GetAbi()
	if err != nil {
		t.Fatal("failed to get scroll chain abi", "err", err)
	}

	assert.Equal(t, crypto.Keccak256Hash([]byte("CommitBatch(uint256,bytes32)")), scrollChainABI.Events["CommitBatch"].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("RevertBatch(uint256,bytes32)")), scrollChainABI.Events["RevertBatch"].ID)
	assert.Equal(t, crypto.Keccak256Hash([]byte("FinalizeBatch(uint256,bytes32,bytes32,bytes32)")), scrollChainABI.Events["FinalizeBatch"].ID)
}

func TestUnpackLog(t *testing.T) {
	scrollChainABI, err := scrollChainMetaData.GetAbi()
	require.NoError(t, err)

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
			"CommitBatch",
			types.Log{
				Data:   []byte{},
				Topics: []common.Hash{scrollChainABI.Events["CommitBatch"].ID, common.BigToHash(mockBatchIndex), mockBatchHash},
			},
			&L1CommitBatchEvent{BatchIndex: mockBatchIndex, BatchHash: mockBatchHash},
			&L1CommitBatchEvent{},
		},
		{
			"RevertBatch",
			types.Log{
				Data:   []byte{},
				Topics: []common.Hash{scrollChainABI.Events["RevertBatch"].ID, common.BigToHash(mockBatchIndex), mockBatchHash},
			},
			&L1RevertBatchEvent{BatchIndex: mockBatchIndex, BatchHash: mockBatchHash},
			&L1RevertBatchEvent{},
		},
		{
			"FinalizeBatch",
			types.Log{
				Data:   append(mockStateRoot.Bytes(), mockWithdrawRoot.Bytes()...),
				Topics: []common.Hash{scrollChainABI.Events["FinalizeBatch"].ID, common.BigToHash(mockBatchIndex), mockBatchHash},
			},
			&L1FinalizeBatchEvent{
				BatchIndex:   mockBatchIndex,
				BatchHash:    mockBatchHash,
				StateRoot:    mockStateRoot,
				WithdrawRoot: mockWithdrawRoot,
			},
			&L1FinalizeBatchEvent{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.eventName, func(t *testing.T) {
			err := UnpackLog(scrollChainABI, tt.out, tt.eventName, tt.mockLog)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, tt.out)
		})
	}
}
