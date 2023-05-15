// Copyright 2023 The go-ethereum Authors
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

// Package utesting provides a standalone replacement for package testing.
//
// This package exists because package testing cannot easily be embedded into a
// standalone go program. It provides an API that mirrors the standard library
// testing API.

package blocktest

import "testing"

func TestHash(t *testing.T) {
	th := NewHasher()
	hash := th.Hash()
	if have, want := hash.Hex(), "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"; have != want {
		t.Errorf("hash not match, have: %s want: %s", have, want)
	}
	err := th.Update([]byte("foo"), []byte("bar"))
	if err != nil {
		t.Error(err)
	}
	hash = th.Hash()
	if have, want := hash.Hex(), "0x38d18acb67d25c8bb9942764b62f18e17054f66a817bd4295423adf9ed98873e"; have != want {
		t.Errorf("hash not match, have: %s want: %s", have, want)
	}

	th.Reset()
	hash = th.Hash()
	if have, want := hash.Hex(), "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"; have != want {
		t.Errorf("hash not match, have: %s want: %s", have, want)
	}
}
