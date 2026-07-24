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

package abi

import (
	"bytes"
	"strings"
	"testing"
)

// TestPackNoArgMethodDoesNotAliasSelector is a regression test for a bug where
// ABI.Pack of a zero-argument method returned a slice that aliased the cached
// method.ID. method.ID is keccak256(sig)[:4]: length 4 but capacity 32, so
// `append(method.ID, arguments...)` with no arguments returned the selector's
// own backing array. Mutating the returned calldata then corrupted the cached
// selector for every subsequent encode of that method.
func TestPackNoArgMethodDoesNotAliasSelector(t *testing.T) {
	const def = `[{ "type" : "function", "name" : "balance", "stateMutability" : "view" }]`
	abi, err := JSON(strings.NewReader(def))
	if err != nil {
		t.Fatal(err)
	}
	// selector aliases the cached method.ID; want is an independent copy of the
	// expected value, so it stays correct even if the bug corrupts the cache.
	selector := abi.Methods["balance"].ID
	want := bytes.Clone(selector)

	data1, err := abi.Pack("balance")
	if err != nil {
		t.Fatal(err)
	}
	// The returned calldata must be exactly the 4-byte selector...
	if !bytes.Equal(data1, want) {
		t.Fatalf("unexpected calldata: got %x, want %x", data1, want)
	}
	// ...but it must not share storage with the cached selector.
	if len(data1) > 0 && len(selector) > 0 && &data1[0] == &selector[0] {
		t.Fatal("Pack returned a slice aliasing the cached method.ID backing array")
	}

	// Mutating the first result must not affect the selector or later packs.
	data1[0] ^= 0xff

	data2, err := abi.Pack("balance")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data2, want) {
		t.Fatalf("second pack corrupted: got %x, want %x", data2, want)
	}
	if !bytes.Equal(abi.Methods["balance"].ID, want) {
		t.Fatalf("cached method.ID corrupted: got %x, want %x", abi.Methods["balance"].ID, want)
	}
}
