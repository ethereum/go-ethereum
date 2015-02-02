package rle

import (
	"testing"

	checker "gopkg.in/check.v1"
)

func Test(t *testing.T) { checker.TestingT(t) }

type CompressionRleSuite struct{}

var _ = checker.Suite(&CompressionRleSuite{})

func (s *CompressionRleSuite) TestDecompressSimple(c *checker.C) {
	exp := []byte{0xc5, 0xd2, 0x46, 0x1, 0x86, 0xf7, 0x23, 0x3c, 0x92, 0x7e, 0x7d, 0xb2, 0xdc, 0xc7, 0x3, 0xc0, 0xe5, 0x0, 0xb6, 0x53, 0xca, 0x82, 0x27, 0x3b, 0x7b, 0xfa, 0xd8, 0x4, 0x5d, 0x85, 0xa4, 0x70}
	res, err := Decompress([]byte{token, 0xfd})
	c.Assert(err, checker.IsNil)
	c.Assert(res, checker.DeepEquals, exp)
	// if bytes.Compare(res, exp) != 0 {
	// 	t.Error("empty sha3", res)
	// }

	exp = []byte{0x56, 0xe8, 0x1f, 0x17, 0x1b, 0xcc, 0x55, 0xa6, 0xff, 0x83, 0x45, 0xe6, 0x92, 0xc0, 0xf8, 0x6e, 0x5b, 0x48, 0xe0, 0x1b, 0x99, 0x6c, 0xad, 0xc0, 0x1, 0x62, 0x2f, 0xb5, 0xe3, 0x63, 0xb4, 0x21}
	res, err = Decompress([]byte{token, 0xfe})
	c.Assert(err, checker.IsNil)
	c.Assert(res, checker.DeepEquals, exp)
	// if bytes.Compare(res, exp) != 0 {
	// 	t.Error("0x80 sha3", res)
	// }

	res, err = Decompress([]byte{token, 0xff})
	c.Assert(err, checker.IsNil)
	c.Assert(res, checker.DeepEquals, []byte{token})
	// if bytes.Compare(res, []byte{token}) != 0 {
	// 	t.Error("token", res)
	// }

	res, err = Decompress([]byte{token, 12})
	c.Assert(err, checker.IsNil)
	c.Assert(res, checker.DeepEquals, make([]byte, 10))
	// if bytes.Compare(res, make([]byte, 10)) != 0 {
	// 	t.Error("10 * zero", res)
	// }
}

// func TestDecompressMulti(t *testing.T) {
// 	res, err := Decompress([]byte{token, 0xfd, token, 0xfe, token, 12})
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	var exp []byte
// 	exp = append(exp, crypto.Sha3([]byte(""))...)
// 	exp = append(exp, crypto.Sha3([]byte{0x80})...)
// 	exp = append(exp, make([]byte, 10)...)

// 	if bytes.Compare(res, res) != 0 {
// 		t.Error("Expected", exp, "result", res)
// 	}
// }

// func TestCompressSimple(t *testing.T) {
// 	res := Compress([]byte{0, 0, 0, 0, 0})
// 	if bytes.Compare(res, []byte{token, 7}) != 0 {
// 		t.Error("5 * zero", res)
// 	}

// 	res = Compress(crypto.Sha3([]byte("")))
// 	if bytes.Compare(res, []byte{token, emptyShaToken}) != 0 {
// 		t.Error("empty sha", res)
// 	}

// 	res = Compress(crypto.Sha3([]byte{0x80}))
// 	if bytes.Compare(res, []byte{token, emptyListShaToken}) != 0 {
// 		t.Error("empty list sha", res)
// 	}

// 	res = Compress([]byte{token})
// 	if bytes.Compare(res, []byte{token, tokenToken}) != 0 {
// 		t.Error("token", res)
// 	}
// }

// func TestCompressMulti(t *testing.T) {
// 	in := []byte{0, 0, 0, 0, 0}
// 	in = append(in, crypto.Sha3([]byte(""))...)
// 	in = append(in, crypto.Sha3([]byte{0x80})...)
// 	in = append(in, token)
// 	res := Compress(in)

// 	exp := []byte{token, 7, token, emptyShaToken, token, emptyListShaToken, token, tokenToken}
// 	if bytes.Compare(res, exp) != 0 {
// 		t.Error("expected", exp, "got", res)
// 	}
// }

// func TestCompressDecompress(t *testing.T) {
// 	var in []byte

// 	for i := 0; i < 20; i++ {
// 		in = append(in, []byte{0, 0, 0, 0, 0}...)
// 		in = append(in, crypto.Sha3([]byte(""))...)
// 		in = append(in, crypto.Sha3([]byte{0x80})...)
// 		in = append(in, []byte{123, 2, 19, 89, 245, 254, 255, token, 98, 233}...)
// 		in = append(in, token)
// 	}

// 	c := Compress(in)
// 	d, err := Decompress(c)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if bytes.Compare(d, in) != 0 {
// 		t.Error("multi failed\n", d, "\n", in)
// 	}
// }
