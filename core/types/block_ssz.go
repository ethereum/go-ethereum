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

package types

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/ssz"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// payloadFields is the active_fields configuration of ExecutionPayload,
// ProgressiveContainer(active_fields=[1] * 18), and pairFields that of its
// two inner containers, [1] * 2.
var (
	payloadFields = ssz.MustActiveFields(1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1)
	pairFields    = ssz.MustActiveFields(1, 1)
)

// GasAmounts is the EIP-7807 GasAmounts container: a regular and a blob gas
// amount, each a uint64.
type GasAmounts struct {
	Regular uint64
	Blob    uint64
}

// Root computes the hash tree root of the container.
func (g GasAmounts) Root() common.Hash {
	roots := [][32]byte{uint64Leaf(g.Regular), uint64Leaf(g.Blob)}
	return common.Hash(ssz.HashTreeRootProgressiveContainer(pairFields, roots))
}

// BlobFeesPerGas is the EIP-7807 BlobFeesPerGas container: a regular and a
// blob fee per gas, each a uint256.
type BlobFeesPerGas struct {
	Regular uint256.Int
	Blob    uint256.Int
}

// Root computes the hash tree root of the container.
func (f BlobFeesPerGas) Root() common.Hash {
	roots := [][32]byte{uint256Leaf(&f.Regular), uint256Leaf(&f.Blob)}
	return common.Hash(ssz.HashTreeRootProgressiveContainer(pairFields, roots))
}

// ExecutionPayload is the EIP-7807 execution block, the eighteen-field
// progressive container whose hash tree root the EIP defines as the block
// hash. The fields are plain values in the EIP's order: the list elements
// keep their existing encodings (EIP-2718 envelopes for transactions, RLP for
// withdrawals and the block access list), and the three roots a Geth block
// does not carry are inputs. ReceiptsRoot is the progressive root over the
// receipt envelopes, computed the way Summary computes TransactionsRoot;
// RequestsHash is the consensus layer's ExecutionRequests root, not the
// EIP-7685 requests hash of the header.
type ExecutionPayload struct {
	ParentHash            common.Hash
	Miner                 common.Address
	StateRoot             common.Hash
	Transactions          [][]byte
	ReceiptsRoot          common.Hash
	Number                uint64
	GasLimits             GasAmounts
	GasUsed               GasAmounts
	Timestamp             uint64
	ExtraData             []byte
	MixHash               common.Hash
	BaseFeesPerGas        BlobFeesPerGas
	Withdrawals           [][]byte
	ExcessGas             GasAmounts
	ParentBeaconBlockRoot common.Hash
	RequestsHash          common.Hash
	BlockAccessList       []byte
	SlotNumber            uint64
}

// ExecutionPayloadSummary is the SSZ summary of an ExecutionPayload that the
// EIP defines as the execution block header: the same container with the
// transactions, withdrawals and block_access_list fields replaced by their
// hash tree roots. It has the root of the payload it summarizes, which is what
// lets a header-only verifier compute the block hash.
type ExecutionPayloadSummary struct {
	ParentHash            common.Hash
	Miner                 common.Address
	StateRoot             common.Hash
	TransactionsRoot      common.Hash
	ReceiptsRoot          common.Hash
	Number                uint64
	GasLimits             GasAmounts
	GasUsed               GasAmounts
	Timestamp             uint64
	ExtraData             []byte
	MixHash               common.Hash
	BaseFeesPerGas        BlobFeesPerGas
	WithdrawalsRoot       common.Hash
	ExcessGas             GasAmounts
	ParentBeaconBlockRoot common.Hash
	RequestsHash          common.Hash
	BlockAccessListRoot   common.Hash
	SlotNumber            uint64
}

// Summary returns the header form of the payload, with the three list fields
// replaced by their roots. The extra data is shared, not copied.
func (p *ExecutionPayload) Summary() *ExecutionPayloadSummary {
	return &ExecutionPayloadSummary{
		ParentHash:            p.ParentHash,
		Miner:                 p.Miner,
		StateRoot:             p.StateRoot,
		TransactionsRoot:      ByteListsRoot(p.Transactions),
		ReceiptsRoot:          p.ReceiptsRoot,
		Number:                p.Number,
		GasLimits:             p.GasLimits,
		GasUsed:               p.GasUsed,
		Timestamp:             p.Timestamp,
		ExtraData:             p.ExtraData,
		MixHash:               p.MixHash,
		BaseFeesPerGas:        p.BaseFeesPerGas,
		WithdrawalsRoot:       ByteListsRoot(p.Withdrawals),
		ExcessGas:             p.ExcessGas,
		ParentBeaconBlockRoot: p.ParentBeaconBlockRoot,
		RequestsHash:          p.RequestsHash,
		BlockAccessListRoot:   common.Hash(ssz.HashTreeRootProgressiveBytes(p.BlockAccessList)),
		SlotNumber:            p.SlotNumber,
	}
}

// Root computes the hash tree root of the payload, the EIP-7807 block hash.
// The payload hashes through its summary: a list field enters the container
// as its root either way, which is the SSZ summary property the EIP relies
// on. It fails only for extra data longer than MAX_EXTRA_DATA_BYTES, which
// the container cannot represent.
func (p *ExecutionPayload) Root() (common.Hash, error) {
	return p.Summary().Root()
}

// Root computes the hash tree root of the summary, which is the root of the
// payload it summarizes: the eighteen field roots in the EIP's order, folded
// as a progressive container.
func (s *ExecutionPayloadSummary) Root() (common.Hash, error) {
	if uint64(len(s.ExtraData)) > params.MaximumExtraDataSize {
		return common.Hash{}, fmt.Errorf("extra data is %d bytes, ByteList[MAX_EXTRA_DATA_BYTES] holds at most %d",
			len(s.ExtraData), params.MaximumExtraDataSize)
	}
	roots := [][32]byte{
		s.ParentHash,
		addressLeaf(s.Miner),
		s.StateRoot,
		s.TransactionsRoot,
		s.ReceiptsRoot,
		uint64Leaf(s.Number),
		s.GasLimits.Root(),
		s.GasUsed.Root(),
		uint64Leaf(s.Timestamp),
		extraDataRoot(s.ExtraData),
		s.MixHash,
		s.BaseFeesPerGas.Root(),
		s.WithdrawalsRoot,
		s.ExcessGas.Root(),
		s.ParentBeaconBlockRoot,
		s.RequestsHash,
		s.BlockAccessListRoot,
		uint64Leaf(s.SlotNumber),
	}
	return common.Hash(ssz.HashTreeRootProgressiveContainer(payloadFields, roots)), nil
}

// uint64Leaf is the chunk of a uint64 field: its little-endian serialization,
// zero-padded.
func uint64Leaf(v uint64) (leaf [32]byte) {
	ssz.AppendUint64(leaf[:0], v)
	return leaf
}

// uint256Leaf is the chunk of a uint256 field: its little-endian serialization.
func uint256Leaf(v *uint256.Int) (leaf [32]byte) {
	ssz.AppendUint256(leaf[:0], v)
	return leaf
}

// addressLeaf is the chunk of an ExecutionAddress field: the twenty bytes,
// zero-padded.
func addressLeaf(a common.Address) (leaf [32]byte) {
	copy(leaf[:], a[:])
	return leaf
}

// extraDataRoot computes the root of the ByteList[MAX_EXTRA_DATA_BYTES]
// extra_data field: a classic byte list of at most one chunk. The caller has
// checked the length, which Merkleize requires.
func extraDataRoot(data []byte) [32]byte {
	return ssz.MixInLength(ssz.Merkleize(ssz.Pack(data), 1), uint64(len(data)))
}

// ByteListsRoot computes the root of a ProgressiveList[ProgressiveByteList]
// from its elements' bytes, the type of the transactions, withdrawals and
// receipts lists of ExecutionPayload. The receipts are not part of the
// payload, so the code that fills ReceiptsRoot from a block's receipts uses
// it from outside.
func ByteListsRoot(elems [][]byte) common.Hash {
	roots := make([][32]byte, len(elems))
	for i, elem := range elems {
		roots[i] = ssz.HashTreeRootProgressiveBytes(elem)
	}
	return common.Hash(ssz.HashTreeRootProgressiveRoots(roots))
}
