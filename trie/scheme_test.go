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

func TestEncodeNodeKey(t *testing.T) {
	var (
		randomHash  = common.HexToHash("0xa5d8b963aee47d5cafe7135a12f03e076d656ff5b77789afa867eb95119ab175")
		randomOwner = common.HexToHash("0x65710c2c33ddfda00132ce3ab21de97bfa01ea7a1403cfa8a8e3a9dccbb66422")
	)
	var cases = []struct {
		owner  common.Hash
		path   []byte
		hash   common.Hash
		expect []byte
	}{
		// metaroot
		{common.Hash{}, nil, common.Hash{}, nil},
		// no owner, empty keys, with and without terminator.
		{common.HexToHash(""), []byte{}, randomHash, append([]byte{0x00}, randomHash.Bytes()...)},
		{common.HexToHash(""), []byte{16}, randomHash, append([]byte{0x20}, randomHash.Bytes()...)},

		// no owner, odd length, no terminator
		{common.HexToHash(""), []byte{1, 2, 3, 4, 5}, randomHash, append([]byte{0x11, 0x23, 0x45}, randomHash.Bytes()...)},
		// no owner, even length, no terminator
		{common.HexToHash(""), []byte{0, 1, 2, 3, 4, 5}, randomHash, append([]byte{0x00, 0x01, 0x23, 0x45}, randomHash.Bytes()...)},
		// no owner, odd length, terminator
		{common.HexToHash(""), []byte{15, 1, 12, 11, 8, 16 /*term*/}, randomHash, append([]byte{0x3f, 0x1c, 0xb8}, randomHash.Bytes()...)},
		// no owner, even length, terminator
		{common.HexToHash(""), []byte{0, 15, 1, 12, 11, 8, 16 /*term*/}, randomHash, append([]byte{0x20, 0x0f, 0x1c, 0xb8}, randomHash.Bytes()...)},

		// with owner, empty keys, with and without terminator.
		{randomOwner, []byte{}, randomHash, append(append(randomOwner.Bytes(), []byte{0x00}...), randomHash.Bytes()...)},
		{randomOwner, []byte{16}, randomHash, append(append(randomOwner.Bytes(), []byte{0x20}...), randomHash.Bytes()...)},
		// with owner, odd length, no terminator
		{randomOwner, []byte{1, 2, 3, 4, 5}, randomHash, append(append(randomOwner.Bytes(), []byte{0x11, 0x23, 0x45}...), randomHash.Bytes()...)},
		// with owner, even length, no terminator
		{randomOwner, []byte{0, 1, 2, 3, 4, 5}, randomHash, append(append(randomOwner.Bytes(), []byte{0x00, 0x01, 0x23, 0x45}...), randomHash.Bytes()...)},
		// with owner, odd length, terminator
		{randomOwner, []byte{15, 1, 12, 11, 8, 16 /*term*/}, randomHash, append(append(randomOwner.Bytes(), []byte{0x3f, 0x1c, 0xb8}...), randomHash.Bytes()...)},
		// with owner, even length, terminator
		{randomOwner, []byte{0, 15, 1, 12, 11, 8, 16 /*term*/}, randomHash, append(append(randomOwner.Bytes(), []byte{0x20, 0x0f, 0x1c, 0xb8}...), randomHash.Bytes()...)},
	}
	for _, c := range cases {
		got := EncodeNodeKey(c.owner, c.path, c.hash)
		if !bytes.Equal(got, c.expect) {
			t.Fatal("Encoding result mismatch", "want", c.expect, "got", got)
		}
	}
}

func TestDecodeNodeKey(t *testing.T) {
	var (
		randomHash  = common.HexToHash("0xa5d8b963aee47d5cafe7135a12f03e076d656ff5b77789afa867eb95119ab175")
		randomOwner = common.HexToHash("0x65710c2c33ddfda00132ce3ab21de97bfa01ea7a1403cfa8a8e3a9dccbb66422")
	)
	var cases = []struct {
		owner common.Hash
		path  []byte
		hash  common.Hash
		input []byte
	}{
		// metaroot
		{common.Hash{}, nil, common.Hash{}, nil},
		// no owner, empty keys, with and without terminator.
		{common.HexToHash(""), []byte{}, randomHash, append([]byte{0x00}, randomHash.Bytes()...)},
		{common.HexToHash(""), []byte{16}, randomHash, append([]byte{0x20}, randomHash.Bytes()...)},

		// no owner, odd length, no terminator
		{common.HexToHash(""), []byte{1, 2, 3, 4, 5}, randomHash, append([]byte{0x11, 0x23, 0x45}, randomHash.Bytes()...)},
		// no owner, even length, no terminator
		{common.HexToHash(""), []byte{0, 1, 2, 3, 4, 5}, randomHash, append([]byte{0x00, 0x01, 0x23, 0x45}, randomHash.Bytes()...)},
		// no owner, odd length, terminator
		{common.HexToHash(""), []byte{15, 1, 12, 11, 8, 16 /*term*/}, randomHash, append([]byte{0x3f, 0x1c, 0xb8}, randomHash.Bytes()...)},
		// no owner, even length, terminator
		{common.HexToHash(""), []byte{0, 15, 1, 12, 11, 8, 16 /*term*/}, randomHash, append([]byte{0x20, 0x0f, 0x1c, 0xb8}, randomHash.Bytes()...)},

		// with owner, empty keys, with and without terminator.
		{randomOwner, []byte{}, randomHash, append(append(randomOwner.Bytes(), []byte{0x00}...), randomHash.Bytes()...)},
		{randomOwner, []byte{16}, randomHash, append(append(randomOwner.Bytes(), []byte{0x20}...), randomHash.Bytes()...)},
		// with owner, odd length, no terminator
		{randomOwner, []byte{1, 2, 3, 4, 5}, randomHash, append(append(randomOwner.Bytes(), []byte{0x11, 0x23, 0x45}...), randomHash.Bytes()...)},
		// with owner, even length, no terminator
		{randomOwner, []byte{0, 1, 2, 3, 4, 5}, randomHash, append(append(randomOwner.Bytes(), []byte{0x00, 0x01, 0x23, 0x45}...), randomHash.Bytes()...)},
		// with owner, odd length, terminator
		{randomOwner, []byte{15, 1, 12, 11, 8, 16 /*term*/}, randomHash, append(append(randomOwner.Bytes(), []byte{0x3f, 0x1c, 0xb8}...), randomHash.Bytes()...)},
		// with owner, even length, terminator
		{randomOwner, []byte{0, 15, 1, 12, 11, 8, 16 /*term*/}, randomHash, append(append(randomOwner.Bytes(), []byte{0x20, 0x0f, 0x1c, 0xb8}...), randomHash.Bytes()...)},
	}
	for _, c := range cases {
		owner, path, hash := DecodeNodeKey(c.input)
		if !bytes.Equal(owner.Bytes(), c.owner.Bytes()) {
			t.Fatal("Decode owner mismatch", "want", c.owner, "got", owner)
		}
		if !bytes.Equal(path, c.path) {
			t.Fatal("Decode path mismatch", "want", c.path, "got", path)
		}
		if !bytes.Equal(hash.Bytes(), c.hash.Bytes()) {
			t.Fatal("Decode hash mismatch", "want", c.hash, "got", hash)
		}
	}
}
