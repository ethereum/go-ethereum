// Copyright 2022 The go-ethereum Authors
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

package beacon

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
)

//go:generate go run github.com/fjl/gencodec -type PayloadAttributes -field-override payloadAttributesMarshaling -out gen_blockparams.go

// PayloadAttributes describes the environment context in which a block should
// be built.
type PayloadAttributes struct {
	Timestamp             uint64              `json:"timestamp"             gencodec:"required"`
	Random                common.Hash         `json:"prevRandao"            gencodec:"required"`
	SuggestedFeeRecipient common.Address      `json:"suggestedFeeRecipient" gencodec:"required"`
	Withdrawals           []*types.Withdrawal `json:"withdrawals"`
}

// JSON type overrides for PayloadAttributes.
type payloadAttributesMarshaling struct {
	Timestamp hexutil.Uint64
}

// BlobsBundle holds the blobs of an execution payload, to be retrieved separately
type BlobsBundle struct {
	BlockHash common.Hash           `json:"blockHash"     gencodec:"required"`
	KZGs      []types.KZGCommitment `json:"kzgs"      gencodec:"required"`
	Blobs     []types.Blob          `json:"blobs"      gencodec:"required"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutableData -field-override executableDataMarshaling -out gen_ed.go

// ExecutableData is the data necessary to execute an EL payload.
type ExecutableData struct {
	ParentHash    common.Hash         `json:"parentHash"    gencodec:"required"`
	FeeRecipient  common.Address      `json:"feeRecipient"  gencodec:"required"`
	StateRoot     common.Hash         `json:"stateRoot"     gencodec:"required"`
	ReceiptsRoot  common.Hash         `json:"receiptsRoot"  gencodec:"required"`
	LogsBloom     []byte              `json:"logsBloom"     gencodec:"required"`
	Random        common.Hash         `json:"prevRandao"    gencodec:"required"`
	Number        uint64              `json:"blockNumber"   gencodec:"required"`
	GasLimit      uint64              `json:"gasLimit"      gencodec:"required"`
	GasUsed       uint64              `json:"gasUsed"       gencodec:"required"`
	Timestamp     uint64              `json:"timestamp"     gencodec:"required"`
	ExtraData     []byte              `json:"extraData"     gencodec:"required"`
	BaseFeePerGas *big.Int            `json:"baseFeePerGas" gencodec:"required"`
	BlockHash     common.Hash         `json:"blockHash"     gencodec:"required"`
	Transactions  [][]byte            `json:"transactions"  gencodec:"required"`
	ExcessDataGas *big.Int            `json:"excessDataGas"` // New in EIP-4844
	Withdrawals   []*types.Withdrawal `json:"withdrawals"`
}

// JSON type overrides for executableData.
type executableDataMarshaling struct {
	Number        hexutil.Uint64
	GasLimit      hexutil.Uint64
	GasUsed       hexutil.Uint64
	Timestamp     hexutil.Uint64
	BaseFeePerGas *hexutil.Big
	ExcessDataGas *hexutil.Big
	ExtraData     hexutil.Bytes
	LogsBloom     hexutil.Bytes
	Transactions  []hexutil.Bytes
}

type ExecutableDataV2 struct {
	ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
	BlockValue       *big.Int        `json:"blockValue"  gencodec:"required"`
}

type PayloadStatusV1 struct {
	Status          string       `json:"status"`
	LatestValidHash *common.Hash `json:"latestValidHash"`
	ValidationError *string      `json:"validationError"`
}

type TransitionConfigurationV1 struct {
	TerminalTotalDifficulty *hexutil.Big   `json:"terminalTotalDifficulty"`
	TerminalBlockHash       common.Hash    `json:"terminalBlockHash"`
	TerminalBlockNumber     hexutil.Uint64 `json:"terminalBlockNumber"`
}

// PayloadID is an identifier of the payload build process
type PayloadID [8]byte

func (b PayloadID) String() string {
	return hexutil.Encode(b[:])
}

func (b PayloadID) MarshalText() ([]byte, error) {
	return hexutil.Bytes(b[:]).MarshalText()
}

func (b *PayloadID) UnmarshalText(input []byte) error {
	err := hexutil.UnmarshalFixedText("PayloadID", input, b[:])
	if err != nil {
		return fmt.Errorf("invalid payload id %q: %w", input, err)
	}
	return nil
}

type ForkChoiceResponse struct {
	PayloadStatus PayloadStatusV1 `json:"payloadStatus"`
	PayloadID     *PayloadID      `json:"payloadId"`
}

type ForkchoiceStateV1 struct {
	HeadBlockHash      common.Hash `json:"headBlockHash"`
	SafeBlockHash      common.Hash `json:"safeBlockHash"`
	FinalizedBlockHash common.Hash `json:"finalizedBlockHash"`
}

func encodeTransactions(txs []*types.Transaction) [][]byte {
	var enc = make([][]byte, len(txs))
	for i, tx := range txs {
		enc[i], _ = tx.MarshalMinimal()
	}
	return enc
}

func decodeTransactions(enc [][]byte) ([]*types.Transaction, error) {
	var txs = make([]*types.Transaction, len(enc))
	for i, encTx := range enc {
		var tx types.Transaction
		if err := tx.UnmarshalMinimal(encTx); err != nil {
			return nil, fmt.Errorf("invalid transaction %d: %v", i, err)
		}
		txs[i] = &tx
	}
	return txs, nil
}

// ExecutableDataToBlock constructs a block from executable data.
// It verifies that the following fields:
//
//	len(extraData) <= 32
//	uncleHash = emptyUncleHash
//	difficulty = 0
//
// and that the blockhash of the constructed block matches the parameters. Nil
// Withdrawals value will propagate through the returned block. Empty
// Withdrawals value must be passed via non-nil, length 0 value in params.
func ExecutableDataToBlock(params ExecutableData) (*types.Block, error) {
	txs, err := decodeTransactions(params.Transactions)
	if err != nil {
		return nil, err
	}
	if len(params.ExtraData) > 32 {
		return nil, fmt.Errorf("invalid extradata length: %v", len(params.ExtraData))
	}
	if len(params.LogsBloom) != 256 {
		return nil, fmt.Errorf("invalid logsBloom length: %v", len(params.LogsBloom))
	}
	// Check that baseFeePerGas is not negative or too big
	if params.BaseFeePerGas != nil && (params.BaseFeePerGas.Sign() == -1 || params.BaseFeePerGas.BitLen() > 256) {
		return nil, fmt.Errorf("invalid baseFeePerGas: %v", params.BaseFeePerGas)
	}
	// Only set withdrawalsRoot if it is non-nil. This allows CLs to use
	// ExecutableData before withdrawals are enabled by marshaling
	// Withdrawals as the json null value.
	var withdrawalsRoot *common.Hash
	if params.Withdrawals != nil {
		h := types.DeriveSha(types.Withdrawals(params.Withdrawals), trie.NewStackTrie(nil))
		withdrawalsRoot = &h
	}
	header := &types.Header{
		ParentHash:      params.ParentHash,
		UncleHash:       types.EmptyUncleHash,
		Coinbase:        params.FeeRecipient,
		Root:            params.StateRoot,
		TxHash:          types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash:     params.ReceiptsRoot,
		Bloom:           types.BytesToBloom(params.LogsBloom),
		Difficulty:      common.Big0,
		Number:          new(big.Int).SetUint64(params.Number),
		GasLimit:        params.GasLimit,
		GasUsed:         params.GasUsed,
		Time:            params.Timestamp,
		BaseFee:         params.BaseFeePerGas,
		ExcessDataGas:   params.ExcessDataGas,
		Extra:           params.ExtraData,
		MixDigest:       params.Random,
		WithdrawalsHash: withdrawalsRoot,
	}
	block := types.NewBlockWithHeader(header).WithBody2(txs, nil /* uncles */, params.Withdrawals)
	if block.Hash() != params.BlockHash {
		return nil, fmt.Errorf("blockhash mismatch, want %x, got %x", params.BlockHash, block.Hash())
	}
	return block, nil
}

// BlockToExecutableData constructs the ExecutableDataV2 structure by filling the fields from the
// given block, and setting BlockValue field to 'fees'. It assumes the given block is a post-merge
// block.
func BlockToExecutableData(block *types.Block, fees *big.Int) *ExecutableDataV2 {
	return &ExecutableDataV2{
		ExecutionPayload: &ExecutableData{
			BlockHash:     block.Hash(),
			ParentHash:    block.ParentHash(),
			FeeRecipient:  block.Coinbase(),
			StateRoot:     block.Root(),
			Number:        block.NumberU64(),
			GasLimit:      block.GasLimit(),
			GasUsed:       block.GasUsed(),
			BaseFeePerGas: block.BaseFee(),
			ExcessDataGas: block.ExcessDataGas(),
			Timestamp:     block.Time(),
			ReceiptsRoot:  block.ReceiptHash(),
			LogsBloom:     block.Bloom().Bytes(),
			Transactions:  encodeTransactions(block.Transactions()),
			Random:        block.MixDigest(),
			ExtraData:     block.Extra(),
			Withdrawals:   block.Withdrawals(),
		},
		BlockValue: fees,
	}
}

func BlockToBlobData(block *types.Block) (*BlobsBundle, error) {
	blockHash := block.Hash()
	blobsBundle := &BlobsBundle{
		BlockHash: blockHash,
		Blobs:     []types.Blob{},
		KZGs:      []types.KZGCommitment{},
	}
	for i, tx := range block.Transactions() {
		if tx.Type() == types.BlobTxType {
			versionedHashes, kzgs, blobs, aggProof := tx.BlobWrapData()
			if len(versionedHashes) != len(kzgs) || len(versionedHashes) != len(blobs) {
				return nil, fmt.Errorf("tx %d in block %s has inconsistent blobs (%d) / kzgs (%d)"+
					" / versioned hashes (%d)", i, blockHash, len(blobs), len(kzgs), len(versionedHashes))
			}
			var zProof types.KZGProof
			if zProof == aggProof {
				return nil, errors.New("aggregated proof is not available in blobs")
			}
			blobsBundle.Blobs = append(blobsBundle.Blobs, blobs...)
			blobsBundle.KZGs = append(blobsBundle.KZGs, kzgs...)
		}
	}
	return blobsBundle, nil
}
