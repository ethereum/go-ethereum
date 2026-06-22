package bintrie

import (
	"bytes"
	"testing"
)

// ba builds a BitArray with the given length and leading bytes, for use as an
// expected value. Remaining bytes are zero.
func ba(length uint8, lead ...byte) BitArray {
	var b BitArray
	b.len = length
	copy(b.bytes[:], lead)
	return b
}

func TestNewBitArray(t *testing.T) {
	tests := []struct {
		name   string
		length uint8
		val    uint64
		want   BitArray
	}{
		{"empty", 0, 0, ba(0)},
		{"single 1", 1, 1, ba(1, 0x80)},
		{"single 0", 1, 0, ba(1, 0x00)},
		{"101", 3, 0b101, ba(3, 0xA0)},
		{"full byte", 8, 0xFF, ba(8, 0xFF)},
		{"ten bits", 10, 0x3FF, ba(10, 0xFF, 0xC0)},
		{"high bits ignored beyond length", 3, 0b11101, ba(3, 0xA0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBitArray(tt.length, tt.val)
			if !got.Equal(&tt.want) {
				t.Errorf("NewBitArray(%d, %#x) = %x (len %d), want %x (len %d)",
					tt.length, tt.val, got.bytes, got.len, tt.want.bytes, tt.want.len)
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
		{"empty", 0, []byte{0xFF}, ba(0)},
		{"full byte", 8, []byte{0xAB}, ba(8, 0xAB)},
		{"top 4 bits", 4, []byte{0xFF}, ba(4, 0xF0)},
		{"11 bits masks tail", 11, []byte{0xFF, 0xFF}, ba(11, 0xFF, 0xE0)},
		{"data longer than length", 4, []byte{0xFF, 0xFF}, ba(4, 0xF0)},
		{"data shorter than length", 16, []byte{0xAB}, ba(16, 0xAB, 0x00)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := new(BitArray).SetBytes(tt.length, tt.data)
			if !got.Equal(&tt.want) {
				t.Errorf("SetBytes(%d, %x) = %x (len %d), want %x (len %d)",
					tt.length, tt.data, got.bytes, got.len, tt.want.bytes, tt.want.len)
			}
		})
	}
}

func TestSetBytesFull(t *testing.T) {
	data := bytes.Repeat([]byte{0xFF}, 32)
	got := new(BitArray).SetBytes(248, data)
	want := ba(248)
	for i := 0; i < 31; i++ {
		want.bytes[i] = 0xFF
	}
	if !got.Equal(&want) {
		t.Errorf("SetBytes(248, 0xFF*32): byte 31 must be zeroed; got %x", got.bytes)
	}
}

func TestMSBs(t *testing.T) {
	x := new(BitArray).SetBytes(16, []byte{0xAB, 0xCD})
	tests := []struct {
		name string
		n    uint8
		want BitArray
	}{
		{"prefix byte", 8, ba(8, 0xAB)},
		{"prefix nibble", 4, ba(4, 0xA0)},
		{"zero", 0, ba(0)},
		{"n equals len", 16, ba(16, 0xAB, 0xCD)},
		{"n exceeds len copies x", 20, ba(16, 0xAB, 0xCD)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := new(BitArray).MSBs(x, tt.n)
			if !got.Equal(&tt.want) {
				t.Errorf("MSBs(x, %d) = %x (len %d), want %x (len %d)",
					tt.n, got.bytes, got.len, tt.want.bytes, tt.want.len)
			}
		})
	}
}

func TestAppendBit(t *testing.T) {
	// Build "101" one bit at a time from empty.
	var p BitArray
	for _, bit := range []uint8{1, 0, 1} {
		p.AppendBit(&p, bit) // receiver aliases argument
	}
	if want := ba(3, 0xA0); !p.Equal(&want) {
		t.Fatalf("append 1,0,1 = %x (len %d), want %x (len 3)", p.bytes, p.len, want.bytes)
	}

	// Append across a byte boundary: 8 ones then a 1 → 9 bits.
	var q BitArray
	for i := 0; i < 9; i++ {
		q.AppendBit(&q, 1)
	}
	if want := ba(9, 0xFF, 0x80); !q.Equal(&want) {
		t.Fatalf("append nine 1s = %x (len %d), want %x (len 9)", q.bytes, q.len, want.bytes)
	}

	// Appending to a copy must not mutate the source.
	src := new(BitArray).SetBytes(4, []byte{0xF0})
	child := *src
	child.AppendBit(&child, 0)
	if want := ba(4, 0xF0); !src.Equal(&want) {
		t.Errorf("source mutated by append on copy: %x", src.bytes)
	}
	if want := ba(5, 0xF0); !child.Equal(&want) {
		t.Errorf("append 0 = %x (len %d), want %x (len 5)", child.bytes, child.len, want.bytes)
	}
}

func TestSetBit(t *testing.T) {
	if got, want := new(BitArray).SetBit(1), ba(1, 0x80); !got.Equal(&want) {
		t.Errorf("SetBit(1) = %x (len %d), want %x", got.bytes, got.len, want.bytes)
	}
	if got, want := new(BitArray).SetBit(0), ba(1, 0x00); !got.Equal(&want) {
		t.Errorf("SetBit(0) = %x (len %d), want %x", got.bytes, got.len, want.bytes)
	}
}

func TestEqual(t *testing.T) {
	a := NewBitArray(3, 0b101)
	b := NewBitArray(3, 0b101)
	if !a.Equal(&b) {
		t.Error("equal arrays reported unequal")
	}
	// Same active bytes, different length must be unequal.
	c := NewBitArray(2, 0b10) // "10" -> byte 0x80, len 2
	d := ba(3, c.bytes[0])    // same byte, len 3
	if c.Equal(&d) {
		t.Error("arrays with different length reported equal")
	}
}

func TestKeyBytesRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		length uint8
		data   []byte
		want   []byte // expected KeyBytes output
	}{
		{"empty", 0, nil, nil},
		{"one bit", 1, []byte{0x80}, []byte{0x80, 1}},
		{"full byte", 8, []byte{0x80}, []byte{0x80, 8}},
		{"eleven bits", 11, []byte{0xFF, 0xFF}, []byte{0xFF, 0xE0, 11}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := new(BitArray).SetBytes(tt.length, tt.data)
			key := src.KeyBytes()
			if !bytes.Equal(key, tt.want) {
				t.Fatalf("KeyBytes() = %x, want %x", key, tt.want)
			}

			// PutKeyBytes must agree with KeyBytes.
			var buf [33]byte
			if put := src.PutKeyBytes(buf[:]); !bytes.Equal(put, tt.want) {
				t.Fatalf("PutKeyBytes() = %x, want %x", put, tt.want)
			}

			// Re-parse the active bytes and confirm the path round-trips.
			if tt.length == 0 {
				return
			}
			lengthByte := key[len(key)-1]
			reparsed := new(BitArray).SetBytes(lengthByte, key[:len(key)-1])
			if !reparsed.Equal(src) {
				t.Fatalf("round-trip mismatch: %x (len %d) != %x (len %d)",
					reparsed.bytes, reparsed.len, src.bytes, src.len)
			}
		})
	}
}

func TestCopyIsIndependent(t *testing.T) {
	src := new(BitArray).SetBytes(8, []byte{0xAB})
	cp := src.Copy()
	cp.AppendBit(&cp, 1)
	if want := ba(8, 0xAB); !src.Equal(&want) {
		t.Errorf("Copy not independent: source became %x (len %d)", src.bytes, src.len)
	}
}
