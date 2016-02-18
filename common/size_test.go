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
	"math/big"

	checker "gopkg.in/check.v1"
)

type SizeSuite struct{}

var _ = checker.Suite(&SizeSuite{})

func (s *SizeSuite) TestStorageSizeString(c *checker.C) {
	data1 := 2381273
	data2 := 2192
	data3 := 12

	exp1 := "2.38 mB"
	exp2 := "2.19 kB"
	exp3 := "12.00 B"

	c.Assert(StorageSize(data1).String(), checker.Equals, exp1)
	c.Assert(StorageSize(data2).String(), checker.Equals, exp2)
	c.Assert(StorageSize(data3).String(), checker.Equals, exp3)
}

func (s *SizeSuite) TestCommon(c *checker.C) {
	Echy := CurrencyToString(BigPow(10, 19))
	Pchy := CurrencyToString(BigPow(10, 16))
	Tchy := CurrencyToString(BigPow(10, 13))
	Gchy := CurrencyToString(BigPow(10, 10))
	Mchy := CurrencyToString(BigPow(10, 7))
	Kchy := CurrencyToString(BigPow(10, 4))
	Chy := CurrencyToString(big.NewInt(10))

	c.Assert(Echy, checker.Equals, "10 Echy")
	c.Assert(Pchy, checker.Equals, "10 Pchy")
	c.Assert(Tchy, checker.Equals, "10 Tchy")
	c.Assert(Gchy, checker.Equals, "10 Gchy")
	c.Assert(Mchy, checker.Equals, "10 Mchy")
	c.Assert(Kchy, checker.Equals, "10 Kchy")
	c.Assert(Chy, checker.Equals, "10 Chy")
}
