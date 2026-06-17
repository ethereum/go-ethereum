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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/karalabe/ssz"
)

// Fork-boundary filters. Fields gated by these only participate in size,
// encode, decode and hash when Codec.fork is in range — see forkmap.go for how
// the geth fork maps onto ssz.Fork. Grouping the filters here (rather than one
// flag per field) keeps the "field-group" structure visible: each filter is a
// group boundary, and DefineSSZ applies it to every field the group owns.
var (
	// withdrawals group, introduced in Shanghai (Shapella).
	fromShanghai = ssz.ForkFilter{Added: ssz.ForkShapella}
	// blob-gas group, introduced in Cancun (Dencun).
	fromCancun = ssz.ForkFilter{Added: ssz.ForkDencun}
	// beacon-root attribute, introduced in Cancun (Dencun).
	// (shares the Cancun boundary but kept named for clarity at call sites)
	// execution-requests group, introduced in Prague (Pectra).
	fromPrague = ssz.ForkFilter{Added: ssz.ForkPectra}
	// single-proof blobs bundle (BlobsBundleV1), active Cancun..Prague; replaced
	// by the cell-proof BlobsBundleV2 from Osaka on.
	cancunToOsaka = ssz.ForkFilter{Added: ssz.ForkDencun, Removed: forkOsaka}
	// cell-proof blobs bundle (BlobsBundleV2), introduced in Osaka.
	fromOsaka = ssz.ForkFilter{Added: forkOsaka}
	// bal + slot group, introduced in Amsterdam.
	fromAmsterdam = ssz.ForkFilter{Added: forkAmsterdam}
)

// ExecutionPayload is the monolithic execution payload spanning every fork from
// Paris onward. Fork-specific fields are gated with ...OnFork; the in-memory
// struct is a superset and the fork passed to the codec selects the wire shape.
//
// Field groups (see the filter vars above):
//   - base (Paris): ParentHash … Transactions
//   - withdrawals (Shanghai): Withdrawals
//   - blobGas (Cancun): BlobGasUsed, ExcessBlobGas
//   - bal (Amsterdam): BlockAccessList, SlotNumber
//
// Gated scalars are pointers so the codec can distinguish "absent for this
// fork" (nil) from "present and zero"; Validate(fork) enforces that invariant.
type ExecutionPayload struct {
	// base group (Paris)
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

	// withdrawals group (Shanghai)
	Withdrawals []*Withdrawal

	// blobGas group (Cancun)
	BlobGasUsed   *uint64
	ExcessBlobGas *uint64

	// bal group (Amsterdam)
	BlockAccessList []byte
	SlotNumber      *uint64
}

func (p *ExecutionPayload) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// Fixed portion is fork-dependent; let the codec primitives account for it
	// via the active fork rather than hand-summing magic byte constants.
	var size uint32
	// base fixed: 5 hashes(32) + addr(20) + bloom(256) + 4 u64s + uint256(32)
	// + 2 offsets(extra, txs)(4)
	size += 5*32 + 20 + 256 + 4*8 + 32 + 2*4
	if siz.Fork() >= ssz.ForkShapella {
		size += 4 // withdrawals offset
	}
	if siz.Fork() >= ssz.ForkDencun {
		size += 2 * 8 // blob_gas_used, excess_blob_gas
	}
	if siz.Fork() >= forkAmsterdam {
		size += 4 + 8 // block_access_list offset, slot_number
	}
	if fixed {
		return size
	}
	size += ssz.SizeDynamicBytes(siz, p.ExtraData)
	size += ssz.SizeSliceOfDynamicBytes(siz, p.Transactions)
	if siz.Fork() >= ssz.ForkShapella {
		size += ssz.SizeSliceOfStaticObjects(siz, p.Withdrawals)
	}
	if siz.Fork() >= forkAmsterdam {
		size += ssz.SizeDynamicBytes(siz, p.BlockAccessList)
	}
	return size
}

func (p *ExecutionPayload) DefineSSZ(c *ssz.Codec) {
	// --- offset phase ---
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
	ssz.DefineSliceOfStaticObjectsOffsetOnFork(c, &p.Withdrawals, MaxWithdrawalsPerPayload, fromShanghai)
	ssz.DefineUint64PointerOnFork(c, &p.BlobGasUsed, fromCancun)
	ssz.DefineUint64PointerOnFork(c, &p.ExcessBlobGas, fromCancun)
	ssz.DefineDynamicBytesOffsetOnFork(c, &p.BlockAccessList, MaxBalBytes, fromAmsterdam)
	ssz.DefineUint64PointerOnFork(c, &p.SlotNumber, fromAmsterdam)

	// --- content phase ---
	ssz.DefineDynamicBytesContent(c, &p.ExtraData, MaxExtraDataBytes)
	ssz.DefineSliceOfDynamicBytesContent(c, &p.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsContentOnFork(c, &p.Withdrawals, MaxWithdrawalsPerPayload, fromShanghai)
	ssz.DefineDynamicBytesContentOnFork(c, &p.BlockAccessList, MaxBalBytes, fromAmsterdam)
}

// Validate checks that the in-memory field set matches what the given fork's
// wire shape carries: gated scalars must be present iff active, and absent
// fields must be empty so the codec doesn't silently drop populated data.
func (p *ExecutionPayload) Validate(fork ssz.Fork) error {
	// withdrawals (Shanghai+)
	if fork < ssz.ForkShapella && len(p.Withdrawals) > 0 {
		return fmt.Errorf("withdrawals set but fork %d predates Shanghai", fork)
	}
	// blobGas (Cancun+)
	if fork >= ssz.ForkDencun {
		if p.BlobGasUsed == nil || p.ExcessBlobGas == nil {
			return fmt.Errorf("blob gas fields required at fork %d", fork)
		}
	} else if p.BlobGasUsed != nil || p.ExcessBlobGas != nil {
		return fmt.Errorf("blob gas fields set but fork %d predates Cancun", fork)
	}
	// bal + slot (Amsterdam+)
	if fork >= forkAmsterdam {
		if p.SlotNumber == nil {
			return fmt.Errorf("slot_number required at fork %d", fork)
		}
	} else {
		if len(p.BlockAccessList) > 0 {
			return fmt.Errorf("block_access_list set but fork %d predates Amsterdam", fork)
		}
		if p.SlotNumber != nil {
			return fmt.Errorf("slot_number set but fork %d predates Amsterdam", fork)
		}
	}
	return nil
}

// PayloadAttributes is the monolithic build-attributes container spanning every
// fork from Paris onward.
//
// Field groups:
//   - base (Paris): Timestamp, PrevRandao, SuggestedFeeRecipient
//   - withdrawals (Shanghai): Withdrawals
//   - beaconRoot (Cancun): ParentBeaconBlockRoot
//   - amsterdam (Amsterdam): SlotNumber, TargetGasLimit
type PayloadAttributes struct {
	Timestamp             uint64
	PrevRandao            common.Hash
	SuggestedFeeRecipient common.Address

	Withdrawals []*Withdrawal // Shanghai+

	ParentBeaconBlockRoot *common.Hash // Cancun+

	SlotNumber     *uint64 // Amsterdam+
	TargetGasLimit *uint64 // Amsterdam+
}

func (a *PayloadAttributes) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// base fixed: timestamp(8) + randao(32) + fee_recipient(20)
	var size uint32 = 8 + 32 + 20
	if siz.Fork() >= ssz.ForkShapella {
		size += 4 // withdrawals offset
	}
	if siz.Fork() >= ssz.ForkDencun {
		size += 32 // parent_beacon_block_root
	}
	if siz.Fork() >= forkAmsterdam {
		size += 8 + 8 // slot_number, target_gas_limit
	}
	if fixed {
		return size
	}
	if siz.Fork() >= ssz.ForkShapella {
		size += ssz.SizeSliceOfStaticObjects(siz, a.Withdrawals)
	}
	return size
}

func (a *PayloadAttributes) DefineSSZ(c *ssz.Codec) {
	// offset phase
	ssz.DefineUint64(c, &a.Timestamp)
	ssz.DefineStaticBytes(c, &a.PrevRandao)
	ssz.DefineStaticBytes(c, &a.SuggestedFeeRecipient)
	ssz.DefineSliceOfStaticObjectsOffsetOnFork(c, &a.Withdrawals, MaxWithdrawalsPerPayload, fromShanghai)
	ssz.DefineStaticBytesPointerOnFork(c, &a.ParentBeaconBlockRoot, fromCancun)
	ssz.DefineUint64PointerOnFork(c, &a.SlotNumber, fromAmsterdam)
	ssz.DefineUint64PointerOnFork(c, &a.TargetGasLimit, fromAmsterdam)

	// content phase
	ssz.DefineSliceOfStaticObjectsContentOnFork(c, &a.Withdrawals, MaxWithdrawalsPerPayload, fromShanghai)
}

// Validate enforces presence/absence of gated fields for the given fork.
func (a *PayloadAttributes) Validate(fork ssz.Fork) error {
	if fork < ssz.ForkShapella && len(a.Withdrawals) > 0 {
		return fmt.Errorf("withdrawals set but fork %d predates Shanghai", fork)
	}
	if fork >= ssz.ForkDencun {
		if a.ParentBeaconBlockRoot == nil {
			return fmt.Errorf("parent_beacon_block_root required at fork %d", fork)
		}
	} else if a.ParentBeaconBlockRoot != nil {
		return fmt.Errorf("parent_beacon_block_root set but fork %d predates Cancun", fork)
	}
	if fork >= forkAmsterdam {
		if a.SlotNumber == nil || a.TargetGasLimit == nil {
			return fmt.Errorf("slot_number and target_gas_limit required at fork %d", fork)
		}
	} else if a.SlotNumber != nil || a.TargetGasLimit != nil {
		return fmt.Errorf("amsterdam attributes set but fork %d predates Amsterdam", fork)
	}
	return nil
}
