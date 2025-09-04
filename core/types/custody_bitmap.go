package types

import (
	"errors"
	"math/bits"
)

// `CustodyBitmap` is a bitmap to represent which custody index to store (little endian)
type CustodyBitmap [16]byte

const CustodySize = 128

func (b CustodyBitmap) IsSet(i uint) bool {
	if i >= CustodySize {
		return false
	}
	index := i / 8
	offset := i % 8
	return ((b[index] >> offset) & 1) == 1
}

// Set ith bit
func (b *CustodyBitmap) Set(i uint) error {
	if i >= CustodySize {
		return errors.New("bit index out of range")
	}
	index := i / 8
	offset := i % 8
	b[index] |= 1 << offset
	return nil
}

// Clear ith bit
func (b *CustodyBitmap) Clear(i uint) error {
	if i >= CustodySize {
		return errors.New("bit index out of range")
	}
	index := i / 8
	offset := i % 8
	b[index] &^= 1 << offset
	return nil
}

// Number of bits set to 1
func (b CustodyBitmap) OneCount() int {
	total := 0
	for _, byte := range b {
		total += bits.OnesCount8(uint8(byte))
	}
	return total
}

// Return bit indices set to 1, ascending order
func (b CustodyBitmap) Indices() []uint64 {
	out := make([]uint64, 0, b.OneCount())
	for byteIdx, val := range b {
		v := val
		for v != 0 {
			tz := bits.TrailingZeros8(uint8(v)) // 0..7
			idx := uint64(byteIdx*8 + tz)
			out = append(out, idx)
			v &^= 1 << tz
		}
	}
	return out
}

func (b CustodyBitmap) SetIndices(indices []uint64) error {
	for _, i := range indices {
		if i >= CustodySize {
			return errors.New("bit index out of range")
		}
		byteIdx := i / 8
		bitOff := i % 8
		b[byteIdx] |= 1 << bitOff
	}
	return nil
}

func (b CustodyBitmap) SetAll() CustodyBitmap {
	for i := 0; i < len(b); i++ {
		b[i] = 0xFF
	}
	return b
}
