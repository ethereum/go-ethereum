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

package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestHeaderHash pins Header.Hash to known roots. The zero case is the
// hash_tree_root of an empty BeaconBlockHeader.
func TestHeaderHash(t *testing.T) {
	tests := []struct {
		name   string
		header Header
		want   common.Hash
	}{
		{
			name: "zero",
			want: common.HexToHash("0xc78009fdf07fc56a11f122370658a353aaa542ed63e44c4bc15ff4cd105ab33c"),
		},
		{
			name: "small",
			header: Header{
				Slot:          1,
				ProposerIndex: 2,
				ParentRoot:    common.HexToHash("0x01"),
				StateRoot:     common.HexToHash("0x02"),
				BodyRoot:      common.HexToHash("0x03"),
			},
			want: common.HexToHash("0x49a040d48fe4f4e9cd6212143dca4ec87bec2d42f29e4984de69eab6ffefc778"),
		},
		{
			name: "max",
			header: Header{
				Slot:          ^uint64(0),
				ProposerIndex: ^uint64(0),
				ParentRoot:    common.MaxHash,
				StateRoot:     common.MaxHash,
				BodyRoot:      common.MaxHash,
			},
			want: common.HexToHash("0x5ebe9f2b0267944bd80dd5cde20317a91d07225ff12e9cd5ba1e834c05cc2b05"),
		},
		{
			name: "mainnet-ish",
			header: Header{
				Slot:          9_000_000,
				ProposerIndex: 123456,
				ParentRoot:    common.HexToHash("0xa1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90"),
				StateRoot:     common.HexToHash("0x0f1e2d3c4b5a69788796a5b4c3d2e1f00f1e2d3c4b5a69788796a5b4c3d2e1f0"),
				BodyRoot:      common.HexToHash("0xdeadbeefcafebabe0123456789abcdef0123456789abcdeffedcba9876543210"),
			},
			want: common.HexToHash("0xb75d049d2d20ba544f2bde428e80c35144af7c204421c17d08a03c878e374037"),
		},
	}
	for _, test := range tests {
		if got := test.header.Hash(); got != test.want {
			t.Errorf("%s: hash mismatch: got %x, want %x", test.name, got, test.want)
		}
	}
}
