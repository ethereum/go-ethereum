// Copyright 2020 The go-ethereum Authors
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

package les

import "math"

// expiredValue is a scalar value that is continuously expired (decreased
// exponentially) based on the provided logarithmic expiration offset value.
//
// The formula for value calculation is: base*2^(exp-logOffset). In order to
// simplify the calculation of expiredValue, its value is expressed in the form
// of an exponent with a base of 2.
//
// Also here is a trick to reduce a lot of calculations. In theory, when a value X
// decays over time and then a new value Y is added, the final result should be
// X*2^(exp-logOffset)+Y. However it's very hard to represent in memory.
// So the trick is using the idea of inflation instead of exponential decay. At this
// moment the temporary value becomes: X*2^exp+Y*2^logOffset_1, apply the exponential
// decay when we actually want to calculate the value.
//
// e.g.
// t0: V = 100
// t1: add 30, inflationary value is: 100 + 30/0.3, 0.3 is the decay coefficient
// t2: get value, decay coefficient is 0.2 now, final result is: 200*0.2 = 40
type expiredValue struct {
	base, exp uint64
}

// value calculates the value at the given moment.
func (e expiredValue) value(logOffset fixed64) uint64 {
	offset := uint64ToFixed64(e.exp) - logOffset
	return uint64(float64(e.base) * offset.pow2())
}

// add adds a signed value at the given moment
func (e *expiredValue) add(amount int64, logOffset fixed64) int64 {
	integer, frac := logOffset.toUint64(), logOffset.fraction()
	factor := frac.pow2()
	base := factor * float64(amount)
	if integer < e.exp {
		base /= math.Pow(2, float64(e.exp-integer))
	}
	if integer > e.exp {
		e.base >>= (integer - e.exp)
		e.exp = integer
	}
	if base >= 0 || uint64(-base) <= e.base {
		e.base += uint64(base)
		return amount
	}
	net := int64(-float64(e.base) / factor)
	e.base = 0
	return net
}

// addExp adds another expiredValue
func (e *expiredValue) addExp(a expiredValue) {
	if e.exp > a.exp {
		a.base >>= (e.exp - a.exp)
	}
	if e.exp < a.exp {
		e.base >>= (a.exp - e.exp)
		e.exp = a.exp
	}
	e.base += a.base
}

// subExp subtracts another expiredValue
func (e *expiredValue) subExp(a expiredValue) {
	if e.exp > a.exp {
		a.base >>= (e.exp - a.exp)
	}
	if e.exp < a.exp {
		e.base >>= (a.exp - e.exp)
		e.exp = a.exp
	}
	if e.base > a.base {
		e.base -= a.base
	} else {
		e.base = 0
	}
}

// fixedFactor is the fixed point multiplier factor used by fixed64.
const fixedFactor = 0x1000000

// fixed64 implements 64-bit fixed point arithmetic functions.
type fixed64 int64

// uint64ToFixed64 converts uint64 integer to fixed64 format.
func uint64ToFixed64(f uint64) fixed64 {
	return fixed64(f * fixedFactor)
}

// float64ToFixed64 converts float64 to fixed64 format.
func float64ToFixed64(f float64) fixed64 {
	return fixed64(f * fixedFactor)
}

// toUint64 converts fixed64 format to uint64.
func (f64 fixed64) toUint64() uint64 {
	return uint64(f64) / fixedFactor
}

// fraction returns the fractional part of a fixed64 value.
func (f64 fixed64) fraction() fixed64 {
	return f64 % fixedFactor
}

var fixedLogFactor = math.Log(2) / float64(fixedFactor)

// pow2Fixed returns the base 2 power of the fixed point value.
func (f64 fixed64) pow2() float64 {
	return math.Exp(float64(f64) * fixedLogFactor)
}
