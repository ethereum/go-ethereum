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

// ExecutionPayloadEnvelopeAmsterdam is the monolithic request body of
// POST /{fork}/payloads spanning every fork from Paris onward. The wire shape
// is fork-driven by the codec (see the per-fork ExecutionPayloadEnvelope
// catalogue in refactor-ssz.md):
//
//   - Paris / Shanghai: bare payload, no envelope fields
//   - Cancun+:          + ParentBeaconBlockRoot
//   - Prague+:          + ExecutionRequests
//
// ParentBeaconBlockRoot is a pointer so the codec can distinguish "absent for
// this fork" (nil, Paris/Shanghai) from "present and zero".
type ExecutionPayloadEnvelopeAmsterdam struct {
	Payload               *ExecutionPayload
	ParentBeaconBlockRoot *common.Hash // Cancun+
	ExecutionRequests     [][]byte     // Prague+
}

func (e *ExecutionPayloadEnvelopeAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// offset(payload) is always present; the root and the requests offset only
	// exist from Cancun and Prague respectively (matching the OnFork gating).
	size := uint32(4)
	if siz.Fork() >= ssz.ForkDencun {
		size += 32 // ParentBeaconBlockRoot
	}
	if siz.Fork() >= ssz.ForkPectra {
		size += 4 // offset(requests)
	}
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, e.Payload)
	if siz.Fork() >= ssz.ForkPectra {
		size += ssz.SizeSliceOfDynamicBytes(siz, e.ExecutionRequests)
	}
	return size
}

func (e *ExecutionPayloadEnvelopeAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineDynamicObjectOffset(c, &e.Payload)
	ssz.DefineStaticBytesPointerOnFork(c, &e.ParentBeaconBlockRoot, fromCancun)
	ssz.DefineSliceOfDynamicBytesOffsetOnFork(c, &e.ExecutionRequests, MaxExecutionRequestsPerPayload, MaxBytesPerExecutionRequest, fromPrague)

	ssz.DefineDynamicObjectContent(c, &e.Payload)
	ssz.DefineSliceOfDynamicBytesContentOnFork(c, &e.ExecutionRequests, MaxExecutionRequestsPerPayload, MaxBytesPerExecutionRequest, fromPrague)
}

// BlobsBundleV1 is the single-proof blob bundle carried by BuiltPayload for
// Cancun and Prague: one KZG proof per blob.
type BlobsBundleV1 struct {
	Commitments [][48]byte
	Proofs      [][48]byte
	Blobs       []*Blob
}

func (b *BlobsBundleV1) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
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

func (b *BlobsBundleV1) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfStaticBytesOffset(c, &b.Commitments, MaxBlobCommitmentsPerBlock)
	ssz.DefineSliceOfStaticBytesOffset(c, &b.Proofs, MaxBlobCommitmentsPerBlock)
	ssz.DefineSliceOfStaticObjectsOffset(c, &b.Blobs, MaxBlobCommitmentsPerBlock)

	ssz.DefineSliceOfStaticBytesContent(c, &b.Commitments, MaxBlobCommitmentsPerBlock)
	ssz.DefineSliceOfStaticBytesContent(c, &b.Proofs, MaxBlobCommitmentsPerBlock)
	ssz.DefineSliceOfStaticObjectsContent(c, &b.Blobs, MaxBlobCommitmentsPerBlock)
}

// BlobsBundleV2 is the cell-proof blob bundle carried by BuiltPayload from Osaka
// onwards: CellsPerExtBlob cell proofs per blob. The wire layout matches
// BlobsBundleV1 (three slices); only the proofs-list bound differs.
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

// BuiltPayloadAmsterdam is the monolithic response of GET /{fork}/payloads/{id}
// spanning every fork from Paris onward. The wire shape is fork-driven by the
// codec (see the per-fork BuiltPayload catalogue in refactor-ssz.md):
//
//   - Paris / Shanghai: Payload, BlockValue only
//   - Cancun:           + BlobsBundleV1, ShouldOverrideBuilder
//   - Prague:           + ExecutionRequests (placed before ShouldOverrideBuilder)
//   - Osaka+:           BlobsBundle revision becomes V2 (cell proofs)
//
// Exactly one of BlobsBundleV1 / BlobsBundleV2 is active for any given fork
// (V1 = Cancun..Prague, V2 = Osaka+); the other is nil and contributes neither
// an offset nor content. ShouldOverrideBuilder is a pointer so the codec can
// distinguish "absent for this fork" (nil, Paris/Shanghai) from "present and
// false".
type BuiltPayloadAmsterdam struct {
	Payload               *ExecutionPayload
	BlockValue            *uint256.Int
	BlobsBundleV1         *BlobsBundleV1 // Cancun, Prague
	BlobsBundleV2         *BlobsBundleV2 // Osaka+
	ExecutionRequests     [][]byte       // Prague+
	ShouldOverrideBuilder *bool          // Cancun+
}

func (p *BuiltPayloadAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// offset(payload) + 32(value) are always present; the bundle offset and the
	// override byte exist from Cancun, and the requests offset from Prague.
	size := uint32(4 + 32)
	if siz.Fork() >= ssz.ForkDencun {
		size += 4 // offset(blobs_bundle)
		size++    // should_override_builder
	}
	if siz.Fork() >= ssz.ForkPectra {
		size += 4 // offset(execution_requests)
	}
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, p.Payload)
	if siz.Fork() >= forkOsaka {
		size += ssz.SizeDynamicObject(siz, p.BlobsBundleV2)
	} else if siz.Fork() >= ssz.ForkDencun {
		size += ssz.SizeDynamicObject(siz, p.BlobsBundleV1)
	}
	if siz.Fork() >= ssz.ForkPectra {
		size += ssz.SizeSliceOfDynamicBytes(siz, p.ExecutionRequests)
	}
	return size
}

func (p *BuiltPayloadAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineDynamicObjectOffset(c, &p.Payload)
	ssz.DefineUint256(c, &p.BlockValue)
	ssz.DefineDynamicObjectOffsetOnFork(c, &p.BlobsBundleV1, cancunToOsaka)
	ssz.DefineDynamicObjectOffsetOnFork(c, &p.BlobsBundleV2, fromOsaka)
	ssz.DefineSliceOfDynamicBytesOffsetOnFork(c, &p.ExecutionRequests, MaxExecutionRequestsPerPayload, MaxBytesPerExecutionRequest, fromPrague)
	ssz.DefineBoolPointerOnFork(c, &p.ShouldOverrideBuilder, fromCancun)

	ssz.DefineDynamicObjectContent(c, &p.Payload)
	ssz.DefineDynamicObjectContentOnFork(c, &p.BlobsBundleV1, cancunToOsaka)
	ssz.DefineDynamicObjectContentOnFork(c, &p.BlobsBundleV2, fromOsaka)
	ssz.DefineSliceOfDynamicBytesContentOnFork(c, &p.ExecutionRequests, MaxExecutionRequestsPerPayload, MaxBytesPerExecutionRequest, fromPrague)
}

// ForkchoiceUpdateAmsterdam is the request body of POST /amsterdam/forkchoice.
//
// PayloadAttributes is modelled as Optional[PayloadAttributesAmsterdam] = a slice
// of length 0 or 1. CustodyColumns is Optional[Bitvector[128]] — a slice of
// length 0 or 1 of the 16-byte Bitvector wrapper.
type ForkchoiceUpdateAmsterdam struct {
	ForkchoiceState   *ForkchoiceState
	PayloadAttributes []*PayloadAttributes
	CustodyColumns    []*Bitvector128
}

func (f *ForkchoiceUpdateAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// 96 (state) + 4(offset attrs); the custody offset only exists from
	// Amsterdam, matching the OnFork gating below.
	size := uint32(96 + 4)
	if siz.Fork() >= forkAmsterdam {
		size += 4 // offset custody
	}
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicObjects(siz, f.PayloadAttributes)
	if siz.Fork() >= forkAmsterdam {
		size += ssz.SizeSliceOfStaticObjects(siz, f.CustodyColumns)
	}
	return size
}

func (f *ForkchoiceUpdateAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineStaticObject(c, &f.ForkchoiceState)
	ssz.DefineSliceOfDynamicObjectsOffset(c, &f.PayloadAttributes, 1)
	ssz.DefineSliceOfStaticObjectsOffsetOnFork(c, &f.CustodyColumns, 1, fromAmsterdam)

	ssz.DefineSliceOfDynamicObjectsContent(c, &f.PayloadAttributes, 1)
	ssz.DefineSliceOfStaticObjectsContentOnFork(c, &f.CustodyColumns, 1, fromAmsterdam)
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
