package types

import (
	"math/bits"

	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// `CustodyBitmap` is a bitmap to represent which custody index to store (little endian)
type CustodyBitmap [16]byte

var (
	CustodyBitmapAll = func() *CustodyBitmap {
		var result CustodyBitmap
		for i := 0; i < len(result); i++ {
			result[i] = 0xFF
		}
		return &result
	}()

	CustodyBitmapData = func() *CustodyBitmap {
		var result CustodyBitmap
		for i := 0; i < kzg4844.DataPerBlob/8; i++ {
			result[i] = 0xFF
		}
		return &result
	}()
)

// Return bit indices set to 1, ascending order
func (b CustodyBitmap) Indices() []uint64 {
	out := make([]uint64, 0, b.OneCount())
	for byteIdx, val := range b {
		v := val
		for v != 0 {
			tz := bits.TrailingZeros8(v)
			idx := uint64(byteIdx*8 + tz)
			out = append(out, idx)
			v &^= 1 << tz
		}
	}
	return out
}

// Number of bits set to 1
func (b CustodyBitmap) OneCount() int {
	total := 0
	for _, data := range b {
		total += bits.OnesCount8(data)
	}
	return total
}
