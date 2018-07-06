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

package multihash

import (
	"bytes"
	"math/rand"
	"testing"
)

// parse multihash, and check that invalid multihashes fail
func TestCheckMultihash(t *testing.T) {
	hashbytes := make([]byte, 32)
	c, err := rand.Read(hashbytes)
	if err != nil {
		t.Fatal(err)
	} else if c < 32 {
		t.Fatal("short read")
	}

	expected := ToMultihash(hashbytes)

	l, hl, _ := GetMultihashLength(expected)
	if l != 32 {
		t.Fatalf("expected length %d, got %d", 32, l)
	} else if hl != 2 {
		t.Fatalf("expected header length %d, got %d", 2, hl)
	}
	if _, _, err := GetMultihashLength(expected[1:]); err == nil {
		t.Fatal("expected failure on corrupt header")
	}
	if _, _, err := GetMultihashLength(expected[:len(expected)-2]); err == nil {
		t.Fatal("expected failure on short content")
	}
	dh, _ := FromMultihash(expected)
	if !bytes.Equal(dh, hashbytes) {
		t.Fatalf("expected content hash %x, got %x", hashbytes, dh)
	}
}
