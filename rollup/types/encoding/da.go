package encoding

import (
	"fmt"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/common/hexutil"
	"github.com/scroll-tech/go-ethereum/core/types"
)

// CodecVersion defines the version of encoder and decoder.
type CodecVersion int

const (
	// CodecV0 represents the version 0 of the encoder and decoder.
	CodecV0 CodecVersion = iota

	// CodecV1 represents the version 1 of the encoder and decoder.
	CodecV1
)

// Block represents an L2 block.
type Block struct {
	Header         *types.Header
	Transactions   []*types.TransactionData
	WithdrawRoot   common.Hash           `json:"withdraw_trie_root,omitempty"`
	RowConsumption *types.RowConsumption `json:"row_consumption,omitempty"`
}

// Chunk represents a group of blocks.
type Chunk struct {
	Blocks []*Block `json:"blocks"`
}

// Batch represents a batch of chunks.
type Batch struct {
	Index                      uint64
	TotalL1MessagePoppedBefore uint64
	ParentBatchHash            common.Hash
	Chunks                     []*Chunk
}

// NumL1Messages returns the number of L1 messages in this block.
// This number is the sum of included and skipped L1 messages.
func (b *Block) NumL1Messages(totalL1MessagePoppedBefore uint64) uint64 {
	var lastQueueIndex *uint64
	for _, txData := range b.Transactions {
		if txData.Type == types.L1MessageTxType {
			lastQueueIndex = &txData.Nonce
		}
	}
	if lastQueueIndex == nil {
		return 0
	}
	// note: last queue index included before this block is totalL1MessagePoppedBefore - 1
	// TODO: cache results
	return *lastQueueIndex - totalL1MessagePoppedBefore + 1
}

// NumL2Transactions returns the number of L2 transactions in this block.
func (b *Block) NumL2Transactions() uint64 {
	var count uint64
	for _, txData := range b.Transactions {
		if txData.Type != types.L1MessageTxType {
			count++
		}
	}
	return count
}

// NumL1Messages returns the number of L1 messages in this chunk.
// This number is the sum of included and skipped L1 messages.
func (c *Chunk) NumL1Messages(totalL1MessagePoppedBefore uint64) uint64 {
	var numL1Messages uint64
	for _, block := range c.Blocks {
		numL1MessagesInBlock := block.NumL1Messages(totalL1MessagePoppedBefore)
		numL1Messages += numL1MessagesInBlock
		totalL1MessagePoppedBefore += numL1MessagesInBlock
	}
	// TODO: cache results
	return numL1Messages
}

// ConvertTxDataToRLPEncoding transforms []*TransactionData into []*types.Transaction.
func ConvertTxDataToRLPEncoding(txData *types.TransactionData) ([]byte, error) {
	data, err := hexutil.Decode(txData.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode txData.Data: data=%v, err=%w", txData.Data, err)
	}

	var tx *types.Transaction
	switch txData.Type {
	case types.LegacyTxType:
		tx = types.NewTx(&types.LegacyTx{
			Nonce:    txData.Nonce,
			To:       txData.To,
			Value:    txData.Value.ToInt(),
			Gas:      txData.Gas,
			GasPrice: txData.GasPrice.ToInt(),
			Data:     data,
			V:        txData.V.ToInt(),
			R:        txData.R.ToInt(),
			S:        txData.S.ToInt(),
		})

	case types.AccessListTxType:
		tx = types.NewTx(&types.AccessListTx{
			ChainID:    txData.ChainId.ToInt(),
			Nonce:      txData.Nonce,
			To:         txData.To,
			Value:      txData.Value.ToInt(),
			Gas:        txData.Gas,
			GasPrice:   txData.GasPrice.ToInt(),
			Data:       data,
			AccessList: txData.AccessList,
			V:          txData.V.ToInt(),
			R:          txData.R.ToInt(),
			S:          txData.S.ToInt(),
		})

	case types.DynamicFeeTxType:
		tx = types.NewTx(&types.DynamicFeeTx{
			ChainID:    txData.ChainId.ToInt(),
			Nonce:      txData.Nonce,
			To:         txData.To,
			Value:      txData.Value.ToInt(),
			Gas:        txData.Gas,
			GasTipCap:  txData.GasTipCap.ToInt(),
			GasFeeCap:  txData.GasFeeCap.ToInt(),
			Data:       data,
			AccessList: txData.AccessList,
			V:          txData.V.ToInt(),
			R:          txData.R.ToInt(),
			S:          txData.S.ToInt(),
		})

	case types.L1MessageTxType: // L1MessageTxType is not supported
	default:
		return nil, fmt.Errorf("unsupported tx type: %d", txData.Type)
	}

	rlpTxData, err := tx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal binary of the tx: tx=%v, err=%w", tx, err)
	}

	return rlpTxData, nil
}

// CrcMax calculates the maximum row consumption of crc.
func (c *Chunk) CrcMax() (uint64, error) {
	// Map sub-circuit name to row count
	crc := make(map[string]uint64)

	// Iterate over blocks, accumulate row consumption
	for _, block := range c.Blocks {
		if block.RowConsumption == nil {
			return 0, fmt.Errorf("block (%d, %v) has nil RowConsumption", block.Header.Number, block.Header.Hash().Hex())
		}
		for _, subCircuit := range *block.RowConsumption {
			crc[subCircuit.Name] += subCircuit.RowNumber
		}
	}

	// Find the maximum row consumption
	var maxVal uint64
	for _, value := range crc {
		if value > maxVal {
			maxVal = value
		}
	}

	// Return the maximum row consumption
	return maxVal, nil
}

// NumTransactions calculates the total number of transactions in a Chunk.
func (c *Chunk) NumTransactions() uint64 {
	var totalTxNum uint64
	for _, block := range c.Blocks {
		totalTxNum += uint64(len(block.Transactions))
	}
	return totalTxNum
}

// NumL2Transactions calculates the total number of L2 transactions in a Chunk.
func (c *Chunk) NumL2Transactions() uint64 {
	var totalTxNum uint64
	for _, block := range c.Blocks {
		totalTxNum += block.NumL2Transactions()
	}
	return totalTxNum
}

// L2GasUsed calculates the total gas of L2 transactions in a Chunk.
func (c *Chunk) L2GasUsed() uint64 {
	var totalTxNum uint64
	for _, block := range c.Blocks {
		totalTxNum += block.Header.GasUsed
	}
	return totalTxNum
}

// StateRoot gets the state root after committing/finalizing the batch.
func (b *Batch) StateRoot() common.Hash {
	numChunks := len(b.Chunks)
	if len(b.Chunks) == 0 {
		return common.Hash{}
	}
	lastChunkBlockNum := len(b.Chunks[numChunks-1].Blocks)
	return b.Chunks[len(b.Chunks)-1].Blocks[lastChunkBlockNum-1].Header.Root
}

// WithdrawRoot gets the withdraw root after committing/finalizing the batch.
func (b *Batch) WithdrawRoot() common.Hash {
	numChunks := len(b.Chunks)
	if len(b.Chunks) == 0 {
		return common.Hash{}
	}
	lastChunkBlockNum := len(b.Chunks[numChunks-1].Blocks)
	return b.Chunks[len(b.Chunks)-1].Blocks[lastChunkBlockNum-1].WithdrawRoot
}

// TxsToTxsData converts transactions to a TransactionData array.
func TxsToTxsData(txs types.Transactions) []*types.TransactionData {
	txsData := make([]*types.TransactionData, len(txs))
	for i, tx := range txs {
		v, r, s := tx.RawSignatureValues()

		nonce := tx.Nonce()

		// We need QueueIndex in `NewBatchHeader`. However, `TransactionData`
		// does not have this field. Since `L1MessageTx` do not have a nonce,
		// we reuse this field for storing the queue index.
		if msg := tx.AsL1MessageTx(); msg != nil {
			nonce = msg.QueueIndex
		}

		txsData[i] = &types.TransactionData{
			Type:       tx.Type(),
			TxHash:     tx.Hash().String(),
			Nonce:      nonce,
			ChainId:    (*hexutil.Big)(tx.ChainId()),
			Gas:        tx.Gas(),
			GasPrice:   (*hexutil.Big)(tx.GasPrice()),
			GasTipCap:  (*hexutil.Big)(tx.GasTipCap()),
			GasFeeCap:  (*hexutil.Big)(tx.GasFeeCap()),
			To:         tx.To(),
			Value:      (*hexutil.Big)(tx.Value()),
			Data:       hexutil.Encode(tx.Data()),
			IsCreate:   tx.To() == nil,
			AccessList: tx.AccessList(),
			V:          (*hexutil.Big)(v),
			R:          (*hexutil.Big)(r),
			S:          (*hexutil.Big)(s),
		}
	}
	return txsData
}
