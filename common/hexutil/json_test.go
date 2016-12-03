// Copyright 2016 The go-ethereum Authors
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

package hexutil

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"
)

func checkError(t *testing.T, input string, got, want error) bool {
	if got == nil {
		if want != nil {
			t.Errorf("input %s: got no error, want %q", input, want)
			return false
		}
		return true
	}
	if want == nil {
		t.Errorf("input %s: unexpected error %q", input, got)
	} else if got.Error() != want.Error() {
		t.Errorf("input %s: got error %q, want %q", input, got, want)
	}
	return false
}

func referenceBig(s string) *big.Int {
	b, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("invalid")
	}
	return b
}

func referenceBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

var unmarshalBytesTests = []unmarshalTest{
	// invalid encoding
	{input: "", wantErr: errNonString},
	{input: "null", wantErr: errNonString},
	{input: "10", wantErr: errNonString},
	{input: `""`, wantErr: ErrEmptyString},
	{input: `"0"`, wantErr: ErrMissingPrefix},
	{input: `"0x0"`, wantErr: ErrOddLength},
	{input: `"0xxx"`, wantErr: ErrSyntax},
	{input: `"0x01zz01"`, wantErr: ErrSyntax},

	// valid encoding
	{input: `"0x"`, want: referenceBytes("")},
	{input: `"0x02"`, want: referenceBytes("02")},
	{input: `"0X02"`, want: referenceBytes("02")},
	{input: `"0xffffffffff"`, want: referenceBytes("ffffffffff")},
	{
		input: `"0xffffffffffffffffffffffffffffffffffff"`,
		want:  referenceBytes("ffffffffffffffffffffffffffffffffffff"),
	},
}

func TestUnmarshalBytes(t *testing.T) {
	for _, test := range unmarshalBytesTests {
		var v Bytes
		err := v.UnmarshalJSON([]byte(test.input))
		if !checkError(t, test.input, err, test.wantErr) {
			continue
		}
		if !bytes.Equal(test.want.([]byte), []byte(v)) {
			t.Errorf("input %s: value mismatch: got %x, want %x", test.input, &v, test.want)
			continue
		}
	}
}

func BenchmarkUnmarshalBytes(b *testing.B) {
	input := []byte(`"0x123456789abcdef123456789abcdef"`)
	for i := 0; i < b.N; i++ {
		var v Bytes
		if err := v.UnmarshalJSON(input); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMarshalBytes(t *testing.T) {
	for _, test := range encodeBytesTests {
		in := test.input.([]byte)
		out, err := Bytes(in).MarshalJSON()
		if err != nil {
			t.Errorf("%x: %v", in, err)
			continue
		}
		if want := `"` + test.want + `"`; string(out) != want {
			t.Errorf("%x: MarshalJSON output mismatch: got %q, want %q", in, out, want)
			continue
		}
		if out := Bytes(in).String(); out != test.want {
			t.Errorf("%x: String mismatch: got %q, want %q", in, out, test.want)
			continue
		}
	}
}

var unmarshalBigTests = []unmarshalTest{
	// invalid encoding
	{input: "", wantErr: errNonString},
	{input: "null", wantErr: errNonString},
	{input: "10", wantErr: errNonString},
	{input: `""`, wantErr: ErrEmptyString},
	{input: `"0"`, wantErr: ErrMissingPrefix},
	{input: `"0x"`, wantErr: ErrEmptyNumber},
	{input: `"0x01"`, wantErr: ErrLeadingZero},
	{input: `"0xx"`, wantErr: ErrSyntax},
	{input: `"0x1zz01"`, wantErr: ErrSyntax},

	// valid encoding
	{input: `"0x0"`, want: big.NewInt(0)},
	{input: `"0x2"`, want: big.NewInt(0x2)},
	{input: `"0x2F2"`, want: big.NewInt(0x2f2)},
	{input: `"0X2F2"`, want: big.NewInt(0x2f2)},
	{input: `"0x1122aaff"`, want: big.NewInt(0x1122aaff)},
	{input: `"0xbBb"`, want: big.NewInt(0xbbb)},
	{input: `"0xfffffffff"`, want: big.NewInt(0xfffffffff)},
	{
		input: `"0x112233445566778899aabbccddeeff"`,
		want:  referenceBig("112233445566778899aabbccddeeff"),
	},
	{
		input: `"0xffffffffffffffffffffffffffffffffffff"`,
		want:  referenceBig("ffffffffffffffffffffffffffffffffffff"),
	},
}

func TestUnmarshalBig(t *testing.T) {
	for _, test := range unmarshalBigTests {
		var v Big
		err := v.UnmarshalJSON([]byte(test.input))
		if !checkError(t, test.input, err, test.wantErr) {
			continue
		}
		if test.want != nil && test.want.(*big.Int).Cmp((*big.Int)(&v)) != 0 {
			t.Errorf("input %s: value mismatch: got %x, want %x", test.input, (*big.Int)(&v), test.want)
			continue
		}
	}
}

func BenchmarkUnmarshalBig(b *testing.B) {
	input := []byte(`"0x123456789abcdef123456789abcdef"`)
	for i := 0; i < b.N; i++ {
		var v Big
		if err := v.UnmarshalJSON(input); err != nil {
			b.Fatal(err)
		}
	}
}

func TestMarshalBig(t *testing.T) {
	for _, test := range encodeBigTests {
		in := test.input.(*big.Int)
		out, err := (*Big)(in).MarshalJSON()
		if err != nil {
			t.Errorf("%d: %v", in, err)
			continue
		}
		if want := `"` + test.want + `"`; string(out) != want {
			t.Errorf("%d: MarshalJSON output mismatch: got %q, want %q", in, out, want)
			continue
		}
		if out := (*Big)(in).String(); out != test.want {
			t.Errorf("%x: String mismatch: got %q, want %q", in, out, test.want)
			continue
		}
	}
}

var unmarshalUint64Tests = []unmarshalTest{
	// invalid encoding
	{input: "", wantErr: errNonString},
	{input: "null", wantErr: errNonString},
	{input: "10", wantErr: errNonString},
	{input: `""`, wantErr: ErrEmptyString},
	{input: `"0"`, wantErr: ErrMissingPrefix},
	{input: `"0x"`, wantErr: ErrEmptyNumber},
	{input: `"0x01"`, wantErr: ErrLeadingZero},
	{input: `"0xfffffffffffffffff"`, wantErr: ErrUint64Range},
	{input: `"0xx"`, wantErr: ErrSyntax},
	{input: `"0x1zz01"`, wantErr: ErrSyntax},

	// valid encoding
	{input: `"0x0"`, want: uint64(0)},
	{input: `"0x2"`, want: uint64(0x2)},
	{input: `"0x2F2"`, want: uint64(0x2f2)},
	{input: `"0X2F2"`, want: uint64(0x2f2)},
	{input: `"0x1122aaff"`, want: uint64(0x1122aaff)},
	{input: `"0xbbb"`, want: uint64(0xbbb)},
	{input: `"0xffffffffffffffff"`, want: uint64(0xffffffffffffffff)},
}

func TestUnmarshalUint64(t *testing.T) {
	for _, test := range unmarshalUint64Tests {
		var v Uint64
		err := v.UnmarshalJSON([]byte(test.input))
		if !checkError(t, test.input, err, test.wantErr) {
			continue
		}
		if uint64(v) != test.want.(uint64) {
			t.Errorf("input %s: value mismatch: got %d, want %d", test.input, v, test.want)
			continue
		}
	}
}

func BenchmarkUnmarshalUint64(b *testing.B) {
	input := []byte(`"0x123456789abcdf"`)
	for i := 0; i < b.N; i++ {
		var v Uint64
		v.UnmarshalJSON(input)
	}
}

func TestMarshalUint64(t *testing.T) {
	for _, test := range encodeUint64Tests {
		in := test.input.(uint64)
		out, err := Uint64(in).MarshalJSON()
		if err != nil {
			t.Errorf("%d: %v", in, err)
			continue
		}
		if want := `"` + test.want + `"`; string(out) != want {
			t.Errorf("%d: MarshalJSON output mismatch: got %q, want %q", in, out, want)
			continue
		}
		if out := (Uint64)(in).String(); out != test.want {
			t.Errorf("%x: String mismatch: got %q, want %q", in, out, test.want)
			continue
		}
	}
}
