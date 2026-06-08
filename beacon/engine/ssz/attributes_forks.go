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
	"github.com/karalabe/ssz"
)

// PayloadAttributesCancun is the Cancun/Prague/Osaka build attribute shape.
type PayloadAttributesCancun struct {
	Timestamp             uint64
	PrevRandao            common.Hash
	SuggestedFeeRecipient common.Address
	Withdrawals           []*Withdrawal
	ParentBeaconBlockRoot common.Hash
}

func (a *PayloadAttributesCancun) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// 8 + 32 + 20 + 4(offset) + 32 = 96
	size := uint32(96)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticObjects(siz, a.Withdrawals)
	return size
}

func (a *PayloadAttributesCancun) DefineSSZ(c *ssz.Codec) {
	ssz.DefineUint64(c, &a.Timestamp)
	ssz.DefineStaticBytes(c, &a.PrevRandao)
	ssz.DefineStaticBytes(c, &a.SuggestedFeeRecipient)
	ssz.DefineSliceOfStaticObjectsOffset(c, &a.Withdrawals, MaxWithdrawalsPerPayload)
	ssz.DefineStaticBytes(c, &a.ParentBeaconBlockRoot)

	ssz.DefineSliceOfStaticObjectsContent(c, &a.Withdrawals, MaxWithdrawalsPerPayload)
}

// PayloadAttributesShanghai = Cancun minus parent_beacon_block_root.
type PayloadAttributesShanghai struct {
	Timestamp             uint64
	PrevRandao            common.Hash
	SuggestedFeeRecipient common.Address
	Withdrawals           []*Withdrawal
}

func (a *PayloadAttributesShanghai) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(8 + 32 + 20 + 4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticObjects(siz, a.Withdrawals)
	return size
}

func (a *PayloadAttributesShanghai) DefineSSZ(c *ssz.Codec) {
	ssz.DefineUint64(c, &a.Timestamp)
	ssz.DefineStaticBytes(c, &a.PrevRandao)
	ssz.DefineStaticBytes(c, &a.SuggestedFeeRecipient)
	ssz.DefineSliceOfStaticObjectsOffset(c, &a.Withdrawals, MaxWithdrawalsPerPayload)

	ssz.DefineSliceOfStaticObjectsContent(c, &a.Withdrawals, MaxWithdrawalsPerPayload)
}

// PayloadAttributesParis is the original (no withdrawals) shape.
type PayloadAttributesParis struct {
	Timestamp             uint64
	PrevRandao            common.Hash
	SuggestedFeeRecipient common.Address
}

func (*PayloadAttributesParis) SizeSSZ(*ssz.Sizer) uint32 { return 60 }

func (a *PayloadAttributesParis) DefineSSZ(c *ssz.Codec) {
	ssz.DefineUint64(c, &a.Timestamp)
	ssz.DefineStaticBytes(c, &a.PrevRandao)
	ssz.DefineStaticBytes(c, &a.SuggestedFeeRecipient)
}

// ExecutionPayloadBodyCancun = Shanghai-shaped body (no BAL).
type ExecutionPayloadBodyCancun struct {
	Transactions [][]byte
	Withdrawals  []*Withdrawal
}

func (b *ExecutionPayloadBodyCancun) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(8)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicBytes(siz, b.Transactions)
	size += ssz.SizeSliceOfStaticObjects(siz, b.Withdrawals)
	return size
}

func (b *ExecutionPayloadBodyCancun) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicBytesOffset(c, &b.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsOffset(c, &b.Withdrawals, MaxWithdrawalsPerPayload)

	ssz.DefineSliceOfDynamicBytesContent(c, &b.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsContent(c, &b.Withdrawals, MaxWithdrawalsPerPayload)
}

// ExecutionPayloadBodyParis is the original transactions-only body.
type ExecutionPayloadBodyParis struct {
	Transactions [][]byte
}

func (b *ExecutionPayloadBodyParis) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicBytes(siz, b.Transactions)
	return size
}

func (b *ExecutionPayloadBodyParis) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicBytesOffset(c, &b.Transactions, MaxTxsPerPayload, MaxBytesPerTx)

	ssz.DefineSliceOfDynamicBytesContent(c, &b.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
}
