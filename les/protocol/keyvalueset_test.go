// Copyright 2020 The go-ethereum Authors
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

package protocol

import (
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
)

func TestKeyValueSet(t *testing.T) {
	var cases = []struct {
		key   string
		value interface{}
	}{
		// {"key1", uint64(10)},
		{"key2", false},
		{"key3", nil},
	}
	var list KeyValueList
	for _, c := range cases {
		list = list.Add(c.key, c.value)
	}
	blob, err := rlp.EncodeToBytes(list)
	if err != nil {
		t.Fatalf("Failed to encode keyvalue list: %v", err)
	}
	var dec KeyValueList
	err = rlp.DecodeBytes(blob, &dec)
	if err != nil {
		t.Fatalf("Failed to decode keyvalue list: %v", err)
	}
}
