package bintrie

import (
	"bytes"
	"encoding/binary"
	"math/bits"
	"testing"
)

const (
	ones63 = 0x7FFFFFFFFFFFFFFF // 63 bits of 1
)

func TestBytes(t *testing.T) {
	tests := []struct {
		name string
		ba   BitArray
		want [32]byte
	}{
		{
			name: "length == 0",
			ba:   BitArray{len: 0, words: [4]uint64{0, 0, 0, 0}},
			want: [32]byte{},
		},
		{
			name: "length < 64",
			ba:   BitArray{len: 38, words: [4]uint64{0x3FFFFFFFFF, 0, 0, 0}},
			want: func() [32]byte {
				var b [32]byte
				binary.BigEndian.PutUint64(b[24:32], 0x3FFFFFFFFF)
				return b
			}(),
		},
		{
			name: "64 <= length < 128",
			ba:   BitArray{len: 100, words: [4]uint64{maxUint64, 0xFFFFFFFFF, 0, 0}},
			want: func() [32]byte {
				var b [32]byte
				binary.BigEndian.PutUint64(b[16:24], 0xFFFFFFFFF)
				binary.BigEndian.PutUint64(b[24:32], maxUint64)
				return b
			}(),
		},
		{
			name: "128 <= length < 192",
			ba:   BitArray{len: 130, words: [4]uint64{maxUint64, maxUint64, 0x3, 0}},
			want: func() [32]byte {
				var b [32]byte
				binary.BigEndian.PutUint64(b[8:16], 0x3)
				binary.BigEndian.PutUint64(b[16:24], maxUint64)
				binary.BigEndian.PutUint64(b[24:32], maxUint64)
				return b
			}(),
		},
		{
			name: "192 <= length < 255",
			ba:   BitArray{len: 201, words: [4]uint64{maxUint64, maxUint64, maxUint64, 0x1FF}},
			want: func() [32]byte {
				var b [32]byte
				binary.BigEndian.PutUint64(b[0:8], 0x1FF)
				binary.BigEndian.PutUint64(b[8:16], maxUint64)
				binary.BigEndian.PutUint64(b[16:24], maxUint64)
				binary.BigEndian.PutUint64(b[24:32], maxUint64)
				return b
			}(),
		},
		{
			name: "length == 254",
			ba:   BitArray{len: 254, words: [4]uint64{maxUint64, maxUint64, maxUint64, 0x3FFFFFFFFFFFFFFF}},
			want: func() [32]byte {
				var b [32]byte
				binary.BigEndian.PutUint64(b[0:8], 0x3FFFFFFFFFFFFFFF)
				binary.BigEndian.PutUint64(b[8:16], maxUint64)
				binary.BigEndian.PutUint64(b[16:24], maxUint64)
				binary.BigEndian.PutUint64(b[24:32], maxUint64)
				return b
			}(),
		},
		{
			name: "length == 255",
			ba:   BitArray{len: 255, words: [4]uint64{maxUint64, maxUint64, maxUint64, ones63}},
			want: func() [32]byte {
				var b [32]byte
				binary.BigEndian.PutUint64(b[0:8], ones63)
				binary.BigEndian.PutUint64(b[8:16], maxUint64)
				binary.BigEndian.PutUint64(b[16:24], maxUint64)
				binary.BigEndian.PutUint64(b[24:32], maxUint64)
				return b
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.ba.Bytes()
			if !bytes.Equal(got[:], tt.want[:]) {
				t.Errorf("BitArray.Bytes() = %v, want %v", got, tt.want)
			}

			// check if the received bytes has the same bit count as the BitArray.len
			count := 0
			for _, b := range got {
				count += bits.OnesCount8(b)
			}
			if count != int(tt.ba.len) {
				t.Errorf("BitArray.Bytes() bit count = %v, want %v", count, tt.ba.len)
			}
		})
	}
}

func TestRsh(t *testing.T) {
	tests := []struct {
		name     string
		initial  *BitArray
		shiftBy  uint8
		expected *BitArray
	}{
		{
			name: "zero length array",
			initial: &BitArray{
				len:   0,
				words: [4]uint64{0, 0, 0, 0},
			},
			shiftBy: 5,
			expected: &BitArray{
				len:   0,
				words: [4]uint64{0, 0, 0, 0},
			},
		},
		{
			name: "shift by 0",
			initial: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			shiftBy: 0,
			expected: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name: "shift by more than length",
			initial: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
			shiftBy: 65,
			expected: &BitArray{
				len:   0,
				words: [4]uint64{0, 0, 0, 0},
			},
		},
		{
			name: "shift by less than 64",
			initial: &BitArray{
				len:   128,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
			shiftBy: 32,
			expected: &BitArray{
				len:   96,
				words: [4]uint64{maxUint64, 0x00000000FFFFFFFF, 0, 0},
			},
		},
		{
			name: "shift by exactly 64",
			initial: &BitArray{
				len:   128,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
			shiftBy: 64,
			expected: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name: "shift by 127",
			initial: &BitArray{
				len:   255,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, ones63},
			},
			shiftBy: 127,
			expected: &BitArray{
				len:   128,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
		},
		{
			name: "shift by 128",
			initial: &BitArray{
				len:   251,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, maxUint64},
			},
			shiftBy: 128,
			expected: &BitArray{
				len:   123,
				words: [4]uint64{maxUint64, 0x7FFFFFFFFFFFFFF, 0, 0},
			},
		},
		{
			name: "shift by 192",
			initial: &BitArray{
				len:   251,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, maxUint64},
			},
			shiftBy: 192,
			expected: &BitArray{
				len:   59,
				words: [4]uint64{0x7FFFFFFFFFFFFFF, 0, 0, 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := new(BitArray).rsh(tt.initial, tt.shiftBy)
			if !result.Equal(tt.expected) {
				t.Errorf("rsh() got = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestLsh(t *testing.T) {
	tests := []struct {
		name string
		x    *BitArray
		n    uint8
		want *BitArray
	}{
		{
			name: "empty array",
			x:    emptyBitArray,
			n:    5,
			want: emptyBitArray,
		},
		{
			name: "shift by 0",
			x: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			n: 0,
			want: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name: "shift within first word",
			x: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
			n: 4,
			want: &BitArray{
				len:   8,
				words: [4]uint64{0xF0, 0, 0, 0}, // 11110000
			},
		},
		{
			name: "shift across word boundary",
			x: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
			n: 62,
			want: &BitArray{
				len:   66,
				words: [4]uint64{0xC000000000000000, 0x3, 0, 0},
			},
		},
		{
			name: "shift by 64 (full word)",
			x: &BitArray{
				len:   8,
				words: [4]uint64{0xFF, 0, 0, 0}, // 11111111
			},
			n: 64,
			want: &BitArray{
				len:   72,
				words: [4]uint64{0, 0xFF, 0, 0},
			},
		},
		{
			name: "shift by 128",
			x: &BitArray{
				len:   8,
				words: [4]uint64{0xFF, 0, 0, 0}, // 11111111
			},
			n: 128,
			want: &BitArray{
				len:   136,
				words: [4]uint64{0, 0, 0xFF, 0},
			},
		},
		{
			name: "shift by 192",
			x: &BitArray{
				len:   8,
				words: [4]uint64{0xFF, 0, 0, 0}, // 11111111
			},
			n: 192,
			want: &BitArray{
				len:   200,
				words: [4]uint64{0, 0, 0, 0xFF},
			},
		},
		{
			name: "shift causing length overflow",
			x: &BitArray{
				len:   200,
				words: [4]uint64{0xFF, 0, 0, 0},
			},
			n: 60,
			want: &BitArray{
				len: 255, // capped at maxUint8
				words: [4]uint64{
					0xF000000000000000,
					0xF,
					0,
					0,
				},
			},
		},
		{
			name: "shift sparse bits",
			x: &BitArray{
				len:   8,
				words: [4]uint64{0xAA, 0, 0, 0}, // 10101010
			},
			n: 4,
			want: &BitArray{
				len:   12,
				words: [4]uint64{0xAA0, 0, 0, 0}, // 101010100000
			},
		},
		{
			name: "shift partial word across boundary",
			x: &BitArray{
				len:   100,
				words: [4]uint64{0xFF, 0xFF, 0, 0},
			},
			n: 60,
			want: &BitArray{
				len: 160,
				words: [4]uint64{
					0xF000000000000000,
					0xF00000000000000F,
					0xF,
					0,
				},
			},
		},
		{
			name: "near maximum length shift",
			x: &BitArray{
				len:   251,
				words: [4]uint64{0xFF, 0, 0, 0},
			},
			n: 4,
			want: &BitArray{
				len:   255, // capped at maxUint8
				words: [4]uint64{0xFF0, 0, 0, 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := new(BitArray).lsh(tt.x, tt.n)
			if !got.Equal(tt.want) {
				t.Errorf("Lsh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAppend(t *testing.T) {
	tests := []struct {
		name string
		x    *BitArray
		y    *BitArray
		want *BitArray
	}{
		{
			name: "both empty arrays",
			x:    emptyBitArray,
			y:    emptyBitArray,
			want: emptyBitArray,
		},
		{
			name: "first array empty",
			x:    emptyBitArray,
			y: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
			want: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
		},
		{
			name: "second array empty",
			x: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
			y: emptyBitArray,
			want: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
		},
		{
			name: "within first word",
			x: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
			y: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
			want: &BitArray{
				len:   8,
				words: [4]uint64{0xFF, 0, 0, 0}, // 11111111
			},
		},
		{
			name: "different lengths within word",
			x: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
			y: &BitArray{
				len:   2,
				words: [4]uint64{0x3, 0, 0, 0}, // 11
			},
			want: &BitArray{
				len:   6,
				words: [4]uint64{0x3F, 0, 0, 0}, // 111111
			},
		},
		{
			name: "across word boundary",
			x: &BitArray{
				len:   62,
				words: [4]uint64{0x3FFFFFFFFFFFFFFF, 0, 0, 0},
			},
			y: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0}, // 1111
			},
			want: &BitArray{
				len:   66,
				words: [4]uint64{maxUint64, 0x3, 0, 0},
			},
		},
		{
			name: "across multiple words",
			x: &BitArray{
				len:   128,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
			y: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			want: &BitArray{
				len:   192,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, 0},
			},
		},
		{
			name: "sparse bits",
			x: &BitArray{
				len:   8,
				words: [4]uint64{0xAA, 0, 0, 0}, // 10101010
			},
			y: &BitArray{
				len:   8,
				words: [4]uint64{0x55, 0, 0, 0}, // 01010101
			},
			want: &BitArray{
				len:   16,
				words: [4]uint64{0xAA55, 0, 0, 0}, // 1010101001010101
			},
		},
		{
			name: "result exactly at length limit",
			x: &BitArray{
				len:   251,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, 0x7FFFFFFFFFFFFFFF},
			},
			y: &BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0},
			},
			want: &BitArray{
				len:   255,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, 0x7FFFFFFFFFFFFFFF},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := new(BitArray).Append(tt.x, tt.y)
			if !got.Equal(tt.want) {
				t.Errorf("Append() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLSBs(t *testing.T) {
	tests := []struct {
		name string
		x    *BitArray
		pos  uint8
		want *BitArray
	}{
		{
			name: "zero position",
			x: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			pos: 0,
			want: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name: "position beyond length",
			x: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			pos: 65,
			want: &BitArray{
				len:   0,
				words: [4]uint64{0, 0, 0, 0},
			},
		},
		{
			name: "get last 4 bits",
			x: &BitArray{
				len:   8,
				words: [4]uint64{0xFF, 0, 0, 0}, // 11111111
			},
			pos: 4,
			want: &BitArray{
				len:   4,
				words: [4]uint64{0x0F, 0, 0, 0}, // 1111
			},
		},
		{
			name: "get bits across word boundary",
			x: &BitArray{
				len:   128,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
			pos: 64,
			want: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name: "get bits from max length array",
			x: &BitArray{
				len:   251,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, 0x7FFFFFFFFFFFFFF},
			},
			pos: 200,
			want: &BitArray{
				len:   51,
				words: [4]uint64{0x7FFFFFFFFFFFF, 0, 0, 0},
			},
		},
		{
			name: "empty array",
			x:    emptyBitArray,
			pos:  1,
			want: &BitArray{
				len:   0,
				words: [4]uint64{0, 0, 0, 0},
			},
		},
		{
			name: "sparse bits",
			x: &BitArray{
				len:   16,
				words: [4]uint64{0xAAAA, 0, 0, 0}, // 1010101010101010
			},
			pos: 8,
			want: &BitArray{
				len:   8,
				words: [4]uint64{0xAA, 0, 0, 0}, // 10101010
			},
		},
		{
			name: "position equals length",
			x: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			pos: 64,
			want: &BitArray{
				len:   0,
				words: [4]uint64{0, 0, 0, 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := new(BitArray).lsb(tt.x, tt.pos)
			if !got.Equal(tt.want) {
				t.Errorf("LSBs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLSBsFromLSB(t *testing.T) {
	tests := []struct {
		name     string
		initial  BitArray
		length   uint8
		expected BitArray
	}{
		{
			name: "zero",
			initial: BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			length: 0,
			expected: BitArray{
				len:   0,
				words: [4]uint64{0, 0, 0, 0},
			},
		},
		{
			name: "get 32 LSBs",
			initial: BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			length: 32,
			expected: BitArray{
				len:   32,
				words: [4]uint64{0x00000000FFFFFFFF, 0, 0, 0},
			},
		},
		{
			name: "get 1 LSB",
			initial: BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			length: 1,
			expected: BitArray{
				len:   1,
				words: [4]uint64{0x1, 0, 0, 0},
			},
		},
		{
			name: "get 100 LSBs across words",
			initial: BitArray{
				len:   128,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
			length: 100,
			expected: BitArray{
				len:   100,
				words: [4]uint64{maxUint64, 0x0000000FFFFFFFFF, 0, 0},
			},
		},
		{
			name: "get 64 LSBs at word boundary",
			initial: BitArray{
				len:   128,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
			length: 64,
			expected: BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name: "get 128 LSBs at word boundary",
			initial: BitArray{
				len:   192,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, 0},
			},
			length: 128,
			expected: BitArray{
				len:   128,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
		},
		{
			name: "get 150 LSBs in third word",
			initial: BitArray{
				len:   192,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, 0},
			},
			length: 150,
			expected: BitArray{
				len:   150,
				words: [4]uint64{maxUint64, maxUint64, 0x3FFFFF, 0},
			},
		},
		{
			name: "get 220 LSBs in fourth word",
			initial: BitArray{
				len:   255,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, maxUint64},
			},
			length: 220,
			expected: BitArray{
				len:   220,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, 0xFFFFFFF},
			},
		},
		{
			name: "get 251 LSBs",
			initial: BitArray{
				len:   255,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, maxUint64},
			},
			length: 251,
			expected: BitArray{
				len:   251,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, 0x7FFFFFFFFFFFFFF},
			},
		},
		{
			name: "get 100 LSBs from sparse bits",
			initial: BitArray{
				len:   128,
				words: [4]uint64{0xAAAAAAAAAAAAAAAA, 0x5555555555555555, 0, 0},
			},
			length: 100,
			expected: BitArray{
				len:   100,
				words: [4]uint64{0xAAAAAAAAAAAAAAAA, 0x0000000555555555, 0, 0},
			},
		},
		{
			name: "no change when new length equals current length",
			initial: BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			length: 64,
			expected: BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name: "no change when new length greater than current length",
			initial: BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			length: 128,
			expected: BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := new(BitArray).copyLsb(&tt.initial, tt.length)
			if !result.Equal(&tt.expected) {
				t.Errorf("Truncate() got = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestMSBs(t *testing.T) {
	tests := []struct {
		name string
		x    *BitArray
		n    uint8
		want *BitArray
	}{
		{
			name: "empty array",
			x:    emptyBitArray,
			n:    0,
			want: emptyBitArray,
		},
		{
			name: "get all bits",
			x: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			n: 64,
			want: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name: "get more bits than available",
			x: &BitArray{
				len:   32,
				words: [4]uint64{0xFFFFFFFF, 0, 0, 0},
			},
			n: 64,
			want: &BitArray{
				len:   32,
				words: [4]uint64{0xFFFFFFFF, 0, 0, 0},
			},
		},
		{
			name: "get half of available bits",
			x: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			n: 32,
			want: &BitArray{
				len:   32,
				words: [4]uint64{0xFFFFFFFF00000000 >> 32, 0, 0, 0},
			},
		},
		{
			name: "get MSBs across word boundary",
			x: &BitArray{
				len:   128,
				words: [4]uint64{maxUint64, maxUint64, 0, 0},
			},
			n: 100,
			want: &BitArray{
				len:   100,
				words: [4]uint64{maxUint64, maxUint64 >> 28, 0, 0},
			},
		},
		{
			name: "get MSBs from max length array",
			x: &BitArray{
				len:   255,
				words: [4]uint64{maxUint64, maxUint64, maxUint64, ones63},
			},
			n: 64,
			want: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name: "get zero bits",
			x: &BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
			n: 0,
			want: &BitArray{
				len:   0,
				words: [4]uint64{0, 0, 0, 0},
			},
		},
		{
			name: "sparse bits",
			x: &BitArray{
				len:   128,
				words: [4]uint64{0xAAAAAAAAAAAAAAAA, 0x5555555555555555, 0, 0},
			},
			n: 64,
			want: &BitArray{
				len:   64,
				words: [4]uint64{0x5555555555555555, 0, 0, 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := new(BitArray).MSBs(tt.x, tt.n)
			if !got.Equal(tt.want) {
				t.Errorf("MSBs() = %v, want %v", got, tt.want)
			}

			if got.len != tt.want.len {
				t.Errorf("MSBs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetBit(t *testing.T) {
	tests := []struct {
		name string
		bit  uint8
		want BitArray
	}{
		{
			name: "set bit 0",
			bit:  0,
			want: BitArray{
				len:   1,
				words: [4]uint64{0, 0, 0, 0},
			},
		},
		{
			name: "set bit 1",
			bit:  1,
			want: BitArray{
				len:   1,
				words: [4]uint64{1, 0, 0, 0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := new(BitArray).SetBit(tt.bit)
			if !got.Equal(&tt.want) {
				t.Errorf("SetBit(%v) = %v, want %v", tt.bit, got, tt.want)
			}
		})
	}
}

func TestSetBytes(t *testing.T) {
	tests := []struct {
		name   string
		length uint8
		data   []byte
		want   BitArray
	}{
		{
			name:   "empty data",
			length: 0,
			data:   []byte{},
			want: BitArray{
				len:   0,
				words: [4]uint64{0, 0, 0, 0},
			},
		},
		{
			name:   "single byte",
			length: 8,
			data:   []byte{0xFF},
			want: BitArray{
				len:   8,
				words: [4]uint64{0xFF, 0, 0, 0},
			},
		},
		{
			name:   "two bytes",
			length: 16,
			data:   []byte{0xAA, 0xFF},
			want: BitArray{
				len:   16,
				words: [4]uint64{0xAAFF, 0, 0, 0},
			},
		},
		{
			name:   "three bytes",
			length: 24,
			data:   []byte{0xAA, 0xBB, 0xCC},
			want: BitArray{
				len:   24,
				words: [4]uint64{0xAABBCC, 0, 0, 0},
			},
		},
		{
			name:   "four bytes",
			length: 32,
			data:   []byte{0xAA, 0xBB, 0xCC, 0xDD},
			want: BitArray{
				len:   32,
				words: [4]uint64{0xAABBCCDD, 0, 0, 0},
			},
		},
		{
			name:   "eight bytes (full word)",
			length: 64,
			data:   []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			want: BitArray{
				len:   64,
				words: [4]uint64{maxUint64, 0, 0, 0},
			},
		},
		{
			name:   "sixteen bytes (two words)",
			length: 128,
			data: []byte{
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA, 0xAA,
			},
			want: BitArray{
				len: 128,
				words: [4]uint64{
					0xAAAAAAAAAAAAAAAA,
					0xFFFFFFFFFFFFFFFF,
					0, 0,
				},
			},
		},
		{
			name:   "thirty-two bytes (full array)",
			length: 251,
			data: []byte{
				0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
			},
			want: BitArray{
				len: 251,
				words: [4]uint64{
					maxUint64,
					maxUint64,
					maxUint64,
					0x7FFFFFFFFFFFFFF,
				},
			},
		},
		{
			name:   "truncate to length",
			length: 4,
			data:   []byte{0xFF},
			want: BitArray{
				len:   4,
				words: [4]uint64{0xF, 0, 0, 0},
			},
		},
		{
			name:   "data larger than 32 bytes",
			length: 251,
			data: []byte{
				0x7F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
				0xFF, 0xFF, 0xFF, 0xFF, // extra bytes should be ignored
			},
			want: BitArray{
				len: 251,
				words: [4]uint64{
					maxUint64,
					maxUint64,
					maxUint64,
					0x7FFFFFFFFFFFFFF,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := new(BitArray).SetBytes(tt.length, tt.data)
			if !got.Equal(&tt.want) {
				t.Errorf("SetBytes(%d, %v) = %v, want %v", tt.length, tt.data, got, tt.want)
			}
		})
	}
}
