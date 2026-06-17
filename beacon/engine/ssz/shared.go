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

// Blob wraps a single 131072-byte blob. karalabe/ssz's generic
// constraint doesn't list 131072 as a "common" fixed length, so we
// wrap it as a static-object containing a length-checked []byte.
type Blob struct {
	Bytes []byte
}

func (*Blob) SizeSSZ(*ssz.Sizer) uint32 { return BytesPerBlob }

func (b *Blob) DefineSSZ(c *ssz.Codec) {
	ssz.DefineCheckedStaticBytes(c, &b.Bytes, BytesPerBlob)
}

// BlobCell wraps a single BYTES_PER_CELL = 2048 byte cell. Same wrapper
// rationale as Blob.
type BlobCell struct {
	Bytes []byte
}

func (*BlobCell) SizeSSZ(*ssz.Sizer) uint32 { return BytesPerCell }

func (b *BlobCell) DefineSSZ(c *ssz.Codec) {
	ssz.DefineCheckedStaticBytes(c, &b.Bytes, BytesPerCell)
}

// Bitvector128 wraps a packed bitvector of length CELLS_PER_EXT_BLOB = 128
// (16 bytes). Same wrapper rationale as Blob.
type Bitvector128 struct {
	Bytes []byte
}

func (*Bitvector128) SizeSSZ(*ssz.Sizer) uint32 { return CellsPerExtBlob / 8 }

func (b *Bitvector128) DefineSSZ(c *ssz.Codec) {
	ssz.DefineCheckedStaticBytes(c, &b.Bytes, CellsPerExtBlob/8)
}

// OptionalBlobCell models `Optional[ByteVector[BYTES_PER_CELL]]` =
// `List[BlobCell, 1]`. Length 0 = absent, length 1 = present (2048 bytes).
type OptionalBlobCell struct {
	Cells []*BlobCell
}

func (o *OptionalBlobCell) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	if fixed {
		return 0
	}
	return ssz.SizeSliceOfStaticObjects(siz, o.Cells)
}

func (o *OptionalBlobCell) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfStaticObjectsContent(c, &o.Cells, 1)
}

func (o *OptionalBlobCell) Validate() error { return checkOptional(o.Cells) }

// OptionalProof models `Optional[Bytes48]` = `List[Bytes48, 1]`.
type OptionalProof struct {
	Proofs [][48]byte
}

func (o *OptionalProof) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	if fixed {
		return 0
	}
	return ssz.SizeSliceOfStaticBytes(siz, o.Proofs)
}

func (o *OptionalProof) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfStaticBytesContent(c, &o.Proofs, 1)
}

func (o *OptionalProof) Validate() error { return checkOptional(o.Proofs) }

// Withdrawal mirrors the consensus-specs Withdrawal container.
type Withdrawal struct {
	Index          uint64
	ValidatorIndex uint64
	Address        common.Address
	Amount         uint64
}

func (*Withdrawal) SizeSSZ(*ssz.Sizer) uint32 { return 44 }

func (w *Withdrawal) DefineSSZ(c *ssz.Codec) {
	ssz.DefineUint64(c, &w.Index)
	ssz.DefineUint64(c, &w.ValidatorIndex)
	ssz.DefineStaticBytes(c, &w.Address)
	ssz.DefineUint64(c, &w.Amount)
}

// ForkchoiceState carries head/safe/finalized block hashes.
type ForkchoiceState struct {
	HeadBlockHash      common.Hash
	SafeBlockHash      common.Hash
	FinalizedBlockHash common.Hash
}

func (*ForkchoiceState) SizeSSZ(*ssz.Sizer) uint32 { return 96 }

func (f *ForkchoiceState) DefineSSZ(c *ssz.Codec) {
	ssz.DefineStaticBytes(c, &f.HeadBlockHash)
	ssz.DefineStaticBytes(c, &f.SafeBlockHash)
	ssz.DefineStaticBytes(c, &f.FinalizedBlockHash)
}

// PayloadStatus is the response shape for /payloads and the inner status of
// /forkchoice. Status values are defined as constants in constants.go.
type PayloadStatus struct {
	Status          uint8
	LatestValidHash []common.Hash // Optional[Hash32] — length 0 or 1
	ValidationError [][]byte      // Optional[String]  — length 0 or 1
}

func (p *PayloadStatus) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	// status(1) + offset(4) + offset(4)
	size := uint32(9)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfStaticBytes(siz, p.LatestValidHash)
	size += ssz.SizeSliceOfDynamicBytes(siz, p.ValidationError)
	return size
}

func (p *PayloadStatus) DefineSSZ(c *ssz.Codec) {
	ssz.DefineUint8(c, &p.Status)
	ssz.DefineSliceOfStaticBytesOffset(c, &p.LatestValidHash, 1)
	ssz.DefineSliceOfDynamicBytesOffset(c, &p.ValidationError, 1, MaxErrorBytes)

	ssz.DefineSliceOfStaticBytesContent(c, &p.LatestValidHash, 1)
	ssz.DefineSliceOfDynamicBytesContent(c, &p.ValidationError, 1, MaxErrorBytes)
}

// Validate enforces the Optional[T] = List[T,1] length invariants.
func (p *PayloadStatus) Validate() error {
	if err := checkOptional(p.LatestValidHash); err != nil {
		return err
	}
	return checkOptional(p.ValidationError)
}
