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
	"encoding/json"
	"testing"
)

func TestBlobAndProofListMarshalJSONNil(t *testing.T) {
	tests := []struct {
		name string
		list any
	}{
		{
			name: "should encode null if BlobAndProofListV1 is nil",
			list: BlobAndProofListV1(nil),
		},
		{
			name: "should encode null if BlobAndProofListV2 is nil",
			list: BlobAndProofListV2(nil),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enc, err := json.Marshal(test.list)
			if err != nil {
				t.Fatal(err)
			}
			if string(enc) != "null" {
				t.Fatalf("got %s, want null", enc)
			}
		})
	}
}
