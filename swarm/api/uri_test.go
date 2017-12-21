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
		uri                       string
		expectURI                 *URI
		expectErr                 bool
		expectRaw                 bool
		expectImmutable           bool
		expectList                bool
		expectDeprecatedRaw       bool
		expectDeprecatedImmutable bool
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
			uri:                 "bzzr:",
			expectURI:           &URI{Scheme: "bzzr"},
			expectDeprecatedRaw: true,
		},
		{
			uri:                 "bzzr:/",
			expectURI:           &URI{Scheme: "bzzr"},
			expectDeprecatedRaw: true,
		},
		{
			uri:                       "bzzi:",
			expectURI:                 &URI{Scheme: "bzzi"},
			expectDeprecatedImmutable: true,
		},
		{
			uri:                       "bzzi:/",
			expectURI:                 &URI{Scheme: "bzzi"},
			expectDeprecatedImmutable: true,
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
		if actual.DeprecatedRaw() != x.expectDeprecatedRaw {
			t.Fatalf("expected %s deprecated raw to be %t, got %t", x.uri, x.expectDeprecatedRaw, actual.DeprecatedRaw())
		}
		if actual.DeprecatedImmutable() != x.expectDeprecatedImmutable {
			t.Fatalf("expected %s deprecated immutable to be %t, got %t", x.uri, x.expectDeprecatedImmutable, actual.DeprecatedImmutable())
		}
	}
}
