package trie

import (
	checker "gopkg.in/check.v1"
)

type TrieEncodingSuite struct{}

var _ = checker.Suite(&TrieEncodingSuite{})

func (s *TrieEncodingSuite) TestCompactEncode(c *checker.C) {
	// even compact encode
	test1 := []byte{1, 2, 3, 4, 5}
	res1 := CompactEncode(test1)
	c.Assert(res1, checker.Equals, "\x11\x23\x45")

	// odd compact encode
	test2 := []byte{0, 1, 2, 3, 4, 5}
	res2 := CompactEncode(test2)
	c.Assert(res2, checker.Equals, "\x00\x01\x23\x45")

	//odd terminated compact encode
	test3 := []byte{0, 15, 1, 12, 11, 8 /*term*/, 16}
	res3 := CompactEncode(test3)
	c.Assert(res3, checker.Equals, "\x20\x0f\x1c\xb8")

	// even terminated compact encode
	test4 := []byte{15, 1, 12, 11, 8 /*term*/, 16}
	res4 := CompactEncode(test4)
	c.Assert(res4, checker.Equals, "\x3f\x1c\xb8")
}

func (s *TrieEncodingSuite) TestCompactHexDecode(c *checker.C) {
	exp := []byte{7, 6, 6, 5, 7, 2, 6, 2, 16}
	res := CompactHexDecode("verb")
	c.Assert(res, checker.DeepEquals, exp)
}

func (s *TrieEncodingSuite) TestCompactDecode(c *checker.C) {
	// odd compact decode
	exp := []byte{1, 2, 3, 4, 5}
	res := CompactDecode("\x11\x23\x45")
	c.Assert(res, checker.DeepEquals, exp)

	// even compact decode
	exp = []byte{0, 1, 2, 3, 4, 5}
	res = CompactDecode("\x00\x01\x23\x45")
	c.Assert(res, checker.DeepEquals, exp)

	// even terminated compact decode
	exp = []byte{0, 15, 1, 12, 11, 8 /*term*/, 16}
	res = CompactDecode("\x20\x0f\x1c\xb8")
	c.Assert(res, checker.DeepEquals, exp)

	// even terminated compact decode
	exp = []byte{15, 1, 12, 11, 8 /*term*/, 16}
	res = CompactDecode("\x3f\x1c\xb8")
	c.Assert(res, checker.DeepEquals, exp)
}
