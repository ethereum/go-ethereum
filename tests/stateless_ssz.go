// Copyright 2024 The go-ethereum Authors
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

package tests

//go:generate go run github.com/ferranbt/fastssz/sszgen -path . -objs SszWithdrawal,SszExecutionPayload,SszNewPayloadRequest,SszExecutionWitness,SszChainConfig,SszStatelessInput,SszStatelessValidationResult -output stateless_ssz_encoding.go

// SszWithdrawal mirrors the SSZ encoding of a withdrawal.
type SszWithdrawal struct {
	Index          uint64   `ssz-size:"8"`
	ValidatorIndex uint64   `ssz-size:"8"`
	Address        [20]byte `ssz-size:"20"`
	Amount         [32]byte `ssz-size:"32"`
}

// SszExecutionPayload mirrors the SSZ encoding of an execution payload.
type SszExecutionPayload struct {
	ParentHash      [32]byte  `ssz-size:"32"`
	FeeRecipient    [20]byte  `ssz-size:"20"`
	StateRoot       [32]byte  `ssz-size:"32"`
	ReceiptsRoot    [32]byte  `ssz-size:"32"`
	LogsBloom       [256]byte `ssz-size:"256"`
	PrevRandao      [32]byte  `ssz-size:"32"`
	BlockNumber     uint64
	GasLimit        uint64
	GasUsed         uint64
	Timestamp       uint64
	ExtraData       []byte           `ssz-max:"32"`
	BaseFeePerGas   [32]byte         `ssz-size:"32"`
	BlockHash       [32]byte         `ssz-size:"32"`
	Transactions    [][]byte         `ssz-max:"1048576,1073741824"`
	Withdrawals     []*SszWithdrawal `ssz-max:"65536"` // TODO: this is here because of a spec test going over 16
	BlobGasUsed     uint64
	ExcessBlobGas   uint64
	BlockAccessList []byte `ssz-max:"16777216"`
	SlotNumber      uint64
}

// SszNewPayloadRequest mirrors the SSZ encoding of a new payload request.
type SszNewPayloadRequest struct {
	ExecutionPayload      *SszExecutionPayload
	VersionedHashes       [][32]byte `ssz-max:"4096" ssz-size:"?,32"`
	ParentBeaconBlockRoot [32]byte   `ssz-size:"32"`
	ExecutionRequests     [][]byte   `ssz-max:"16,1048576"`
}

// SszExecutionWitness mirrors the SSZ encoding of an execution witness.
type SszExecutionWitness struct {
	State   [][]byte `ssz-max:"1048576,1048576"`
	Codes   [][]byte `ssz-max:"65536,16777216"`
	Headers [][]byte `ssz-max:"256,1024"`
}

// SszChainConfig mirrors the SSZ encoding of a chain config.
type SszChainConfig struct {
	ChainId uint64
}

// SszStatelessInput mirrors the SSZ encoding of a stateless execution input.
type SszStatelessInput struct {
	NewPayloadRequest *SszNewPayloadRequest
	Witness           *SszExecutionWitness
	ChainConfig       *SszChainConfig
	PublicKeys        [][]byte `ssz-max:"1048576,48"`
}

// SszStatelessValidationResult mirrors the SSZ encoding of a stateless validation result.
type SszStatelessValidationResult struct {
	NewPayloadRequestRoot [32]byte `ssz-size:"32"`
	SuccessfulValidation  bool
	ChainConfig           *SszChainConfig
}
