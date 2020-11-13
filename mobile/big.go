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

// Contains all the wrappers from the math/big package.

package geth

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// A BigInt represents a signed multi-precision integer.
type BigInt struct {
	bigint *big.Int
}

// NewBigInt allocates and returns a new BigInt set to x.
func NewBigInt(x int64) *BigInt {
	return &BigInt{big.NewInt(x)}
}

// NewBigIntFromString allocates and returns a new BigInt set to x
// interpreted in the provided base.
func NewBigIntFromString(x string, base int) *BigInt {
	b, success := new(big.Int).SetString(x, base)
	if !success {
		return nil
	}
	return &BigInt{b}
}

// GetBytes returns the absolute value of x as a big-endian byte slice.
func (bi *BigInt) GetBytes() []byte {
	return bi.bigint.Bytes()
}

// String returns the value of x as a formatted decimal string.
func (bi *BigInt) String() string {
	return bi.bigint.String()
}

// GetInt64 returns the int64 representation of x. If x cannot be represented in
// an int64, the result is undefined.
func (bi *BigInt) GetInt64() int64 {
	return bi.bigint.Int64()
}

// SetBytes interprets buf as the bytes of a big-endian unsigned integer and sets
// the big int to that value.
func (bi *BigInt) SetBytes(buf []byte) {
	bi.bigint.SetBytes(common.CopyBytes(buf))
}

// SetInt64 sets the big int to x.
func (bi *BigInt) SetInt64(x int64) {
	bi.bigint.SetInt64(x)
}

// Sign returns:
//
//	-1 if x <  0
//	 0 if x == 0
//	+1 if x >  0
//
func (bi *BigInt) Sign() int {
	return bi.bigint.Sign()
}

// SetString sets the big int to x.
//
// The string prefix determines the actual conversion base. A prefix of "0x" or
// "0X" selects base 16; the "0" prefix selects base 8, and a "0b" or "0B" prefix
// selects base 2. Otherwise the selected base is 10.
func (bi *BigInt) SetString(x string, base int) {
	bi.bigint.SetString(x, base)
}

// BigInts represents a slice of big ints.
type BigInts struct{ bigints []*big.Int }

// NewBigInts creates a slice of uninitialized big numbers.
func NewBigInts(size int) *BigInts {
	return &BigInts{
		bigints: make([]*big.Int, size),
	}
}

// Size returns the number of big ints in the slice.
func (bi *BigInts) Size() int {
	return len(bi.bigints)
}

// Get returns the bigint at the given index from the slice.
func (bi *BigInts) Get(index int) (bigint *BigInt, _ error) {
	if index < 0 || index >= len(bi.bigints) {
		return nil, errors.New("index out of bounds")
	}
	return &BigInt{bi.bigints[index]}, nil
}

// Set sets the big int at the given index in the slice.
func (bi *BigInts) Set(index int, bigint *BigInt) error {
	if index < 0 || index >= len(bi.bigints) {
		return errors.New("index out of bounds")
	}
	bi.bigints[index] = bigint.bigint
	return nil
}

// GetString returns the value of x as a formatted string in some number base.
func (bi *BigInt) GetString(base int) string {
	return bi.bigint.Text(base)
}
