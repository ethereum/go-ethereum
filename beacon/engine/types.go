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

package engine

import (
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

// PayloadVersion denotes the version of PayloadAttributes used to request the
// building of the payload to commence.
type PayloadVersion byte

var (
	PayloadV1 PayloadVersion = 0x1
	PayloadV2 PayloadVersion = 0x2
	PayloadV3 PayloadVersion = 0x3
)

//go:generate go run github.com/fjl/gencodec -type PayloadAttributes -field-override payloadAttributesMarshaling -out gen_blockparams.go

// PayloadAttributes describes the environment context in which a block should
// be built.
type PayloadAttributes struct {
	Timestamp             uint64              `json:"timestamp"             gencodec:"required"`
	Random                common.Hash         `json:"prevRandao"            gencodec:"required"`
	SuggestedFeeRecipient common.Address      `json:"suggestedFeeRecipient" gencodec:"required"`
	Withdrawals           []*types.Withdrawal `json:"withdrawals"`
	BeaconRoot            *common.Hash        `json:"parentBeaconBlockRoot"`
}

// JSON type overrides for PayloadAttributes.
type payloadAttributesMarshaling struct {
	Timestamp hexutil.Uint64
}

//go:generate go run github.com/fjl/gencodec -type ExecutableData -field-override executableDataMarshaling -out gen_ed.go

// ExecutableData is the data necessary to execute an EL payload.
type ExecutableData struct {
	ParentHash       common.Hash             `json:"parentHash"    gencodec:"required"`
	FeeRecipient     common.Address          `json:"feeRecipient"  gencodec:"required"`
	StateRoot        common.Hash             `json:"stateRoot"     gencodec:"required"`
	ReceiptsRoot     common.Hash             `json:"receiptsRoot"  gencodec:"required"`
	LogsBloom        []byte                  `json:"logsBloom"     gencodec:"required"`
	Random           common.Hash             `json:"prevRandao"    gencodec:"required"`
	Number           uint64                  `json:"blockNumber"   gencodec:"required"`
	GasLimit         uint64                  `json:"gasLimit"      gencodec:"required"`
	GasUsed          uint64                  `json:"gasUsed"       gencodec:"required"`
	Timestamp        uint64                  `json:"timestamp"     gencodec:"required"`
	ExtraData        []byte                  `json:"extraData"     gencodec:"required"`
	BaseFeePerGas    *big.Int                `json:"baseFeePerGas" gencodec:"required"`
	BlockHash        common.Hash             `json:"blockHash"     gencodec:"required"`
	Transactions     [][]byte                `json:"transactions"  gencodec:"required"`
	Withdrawals      []*types.Withdrawal     `json:"withdrawals"`
	BlobGasUsed      *uint64                 `json:"blobGasUsed"`
	ExcessBlobGas    *uint64                 `json:"excessBlobGas"`
	ExecutionWitness *types.ExecutionWitness `json:"executionWitness,omitempty"`
}

// JSON type overrides for executableData.
type executableDataMarshaling struct {
	Number        hexutil.Uint64
	GasLimit      hexutil.Uint64
	GasUsed       hexutil.Uint64
	Timestamp     hexutil.Uint64
	BaseFeePerGas *hexutil.Big
	ExtraData     hexutil.Bytes
	LogsBloom     hexutil.Bytes
	Transactions  []hexutil.Bytes
	BlobGasUsed   *hexutil.Uint64
	ExcessBlobGas *hexutil.Uint64
}

// StatelessPayloadStatusV1 is the result of a stateless payload execution.
type StatelessPayloadStatusV1 struct {
	Status          string      `json:"status"`
	StateRoot       common.Hash `json:"stateRoot"`
	ReceiptsRoot    common.Hash `json:"receiptsRoot"`
	ValidationError *string     `json:"validationError"`
}

//go:generate go run github.com/fjl/gencodec -type ExecutionPayloadEnvelope -field-override executionPayloadEnvelopeMarshaling -out gen_epe.go

type ExecutionPayloadEnvelope struct {
	ExecutionPayload *ExecutableData `json:"executionPayload"  gencodec:"required"`
	BlockValue       *big.Int        `json:"blockValue"  gencodec:"required"`
	BlobsBundle      *BlobsBundleV1  `json:"blobsBundle"`
	Requests         [][]byte        `json:"executionRequests"`
	Override         bool            `json:"shouldOverrideBuilder"`
	Witness          *hexutil.Bytes  `json:"witness,omitempty"`
}

type BlobsBundleV1 struct {
	Commitments []hexutil.Bytes `json:"commitments"`
	Proofs      []hexutil.Bytes `json:"proofs"`
	Blobs       []hexutil.Bytes `json:"blobs"`
}

type BlobAndProofV1 struct {
	Blob  hexutil.Bytes `json:"blob"`
	Proof hexutil.Bytes `json:"proof"`
}

// JSON type overrides for ExecutionPayloadEnvelope.
type executionPayloadEnvelopeMarshaling struct {
	BlockValue *hexutil.Big
	Requests   []hexutil.Bytes
}

type PayloadStatusV1 struct {
	Status          string         `json:"status"`
	Witness         *hexutil.Bytes `json:"witness,omitempty"`
	LatestValidHash *common.Hash   `json:"latestValidHash"`
	ValidationError *string        `json:"validationError"`
}

type TransitionConfigurationV1 struct {
	TerminalTotalDifficulty *hexutil.Big   `json:"terminalTotalDifficulty"`
	TerminalBlockHash       common.Hash    `json:"terminalBlockHash"`
	TerminalBlockNumber     hexutil.Uint64 `json:"terminalBlockNumber"`
}

// PayloadID is an identifier of the payload build process
type PayloadID [8]byte

// Version returns the payload version associated with the identifier.
func (b PayloadID) Version() PayloadVersion {
	return PayloadVersion(b[0])
}

// Is returns whether the identifier matches any of provided payload versions.
func (b PayloadID) Is(versions ...PayloadVersion) bool {
	return slices.Contains(versions, b.Version())
}

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
		enc[i], _ = tx.MarshalBinary()
	}
	return enc
}

func decodeTransactions(enc [][]byte) ([]*types.Transaction, error) {
	var txs = make([]*types.Transaction, len(enc))
	for i, encTx := range enc {
		var tx types.Transaction
		if err := tx.UnmarshalBinary(encTx); err != nil {
			return nil, fmt.Errorf("invalid transaction %d: %v", i, err)
		}
		txs[i] = &tx
	}
	return txs, nil
}

// ExecutableDataToBlock constructs a block from executable data.
// It verifies that the following fields:
//
//		len(extraData) <= 32
//		uncleHash = emptyUncleHash
//		difficulty = 0
//	 	if versionedHashes != nil, versionedHashes match to blob transactions
//
// and that the blockhash of the constructed block matches the parameters. Nil
// Withdrawals value will propagate through the returned block. Empty
// Withdrawals value must be passed via non-nil, length 0 value in data.
func ExecutableDataToBlock(data ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, requests [][]byte) (*types.Block, error) {
	block, err := ExecutableDataToBlockNoHash(data, versionedHashes, beaconRoot, requests)
	if err != nil {
		return nil, err
	}
	if block.Hash() != data.BlockHash {
		return nil, fmt.Errorf("blockhash mismatch, want %x, got %x", data.BlockHash, block.Hash())
	}
	return block, nil
}

// ExecutableDataToBlockNoHash is analogous to ExecutableDataToBlock, but is used
// for stateless execution, so it skips checking if the executable data hashes to
// the requested hash (stateless has to *compute* the root hash, it's not given).
func ExecutableDataToBlockNoHash(data ExecutableData, versionedHashes []common.Hash, beaconRoot *common.Hash, requests [][]byte) (*types.Block, error) {
	txs, err := decodeTransactions(data.Transactions)
	if err != nil {
		return nil, err
	}
	if len(data.ExtraData) > int(params.MaximumExtraDataSize) {
		return nil, fmt.Errorf("invalid extradata length: %v", len(data.ExtraData))
	}
	if len(data.LogsBloom) != 256 {
		return nil, fmt.Errorf("invalid logsBloom length: %v", len(data.LogsBloom))
	}
	// Check that baseFeePerGas is not negative or too big
	if data.BaseFeePerGas != nil && (data.BaseFeePerGas.Sign() == -1 || data.BaseFeePerGas.BitLen() > 256) {
		return nil, fmt.Errorf("invalid baseFeePerGas: %v", data.BaseFeePerGas)
	}
	var blobHashes = make([]common.Hash, 0, len(txs))
	for _, tx := range txs {
		blobHashes = append(blobHashes, tx.BlobHashes()...)
	}
	if len(blobHashes) != len(versionedHashes) {
		return nil, fmt.Errorf("invalid number of versionedHashes: %v blobHashes: %v", versionedHashes, blobHashes)
	}
	for i := 0; i < len(blobHashes); i++ {
		if blobHashes[i] != versionedHashes[i] {
			return nil, fmt.Errorf("invalid versionedHash at %v: %v blobHashes: %v", i, versionedHashes, blobHashes)
		}
	}
	// Only set withdrawalsRoot if it is non-nil. This allows CLs to use
	// ExecutableData before withdrawals are enabled by marshaling
	// Withdrawals as the json null value.
	var withdrawalsRoot *common.Hash
	if data.Withdrawals != nil {
		h := types.DeriveSha(types.Withdrawals(data.Withdrawals), trie.NewStackTrie(nil))
		withdrawalsRoot = &h
	}

	var requestsHash *common.Hash
	if requests != nil {
		h := types.CalcRequestsHash(requests)
		requestsHash = &h
	}

	header := &types.Header{
		ParentHash:       data.ParentHash,
		UncleHash:        types.EmptyUncleHash,
		Coinbase:         data.FeeRecipient,
		Root:             data.StateRoot,
		TxHash:           types.DeriveSha(types.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash:      data.ReceiptsRoot,
		Bloom:            types.BytesToBloom(data.LogsBloom),
		Difficulty:       common.Big0,
		Number:           new(big.Int).SetUint64(data.Number),
		GasLimit:         data.GasLimit,
		GasUsed:          data.GasUsed,
		Time:             data.Timestamp,
		BaseFee:          data.BaseFeePerGas,
		Extra:            data.ExtraData,
		MixDigest:        data.Random,
		WithdrawalsHash:  withdrawalsRoot,
		ExcessBlobGas:    data.ExcessBlobGas,
		BlobGasUsed:      data.BlobGasUsed,
		ParentBeaconRoot: beaconRoot,
		RequestsHash:     requestsHash,
	}
	return types.NewBlockWithHeader(header).
			WithBody(types.Body{Transactions: txs, Uncles: nil, Withdrawals: data.Withdrawals}).
			WithWitness(data.ExecutionWitness),
		nil
}

// BlockToExecutableData constructs the ExecutableData structure by filling the
// fields from the given block. It assumes the given block is post-merge block.
func BlockToExecutableData(block *types.Block, fees *big.Int, sidecars []*types.BlobTxSidecar, requests [][]byte) *ExecutionPayloadEnvelope {
	data := &ExecutableData{
		BlockHash:        block.Hash(),
		ParentHash:       block.ParentHash(),
		FeeRecipient:     block.Coinbase(),
		StateRoot:        block.Root(),
		Number:           block.NumberU64(),
		GasLimit:         block.GasLimit(),
		GasUsed:          block.GasUsed(),
		BaseFeePerGas:    block.BaseFee(),
		Timestamp:        block.Time(),
		ReceiptsRoot:     block.ReceiptHash(),
		LogsBloom:        block.Bloom().Bytes(),
		Transactions:     encodeTransactions(block.Transactions()),
		Random:           block.MixDigest(),
		ExtraData:        block.Extra(),
		Withdrawals:      block.Withdrawals(),
		BlobGasUsed:      block.BlobGasUsed(),
		ExcessBlobGas:    block.ExcessBlobGas(),
		ExecutionWitness: block.ExecutionWitness(),
	}

	// Add blobs.
	bundle := BlobsBundleV1{
		Commitments: make([]hexutil.Bytes, 0),
		Blobs:       make([]hexutil.Bytes, 0),
		Proofs:      make([]hexutil.Bytes, 0),
	}
	for _, sidecar := range sidecars {
		for j := range sidecar.Blobs {
			bundle.Blobs = append(bundle.Blobs, hexutil.Bytes(sidecar.Blobs[j][:]))
			bundle.Commitments = append(bundle.Commitments, hexutil.Bytes(sidecar.Commitments[j][:]))
			bundle.Proofs = append(bundle.Proofs, hexutil.Bytes(sidecar.Proofs[j][:]))
		}
	}

	return &ExecutionPayloadEnvelope{
		ExecutionPayload: data,
		BlockValue:       fees,
		BlobsBundle:      &bundle,
		Requests:         requests,
		Override:         false,
	}
}

// ExecutionPayloadBody is used in the response to GetPayloadBodiesByHash and GetPayloadBodiesByRange
type ExecutionPayloadBody struct {
	TransactionData []hexutil.Bytes     `json:"transactions"`
	Withdrawals     []*types.Withdrawal `json:"withdrawals"`
}

// Client identifiers to support ClientVersionV1.
const (
	ClientCode = "GE"
	ClientName = "go-ethereum"
)

// ClientVersionV1 contains information which identifies a client implementation.
type ClientVersionV1 struct {
	Code    string `json:"code"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Commit  string `json:"commit"`
}

func (v *ClientVersionV1) String() string {
	return fmt.Sprintf("%s-%s-%s-%s", v.Code, v.Name, v.Version, v.Commit)
}
