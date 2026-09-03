// Copyright 2024 The go-ethereum Authors
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

// TestParseDelegation tests a few possible delegation designator values and
// ensures they are parsed correctly.
func TestParseDelegation(t *testing.T) {
	addr := common.Address{0x42}
	for _, tt := range []struct {
		val  []byte
		want *common.Address
	}{
		{ // simple correct delegation
			val:  append(DelegationPrefix, addr.Bytes()...),
			want: &addr,
		},
		{ // wrong address size
			val: append(DelegationPrefix, addr.Bytes()[0:19]...),
		},
		{ // short address
			val: append(DelegationPrefix, 0x42),
		},
		{ // long address
			val: append(append(DelegationPrefix, addr.Bytes()...), 0x42),
		},
		{ // wrong prefix size
			val: append(DelegationPrefix[:2], addr.Bytes()...),
		},
		{ // wrong prefix
			val: append([]byte{0xef, 0x01, 0x01}, addr.Bytes()...),
		},
		{ // wrong prefix
			val: append([]byte{0xef, 0x00, 0x00}, addr.Bytes()...),
		},
		{ // no prefix
			val: addr.Bytes(),
		},
		{ // no address
			val: DelegationPrefix,
		},
	} {
		got, ok := ParseDelegation(tt.val)
		if ok && tt.want == nil {
			t.Fatalf("expected fail, got %s", got.Hex())
		}
		if !ok && tt.want != nil {
			t.Fatalf("failed to parse, want %s", tt.want.Hex())
		}
	}
}
