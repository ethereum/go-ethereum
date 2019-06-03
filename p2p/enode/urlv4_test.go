// Copyright 2018 The go-ethereum Authors
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

package enode

import (
	"crypto/ecdsa"
	"net"
	"reflect"
	"strings"
	"testing"
)

var parseNodeTests = []struct {
	rawurl     string
	wantError  string
	wantResult *Node
}{
	{
		rawurl:    "http://foobar",
		wantError: `invalid URL scheme, want "enode"`,
	},
	{
		rawurl:    "enode://01010101@123.124.125.126:3",
		wantError: `invalid node ID (wrong length, want 128 hex chars)`,
	},
	// Complete nodes with IP address.
	{
		rawurl:    "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@hostname:3",
		wantError: `invalid IP address`,
	},
	{
		rawurl:    "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@127.0.0.1:foo",
		wantError: `invalid port`,
	},
	{
		rawurl:    "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@127.0.0.1:3?discport=foo",
		wantError: `invalid discport in query`,
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@127.0.0.1:52150",
		wantResult: NewV4(
			hexPubkey("1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			net.IP{0x7f, 0x0, 0x0, 0x1},
			52150,
			52150,
		),
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@[::]:52150",
		wantResult: NewV4(
			hexPubkey("1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			net.ParseIP("::"),
			52150,
			52150,
		),
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@[2001:db8:3c4d:15::abcd:ef12]:52150",
		wantResult: NewV4(
			hexPubkey("1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			net.ParseIP("2001:db8:3c4d:15::abcd:ef12"),
			52150,
			52150,
		),
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@127.0.0.1:52150?discport=22334",
		wantResult: NewV4(
			hexPubkey("1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			net.IP{0x7f, 0x0, 0x0, 0x1},
			52150,
			22334,
		),
	},
	// Incomplete nodes with no address.
	{
		rawurl: "1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439",
		wantResult: NewV4(
			hexPubkey("1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			nil, 0, 0,
		),
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439",
		wantResult: NewV4(
			hexPubkey("1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			nil, 0, 0,
		),
	},
	// Invalid URLs
	{
		rawurl:    "01010101",
		wantError: `invalid node ID (wrong length, want 128 hex chars)`,
	},
	{
		rawurl:    "enode://01010101",
		wantError: `invalid node ID (wrong length, want 128 hex chars)`,
	},
	{
		// This test checks that errors from url.Parse are handled.
		rawurl:    "://foo",
		wantError: `parse ://foo: missing protocol scheme`,
	},
}

func hexPubkey(h string) *ecdsa.PublicKey {
	k, err := parsePubkey(h)
	if err != nil {
		panic(err)
	}
	return k
}

func TestParseNode(t *testing.T) {
	for _, test := range parseNodeTests {
		n, err := ParseV4(test.rawurl)
		if test.wantError != "" {
			if err == nil {
				t.Errorf("test %q:\n  got nil error, expected %#q", test.rawurl, test.wantError)
				continue
			} else if err.Error() != test.wantError {
				t.Errorf("test %q:\n  got error %#q, expected %#q", test.rawurl, err.Error(), test.wantError)
				continue
			}
		} else {
			if err != nil {
				t.Errorf("test %q:\n  unexpected error: %v", test.rawurl, err)
				continue
			}
			if !reflect.DeepEqual(n, test.wantResult) {
				t.Errorf("test %q:\n  result mismatch:\ngot:  %#v\nwant: %#v", test.rawurl, n, test.wantResult)
			}
		}
	}
}

func TestNodeString(t *testing.T) {
	for i, test := range parseNodeTests {
		if test.wantError == "" && strings.HasPrefix(test.rawurl, "enode://") {
			str := test.wantResult.String()
			if str != test.rawurl {
				t.Errorf("test %d: Node.String() mismatch:\ngot:  %s\nwant: %s", i, str, test.rawurl)
			}
		}
	}
}
