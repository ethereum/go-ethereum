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
	"reflect"
	"testing"
)

func TestParseURI(t *testing.T) {
	type test struct {
		uri             string
		expectURI       *URI
		expectErr       bool
		expectRaw       bool
		expectImmutable bool
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
			uri:             "bzzi:",
			expectURI:       &URI{Scheme: "bzzi"},
			expectImmutable: true,
		},
		{
			uri:       "bzzr:",
			expectURI: &URI{Scheme: "bzzr"},
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
			uri:       "bzzr:/",
			expectURI: &URI{Scheme: "bzzr"},
			expectRaw: true,
		},
		{
			uri:       "bzzr:/abc123",
			expectURI: &URI{Scheme: "bzzr", Addr: "abc123"},
			expectRaw: true,
		},
		{
			uri:       "bzzr:/abc123/path/to/entry",
			expectURI: &URI{Scheme: "bzzr", Addr: "abc123", Path: "path/to/entry"},
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
	}
}
