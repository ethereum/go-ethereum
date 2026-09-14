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

package ssz

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/holiman/uint256"
)

func TestAppendBasic(t *testing.T) {
	u128 := uint256.NewInt(0)
	u128[0], u128[1] = 0x0807060504030201, 0x100f0e0d0c0b0a09
	u128[2], u128[3] = 0xffffffffffffffff, 0xffffffffffffffff // ignored upper limbs
	u256 := uint256.NewInt(0).SetBytes([]byte{0xaa, 0xbb})
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"bool true", AppendBool(nil, true), []byte{1}},
		{"bool false", AppendBool(nil, false), []byte{0}},
		{"uint8", AppendUint8(nil, 0xab), []byte{0xab}},
		{"uint16", AppendUint16(nil, 0x1234), []byte{0x34, 0x12}},
		{"uint32", AppendUint32(nil, 0x12345678), []byte{0x78, 0x56, 0x34, 0x12}},
		{"uint64", AppendUint64(nil, 0x0102030405060708), []byte{8, 7, 6, 5, 4, 3, 2, 1}},
		{"uint128", AppendUint128(nil, u128), []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
		{"uint256", AppendUint256(nil, u256), append([]byte{0xbb, 0xaa}, make([]byte, 30)...)},
		{"offset", AppendOffset(nil, 0x01020304), []byte{4, 3, 2, 1}},
	}
	for _, tc := range tests {
		if !bytes.Equal(tc.got, tc.want) {
			t.Errorf("%s = %x, want %x", tc.name, tc.got, tc.want)
		}
	}
	// Appending extends rather than replaces.
	if got := AppendUint16([]byte{0xee}, 1); !bytes.Equal(got, []byte{0xee, 1, 0}) {
		t.Errorf("append onto prefix = %x", got)
	}
}

// pair is a two-field fixed-size container used to exercise the package
// entry points.
type pair struct {
	a uint64
	b [4]byte
}

func (p *pair) SizeSSZ() int { return 12 }
func (p *pair) MarshalSSZTo(dst []byte) ([]byte, error) {
	dst = AppendUint64(dst, p.a)
	return append(dst, p.b[:]...), nil
}
func (p *pair) UnmarshalSSZ(data []byte) error {
	d := NewDecoder(data)
	p.a = d.Uint64()
	d.Bytes(p.b[:])
	return d.Close()
}
func (p *pair) HashTreeRoot() [32]byte {
	buf, _ := p.MarshalSSZTo(nil)
	return Merkleize([][32]byte{Merkleize(Pack(buf[:8]), 1), Merkleize(Pack(buf[8:]), 1)}, 2)
}

func TestEncodeDecode(t *testing.T) {
	in := &pair{a: 42, b: [4]byte{1, 2, 3, 4}}
	blob, err := Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	if want := append(AppendUint64(nil, 42), 1, 2, 3, 4); !bytes.Equal(blob, want) {
		t.Fatalf("Encode = %x, want %x", blob, want)
	}
	var out pair
	if err := Decode(blob, &out); err != nil {
		t.Fatal(err)
	}
	if out != *in {
		t.Errorf("Decode = %+v, want %+v", out, *in)
	}
	if err := Decode(blob[:11], &out); !errors.Is(err, ErrSize) {
		t.Errorf("short input: err = %v, want ErrSize", err)
	}
	if err := Decode(append(blob, 0), &out); !errors.Is(err, ErrTrailing) {
		t.Errorf("long input: err = %v, want ErrTrailing", err)
	}
	if got := HashTreeRoot(in); got != in.HashTreeRoot() {
		t.Errorf("HashTreeRoot entry point disagrees with the method")
	}
}

// huge claims the largest serialized size an int can hold.
type huge struct{ pair }

func (h *huge) SizeSSZ() int { return math.MaxInt }

func TestEncodeTooBig(t *testing.T) {
	if uint64(math.MaxInt) < MaxSize {
		t.Skip("int cannot reach the SSZ envelope on this platform")
	}
	if _, err := Encode(new(huge)); !errors.Is(err, ErrTooBig) {
		t.Errorf("err = %v, want ErrTooBig", err)
	}
}
