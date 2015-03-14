package ethutil

import (
	"bytes"
	"math/big"
	"os"
	"testing"

	checker "gopkg.in/check.v1"
)

type CommonSuite struct{}

var _ = checker.Suite(&CommonSuite{})

func (s *CommonSuite) TestOS(c *checker.C) {
	expwin := (os.PathSeparator == '\\' && os.PathListSeparator == ';')
	res := IsWindows()

	if !expwin {
		c.Assert(res, checker.Equals, expwin, checker.Commentf("IsWindows is", res, "but path is", os.PathSeparator))
	} else {
		c.Assert(res, checker.Not(checker.Equals), expwin, checker.Commentf("IsWindows is", res, "but path is", os.PathSeparator))
	}
}

func (s *CommonSuite) TestWindonziePath(c *checker.C) {
	iswindowspath := os.PathSeparator == '\\'
	path := "/opt/eth/test/file.ext"
	res := WindonizePath(path)
	ressep := string(res[0])

	if !iswindowspath {
		c.Assert(ressep, checker.Equals, "/")
	} else {
		c.Assert(ressep, checker.Not(checker.Equals), "/")
	}
}

func (s *CommonSuite) TestCommon(c *checker.C) {
	douglas := CurrencyToString(BigPow(10, 43))
	einstein := CurrencyToString(BigPow(10, 22))
	ether := CurrencyToString(BigPow(10, 19))
	finney := CurrencyToString(BigPow(10, 16))
	szabo := CurrencyToString(BigPow(10, 13))
	shannon := CurrencyToString(BigPow(10, 10))
	babbage := CurrencyToString(BigPow(10, 7))
	ada := CurrencyToString(BigPow(10, 4))
	wei := CurrencyToString(big.NewInt(10))

	c.Assert(douglas, checker.Equals, "10 Douglas")
	c.Assert(einstein, checker.Equals, "10 Einstein")
	c.Assert(ether, checker.Equals, "10 Ether")
	c.Assert(finney, checker.Equals, "10 Finney")
	c.Assert(szabo, checker.Equals, "10 Szabo")
	c.Assert(shannon, checker.Equals, "10 Shannon")
	c.Assert(babbage, checker.Equals, "10 Babbage")
	c.Assert(ada, checker.Equals, "10 Ada")
	c.Assert(wei, checker.Equals, "10 Wei")
}

func (s *CommonSuite) TestLarge(c *checker.C) {
	douglaslarge := CurrencyToString(BigPow(100000000, 43))
	adalarge := CurrencyToString(BigPow(100000000, 4))
	weilarge := CurrencyToString(big.NewInt(100000000))

	c.Assert(douglaslarge, checker.Equals, "10000E298 Douglas")
	c.Assert(adalarge, checker.Equals, "10000E7 Einstein")
	c.Assert(weilarge, checker.Equals, "100 Babbage")
}

//fromHex
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
