// Copyright 2021 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestEncodeStorageKey(t *testing.T) {
	randomOwner := common.HexToHash("0x65710c2c33ddfda00132ce3ab21de97bfa01ea7a1403cfa8a8e3a9dccbb66422")
	var cases = []struct {
		owner  common.Hash
		path   []byte
		expect []byte
	}{
		// no owner, empty keys, with and without terminator.
		{common.HexToHash(""), []byte{}, []byte{0x00}},
		{common.HexToHash(""), []byte{16}, []byte{0x02}},

		// no owner, odd length, no terminator
		{common.HexToHash(""), []byte{1, 2, 3, 4, 5}, []byte{0x12, 0x34, 0x51}},
		// no owner, even length, no terminator
		{common.HexToHash(""), []byte{0, 1, 2, 3, 4, 5}, []byte{0x01, 0x23, 0x45, 0x00}},
		// no owner, odd length, terminator
		{common.HexToHash(""), []byte{15, 1, 12, 11, 8, 16 /*term*/}, []byte{0xf1, 0xcb, 0x83}},
		// no owner, even length, terminator
		{common.HexToHash(""), []byte{0, 15, 1, 12, 11, 8, 16 /*term*/}, []byte{0x0f, 0x1c, 0xb8, 0x02}},

		// with owner, empty keys, with and without terminator.
		{randomOwner, []byte{}, append(randomOwner.Bytes(), []byte{0x00}...)},
		{randomOwner, []byte{16}, append(randomOwner.Bytes(), []byte{0x02}...)},
		// with owner, odd length, no terminator
		{randomOwner, []byte{1, 2, 3, 4, 5}, append(randomOwner.Bytes(), []byte{0x12, 0x34, 0x51}...)},
		// with owner, even length, no terminator
		{randomOwner, []byte{0, 1, 2, 3, 4, 5}, append(randomOwner.Bytes(), []byte{0x01, 0x23, 0x45, 0x00}...)},
		// with owner, odd length, terminator
		{randomOwner, []byte{15, 1, 12, 11, 8, 16 /*term*/}, append(randomOwner.Bytes(), []byte{0xf1, 0xcb, 0x83}...)},
		// with owner, even length, terminator
		{randomOwner, []byte{0, 15, 1, 12, 11, 8, 16 /*term*/}, append(randomOwner.Bytes(), []byte{0x0f, 0x1c, 0xb8, 0x02}...)},
	}
	for _, c := range cases {
		got := EncodeStorageKey(c.owner, c.path)
		if !bytes.Equal(got, c.expect) {
			t.Fatal("Encoding result mismatch", "want", c.expect, "got", got)
		}
	}
}

func TestDecodeStorageKey(t *testing.T) {
	var (
		randomOwner = common.HexToHash("0x65710c2c33ddfda00132ce3ab21de97bfa01ea7a1403cfa8a8e3a9dccbb66422")
	)
	var cases = []struct {
		owner common.Hash
		path  []byte
		input []byte
	}{
		// no owner, empty keys, with and without terminator.
		{common.HexToHash(""), []byte{}, []byte{0x00}},
		{common.HexToHash(""), []byte{16}, []byte{0x02}},

		// no owner, odd length, no terminator
		{common.HexToHash(""), []byte{1, 2, 3, 4, 5}, []byte{0x12, 0x34, 0x51}},
		// no owner, even length, no terminator
		{common.HexToHash(""), []byte{0, 1, 2, 3, 4, 5}, []byte{0x01, 0x23, 0x45, 0x00}},
		// no owner, odd length, terminator
		{common.HexToHash(""), []byte{15, 1, 12, 11, 8, 16 /*term*/}, []byte{0xf1, 0xcb, 0x83}},
		// no owner, even length, terminator
		{common.HexToHash(""), []byte{0, 15, 1, 12, 11, 8, 16 /*term*/}, []byte{0x0f, 0x1c, 0xb8, 0x02}},

		// with owner, empty keys, with and without terminator.
		{randomOwner, []byte{}, append(randomOwner.Bytes(), []byte{0x00}...)},
		{randomOwner, []byte{16}, append(randomOwner.Bytes(), []byte{0x02}...)},
		// with owner, odd length, no terminator
		{randomOwner, []byte{1, 2, 3, 4, 5}, append(randomOwner.Bytes(), []byte{0x12, 0x34, 0x51}...)},
		// with owner, even length, no terminator
		{randomOwner, []byte{0, 1, 2, 3, 4, 5}, append(randomOwner.Bytes(), []byte{0x01, 0x23, 0x45, 0x00}...)},
		// with owner, odd length, terminator
		{randomOwner, []byte{15, 1, 12, 11, 8, 16 /*term*/}, append(randomOwner.Bytes(), []byte{0xf1, 0xcb, 0x83}...)},
		// with owner, even length, terminator
		{randomOwner, []byte{0, 15, 1, 12, 11, 8, 16 /*term*/}, append(randomOwner.Bytes(), []byte{0x0f, 0x1c, 0xb8, 0x02}...)},
	}
	for _, c := range cases {
		owner, path := DecodeStorageKey(c.input)
		if !bytes.Equal(owner.Bytes(), c.owner.Bytes()) {
			t.Fatal("Decode owner mismatch", "want", c.owner, "got", owner)
		}
		if !bytes.Equal(path, c.path) {
			t.Fatal("Decode path mismatch", "want", c.path, "got", path)
		}
	}
}
