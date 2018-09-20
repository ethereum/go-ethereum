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
			bv.Set(i, true)
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

			bv.Set(i, false)

			if bv.Get(i) {
				t.Errorf("element on index %v is not set to false", i)
			}
		}
	}
}

func TestBitvectorNewFromBytesGet(t *testing.T) {
	bv, err := NewFromBytes([]byte{8}, 8)
	if err != nil {
		t.Error(err)
	}
	if !bv.Get(3) {
		t.Fatalf("element 3 is not set to true: state %08b", bv.b[0])
	}
}
