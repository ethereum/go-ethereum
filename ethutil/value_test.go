package ethutil

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
	c.Assert(NewValue(inter.Interface()).Cmp(NewValue(interExp)), checker.Equals, true)
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

func (s *ValueSuite) TestAdversary(c *checker.C) {
	badRlp := []string{
		"10fa5",
		"10fa56a89200b3c",
		"f971808813c16d60fa56a89200b3c8f3421aaa3626c711a1bed1c",
		"f971808813c16d67555ec800881bc16d674ec8000094bbbd0256041f7aed3ce278c56ee61492de96d0018401312d008061a06162636465666768696a6b6c6d6e6f707172737475767778797a616263646566a06162636465666768696a6b6c6d6e6f707172737475767778797a6162636465",
		"f971808813c16d67555ec800881bc16d674ec8110094bbbd3256041f7aed3ce278c59ee62492de06d001832401312d008061a06162636465666768696a6b6c6d6e6f7071727475ab767778797a616263646566a06162636465666768696a6b6c6d6e6f7071",
	}
	// simply looking for input that panics
	for _, v := range badRlp {
		NewValueFromBytes(Hex2Bytes(v))
	}
}
