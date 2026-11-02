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
	"reflect"
	"strings"
	"testing"
)

// word is a fixed-size element of two bytes.
type word uint16

func (w *word) SizeSSZ() int                            { return 2 }
func (w *word) MarshalSSZTo(dst []byte) ([]byte, error) { return AppendUint16(dst, uint16(*w)), nil }
func (w *word) UnmarshalSSZ(data []byte) error {
	d := NewDecoder(data)
	*w = word(d.Uint16())
	return d.Close()
}
func (w *word) HashTreeRoot() [32]byte { return Merkleize(Pack(AppendUint16(nil, uint16(*w))), 1) }

// blob is a variable-size element: a container holding one byte list, so
// its serialization is a 4-byte offset followed by the bytes.
type blob []byte

func (b *blob) SizeSSZ() int { return BytesPerLengthOffset + len(*b) }
func (b *blob) MarshalSSZTo(dst []byte) ([]byte, error) {
	dst = AppendOffset(dst, BytesPerLengthOffset)
	return append(dst, *b...), nil
}
func (b *blob) UnmarshalSSZ(data []byte) error {
	d := NewDecoder(data)
	d.Offset()
	sections, err := d.Sections()
	if err != nil {
		return err
	}
	*b = append(blob(nil), sections[0]...)
	return nil
}
func (b *blob) HashTreeRoot() [32]byte {
	return Merkleize([][32]byte{MixInLength(Merkleize(Pack(*b), 1), uint64(len(*b)))}, 1)
}

func TestFixedSlice(t *testing.T) {
	for _, in := range [][]word{nil, {1}, {1, 2, 3}} {
		ptrs := make([]*word, len(in))
		for i := range in {
			ptrs[i] = &in[i]
		}
		if size := SizeFixedSlice(ptrs); size != 2*len(in) {
			t.Errorf("SizeFixedSlice(%v) = %d", in, size)
		}
		buf, err := MarshalFixedSlice([]byte{0xee}, ptrs)
		if err != nil {
			t.Fatal(err)
		}
		out, err := UnmarshalFixedSlice[word, *word](buf[1:], 2, 3)
		if err != nil {
			t.Fatal(err)
		}
		if len(in) == 0 {
			in = []word{}
		}
		if !reflect.DeepEqual(out, in) {
			t.Errorf("round trip of %v gave %v", in, out)
		}
	}
}

func TestUnmarshalFixedSliceErrors(t *testing.T) {
	if _, err := UnmarshalFixedSlice[word, *word]([]byte{1, 2, 3}, 2, 8); !errors.Is(err, ErrSize) {
		t.Errorf("misaligned: err = %v, want ErrSize", err)
	}
	if _, err := UnmarshalFixedSlice[word, *word](make([]byte, 6), 2, 2); !errors.Is(err, ErrTooBig) {
		t.Errorf("over limit: err = %v, want ErrTooBig", err)
	}
	defer func() {
		if recover() == nil {
			t.Error("zero element size did not panic")
		}
	}()
	UnmarshalFixedSlice[word, *word](nil, 0, 8)
}

func TestVariableSlice(t *testing.T) {
	for _, in := range [][]blob{nil, {{}}, {{1, 2}}, {{}, {3}, {}}, {{1}, {2, 3}, {4, 5, 6}}} {
		ptrs := make([]*blob, len(in))
		for i := range in {
			ptrs[i] = &in[i]
		}
		buf, err := MarshalVariableSlice([]byte{0xee}, ptrs)
		if err != nil {
			t.Fatal(err)
		}
		if size := SizeVariableSlice(ptrs); size != len(buf)-1 {
			t.Errorf("SizeVariableSlice = %d, marshaled %d", size, len(buf)-1)
		}
		out, err := UnmarshalVariableSlice[blob, *blob](buf[1:], 3)
		if err != nil {
			t.Fatalf("%v: %v", in, err)
		}
		if len(out) != len(in) {
			t.Fatalf("%v: got %d elements", in, len(out))
		}
		for i := range in {
			if !bytes.Equal(out[i], in[i]) {
				t.Errorf("element %d = %x, want %x", i, out[i], in[i])
			}
		}
	}
}

func TestUnmarshalVariableSliceErrors(t *testing.T) {
	two := func() []byte { // two elements: offsets 8 and 10, payloads (4)(1) and (4)(2)
		buf := AppendOffset(nil, 8)
		buf = AppendOffset(buf, 13)
		buf = AppendOffset(buf, 4)
		buf = append(buf, 1)
		buf = AppendOffset(buf, 4)
		return append(buf, 2)
	}
	if _, err := UnmarshalVariableSlice[blob, *blob](two(), 8); err != nil {
		t.Fatalf("baseline: %v", err)
	}
	tests := []struct {
		name string
		data []byte
		err  error
	}{
		{"short of an offset", []byte{1, 2, 3}, ErrSize},
		{"zero first offset", AppendOffset(nil, 0), ErrOffset},
		{"first offset misaligned", append(AppendOffset(nil, 6), 0, 0), ErrOffset},
		{"first offset beyond input", AppendOffset(nil, 8), ErrOffset},
		{"decreasing", func() []byte { b := two(); copy(b[4:8], AppendOffset(nil, 7)); return b }(), ErrOffset},
		{"beyond input", func() []byte { b := two(); copy(b[4:8], AppendOffset(nil, 99)); return b }(), ErrOffset},
		{"element error", func() []byte { b := two(); copy(b[13:17], AppendOffset(nil, 3)); return b }(), ErrOffset},
	}
	for _, tc := range tests {
		_, err := UnmarshalVariableSlice[blob, *blob](tc.data, 8)
		if !errors.Is(err, tc.err) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.err)
		}
	}
	if _, err := UnmarshalVariableSlice[blob, *blob](two(), 1); !errors.Is(err, ErrTooBig) {
		t.Errorf("over limit: err = %v, want ErrTooBig", err)
	}
	if _, err := UnmarshalVariableSlice[blob, *blob](tests[6].data, 8); !strings.HasPrefix(err.Error(), "element 1:") {
		t.Errorf("element error not attributed: %v", err)
	}
}
