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

type ValueSuite struct{}

var _ = checker.Suite(&ValueSuite{})

func (s *ValueSuite) TestValueCmp(c *checker.C) {
	val1 := NewValue("hello")
	val2 := NewValue("world")
	c.Assert(val1.Cmp(val2), checker.Equals, false)

	val3 := NewValue("hello")
	val4 := NewValue("hello")
	c.Assert(val3.Cmp(val4), checker.Equals, true)
}

func (s *ValueSuite) TestValueTypes(c *checker.C) {
	str := NewValue("str")
	num := NewValue(1)
	inter := NewValue([]interface{}{1})
	byt := NewValue([]byte{1, 2, 3, 4})
	bigInt := NewValue(big.NewInt(10))

	strExp := "str"
	numExp := uint64(1)
	interExp := []interface{}{1}
	bytExp := []byte{1, 2, 3, 4}
	bigExp := big.NewInt(10)

	c.Assert(str.Str(), checker.Equals, strExp)
	c.Assert(num.Uint(), checker.Equals, numExp)
	c.Assert(NewValue(inter.Val).Cmp(NewValue(interExp)), checker.Equals, true)
	c.Assert(byt.Bytes(), checker.DeepEquals, bytExp)
	c.Assert(bigInt.BigInt(), checker.DeepEquals, bigExp)
}

func (s *ValueSuite) TestIterator(c *checker.C) {
	value := NewValue([]interface{}{1, 2, 3})
	iter := value.NewIterator()
	values := []uint64{1, 2, 3}
	i := 0
	for iter.Next() {
		c.Assert(values[i], checker.Equals, iter.Value().Uint())
		i++
	}
}

func (s *ValueSuite) TestMath(c *checker.C) {
	data1 := NewValue(1)
	data1.Add(1).Add(1)
	exp1 := NewValue(3)
	data2 := NewValue(2)
	data2.Sub(1).Sub(1)
	exp2 := NewValue(0)

	c.Assert(data1.DeepCmp(exp1), checker.Equals, true)
	c.Assert(data2.DeepCmp(exp2), checker.Equals, true)
}

func (s *ValueSuite) TestString(c *checker.C) {
	data := "10"
	exp := int64(10)
	c.Assert(NewValue(data).Int(), checker.DeepEquals, exp)
}
