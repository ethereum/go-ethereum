package rle

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestDecompressSimple(t *testing.T) {
	res, err := Decompress([]byte{token, 0xfd})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(res, crypto.Sha3([]byte(""))) != 0 {
		t.Error("empty sha3", res)
	}

	res, err = Decompress([]byte{token, 0xfe})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(res, crypto.Sha3([]byte{0x80})) != 0 {
		t.Error("0x80 sha3", res)
	}

	res, err = Decompress([]byte{token, 0xff})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(res, []byte{token}) != 0 {
		t.Error("token", res)
	}

	res, err = Decompress([]byte{token, 12})
	if err != nil {
		t.Error(err)
	}
	if bytes.Compare(res, make([]byte, 10)) != 0 {
		t.Error("10 * zero", res)
	}
}

func TestDecompressMulti(t *testing.T) {
	res, err := Decompress([]byte{token, 0xfd, token, 0xfe, token, 12})
	if err != nil {
		t.Error(err)
	}

	var exp []byte
	exp = append(exp, crypto.Sha3([]byte(""))...)
	exp = append(exp, crypto.Sha3([]byte{0x80})...)
	exp = append(exp, make([]byte, 10)...)

	if bytes.Compare(res, res) != 0 {
		t.Error("Expected", exp, "result", res)
	}
}

func TestCompressSimple(t *testing.T) {
	res := Compress([]byte{0, 0, 0, 0, 0})
	if bytes.Compare(res, []byte{token, 7}) != 0 {
		t.Error("5 * zero", res)
	}

	res = Compress(crypto.Sha3([]byte("")))
	if bytes.Compare(res, []byte{token, emptyShaToken}) != 0 {
		t.Error("empty sha", res)
	}

	res = Compress(crypto.Sha3([]byte{0x80}))
	if bytes.Compare(res, []byte{token, emptyListShaToken}) != 0 {
		t.Error("empty list sha", res)
	}

	res = Compress([]byte{token})
	if bytes.Compare(res, []byte{token, tokenToken}) != 0 {
		t.Error("token", res)
	}
}

func TestCompressMulti(t *testing.T) {
	in := []byte{0, 0, 0, 0, 0}
	in = append(in, crypto.Sha3([]byte(""))...)
	in = append(in, crypto.Sha3([]byte{0x80})...)
	in = append(in, token)
	res := Compress(in)

	exp := []byte{token, 7, token, emptyShaToken, token, emptyListShaToken, token, tokenToken}
	if bytes.Compare(res, exp) != 0 {
		t.Error("expected", exp, "got", res)
	}
}

func TestCompressDecompress(t *testing.T) {
	var in []byte

	for i := 0; i < 20; i++ {
		in = append(in, []byte{0, 0, 0, 0, 0}...)
		in = append(in, crypto.Sha3([]byte(""))...)
		in = append(in, crypto.Sha3([]byte{0x80})...)
		in = append(in, []byte{123, 2, 19, 89, 245, 254, 255, token, 98, 233}...)
		in = append(in, token)
	}

	c := Compress(in)
	d, err := Decompress(c)
	if err != nil {
		t.Error(err)
	}

	if bytes.Compare(d, in) != 0 {
		t.Error("multi failed\n", d, "\n", in)
	}
}
