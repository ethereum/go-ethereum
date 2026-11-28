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

package snap

import (
	"encoding/hex"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/rlp"
)

// Tests that the snap ENR entry lists the served versions, and stays an empty
// list when only snap/1 is served.
func TestENREntryEncoding(t *testing.T) {
	tests := []struct {
		snapV2 bool
		want   string
	}{
		{snapV2: false, want: "84736e6170c0"},
		{snapV2: true, want: "84736e6170c20102"},
	}
	for _, tt := range tests {
		protocols := MakeProtocols(nil, tt.snapV2)
		entry := protocols[0].Attributes[0]
		for _, proto := range protocols[1:] {
			if proto.Attributes[0] != entry {
				t.Fatalf("snapV2=%v: protocol versions carry different ENR entries", tt.snapV2)
			}
		}
		var rec enr.Record
		rec.Set(entry)
		key, err := rlp.EncodeToBytes(entry.ENRKey())
		if err != nil {
			t.Fatal(err)
		}
		var value rlp.RawValue
		if err := rec.Load(enr.WithEntry(entry.ENRKey(), &value)); err != nil {
			t.Fatalf("snapV2=%v: %v", tt.snapV2, err)
		}
		if got := hex.EncodeToString(append(key, value...)); got != tt.want {
			t.Errorf("snapV2=%v: got %s, want %s", tt.snapV2, got, tt.want)
		}
	}
}

// Tests that version lists from other nodes decode, including versions this
// node does not know.
func TestENREntryDecoding(t *testing.T) {
	tests := []struct {
		input string
		want  []uint
	}{
		{input: "c0", want: nil},
		{input: "c102", want: []uint{2}},
		{input: "c3010203", want: []uint{1, 2, 3}},
	}
	for _, tt := range tests {
		input, _ := hex.DecodeString(tt.input)
		var entry enrEntry
		if err := rlp.DecodeBytes(input, &entry); err != nil {
			t.Fatalf("%s: %v", tt.input, err)
		}
		if !slices.Equal(entry.Versions, tt.want) {
			t.Errorf("%s: got versions %v, want %v", tt.input, entry.Versions, tt.want)
		}
	}
}
