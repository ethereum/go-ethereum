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
	"testing"

	"github.com/holiman/uint256"
)

func TestDecoderFixedFields(t *testing.T) {
	u128 := uint256.NewInt(0).SetBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	u256 := uint256.NewInt(0).Not(uint256.NewInt(0))
	var buf []byte
	buf = AppendBool(buf, true)
	buf = AppendBool(buf, false)
	buf = AppendUint8(buf, 0xab)
	buf = AppendUint16(buf, 0x1234)
	buf = AppendUint32(buf, 0x12345678)
	buf = AppendUint64(buf, 0x123456789abcdef0)
	buf = AppendUint128(buf, u128)
	buf = AppendUint256(buf, u256)
	buf = append(buf, 7, 8, 9)

	d := NewDecoder(buf)
	if v := d.Bool(); v != true {
		t.Errorf("Bool = %v, want true", v)
	}
	if v := d.Bool(); v != false {
		t.Errorf("Bool = %v, want false", v)
	}
	if v := d.Uint8(); v != 0xab {
		t.Errorf("Uint8 = %#x", v)
	}
	if v := d.Uint16(); v != 0x1234 {
		t.Errorf("Uint16 = %#x", v)
	}
	if v := d.Uint32(); v != 0x12345678 {
		t.Errorf("Uint32 = %#x", v)
	}
	if v := d.Uint64(); v != 0x123456789abcdef0 {
		t.Errorf("Uint64 = %#x", v)
	}
	var got128, got256 uint256.Int
	got128[2], got128[3] = 99, 99 // must be cleared
	d.Uint128(&got128)
	if !got128.Eq(u128) {
		t.Errorf("Uint128 = %v, want %v", &got128, u128)
	}
	d.Uint256(&got256)
	if !got256.Eq(u256) {
		t.Errorf("Uint256 = %v, want %v", &got256, u256)
	}
	var tail [3]byte
	d.Bytes(tail[:])
	if tail != [3]byte{7, 8, 9} {
		t.Errorf("Bytes = %v", tail)
	}
	if err := d.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestDecoderShortInput(t *testing.T) {
	reads := map[string]func(*Decoder){
		"Bool":    func(d *Decoder) { d.Bool() },
		"Uint8":   func(d *Decoder) { d.Uint8() },
		"Uint16":  func(d *Decoder) { d.Uint16() },
		"Uint32":  func(d *Decoder) { d.Uint32() },
		"Uint64":  func(d *Decoder) { d.Uint64() },
		"Uint128": func(d *Decoder) { d.Uint128(new(uint256.Int)) },
		"Uint256": func(d *Decoder) { d.Uint256(new(uint256.Int)) },
		"Bytes":   func(d *Decoder) { d.Bytes(make([]byte, 2)) },
		"Offset":  func(d *Decoder) { d.Offset() },
	}
	for name, read := range reads {
		d := NewDecoder([]byte{1}) // one byte short for everything but Bool/Uint8
		if name == "Bool" || name == "Uint8" {
			d = NewDecoder(nil)
		}
		read(d)
		if err := d.Err(); !errors.Is(err, ErrSize) {
			t.Errorf("%s on short input: err = %v, want ErrSize", name, err)
		}
	}
}

func TestDecoderBadBoolean(t *testing.T) {
	for _, b := range []byte{2, 0x80, 0xff} {
		d := NewDecoder([]byte{b})
		if v := d.Bool(); v {
			t.Errorf("Bool(%#x) = true", b)
		}
		if err := d.Close(); !errors.Is(err, ErrBadBoolean) {
			t.Errorf("Bool(%#x): err = %v, want ErrBadBoolean", b, err)
		}
	}
}

func TestDecoderTrailing(t *testing.T) {
	d := NewDecoder([]byte{1, 2, 3})
	d.Uint16()
	if err := d.Close(); !errors.Is(err, ErrTrailing) {
		t.Errorf("err = %v, want ErrTrailing", err)
	}
}

// TestDecoderLatch checks that the first error sticks: later reads return
// zero without advancing, and Close reports the first failure rather than a
// consequence of it.
func TestDecoderLatch(t *testing.T) {
	d := NewDecoder([]byte{1, 2, 3})
	d.Uint32() // short: latches ErrSize
	if v := d.Uint8(); v != 0 {
		t.Errorf("read after error = %d, want 0", v)
	}
	if d.pos != 0 {
		t.Errorf("position advanced to %d after error", d.pos)
	}
	if err := d.Close(); !errors.Is(err, ErrSize) {
		t.Errorf("Close = %v, want the latched ErrSize", err)
	}
}

func TestDecoderSections(t *testing.T) {
	// Fixed part: one uint16 and two offsets (10 bytes), then the variable
	// part. Sections are built by hand so the offset rules can be broken.
	fixed := func(off1, off2 uint32) []byte {
		buf := AppendUint16(nil, 7)
		buf = AppendOffset(buf, int(off1))
		return AppendOffset(buf, int(off2))
	}
	tests := []struct {
		name  string
		data  []byte
		want  [][]byte
		err   error
		notes string
	}{
		{name: "two sections", data: append(fixed(10, 12), 1, 2, 3, 4, 5), want: [][]byte{{1, 2}, {3, 4, 5}}},
		{name: "empty first", data: append(fixed(10, 10), 1, 2), want: [][]byte{{}, {1, 2}}},
		{name: "empty last", data: append(fixed(10, 12), 1, 2), want: [][]byte{{1, 2}, {}}},
		{name: "both empty", data: fixed(10, 10), want: [][]byte{{}, {}}},
		{name: "first offset short", data: append(fixed(9, 12), 1, 2, 3), err: ErrOffset},
		{name: "first offset long", data: append(fixed(11, 12), 1, 2, 3), err: ErrOffset},
		{name: "decreasing", data: append(fixed(10, 9), 1, 2, 3), err: ErrOffset},
		{name: "beyond input", data: append(fixed(10, 14), 1, 2, 3), err: ErrOffset},
		{name: "max uint32", data: append(fixed(10, 0xffffffff), 1, 2, 3), err: ErrOffset},
		{name: "fixed part truncated", data: fixed(10, 12)[:9], err: ErrSize},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := NewDecoder(tc.data)
			d.Uint16()
			d.Offset()
			d.Offset()
			sections, err := d.Sections()
			if tc.err != nil {
				if !errors.Is(err, tc.err) {
					t.Fatalf("err = %v, want %v", err, tc.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(sections) != len(tc.want) {
				t.Fatalf("%d sections, want %d", len(sections), len(tc.want))
			}
			for i := range sections {
				if !bytes.Equal(sections[i], tc.want[i]) {
					t.Errorf("section %d = %x, want %x", i, sections[i], tc.want[i])
				}
			}
		})
	}
}

func TestDecoderMisuse(t *testing.T) {
	expectPanic := func(name string, fn func()) {
		defer func() {
			if recover() == nil {
				t.Errorf("%s did not panic", name)
			}
		}()
		fn()
	}
	expectPanic("Close with offsets", func() {
		d := NewDecoder(AppendOffset(nil, 4))
		d.Offset()
		d.Close()
	})
	expectPanic("Sections without offsets", func() {
		d := NewDecoder([]byte{1})
		d.Uint8()
		d.Sections()
	})
}

func TestDecodeUint64(t *testing.T) {
	if v, err := DecodeUint64(AppendUint64(nil, 42)); err != nil || v != 42 {
		t.Errorf("DecodeUint64 = %d, %v", v, err)
	}
	for _, n := range []int{0, 7, 9} {
		if _, err := DecodeUint64(make([]byte, n)); !errors.Is(err, ErrSize) {
			t.Errorf("%d bytes: err = %v, want ErrSize", n, err)
		}
	}
}
