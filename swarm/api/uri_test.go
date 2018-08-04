// Copyright 2017 The go-ethereum Authors
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

package api

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/storage"
)

func TestParseURI(t *testing.T) {
	type test struct {
		uri                       string
		expectURI                 *URI
		expectErr                 bool
		expectRaw                 bool
		expectImmutable           bool
		expectList                bool
		expectHash                bool
		expectDeprecatedRaw       bool
		expectDeprecatedImmutable bool
		expectValidKey            bool
		expectAddr                storage.Address
	}
	tests := []test{
		{
			uri:       "",
			expectErr: true,
		},
		{
			uri:       "foo",
			expectErr: true,
		},
		{
			uri:       "bzz",
			expectErr: true,
		},
		{
			uri:       "bzz:",
			expectURI: &URI{Scheme: "bzz"},
		},
		{
			uri:             "bzz-immutable:",
			expectURI:       &URI{Scheme: "bzz-immutable"},
			expectImmutable: true,
		},
		{
			uri:       "bzz-raw:",
			expectURI: &URI{Scheme: "bzz-raw"},
			expectRaw: true,
		},
		{
			uri:       "bzz:/",
			expectURI: &URI{Scheme: "bzz"},
		},
		{
			uri:       "bzz:/abc123",
			expectURI: &URI{Scheme: "bzz", Addr: "abc123"},
		},
		{
			uri:       "bzz:/abc123/path/to/entry",
			expectURI: &URI{Scheme: "bzz", Addr: "abc123", Path: "path/to/entry"},
		},
		{
			uri:       "bzz-raw:/",
			expectURI: &URI{Scheme: "bzz-raw"},
			expectRaw: true,
		},
		{
			uri:       "bzz-raw:/abc123",
			expectURI: &URI{Scheme: "bzz-raw", Addr: "abc123"},
			expectRaw: true,
		},
		{
			uri:       "bzz-raw:/abc123/path/to/entry",
			expectURI: &URI{Scheme: "bzz-raw", Addr: "abc123", Path: "path/to/entry"},
			expectRaw: true,
		},
		{
			uri:       "bzz://",
			expectURI: &URI{Scheme: "bzz"},
		},
		{
			uri:       "bzz://abc123",
			expectURI: &URI{Scheme: "bzz", Addr: "abc123"},
		},
		{
			uri:       "bzz://abc123/path/to/entry",
			expectURI: &URI{Scheme: "bzz", Addr: "abc123", Path: "path/to/entry"},
		},
		{
			uri:        "bzz-hash:",
			expectURI:  &URI{Scheme: "bzz-hash"},
			expectHash: true,
		},
		{
			uri:        "bzz-hash:/",
			expectURI:  &URI{Scheme: "bzz-hash"},
			expectHash: true,
		},
		{
			uri:        "bzz-list:",
			expectURI:  &URI{Scheme: "bzz-list"},
			expectList: true,
		},
		{
			uri:        "bzz-list:/",
			expectURI:  &URI{Scheme: "bzz-list"},
			expectList: true,
		},
		{
			uri: "bzz-raw://4378d19c26590f1a818ed7d6a62c3809e149b0999cab5ce5f26233b3b423bf8c",
			expectURI: &URI{Scheme: "bzz-raw",
				Addr: "4378d19c26590f1a818ed7d6a62c3809e149b0999cab5ce5f26233b3b423bf8c",
			},
			expectValidKey: true,
			expectRaw:      true,
			expectAddr: storage.Address{67, 120, 209, 156, 38, 89, 15, 26,
				129, 142, 215, 214, 166, 44, 56, 9,
				225, 73, 176, 153, 156, 171, 92, 229,
				242, 98, 51, 179, 180, 35, 191, 140,
			},
		},
	}
	for _, x := range tests {
		actual, err := Parse(x.uri)
		if x.expectErr {
			if err == nil {
				t.Fatalf("expected %s to error", x.uri)
			}
			continue
		}
		if err != nil {
			t.Fatalf("error parsing %s: %s", x.uri, err)
		}
		if !reflect.DeepEqual(actual, x.expectURI) {
			t.Fatalf("expected %s to return %#v, got %#v", x.uri, x.expectURI, actual)
		}
		if actual.Raw() != x.expectRaw {
			t.Fatalf("expected %s raw to be %t, got %t", x.uri, x.expectRaw, actual.Raw())
		}
		if actual.Immutable() != x.expectImmutable {
			t.Fatalf("expected %s immutable to be %t, got %t", x.uri, x.expectImmutable, actual.Immutable())
		}
		if actual.List() != x.expectList {
			t.Fatalf("expected %s list to be %t, got %t", x.uri, x.expectList, actual.List())
		}
		if actual.Hash() != x.expectHash {
			t.Fatalf("expected %s hash to be %t, got %t", x.uri, x.expectHash, actual.Hash())
		}
		if x.expectValidKey {
			if actual.Address() == nil {
				t.Fatalf("expected %s to return a valid key, got nil", x.uri)
			} else {
				if !bytes.Equal(x.expectAddr, actual.Address()) {
					t.Fatalf("expected %s to be decoded to %v", x.expectURI.Addr, x.expectAddr)
				}
			}
		}
	}
}
