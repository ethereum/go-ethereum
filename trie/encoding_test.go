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

package trie

import (
	"encoding/hex"
	"testing"

	checker "gopkg.in/check.v1"
)

func TestEncoding(t *testing.T) { checker.TestingT(t) }

type TrieEncodingSuite struct{}

var _ = checker.Suite(&TrieEncodingSuite{})

func (s *TrieEncodingSuite) TestCompactEncode(c *checker.C) {
	for _, test := range []struct {
		hexSlice []byte
		exp      []byte
	}{
		// even compact encode
		{[]byte{1, 2, 3, 4, 5}, []byte("\x11\x23\x45")},
		// odd compact encode
		{[]byte{0, 1, 2, 3, 4, 5}, []byte("\x00\x01\x23\x45")},
		//odd terminated compact encode
		{[]byte{0, 15, 1, 12, 11, 8 /*term*/, 16}, []byte("\x20\x0f\x1c\xb8")},
		// even terminated compact encode
		{[]byte{15, 1, 12, 11, 8 /*term*/, 16}, []byte("\x3f\x1c\xb8")},
	} {
		c.Assert(compactEncode(test.hexSlice), checker.DeepEquals, test.exp)
	}
}

func (s *TrieEncodingSuite) TestCompactHexDecode(c *checker.C) {
	exp := []byte{7, 6, 6, 5, 7, 2, 6, 2, 16}
	res := compactHexDecode([]byte("verb"))
	c.Assert(res, checker.DeepEquals, exp)
}

func (s *TrieEncodingSuite) TestCompactHexEncode(c *checker.C) {
	exp := []byte("verb")
	res := compactHexEncode([]byte{7, 6, 6, 5, 7, 2, 6, 2, 16})
	c.Assert(res, checker.DeepEquals, exp)
}

func (s *TrieEncodingSuite) TestCompactDecode(c *checker.C) {
	for _, test := range []struct {
		str []byte
		exp []byte
	}{
		// odd compact decode
		{[]byte("\x11\x23\x45"), []byte{1, 2, 3, 4, 5}},
		// even compact decode
		{[]byte("\x00\x01\x23\x45"), []byte{0, 1, 2, 3, 4, 5}},
		// even terminated compact decode
		{[]byte("\x20\x0f\x1c\xb8"), []byte{0, 15, 1, 12, 11, 8 /*term*/, 16}},
		// even terminated compact decode
		{[]byte("\x3f\x1c\xb8"), []byte{15, 1, 12, 11, 8 /*term*/, 16}},
	} {
		c.Assert(compactDecode(test.str), checker.DeepEquals, test.exp)
	}
}

func (s *TrieEncodingSuite) TestDecodeCompact(c *checker.C) {
	for _, test := range []struct {
		str string
		key []byte
	}{
		{"012345", []byte{0, 1, 2, 3, 4, 5}},
		{"012345", []byte{0, 1, 2, 3, 4, 5, 16}},
		{"abcdef", []byte{10, 11, 12, 13, 14, 15}},
	} {
		exp, err := hex.DecodeString(test.str)
		c.Assert(err, checker.Equals, nil)
		c.Assert(decodeCompact(test.key), checker.DeepEquals, exp)
	}
}

func BenchmarkCompactEncode(b *testing.B) {

	testBytes := []byte{0, 15, 1, 12, 11, 8 /*term*/, 16}
	for i := 0; i < b.N; i++ {
		compactEncode(testBytes)
	}
}

func BenchmarkCompactDecode(b *testing.B) {
	testBytes := []byte{0, 15, 1, 12, 11, 8 /*term*/, 16}
	for i := 0; i < b.N; i++ {
		compactDecode(testBytes)
	}
}

func BenchmarkCompactHexDecode(b *testing.B) {
	testBytes := []byte{7, 6, 6, 5, 7, 2, 6, 2, 16}
	for i := 0; i < b.N; i++ {
		compactHexDecode(testBytes)
	}
}

func BenchmarkDecodeCompact(b *testing.B) {
	testBytes := []byte{7, 6, 6, 5, 7, 2, 6, 2, 16}
	for i := 0; i < b.N; i++ {
		decodeCompact(testBytes)
	}
}
