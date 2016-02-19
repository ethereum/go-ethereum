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

import (
	"fmt"
	"math/big"
)

type StorageSize float64

func (self StorageSize) String() string {
	if self > 1000000 {
		return fmt.Sprintf("%.2f mB", self/1000000)
	} else if self > 1000 {
		return fmt.Sprintf("%.2f kB", self/1000)
	} else {
		return fmt.Sprintf("%.2f B", self)
	}
}

func (self StorageSize) Int64() int64 {
	return int64(self)
}

// The different number of units
var (
	Xchy  = BigPow(10, 42)
	Zchy  = BigPow(10, 21)
	Echy  = BigPow(10, 18)
	Pchy  = BigPow(10, 15)
	Tchy  = BigPow(10, 12)
	Gchy  = BigPow(10, 9)
	Mchy  = BigPow(10, 6)
	Kchy  = BigPow(10, 3)
	Chy   = big.NewInt(1)
)

//
// Currency to string
// Returns a string representing a human readable format
func CurrencyToString(num *big.Int) string {
	var (
		fin   *big.Int = num
		denom string   = "Chy"
	)

	switch {
	case num.Cmp(Echy) >= 0:
		fin = new(big.Int).Div(num, Echy)
		denom = "Echy"
	case num.Cmp(Pchy) >= 0:
		fin = new(big.Int).Div(num, Pchy)
		denom = "Pchy"
	case num.Cmp(Tchy) >= 0:
		fin = new(big.Int).Div(num, Tchy)
		denom = "Tchy"
	case num.Cmp(Gchy) >= 0:
		fin = new(big.Int).Div(num, Gchy)
		denom = "Gchy"
	case num.Cmp(Mchy) >= 0:
		fin = new(big.Int).Div(num, Mchy)
		denom = "Mchy"
	case num.Cmp(Kchy) >= 0:
		fin = new(big.Int).Div(num, Kchy)
		denom = "Kchy"
	}

	// TODO add comment clarifying expected behavior
	if len(fin.String()) > 5 {
		return fmt.Sprintf("%sE%d %s", fin.String()[0:5], len(fin.String())-5, denom)
	}

	return fmt.Sprintf("%v %s", fin, denom)
}
