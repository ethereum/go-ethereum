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

package bitvector

import "testing"

// TestBitvectorNew checks that enforcements of argument length works in the constructors
func TestBitvectorNew(t *testing.T) {
	_, err := New(0)
	if err != errInvalidLength {
		t.Errorf("expected err %v, got %v", errInvalidLength, err)
	}

	_, err = NewFromBytes(nil, 0)
	if err != errInvalidLength {
		t.Errorf("expected err %v, got %v", errInvalidLength, err)
	}

	_, err = NewFromBytes([]byte{0}, 9)
	if err != errInvalidLength {
		t.Errorf("expected err %v, got %v", errInvalidLength, err)
	}

	_, err = NewFromBytes(make([]byte, 8), 8)
	if err != nil {
		t.Error(err)
	}
}

// TestBitvectorGetSet tests correctness of individual Set and Get commands
func TestBitvectorGetSet(t *testing.T) {
	for _, length := range []int{
		1,
		2,
		4,
		8,
		9,
		15,
		16,
	} {
		bv, err := New(length)
		if err != nil {
			t.Errorf("error for length %v: %v", length, err)
		}

		for i := 0; i < length; i++ {
			if bv.Get(i) {
				t.Errorf("expected false for element on index %v", i)
			}
		}

		func() {
			defer func() {
				if err := recover(); err == nil {
					t.Errorf("expecting panic")
				}
			}()
			bv.Get(length + 8)
		}()

		for i := 0; i < length; i++ {
			bv.Set(i)
			for j := 0; j < length; j++ {
				if j == i {
					if !bv.Get(j) {
						t.Errorf("element on index %v is not set to true", i)
					}
				} else {
					if bv.Get(j) {
						t.Errorf("element on index %v is not false", i)
					}
				}
			}

			bv.Unset(i)

			if bv.Get(i) {
				t.Errorf("element on index %v is not set to false", i)
			}
		}
	}
}

// TestBitvectorNewFromBytesGet tests that bit vector is initialized correctly from underlying byte slice
func TestBitvectorNewFromBytesGet(t *testing.T) {
	bv, err := NewFromBytes([]byte{8}, 8)
	if err != nil {
		t.Error(err)
	}
	if !bv.Get(3) {
		t.Fatalf("element 3 is not set to true: state %08b", bv.b[0])
	}
}

// TestBitVectorString tests that string representation of bit vector is correct
func TestBitVectorString(t *testing.T) {
	b := []byte{0xa5, 0x81}
	expect := "1010010110000001"
	bv, err := NewFromBytes(b, 2)
	if err != nil {
		t.Fatal(err)
	}
	if bv.String() != expect {
		t.Fatalf("bitvector string fail: got %s, expect %s", bv.String(), expect)
	}
}

// TestBitVectorSetUnsetBytes tests that setting and unsetting by byte slice modifies the bit vector correctly
func TestBitVectorSetBytes(t *testing.T) {
	b := []byte{0xff, 0xff}
	cb := []byte{0xa5, 0x81}
	expectUnset := "0101101001111110"
	expectReset := "1111111111111111"
	bv, err := NewFromBytes(b, 2)
	if err != nil {
		t.Fatal(err)
	}
	bv.UnsetBytes(cb)
	if bv.String() != expectUnset {
		t.Fatalf("bitvector unset bytes fail: got %s, expect %s", bv.String(), expectUnset)
	}
	bv.SetBytes(cb)
	if bv.String() != expectReset {
		t.Fatalf("bitvector reset bytes fail: got %s, expect %s", bv.String(), expectReset)
	}
}
