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

// ExecutionPayloadCancun is the Cancun/Prague/Osaka payload shape.
// Prague and Osaka don't change the inner payload — execution_requests
// and the BlobsBundle revision change at the envelope level, not here.
type ExecutionPayloadCancun struct {
	ParentHash    common.Hash
	FeeRecipient  common.Address
	StateRoot     common.Hash
	ReceiptsRoot  common.Hash
	LogsBloom     [256]byte
	PrevRandao    common.Hash
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	ExtraData     []byte
	BaseFeePerGas *uint256.Int
	BlockHash     common.Hash
	Transactions  [][]byte
	Withdrawals   []*Withdrawal
	BlobGasUsed   uint64
	ExcessBlobGas uint64
}

func (p *ExecutionPayloadCancun) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// 5 hashes(32) + addr(20) + bloom(256) + 4 u64s + uint256(32)
	// + 3 offsets(4) + 2 u64s (blob gas) = 528
	size := uint32(5*32 + 20 + 256 + 4*8 + 32 + 3*4 + 2*8)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicBytes(siz, p.ExtraData)
	size += ssz.SizeSliceOfDynamicBytes(siz, p.Transactions)
	size += ssz.SizeSliceOfStaticObjects(siz, p.Withdrawals)
	return size
}

func (p *ExecutionPayloadCancun) DefineSSZ(c *ssz.Codec) {
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

	ssz.DefineDynamicBytesContent(c, &p.ExtraData, MaxExtraDataBytes)
	ssz.DefineSliceOfDynamicBytesContent(c, &p.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsContent(c, &p.Withdrawals, MaxWithdrawalsPerPayload)
}

// ExecutionPayloadShanghai is the Shanghai payload shape (Cancun minus blob gas).
type ExecutionPayloadShanghai struct {
	ParentHash    common.Hash
	FeeRecipient  common.Address
	StateRoot     common.Hash
	ReceiptsRoot  common.Hash
	LogsBloom     [256]byte
	PrevRandao    common.Hash
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	ExtraData     []byte
	BaseFeePerGas *uint256.Int
	BlockHash     common.Hash
	Transactions  [][]byte
	Withdrawals   []*Withdrawal
}

func (p *ExecutionPayloadShanghai) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// 5 hashes(32) + addr(20) + bloom(256) + 4 u64s + uint256(32) + 3 offsets(4) = 512
	size := uint32(5*32 + 20 + 256 + 4*8 + 32 + 3*4)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicBytes(siz, p.ExtraData)
	size += ssz.SizeSliceOfDynamicBytes(siz, p.Transactions)
	size += ssz.SizeSliceOfStaticObjects(siz, p.Withdrawals)
	return size
}

func (p *ExecutionPayloadShanghai) DefineSSZ(c *ssz.Codec) {
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

	ssz.DefineDynamicBytesContent(c, &p.ExtraData, MaxExtraDataBytes)
	ssz.DefineSliceOfDynamicBytesContent(c, &p.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsContent(c, &p.Withdrawals, MaxWithdrawalsPerPayload)
}

// ExecutionPayloadParis is the original Bellatrix/Paris payload shape.
type ExecutionPayloadParis struct {
	ParentHash    common.Hash
	FeeRecipient  common.Address
	StateRoot     common.Hash
	ReceiptsRoot  common.Hash
	LogsBloom     [256]byte
	PrevRandao    common.Hash
	BlockNumber   uint64
	GasLimit      uint64
	GasUsed       uint64
	Timestamp     uint64
	ExtraData     []byte
	BaseFeePerGas *uint256.Int
	BlockHash     common.Hash
	Transactions  [][]byte
}

func (p *ExecutionPayloadParis) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// 5 hashes(32) + addr(20) + bloom(256) + 4 u64s + uint256(32) + 2 offsets(4) = 508
	size := uint32(5*32 + 20 + 256 + 4*8 + 32 + 2*4)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicBytes(siz, p.ExtraData)
	size += ssz.SizeSliceOfDynamicBytes(siz, p.Transactions)
	return size
}

func (p *ExecutionPayloadParis) DefineSSZ(c *ssz.Codec) {
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

	ssz.DefineDynamicBytesContent(c, &p.ExtraData, MaxExtraDataBytes)
	ssz.DefineSliceOfDynamicBytesContent(c, &p.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
}
