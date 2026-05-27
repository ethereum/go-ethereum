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

package engine

import (
	"github.com/fjl/jsonw"
)

// MarshalJSON implements json.Marshaler.
func (list BlobAndProofListV1) MarshalJSON() ([]byte, error) {
	if list == nil {
		return []byte("null"), nil
	}
	var b jsonw.Buffer
	b.Array(func() {
		for _, item := range list {
			marshalBlobAndProofV1(&b, item)
		}
	})
	return b.Output(), nil
}

func marshalBlobAndProofV1(b *jsonw.Buffer, item *BlobAndProofV1) {
	if item == nil {
		b.Null()
	} else {
		b.Object(func() {
			b.Key("blob")
			b.HexBytes(item.Blob)
			b.Key("proof")
			b.HexBytes(item.Proof)
		})
	}
}

// MarshalJSON implements json.Marshaler.
func (list BlobAndProofListV2) MarshalJSON() ([]byte, error) {
	if list == nil {
		return []byte("null"), nil
	}
	var b jsonw.Buffer
	b.Array(func() {
		for _, item := range list {
			marshalBlobAndProofV2(&b, item)
		}
	})
	return b.Output(), nil
}

func marshalBlobAndProofV2(b *jsonw.Buffer, item *BlobAndProofV2) {
	if item == nil {
		b.Null()
	} else {
		b.Object(func() {
			b.Key("blob")
			b.HexBytes(item.Blob)
			b.Key("proofs")
			appendHexBytesArray(b, item.CellProofs)
		})
	}
}
