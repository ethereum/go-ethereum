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
	"github.com/karalabe/ssz"
)

// BodiesByHashRequest is the SSZ body of POST /{fork}/bodies/hash.
type BodiesByHashRequest struct {
	BlockHashes []common.Hash
}

func (r *BodiesByHashRequest) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticBytes(siz, r.BlockHashes)
	return size
}

func (r *BodiesByHashRequest) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfStaticBytesOffset(c, &r.BlockHashes, MaxBodiesRequest)
	ssz.DefineSliceOfStaticBytesContent(c, &r.BlockHashes, MaxBodiesRequest)
}

// ExecutionPayloadBody is the monolithic body shape returned by /{fork}/bodies.
//
// Field groups:
//   - base (Paris): Transactions
//   - withdrawals (Shanghai): Withdrawals
//   - bal (Amsterdam): BlockAccessList
type ExecutionPayloadBody struct {
	Transactions    [][]byte
	Withdrawals     []*Withdrawal // Shanghai+
	BlockAccessList []byte        // Amsterdam+
}

func (b *ExecutionPayloadBody) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	var size uint32 = 4 // transactions offset
	if siz.Fork() >= ssz.ForkShapella {
		size += 4 // withdrawals offset
	}
	if siz.Fork() >= forkAmsterdam {
		size += 4 // block_access_list offset
	}
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicBytes(siz, b.Transactions)
	if siz.Fork() >= ssz.ForkShapella {
		size += ssz.SizeSliceOfStaticObjects(siz, b.Withdrawals)
	}
	if siz.Fork() >= forkAmsterdam {
		size += ssz.SizeDynamicBytes(siz, b.BlockAccessList)
	}
	return size
}

func (b *ExecutionPayloadBody) DefineSSZ(c *ssz.Codec) {
	// offset phase
	ssz.DefineSliceOfDynamicBytesOffset(c, &b.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsOffsetOnFork(c, &b.Withdrawals, MaxWithdrawalsPerPayload, fromShanghai)
	ssz.DefineDynamicBytesOffsetOnFork(c, &b.BlockAccessList, MaxBalBytes, fromAmsterdam)

	// content phase
	ssz.DefineSliceOfDynamicBytesContent(c, &b.Transactions, MaxTxsPerPayload, MaxBytesPerTx)
	ssz.DefineSliceOfStaticObjectsContentOnFork(c, &b.Withdrawals, MaxWithdrawalsPerPayload, fromShanghai)
	ssz.DefineDynamicBytesContentOnFork(c, &b.BlockAccessList, MaxBalBytes, fromAmsterdam)
}

// Validate enforces that fields absent for the given fork are empty.
func (b *ExecutionPayloadBody) Validate(fork ssz.Fork) error {
	if fork < ssz.ForkShapella && len(b.Withdrawals) > 0 {
		return fmt.Errorf("withdrawals set but fork %d predates Shanghai", fork)
	}
	if fork < forkAmsterdam && len(b.BlockAccessList) > 0 {
		return fmt.Errorf("block_access_list set but fork %d predates Amsterdam", fork)
	}
	return nil
}

// BodyEntry is the per-block entry returned by /{fork}/bodies/... The Body's
// wire shape follows the codec fork, so this single type spans all forks.
type BodyEntry struct {
	Available bool
	Body      *ExecutionPayloadBody
}

func (e *BodyEntry) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(1 + 4) // bool + offset
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, e.Body)
	return size
}

func (e *BodyEntry) DefineSSZ(c *ssz.Codec) {
	ssz.DefineBool(c, &e.Available)
	ssz.DefineDynamicObjectOffset(c, &e.Body)

	ssz.DefineDynamicObjectContent(c, &e.Body)
}

// BodiesResponse is the SSZ response of /{fork}/bodies/...
type BodiesResponse struct {
	Entries []*BodyEntry
}

func (r *BodiesResponse) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicObjects(siz, r.Entries)
	return size
}

func (r *BodiesResponse) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicObjectsOffset(c, &r.Entries, MaxBodiesRequest)
	ssz.DefineSliceOfDynamicObjectsContent(c, &r.Entries, MaxBodiesRequest)
}
