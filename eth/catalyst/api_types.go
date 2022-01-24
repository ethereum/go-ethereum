// Copyright 2020 The go-ethereum Authors
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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

//go:generate go run github.com/fjl/gencodec -type PayloadAttributesV1 -field-override payloadAttributesMarshaling -out gen_blockparams.go

// PayloadAttributesV1 structure described at https://github.com/ethereum/execution-apis/pull/74
type PayloadAttributesV1 struct {
	Timestamp             uint64         `json:"timestamp"     gencodec:"required"`
	Random                common.Hash    `json:"random"        gencodec:"required"`
	SuggestedFeeRecipient common.Address `json:"suggestedFeeRecipient"  gencodec:"required"`
}

// JSON type overrides for PayloadAttributesV1.
type payloadAttributesMarshaling struct {
	Timestamp hexutil.Uint64
}

//go:generate go run github.com/fjl/gencodec -type ExecutableDataV1 -field-override executableDataMarshaling -out gen_ed.go

// ExecutableDataV1 structure described at https://github.com/ethereum/execution-apis/src/engine/specification.md
type ExecutableDataV1 struct {
	ParentHash    common.Hash    `json:"parentHash"    gencodec:"required"`
	FeeRecipient  common.Address `json:"feeRecipient"  gencodec:"required"`
	StateRoot     common.Hash    `json:"stateRoot"     gencodec:"required"`
	ReceiptsRoot  common.Hash    `json:"receiptsRoot"   gencodec:"required"`
	LogsBloom     []byte         `json:"logsBloom"     gencodec:"required"`
	Random        common.Hash    `json:"random"        gencodec:"required"`
	Number        uint64         `json:"blockNumber"   gencodec:"required"`
	GasLimit      uint64         `json:"gasLimit"      gencodec:"required"`
	GasUsed       uint64         `json:"gasUsed"       gencodec:"required"`
	Timestamp     uint64         `json:"timestamp"     gencodec:"required"`
	ExtraData     []byte         `json:"extraData"     gencodec:"required"`
	BaseFeePerGas *big.Int       `json:"baseFeePerGas" gencodec:"required"`
	BlockHash     common.Hash    `json:"blockHash"     gencodec:"required"`
	Transactions  [][]byte       `json:"transactions"  gencodec:"required"`
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
}

type NewBlockResponse struct {
	Valid bool `json:"valid"`
}

type GenericResponse struct {
	Success bool `json:"success"`
}

type GenericStringResponse struct {
	Status string `json:"status"`
}

type ExecutePayloadResponse struct {
	Status          string      `json:"status"`
	LatestValidHash common.Hash `json:"latestValidHash"`
}

type ConsensusValidatedParams struct {
	BlockHash common.Hash `json:"blockHash"`
	Status    string      `json:"status"`
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
	Status    string     `json:"status"`
	PayloadID *PayloadID `json:"payloadId"`
}

type ForkchoiceStateV1 struct {
	HeadBlockHash      common.Hash `json:"headBlockHash"`
	SafeBlockHash      common.Hash `json:"safeBlockHash"`
	FinalizedBlockHash common.Hash `json:"finalizedBlockHash"`
}
