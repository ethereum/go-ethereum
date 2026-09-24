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
	"encoding/json"
	"errors"
	"strings"
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

// TestSetCodeAuthorizationJSONYParity checks that the yParity field of an
// authorization is restricted to 0 or 1 when decoded from JSON, matching the
// transaction-level yParity handling, instead of being truncated into uint8.
func TestSetCodeAuthorizationJSONYParity(t *testing.T) {
	for _, tt := range []struct {
		yParity string
		want    uint8
		wantErr error
	}{
		{"0x0", 0, nil},
		{"0x1", 1, nil},
		{"0x2", 0, errInvalidYParity},
		{"0x100", 0, errInvalidYParity},
	} {
		input := `{"chainId":"0x1","address":"0x0000000000000000000000000000000000000001","nonce":"0x0","yParity":"` + tt.yParity + `","r":"0x1","s":"0x1"}`
		var auth SetCodeAuthorization
		err := json.Unmarshal([]byte(input), &auth)
		if !errors.Is(err, tt.wantErr) {
			t.Errorf("yParity %s: error %v, want %v", tt.yParity, err, tt.wantErr)
			continue
		}
		if err == nil && auth.V != tt.want {
			t.Errorf("yParity %s: V = %d, want %d", tt.yParity, auth.V, tt.want)
		}
		if err == nil {
			out, _ := json.Marshal(auth)
			if !strings.Contains(string(out), `"yParity":"`+tt.yParity+`"`) {
				t.Errorf("yParity %s: round trip produced %s", tt.yParity, out)
			}
		}
	}
}
