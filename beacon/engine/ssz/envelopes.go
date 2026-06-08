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

// ExecutionPayloadEnvelopeAmsterdam is the request body of
// POST /amsterdam/payloads.
type ExecutionPayloadEnvelopeAmsterdam struct {
	Payload               *ExecutionPayloadAmsterdam
	ParentBeaconBlockRoot common.Hash
	ExecutionRequests     [][]byte
}

func (e *ExecutionPayloadEnvelopeAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// offset(payload) + 32(root) + offset(requests)
	size := uint32(40)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, e.Payload)
	size += ssz.SizeSliceOfDynamicBytes(siz, e.ExecutionRequests)
	return size
}

func (e *ExecutionPayloadEnvelopeAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineDynamicObjectOffset(c, &e.Payload)
	ssz.DefineStaticBytes(c, &e.ParentBeaconBlockRoot)
	ssz.DefineSliceOfDynamicBytesOffset(c, &e.ExecutionRequests, MaxExecutionRequestsPerPayload, MaxBytesPerExecutionRequest)

	ssz.DefineDynamicObjectContent(c, &e.Payload)
	ssz.DefineSliceOfDynamicBytesContent(c, &e.ExecutionRequests, MaxExecutionRequestsPerPayload, MaxBytesPerExecutionRequest)
}

// BlobsBundleV2 is the cell-proof blob bundle carried by BuiltPayload
// from Osaka onwards.
type BlobsBundleV2 struct {
	Commitments [][48]byte
	Proofs      [][48]byte
	Blobs       []*Blob
}

func (b *BlobsBundleV2) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// 3 offsets
	size := uint32(12)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticBytes(siz, b.Commitments)
	size += ssz.SizeSliceOfStaticBytes(siz, b.Proofs)
	size += ssz.SizeSliceOfStaticObjects(siz, b.Blobs)
	return size
}

func (b *BlobsBundleV2) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfStaticBytesOffset(c, &b.Commitments, MaxBlobCommitmentsPerBlock)
	ssz.DefineSliceOfStaticBytesOffset(c, &b.Proofs, MaxBlobCommitmentsPerBlock*CellsPerExtBlob)
	ssz.DefineSliceOfStaticObjectsOffset(c, &b.Blobs, MaxBlobCommitmentsPerBlock)

	ssz.DefineSliceOfStaticBytesContent(c, &b.Commitments, MaxBlobCommitmentsPerBlock)
	ssz.DefineSliceOfStaticBytesContent(c, &b.Proofs, MaxBlobCommitmentsPerBlock*CellsPerExtBlob)
	ssz.DefineSliceOfStaticObjectsContent(c, &b.Blobs, MaxBlobCommitmentsPerBlock)
}

// BuiltPayloadAmsterdam is the response of GET /amsterdam/payloads/{id}.
type BuiltPayloadAmsterdam struct {
	Payload               *ExecutionPayloadAmsterdam
	BlockValue            *uint256.Int
	BlobsBundle           *BlobsBundleV2
	ExecutionRequests     [][]byte
	ShouldOverrideBuilder bool
}

func (p *BuiltPayloadAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// offset(payload) + 32(value) + offset(bundle) + offset(requests) + 1(override)
	size := uint32(4 + 32 + 4 + 4 + 1)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, p.Payload)
	size += ssz.SizeDynamicObject(siz, p.BlobsBundle)
	size += ssz.SizeSliceOfDynamicBytes(siz, p.ExecutionRequests)
	return size
}

func (p *BuiltPayloadAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineDynamicObjectOffset(c, &p.Payload)
	ssz.DefineUint256(c, &p.BlockValue)
	ssz.DefineDynamicObjectOffset(c, &p.BlobsBundle)
	ssz.DefineSliceOfDynamicBytesOffset(c, &p.ExecutionRequests, MaxExecutionRequestsPerPayload, MaxBytesPerExecutionRequest)
	ssz.DefineBool(c, &p.ShouldOverrideBuilder)

	ssz.DefineDynamicObjectContent(c, &p.Payload)
	ssz.DefineDynamicObjectContent(c, &p.BlobsBundle)
	ssz.DefineSliceOfDynamicBytesContent(c, &p.ExecutionRequests, MaxExecutionRequestsPerPayload, MaxBytesPerExecutionRequest)
}

// ForkchoiceUpdateAmsterdam is the request body of POST /amsterdam/forkchoice.
//
// PayloadAttributes is modelled as Optional[PayloadAttributesAmsterdam] = a slice
// of length 0 or 1. CustodyColumns is Optional[Bitvector[128]] — a slice of
// length 0 or 1 of the 16-byte Bitvector wrapper.
type ForkchoiceUpdateAmsterdam struct {
	ForkchoiceState   *ForkchoiceState
	PayloadAttributes []*PayloadAttributesAmsterdam
	CustodyColumns    []*Bitvector128
}

func (f *ForkchoiceUpdateAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// 96 (state) + 4(offset attrs) + 4(offset custody)
	size := uint32(96 + 8)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicObjects(siz, f.PayloadAttributes)
	size += ssz.SizeSliceOfStaticObjects(siz, f.CustodyColumns)
	return size
}

func (f *ForkchoiceUpdateAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineStaticObject(c, &f.ForkchoiceState)
	ssz.DefineSliceOfDynamicObjectsOffset(c, &f.PayloadAttributes, 1)
	ssz.DefineSliceOfStaticObjectsOffset(c, &f.CustodyColumns, 1)

	ssz.DefineSliceOfDynamicObjectsContent(c, &f.PayloadAttributes, 1)
	ssz.DefineSliceOfStaticObjectsContent(c, &f.CustodyColumns, 1)
}

// Validate enforces the Optional[T] = List[T,1] length invariants.
func (f *ForkchoiceUpdateAmsterdam) Validate() error {
	if err := checkOptional(f.PayloadAttributes); err != nil {
		return err
	}
	return checkOptional(f.CustodyColumns)
}

// ForkchoiceUpdateResponseAmsterdam is the response of POST /amsterdam/forkchoice.
type ForkchoiceUpdateResponseAmsterdam struct {
	PayloadStatus *PayloadStatus
	PayloadID     [][8]byte // Optional[Bytes8]
}

func (r *ForkchoiceUpdateResponseAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// offset(status) + offset(id)
	size := uint32(8)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, r.PayloadStatus)
	size += ssz.SizeSliceOfStaticBytes(siz, r.PayloadID)
	return size
}

func (r *ForkchoiceUpdateResponseAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineDynamicObjectOffset(c, &r.PayloadStatus)
	ssz.DefineSliceOfStaticBytesOffset(c, &r.PayloadID, 1)

	ssz.DefineDynamicObjectContent(c, &r.PayloadStatus)
	ssz.DefineSliceOfStaticBytesContent(c, &r.PayloadID, 1)
}

// Validate enforces optional-list invariants and inner status invariants.
func (r *ForkchoiceUpdateResponseAmsterdam) Validate() error {
	if err := checkOptional(r.PayloadID); err != nil {
		return err
	}
	if r.PayloadStatus == nil {
		return nil
	}
	return r.PayloadStatus.Validate()
}
