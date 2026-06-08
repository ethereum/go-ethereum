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

// BlobsVersionedHashesRequest is the SSZ body of POST /blobs/v{1,2,3}.
type BlobsVersionedHashesRequest struct {
	VersionedHashes []common.Hash
}

func (r *BlobsVersionedHashesRequest) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticBytes(siz, r.VersionedHashes)
	return size
}

func (r *BlobsVersionedHashesRequest) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfStaticBytesOffset(c, &r.VersionedHashes, MaxBlobsRequest)
	ssz.DefineSliceOfStaticBytesContent(c, &r.VersionedHashes, MaxBlobsRequest)
}

// BlobAndProofV1 — Cancun whole-blob with one KZG proof.
type BlobAndProofV1 struct {
	Blob  *Blob
	Proof [48]byte
}

func (*BlobAndProofV1) SizeSSZ(*ssz.Sizer) uint32 { return BytesPerBlob + 48 }

func (e *BlobAndProofV1) DefineSSZ(c *ssz.Codec) {
	ssz.DefineStaticObject(c, &e.Blob)
	ssz.DefineStaticBytes(c, &e.Proof)
}

// BlobV1Entry pairs an availability flag with the contents.
type BlobV1Entry struct {
	Available bool
	Contents  *BlobAndProofV1
}

func (*BlobV1Entry) SizeSSZ(*ssz.Sizer) uint32 { return 1 + BytesPerBlob + 48 }

func (e *BlobV1Entry) DefineSSZ(c *ssz.Codec) {
	ssz.DefineBool(c, &e.Available)
	ssz.DefineStaticObject(c, &e.Contents)
}

// BlobsV1Response is the SSZ response of POST /blobs/v1.
type BlobsV1Response struct {
	Entries []*BlobV1Entry
}

func (r *BlobsV1Response) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticObjects(siz, r.Entries)
	return size
}

func (r *BlobsV1Response) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfStaticObjectsOffset(c, &r.Entries, MaxBlobsRequest)
	ssz.DefineSliceOfStaticObjectsContent(c, &r.Entries, MaxBlobsRequest)
}

// BlobAndProofV2 — Osaka whole-blob with CELLS_PER_EXT_BLOB cell proofs.
// proofs is a List with maxItems=128, so it is dynamic.
type BlobAndProofV2 struct {
	Blob   *Blob
	Proofs [][48]byte
}

func (b *BlobAndProofV2) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// blob (static) + offset(proofs)
	size := uint32(BytesPerBlob + 4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticBytes(siz, b.Proofs)
	return size
}

func (b *BlobAndProofV2) DefineSSZ(c *ssz.Codec) {
	ssz.DefineStaticObject(c, &b.Blob)
	ssz.DefineSliceOfStaticBytesOffset(c, &b.Proofs, CellsPerExtBlob)

	ssz.DefineSliceOfStaticBytesContent(c, &b.Proofs, CellsPerExtBlob)
}

// BlobV2Entry is shared by /blobs/v2 and /blobs/v3 (V3 just enables
// per-entry available=false reporting).
type BlobV2Entry struct {
	Available bool
	Contents  *BlobAndProofV2
}

func (e *BlobV2Entry) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(1 + 4) // bool + offset(contents)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, e.Contents)
	return size
}

func (e *BlobV2Entry) DefineSSZ(c *ssz.Codec) {
	ssz.DefineBool(c, &e.Available)
	ssz.DefineDynamicObjectOffset(c, &e.Contents)

	ssz.DefineDynamicObjectContent(c, &e.Contents)
}

type BlobsV2Response struct {
	Entries []*BlobV2Entry
}

func (r *BlobsV2Response) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicObjects(siz, r.Entries)
	return size
}

func (r *BlobsV2Response) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicObjectsOffset(c, &r.Entries, MaxBlobsRequest)
	ssz.DefineSliceOfDynamicObjectsContent(c, &r.Entries, MaxBlobsRequest)
}

// BlobsV4Request is the SSZ body of POST /blobs/v4 (cell-range selection).
type BlobsV4Request struct {
	VersionedHashes []common.Hash
	IndicesBitarray *Bitvector128
}

func (r *BlobsV4Request) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// offset(hashes) + 16(bitvector)
	size := uint32(4 + CellsPerExtBlob/8)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticBytes(siz, r.VersionedHashes)
	return size
}

func (r *BlobsV4Request) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfStaticBytesOffset(c, &r.VersionedHashes, MaxBlobsRequest)
	ssz.DefineStaticObject(c, &r.IndicesBitarray)

	ssz.DefineSliceOfStaticBytesContent(c, &r.VersionedHashes, MaxBlobsRequest)
}

// BlobCellsAndProofs — Amsterdam cell-range response payload.
type BlobCellsAndProofs struct {
	BlobCells []*OptionalBlobCell // List[Optional[ByteVector[BYTES_PER_CELL]], CELLS_PER_EXT_BLOB]
	Proofs    []*OptionalProof    // List[Optional[Bytes48], CELLS_PER_EXT_BLOB]
}

func (b *BlobCellsAndProofs) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(8) // 2 offsets
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicObjects(siz, b.BlobCells)
	size += ssz.SizeSliceOfDynamicObjects(siz, b.Proofs)
	return size
}

func (b *BlobCellsAndProofs) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicObjectsOffset(c, &b.BlobCells, CellsPerExtBlob)
	ssz.DefineSliceOfDynamicObjectsOffset(c, &b.Proofs, CellsPerExtBlob)

	ssz.DefineSliceOfDynamicObjectsContent(c, &b.BlobCells, CellsPerExtBlob)
	ssz.DefineSliceOfDynamicObjectsContent(c, &b.Proofs, CellsPerExtBlob)
}

// BlobV4Entry pairs availability with the cell-range contents.
type BlobV4Entry struct {
	Available bool
	Contents  *BlobCellsAndProofs
}

func (e *BlobV4Entry) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(1 + 4)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, e.Contents)
	return size
}

func (e *BlobV4Entry) DefineSSZ(c *ssz.Codec) {
	ssz.DefineBool(c, &e.Available)
	ssz.DefineDynamicObjectOffset(c, &e.Contents)

	ssz.DefineDynamicObjectContent(c, &e.Contents)
}

type BlobsV4Response struct {
	Entries []*BlobV4Entry
}

func (r *BlobsV4Response) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicObjects(siz, r.Entries)
	return size
}

func (r *BlobsV4Response) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicObjectsOffset(c, &r.Entries, MaxBlobsRequest)
	ssz.DefineSliceOfDynamicObjectsContent(c, &r.Entries, MaxBlobsRequest)
}
