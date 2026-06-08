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

// BodyEntryAmsterdam is the per-block entry returned by /amsterdam/bodies/...
type BodyEntryAmsterdam struct {
	Available bool
	Body      *ExecutionPayloadBodyAmsterdam
}

func (e *BodyEntryAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(1 + 4) // bool + offset
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, e.Body)
	return size
}

func (e *BodyEntryAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineBool(c, &e.Available)
	ssz.DefineDynamicObjectOffset(c, &e.Body)

	ssz.DefineDynamicObjectContent(c, &e.Body)
}

// BodiesResponseAmsterdam is the SSZ response of /amsterdam/bodies/...
type BodiesResponseAmsterdam struct {
	Entries []*BodyEntryAmsterdam
}

func (r *BodiesResponseAmsterdam) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicObjects(siz, r.Entries)
	return size
}

func (r *BodiesResponseAmsterdam) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicObjectsOffset(c, &r.Entries, MaxBodiesRequest)
	ssz.DefineSliceOfDynamicObjectsContent(c, &r.Entries, MaxBodiesRequest)
}

// BodyEntryCancun is the /cancun/bodies/... entry (Shanghai-shape body).
type BodyEntryCancun struct {
	Available bool
	Body      *ExecutionPayloadBodyCancun
}

func (e *BodyEntryCancun) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(1 + 4)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, e.Body)
	return size
}

func (e *BodyEntryCancun) DefineSSZ(c *ssz.Codec) {
	ssz.DefineBool(c, &e.Available)
	ssz.DefineDynamicObjectOffset(c, &e.Body)

	ssz.DefineDynamicObjectContent(c, &e.Body)
}

type BodiesResponseCancun struct {
	Entries []*BodyEntryCancun
}

func (r *BodiesResponseCancun) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicObjects(siz, r.Entries)
	return size
}

func (r *BodiesResponseCancun) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicObjectsOffset(c, &r.Entries, MaxBodiesRequest)
	ssz.DefineSliceOfDynamicObjectsContent(c, &r.Entries, MaxBodiesRequest)
}

// BodyEntryParis is the /paris/bodies/... entry (transactions only).
type BodyEntryParis struct {
	Available bool
	Body      *ExecutionPayloadBodyParis
}

func (e *BodyEntryParis) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(1 + 4)
	if fixed {
		return size
	}
	size += ssz.SizeDynamicObject(siz, e.Body)
	return size
}

func (e *BodyEntryParis) DefineSSZ(c *ssz.Codec) {
	ssz.DefineBool(c, &e.Available)
	ssz.DefineDynamicObjectOffset(c, &e.Body)

	ssz.DefineDynamicObjectContent(c, &e.Body)
}

type BodiesResponseParis struct {
	Entries []*BodyEntryParis
}

func (r *BodiesResponseParis) SizeSSZ(siz *ssz.Sizer, fixed bool) uint32 {
	size := uint32(4)
	if fixed {
		return size
	}
	size += ssz.SizeSliceOfDynamicObjects(siz, r.Entries)
	return size
}

func (r *BodiesResponseParis) DefineSSZ(c *ssz.Codec) {
	ssz.DefineSliceOfDynamicObjectsOffset(c, &r.Entries, MaxBodiesRequest)
	ssz.DefineSliceOfDynamicObjectsContent(c, &r.Entries, MaxBodiesRequest)
}
