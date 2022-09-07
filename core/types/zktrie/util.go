package zktrie

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/crypto/poseidon"
)

// HashElems performs a recursive poseidon hash over the array of ElemBytes, each hash
// reduce 2 fieds into one
func HashElems(fst, snd *big.Int, elems ...*big.Int) (*Hash, error) {

	l := len(elems)
	baseH, err := poseidon.HashFixed([]*big.Int{fst, snd})
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return NewHashFromBigInt(baseH), nil
	} else if l == 1 {
		return HashElems(baseH, elems[0])
	}

	tmp := make([]*big.Int, l/2)
	for i := range tmp {
		if (i+1)*2 > l {
			tmp[i] = elems[i*2+1]
		} else {
			h, err := poseidon.HashFixed(elems[i*2 : (i+1)*2])
			if err != nil {
				return nil, err
			}
			tmp[i] = h
		}
	}

	return HashElems(baseH, tmp[0], tmp[1:]...)
}

// PreHandlingElems turn persisted byte32 elements into field arrays for our hashElem
// it also has the compressed byte32
func PreHandlingElems(flagArray uint32, elems []Byte32) (*Hash, error) {

	ret := make([]*big.Int, len(elems))
	var err error

	for i, elem := range elems {
		if flagArray&(1<<i) != 0 {
			ret[i], err = elem.Hash()
			if err != nil {
				return nil, err
			}
		} else {
			ret[i] = new(big.Int).SetBytes(elem[:])
		}
	}

	if len(ret) < 2 {
		return NewHashFromBigInt(ret[0]), nil
	}

	return HashElems(ret[0], ret[1], ret[2:]...)

}

// SetBitBigEndian sets the bit n in the bitmap to 1, in Big Endian.
func SetBitBigEndian(bitmap []byte, n uint) {
	bitmap[uint(len(bitmap))-n/8-1] |= 1 << (n % 8)
}

// TestBit tests whether the bit n in bitmap is 1.
func TestBit(bitmap []byte, n uint) bool {
	return bitmap[n/8]&(1<<(n%8)) != 0
}

// TestBitBigEndian tests whether the bit n in bitmap is 1, in Big Endian.
func TestBitBigEndian(bitmap []byte, n uint) bool {
	return bitmap[uint(len(bitmap))-n/8-1]&(1<<(n%8)) != 0
}

var BigOne = big.NewInt(1)
var BigZero = big.NewInt(0)

func BigEndianBitsToBigInt(bits []bool) *big.Int {
	result := big.NewInt(0)
	for _, bit := range bits {
		result.Mul(result, big.NewInt(2))
		if bit {
			result.Add(result, BigOne)
		}
	}
	return result
}
