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

package ssz

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
)

// ExecutionPayloadAmsterdam carries the Amsterdam-shaped payload over the wire.
type ExecutionPayloadAmsterdam struct {
	ParentHash      common.Hash
	FeeRecipient    common.Address
	StateRoot       common.Hash
	ReceiptsRoot    common.Hash
	LogsBloom       [256]byte
	PrevRandao      common.Hash
	BlockNumber     uint64
	GasLimit        uint64
	GasUsed         uint64
	Timestamp       uint64
	ExtraData       []byte
	BaseFeePerGas   *uint256.Int
	BlockHash       common.Hash
	Transactions    [][]byte
	Withdrawals     []*Withdrawal
	BlobGasUsed     uint64
	ExcessBlobGas   uint64
	BlockAccessList []byte
	SlotNumber      uint64
}

func (p *ExecutionPayloadAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// 5 hashes(32) + addr(20) + bloom(256) + 4 u64s + uint256(32)
	// + 4 offsets(4) + 2 u64s (blob gas) + u64 (slot) = 540
	size := uint32(5*32 + 20 + 256 + 4*8 + 32 + 4*4 + 2*8 + 8)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicBytes(siz, p.ExtraData)
	size += ssz.SizeSliceOfDynamicBytes(siz, p.Transactions)
	size += ssz.SizeSliceOfStaticObjects(siz, p.Withdrawals)
	size += ssz.SizeDynamicBytes(siz, p.BlockAccessList)
	return size
}

func (p *ExecutionPayloadAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineStaticBytes(c, &p.ParentHash)
	ssz.DefineStaticBytes(c, &p.FeeRecipient)
	ssz.DefineStaticBytes(c, &p.StateRoot)
	ssz.DefineStaticBytes(c, &p.ReceiptsRoot)
	ssz.DefineStaticBytes(c, &p.LogsBloom)
	ssz.DefineStaticBytes(c, &p.PrevRandao)
	ssz.DefineUint64(c, &p.BlockNumber)
	ssz.DefineUint64(c, &p.GasLimit)
	ssz.DefineUint64(c, &p.GasUsed)
	ssz.DefineUint64(c, &p.Timestamp)
	ssz.DefineDynamicBytesOffset(c, &p.ExtraData, MaxExtraDataBytes)
	ssz.DefineUint256(c, &p.BaseFeePerGas)
	ssz.DefineStaticBytes(c, &p.BlockHash)
	ssz.DefineSliceOfDynamicBytesOffset(c, &p.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsOffset(c, &p.Withdrawals, MaxWithdrawalsPerPayload)
	ssz.DefineUint64(c, &p.BlobGasUsed)
	ssz.DefineUint64(c, &p.ExcessBlobGas)
	ssz.DefineDynamicBytesOffset(c, &p.BlockAccessList, MaxBalBytes)
	ssz.DefineUint64(c, &p.SlotNumber)

	ssz.DefineDynamicBytesContent(c, &p.ExtraData, MaxExtraDataBytes)
	ssz.DefineSliceOfDynamicBytesContent(c, &p.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsContent(c, &p.Withdrawals, MaxWithdrawalsPerPayload)
	ssz.DefineDynamicBytesContent(c, &p.BlockAccessList, MaxBalBytes)
}

// PayloadAttributesAmsterdam carries Amsterdam build attributes.
type PayloadAttributesAmsterdam struct {
	Timestamp             uint64
	PrevRandao            common.Hash
	SuggestedFeeRecipient common.Address
	Withdrawals           []*Withdrawal
	ParentBeaconBlockRoot common.Hash
	SlotNumber            uint64
	TargetGasLimit        uint64
}

func (a *PayloadAttributesAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// timestamp(8) + randao(32) + fee_recipient(20) + offset(4)
	// + parent_beacon_block_root(32) + slot(8) + target_gas(8) = 112
	size := uint32(112)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticObjects(siz, a.Withdrawals)
	return size
}

func (a *PayloadAttributesAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineUint64(c, &a.Timestamp)
	ssz.DefineStaticBytes(c, &a.PrevRandao)
	ssz.DefineStaticBytes(c, &a.SuggestedFeeRecipient)
	ssz.DefineSliceOfStaticObjectsOffset(c, &a.Withdrawals, MaxWithdrawalsPerPayload)
	ssz.DefineStaticBytes(c, &a.ParentBeaconBlockRoot)
	ssz.DefineUint64(c, &a.SlotNumber)
	ssz.DefineUint64(c, &a.TargetGasLimit)

	ssz.DefineSliceOfStaticObjectsContent(c, &a.Withdrawals, MaxWithdrawalsPerPayload)
}

// ExecutionPayloadBodyAmsterdam mirrors the /amsterdam/bodies response shape.
type ExecutionPayloadBodyAmsterdam struct {
	Transactions    [][]byte
	Withdrawals     []*Withdrawal
	BlockAccessList []byte
}

func (b *ExecutionPayloadBodyAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// 3 offsets
	size := uint32(12)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicBytes(siz, b.Transactions)
	size += ssz.SizeSliceOfStaticObjects(siz, b.Withdrawals)
	size += ssz.SizeDynamicBytes(siz, b.BlockAccessList)
	return size
}

func (b *ExecutionPayloadBodyAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicBytesOffset(c, &b.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsOffset(c, &b.Withdrawals, MaxWithdrawalsPerPayload)
	ssz.DefineDynamicBytesOffset(c, &b.BlockAccessList, MaxBalBytes)

	ssz.DefineSliceOfDynamicBytesContent(c, &b.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsContent(c, &b.Withdrawals, MaxWithdrawalsPerPayload)
	ssz.DefineDynamicBytesContent(c, &b.BlockAccessList, MaxBalBytes)
}
