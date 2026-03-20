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
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// canonicalBlobAndProofV1 is a reference type for BlobAndProofV1 that uses
// standard json.Marshal (no custom MarshalJSON).
type canonicalBlobAndProofV1 struct {
	Blob  hexutil.Bytes `json:"blob"`
	Proof hexutil.Bytes `json:"proof"`
}

// canonicalBlobAndProofV2 is a reference type for BlobAndProofV2 that uses
// standard json.Marshal (no custom MarshalJSON).
type canonicalBlobAndProofV2 struct {
	Blob       hexutil.Bytes   `json:"blob"`
	CellProofs []hexutil.Bytes `json:"proofs"`
}

func toCanonicalBlobAndProofListV1(list BlobAndProofListV1) []*canonicalBlobAndProofV1 {
	canonical := make([]*canonicalBlobAndProofV1, len(list))
	for i, item := range list {
		if item == nil {
			continue
		}
		canonical[i] = &canonicalBlobAndProofV1{
			Blob:  item.Blob,
			Proof: item.Proof,
		}
	}
	return canonical
}

func toCanonicalBlobAndProofListV2(list BlobAndProofListV2) []*canonicalBlobAndProofV2 {
	canonical := make([]*canonicalBlobAndProofV2, len(list))
	for i, item := range list {
		if item == nil {
			continue
		}
		canonical[i] = &canonicalBlobAndProofV2{
			Blob:       item.Blob,
			CellProofs: item.CellProofs,
		}
	}
	return canonical
}

func TestBlobAndProofListV1MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		list BlobAndProofListV1
	}{
		{
			name: "multiple items",
			list: BlobAndProofListV1{
				{
					Blob:  hexutil.Bytes{0x01, 0x02},
					Proof: hexutil.Bytes{0x03, 0x04},
				},
				{
					Blob:  hexutil.Bytes{},
					Proof: hexutil.Bytes{0x05},
				},
			},
		},
		{
			name: "nil item",
			list: BlobAndProofListV1{
				nil,
				{
					Blob:  hexutil.Bytes{0xaa},
					Proof: hexutil.Bytes{0xbb},
				},
			},
		},
		{
			name: "empty list",
			list: BlobAndProofListV1{},
		},
		{
			name: "nil list",
			list: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.list.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON error: %v", err)
			}
			want, err := json.Marshal(toCanonicalBlobAndProofListV1(tt.list))
			if err != nil {
				t.Fatalf("canonical marshal error: %v", err)
			}
			if !bytes.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("JSON mismatch\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func TestBlobAndProofListV1UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		json string
		want BlobAndProofListV1
	}{
		{
			name: "multiple items",
			json: `[{"blob":"0x0102","proof":"0x0304"},{"blob":"0x","proof":"0x05"}]`,
			want: BlobAndProofListV1{
				{
					Blob:  hexutil.Bytes{0x01, 0x02},
					Proof: hexutil.Bytes{0x03, 0x04},
				},
				{
					Blob:  hexutil.Bytes{},
					Proof: hexutil.Bytes{0x05},
				},
			},
		},
		{
			name: "nil item",
			json: `[null,{"blob":"0xaa","proof":"0xbb"}]`,
			want: BlobAndProofListV1{
				nil,
				{
					Blob:  hexutil.Bytes{0xaa},
					Proof: hexutil.Bytes{0xbb},
				},
			},
		},
		{
			name: "null list",
			json: `null`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got BlobAndProofListV1
			if err := got.UnmarshalJSON([]byte(tt.json)); err != nil {
				t.Fatalf("UnmarshalJSON error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("decoded mismatch\ngot:  %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestBlobAndProofListV2MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		list BlobAndProofListV2
	}{
		{
			name: "multiple items",
			list: BlobAndProofListV2{
				{
					Blob:       hexutil.Bytes{0x01, 0x02},
					CellProofs: []hexutil.Bytes{{0x03, 0x04}, {0x05}},
				},
				{
					Blob:       hexutil.Bytes{},
					CellProofs: []hexutil.Bytes{},
				},
			},
		},
		{
			name: "nil item",
			list: BlobAndProofListV2{
				nil,
				{
					Blob:       hexutil.Bytes{0xaa},
					CellProofs: []hexutil.Bytes{{0xbb}},
				},
			},
		},
		{
			name: "nil proofs slice",
			list: BlobAndProofListV2{
				{
					Blob:       hexutil.Bytes{0xcc},
					CellProofs: nil,
				},
			},
		},
		{
			name: "empty list",
			list: BlobAndProofListV2{},
		},
		{
			name: "nil list",
			list: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.list.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON error: %v", err)
			}
			want, err := json.Marshal(toCanonicalBlobAndProofListV2(tt.list))
			if err != nil {
				t.Fatalf("canonical marshal error: %v", err)
			}
			if !bytes.Equal(compactJSON(got), compactJSON(want)) {
				t.Errorf("JSON mismatch\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

func TestBlobAndProofListV2UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		json string
		want BlobAndProofListV2
	}{
		{
			name: "multiple items",
			json: `[{"blob":"0x0102","proofs":["0x0304","0x05"]},{"blob":"0x","proofs":[]}]`,
			want: BlobAndProofListV2{
				{
					Blob:       hexutil.Bytes{0x01, 0x02},
					CellProofs: []hexutil.Bytes{{0x03, 0x04}, {0x05}},
				},
				{
					Blob:       hexutil.Bytes{},
					CellProofs: []hexutil.Bytes{},
				},
			},
		},
		{
			name: "nil item and nil proofs",
			json: `[null,{"blob":"0xcc","proofs":null}]`,
			want: BlobAndProofListV2{
				nil,
				{
					Blob:       hexutil.Bytes{0xcc},
					CellProofs: nil,
				},
			},
		},
		{
			name: "null list",
			json: `null`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got BlobAndProofListV2
			if err := got.UnmarshalJSON([]byte(tt.json)); err != nil {
				t.Fatalf("UnmarshalJSON error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("decoded mismatch\ngot:  %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}

func TestBlobAndProofFieldCoverage(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		expected []string
	}{
		{
			name: "BlobAndProofV1",
			typ:  reflect.TypeOf(BlobAndProofV1{}),
			expected: []string{
				"Blob",
				"Proof",
			},
		},
		{
			name: "BlobAndProofV2",
			typ:  reflect.TypeOf(BlobAndProofV2{}),
			expected: []string{
				"Blob",
				"CellProofs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.typ.NumField() != len(tt.expected) {
				t.Fatalf("%s has %d fields, expected %d; update marshal_bap_list.go",
					tt.name, tt.typ.NumField(), len(tt.expected))
			}
			for i, name := range tt.expected {
				if tt.typ.Field(i).Name != name {
					t.Errorf("field %d: got %q, want %q; update marshal_bap_list.go",
						i, tt.typ.Field(i).Name, name)
				}
			}
		})
	}
}
