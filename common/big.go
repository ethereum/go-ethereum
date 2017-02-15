// Copyright 2014 The go-ethereum Authors
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

package common

import "math/big"

// Common big integers often used
var (
	Big1     = big.NewInt(1)
	Big2     = big.NewInt(2)
	Big3     = big.NewInt(3)
	Big0     = big.NewInt(0)
	BigTrue  = Big1
	BigFalse = Big0
	Big32    = big.NewInt(32)
	Big36    = big.NewInt(36)
	Big97    = big.NewInt(97)
	Big98    = big.NewInt(98)
	Big256   = big.NewInt(0xff)
	Big257   = big.NewInt(257)
	MaxBig   = String2Big("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
)

// Big pow
//
// Returns the power of two big integers
func BigPow(a, b int) *big.Int {
	c := new(big.Int)
	c.Exp(big.NewInt(int64(a)), big.NewInt(int64(b)), big.NewInt(0))

	return c
}

// Big
//
// Shortcut for new(big.Int).SetString(..., 0)
func Big(num string) *big.Int {
	n := new(big.Int)
	n.SetString(num, 0)

	return n
}

// Bytes2Big
//
func BytesToBig(data []byte) *big.Int {
	n := new(big.Int)
	n.SetBytes(data)

	return n
}
func Bytes2Big(data []byte) *big.Int { return BytesToBig(data) }
func BigD(data []byte) *big.Int      { return BytesToBig(data) }

func String2Big(num string) *big.Int {
	n := new(big.Int)
	n.SetString(num, 0)
	return n
}

func BitTest(num *big.Int, i int) bool {
	return num.Bit(i) > 0
}

// To256
//
// "cast" the big int to a 256 big int (i.e., limit to)
var tt256 = new(big.Int).Lsh(big.NewInt(1), 256)
var tt256m1 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
var tt255 = new(big.Int).Lsh(big.NewInt(1), 255)

func U256(x *big.Int) *big.Int {
	//if x.Cmp(Big0) < 0 {
	//		return new(big.Int).Add(tt256, x)
	//	}

	x.And(x, tt256m1)

	return x
}

func S256(x *big.Int) *big.Int {
	if x.Cmp(tt255) < 0 {
		return x
	} else {
		// We don't want to modify x, ever
		return new(big.Int).Sub(x, tt256)
	}
}

func FirstBitSet(v *big.Int) int {
	for i := 0; i < v.BitLen(); i++ {
		if v.Bit(i) > 0 {
			return i
		}
	}

	return v.BitLen()
}

// Big to bytes
//
// Returns the bytes of a big integer with the size specified by **base**
// Attempts to pad the byte array with zeros.
func BigToBytes(num *big.Int, base int) []byte {
	ret := make([]byte, base/8)

	if len(num.Bytes()) > base/8 {
		return num.Bytes()
	}

	return append(ret[:len(ret)-len(num.Bytes())], num.Bytes()...)
}

// Big copy
//
// Creates a copy of the given big integer
func BigCopy(src *big.Int) *big.Int {
	return new(big.Int).Set(src)
}

// Big max
//
// Returns the maximum size big integer
func BigMax(x, y *big.Int) *big.Int {
	if x.Cmp(y) < 0 {
		return y
	}

	return x
}

// Big min
//
// Returns the minimum size big integer
func BigMin(x, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return y
	}

	return x
}
