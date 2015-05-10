package common

import (
	"bytes"
	"testing"

	checker "gopkg.in/check.v1"
)

type BytesSuite struct{}

var _ = checker.Suite(&BytesSuite{})

func (s *BytesSuite) TestByteString(c *checker.C) {
	var data Bytes
	data = []byte{102, 111, 111}
	exp := "foo"
	res := data.String()

	c.Assert(res, checker.Equals, exp)
}

/*
func (s *BytesSuite) TestDeleteFromByteSlice(c *checker.C) {
	data := []byte{1, 2, 3, 4}
	slice := []byte{1, 2, 3, 4}
	exp := []byte{1, 4}
	res := DeleteFromByteSlice(data, slice)

	c.Assert(res, checker.DeepEquals, exp)
}

*/
func (s *BytesSuite) TestNumberToBytes(c *checker.C) {
	// data1 := int(1)
	// res1 := NumberToBytes(data1, 16)
	// c.Check(res1, checker.Panics)

	var data2 float64 = 3.141592653
	exp2 := []byte{0xe9, 0x38}
	res2 := NumberToBytes(data2, 16)
	c.Assert(res2, checker.DeepEquals, exp2)
}

func (s *BytesSuite) TestBytesToNumber(c *checker.C) {
	datasmall := []byte{0xe9, 0x38, 0xe9, 0x38}
	datalarge := []byte{0xe9, 0x38, 0xe9, 0x38, 0xe9, 0x38, 0xe9, 0x38}

	var expsmall uint64 = 0xe938e938
	var explarge uint64 = 0x0

	ressmall := BytesToNumber(datasmall)
	reslarge := BytesToNumber(datalarge)

	c.Assert(ressmall, checker.Equals, expsmall)
	c.Assert(reslarge, checker.Equals, explarge)

}

func (s *BytesSuite) TestReadVarInt(c *checker.C) {
	data8 := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	data4 := []byte{1, 2, 3, 4}
	data2 := []byte{1, 2}
	data1 := []byte{1}

	exp8 := uint64(72623859790382856)
	exp4 := uint64(16909060)
	exp2 := uint64(258)
	exp1 := uint64(1)

	res8 := ReadVarInt(data8)
	res4 := ReadVarInt(data4)
	res2 := ReadVarInt(data2)
	res1 := ReadVarInt(data1)

	c.Assert(res8, checker.Equals, exp8)
	c.Assert(res4, checker.Equals, exp4)
	c.Assert(res2, checker.Equals, exp2)
	c.Assert(res1, checker.Equals, exp1)
}

func (s *BytesSuite) TestCopyBytes(c *checker.C) {
	data1 := []byte{1, 2, 3, 4}
	exp1 := []byte{1, 2, 3, 4}
	res1 := CopyBytes(data1)
	c.Assert(res1, checker.DeepEquals, exp1)
}

func (s *BytesSuite) TestIsHex(c *checker.C) {
	data1 := "a9e67e"
	exp1 := false
	res1 := IsHex(data1)
	c.Assert(res1, checker.DeepEquals, exp1)

	data2 := "0xa9e67e00"
	exp2 := true
	res2 := IsHex(data2)
	c.Assert(res2, checker.DeepEquals, exp2)

}

func (s *BytesSuite) TestParseDataString(c *checker.C) {
	res1 := ParseData("hello", "world", "0x0106")
	data := "68656c6c6f000000000000000000000000000000000000000000000000000000776f726c640000000000000000000000000000000000000000000000000000000106000000000000000000000000000000000000000000000000000000000000"
	exp1 := Hex2Bytes(data)
	c.Assert(res1, checker.DeepEquals, exp1)
}

func (s *BytesSuite) TestParseDataBytes(c *checker.C) {
	data1 := []byte{232, 212, 165, 16, 0}
	exp1 := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 232, 212, 165, 16, 0}

	res1 := ParseData(data1)
	c.Assert(res1, checker.DeepEquals, exp1)

}

func (s *BytesSuite) TestLeftPadBytes(c *checker.C) {
	val1 := []byte{1, 2, 3, 4}
	exp1 := []byte{0, 0, 0, 0, 1, 2, 3, 4}

	res1 := LeftPadBytes(val1, 8)
	res2 := LeftPadBytes(val1, 2)

	c.Assert(res1, checker.DeepEquals, exp1)
	c.Assert(res2, checker.DeepEquals, val1)
}

func (s *BytesSuite) TestFormatData(c *checker.C) {
	data1 := ""
	data2 := "0xa9e67e00"
	data3 := "a9e67e"
	data4 := "\"a9e67e00\""

	// exp1 := []byte{}
	exp2 := []byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0xa9, 0xe6, 0x7e, 00}
	exp3 := []byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00}
	exp4 := []byte{0x61, 0x39, 0x65, 0x36, 0x37, 0x65, 0x30, 0x30, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00}

	res1 := FormatData(data1)
	res2 := FormatData(data2)
	res3 := FormatData(data3)
	res4 := FormatData(data4)

	c.Assert(res1, checker.IsNil)
	c.Assert(res2, checker.DeepEquals, exp2)
	c.Assert(res3, checker.DeepEquals, exp3)
	c.Assert(res4, checker.DeepEquals, exp4)
}

func (s *BytesSuite) TestRightPadBytes(c *checker.C) {
	val := []byte{1, 2, 3, 4}
	exp := []byte{1, 2, 3, 4, 0, 0, 0, 0}

	resstd := RightPadBytes(val, 8)
	resshrt := RightPadBytes(val, 2)

	c.Assert(resstd, checker.DeepEquals, exp)
	c.Assert(resshrt, checker.DeepEquals, val)
}

func (s *BytesSuite) TestLeftPadString(c *checker.C) {
	val := "test"
	exp := "\x30\x30\x30\x30" + val

	resstd := LeftPadString(val, 8)
	resshrt := LeftPadString(val, 2)

	c.Assert(resstd, checker.Equals, exp)
	c.Assert(resshrt, checker.Equals, val)
}

func (s *BytesSuite) TestRightPadString(c *checker.C) {
	val := "test"
	exp := val + "\x30\x30\x30\x30"

	resstd := RightPadString(val, 8)
	resshrt := RightPadString(val, 2)

	c.Assert(resstd, checker.Equals, exp)
	c.Assert(resshrt, checker.Equals, val)
}

func TestFromHex(t *testing.T) {
	input := "0x01"
	expected := []byte{1}
	result := FromHex(input)
	if bytes.Compare(expected, result) != 0 {
		t.Errorf("Expected % x got % x", expected, result)
	}
}

func TestFromHexOddLength(t *testing.T) {
	input := "0x1"
	expected := []byte{1}
	result := FromHex(input)
	if bytes.Compare(expected, result) != 0 {
		t.Errorf("Expected % x got % x", expected, result)
	}
}
