// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package number

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var tt256 = new(big.Int).Lsh(big.NewInt(1), 256)
var tt256m1 = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
var tt255 = new(big.Int).Lsh(big.NewInt(1), 255)

func limitUnsigned256(x *Number) *Number {
	x.num.And(x.num, tt256m1)
	return x
}

func limitSigned256(x *Number) *Number {
	if x.num.Cmp(tt255) < 0 {
		return x
	} else {
		x.num.Sub(x.num, tt256)
		return x
	}
}

// Number function
type Initialiser func(n int64) *Number

// A Number represents a generic integer with a bounding function limiter. Limit is called after each operations
// to give "fake" bounded integers. New types of Number can be created through NewInitialiser returning a lambda
// with the new Initialiser.
type Number struct {
	num   *big.Int
	limit func(n *Number) *Number
}

// Returns a new initialiser for a new *Number without having to expose certain fields
func NewInitialiser(limiter func(*Number) *Number) Initialiser {
	return func(n int64) *Number {
		return &Number{big.NewInt(n), limiter}
	}
}

// Return a Number with a UNSIGNED limiter up to 256 bits
func Uint256(n int64) *Number {
	return &Number{big.NewInt(n), limitUnsigned256}
}

// Return a Number with a SIGNED limiter up to 256 bits
func Int256(n int64) *Number {
	return &Number{big.NewInt(n), limitSigned256}
}

// Returns a Number with a SIGNED unlimited size
func Big(n int64) *Number {
	return &Number{big.NewInt(n), func(x *Number) *Number { return x }}
}

// Sets i to sum of x+y
func (i *Number) Add(x, y *Number) *Number {
	i.num.Add(x.num, y.num)
	return i.limit(i)
}

// Sets i to difference of x-y
func (i *Number) Sub(x, y *Number) *Number {
	i.num.Sub(x.num, y.num)
	return i.limit(i)
}

// Sets i to product of x*y
func (i *Number) Mul(x, y *Number) *Number {
	i.num.Mul(x.num, y.num)
	return i.limit(i)
}

// Sets i to the quotient prodject of x/y
func (i *Number) Div(x, y *Number) *Number {
	i.num.Div(x.num, y.num)
	return i.limit(i)
}

// Sets i to x % y
func (i *Number) Mod(x, y *Number) *Number {
	i.num.Mod(x.num, y.num)
	return i.limit(i)
}

// Sets i to x << s
func (i *Number) Lsh(x *Number, s uint) *Number {
	i.num.Lsh(x.num, s)
	return i.limit(i)
}

// Sets i to x^y
func (i *Number) Pow(x, y *Number) *Number {
	i.num.Exp(x.num, y.num, big.NewInt(0))
	return i.limit(i)
}

// Setters

// Set x to i
func (i *Number) Set(x *Number) *Number {
	i.num.Set(x.num)
	return i.limit(i)
}

// Set x bytes to i
func (i *Number) SetBytes(x []byte) *Number {
	i.num.SetBytes(x)
	return i.limit(i)
}

// Cmp compares x and y and returns:
//
//     -1 if x <  y
//     0 if x == y
//     +1 if x >  y
func (i *Number) Cmp(x *Number) int {
	return i.num.Cmp(x.num)
}

// Getters

// Returns the string representation of i
func (i *Number) String() string {
	return i.num.String()
}

// Returns the byte representation of i
func (i *Number) Bytes() []byte {
	return i.num.Bytes()
}

// Uint64 returns the Uint64 representation of x. If x cannot be represented in an int64, the result is undefined.
func (i *Number) Uint64() uint64 {
	return i.num.Uint64()
}

// Int64 returns the int64 representation of x. If x cannot be represented in an int64, the result is undefined.
func (i *Number) Int64() int64 {
	return i.num.Int64()
}

// Returns the signed version of i
func (i *Number) Int256() *Number {
	return Int(0).Set(i)
}

// Returns the unsigned version of i
func (i *Number) Uint256() *Number {
	return Uint(0).Set(i)
}

// Returns the index of the first bit that's set to 1
func (i *Number) FirstBitSet() int {
	for j := 0; j < i.num.BitLen(); j++ {
		if i.num.Bit(j) > 0 {
			return j
		}
	}

	return i.num.BitLen()
}

// Variables

var (
	Zero       = Uint(0)
	One        = Uint(1)
	Two        = Uint(2)
	MaxUint256 = Uint(0).SetBytes(common.Hex2Bytes("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"))

	MinOne = Int(-1)

	// "typedefs"
	Uint = Uint256
	Int  = Int256
)
