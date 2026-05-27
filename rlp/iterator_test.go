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

package rlp

import (
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// TestIterator tests some basic things about the ListIterator. A more
// comprehensive test can be found in core/rlp_test.go, where we can
// use both types and rlp without dependency cycles
func TestIterator(t *testing.T) {
	bodyRlpHex := "0xf902cbf8d6f869800182c35094000000000000000000000000000000000000aaaa808a000000000000000000001ba01025c66fad28b4ce3370222624d952c35529e602af7cbe04f667371f61b0e3b3a00ab8813514d1217059748fd903288ace1b4001a4bc5fbde2790debdc8167de2ff869010182c35094000000000000000000000000000000000000aaaa808a000000000000000000001ca05ac4cf1d19be06f3742c21df6c49a7e929ceb3dbaf6a09f3cfb56ff6828bd9a7a06875970133a35e63ac06d360aa166d228cc013e9b96e0a2cae7f55b22e1ee2e8f901f0f901eda0c75448377c0e426b8017b23c5f77379ecf69abc1d5c224284ad3ba1c46c59adaa00000000000000000000000000000000000000000000000000000000000000000940000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000a00000000000000000000000000000000000000000000000000000000000000000b9010000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000808080808080a00000000000000000000000000000000000000000000000000000000000000000880000000000000000"
	bodyRlp := hexutil.MustDecode(bodyRlpHex)

	it, err := NewListIterator(bodyRlp)
	if err != nil {
		t.Fatal(err)
	}
	// Check that txs exist
	if !it.Next() {
		t.Fatal("expected two elems, got zero")
	}
	txs := it.Value()
	if offset := it.Offset(); offset != 3 {
		t.Fatal("wrong offset", offset, "want 3")
	}

	// Check that uncles exist
	if !it.Next() {
		t.Fatal("expected two elems, got one")
	}
	if offset := it.Offset(); offset != 219 {
		t.Fatal("wrong offset", offset, "want 219")
	}

	txit, err := NewListIterator(txs)
	if err != nil {
		t.Fatal(err)
	}
	if c := txit.Count(); c != 2 {
		t.Fatal("wrong Count:", c)
	}
	var i = 0
	for txit.Next() {
		if txit.err != nil {
			t.Fatal(txit.err)
		}
		i++
	}
	if exp := 2; i != exp {
		t.Errorf("count wrong, expected %d got %d", i, exp)
	}
}

func TestIteratorErrors(t *testing.T) {
	tests := []struct {
		input     []byte
		wantCount int // expected Count before iterating
		wantErr   error
	}{
		// Second item string header claims 3 bytes content, but only 2 remain.
		{unhex("C4 01 83AABB"), 2, ErrValueTooLarge},
		// Second item truncated: B9 requires 2 size bytes, none available.
		{unhex("C2 01 B9"), 2, io.ErrUnexpectedEOF},
		// 0x05 should be encoded directly, not as 81 05.
		{unhex("C3 01 8105"), 2, ErrCanonSize},
		// Long-form string header B8 used for 1-byte content (< 56).
		{unhex("C4 01 B801AA"), 2, ErrCanonSize},
		// Long-form list header F8 used for 1-byte content (< 56).
		{unhex("C4 01 F80101"), 2, ErrCanonSize},
	}
	for _, tt := range tests {
		it, err := NewListIterator(tt.input)
		if err != nil {
			t.Fatal("NewListIterator error:", err)
		}
		if c := it.Count(); c != tt.wantCount {
			t.Fatalf("%x: Count = %d, want %d", tt.input, c, tt.wantCount)
		}
		n := 0
		for it.Next() {
			if it.Err() != nil {
				break
			}
			n++
		}
		if wantN := tt.wantCount - 1; n != wantN {
			t.Fatalf("%x: got %d valid items, want %d", tt.input, n, wantN)
		}
		if it.Err() != tt.wantErr {
			t.Fatalf("%x: got error %v, want %v", tt.input, it.Err(), tt.wantErr)
		}
		if it.Next() {
			t.Fatalf("%x: Next returned true after error", tt.input)
		}
	}
}

func FuzzIteratorCount(f *testing.F) {
	examples := [][]byte{unhex("010203"), unhex("018142"), unhex("01830202")}
	for _, e := range examples {
		f.Add(e)
	}
	f.Fuzz(func(t *testing.T, in []byte) {
		it := newIterator(in, 0)
		count := it.Count()
		i := 0
		for it.Next() {
			i++
		}
		if i != count {
			t.Fatalf("%x: count %d not equal to %d iterations", in, count, i)
		}
	})
}
