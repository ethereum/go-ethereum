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

package crypto

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

// These tests are sanity checks.
// They should ensure that we don't e.g. use Sha3-224 instead of Sha3-256
// and that the sha3 library uses keccak-f permutation.

func TestSha3(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	checkhash(t, "Sha3-256", func(in []byte) []byte { return Sha3(in) }, msg, exp)
}

func TestSha3Hash(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45")
	checkhash(t, "Sha3-256-array", func(in []byte) []byte { h := Sha3Hash(in); return h[:] }, msg, exp)
}

func TestSha256(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad")
	checkhash(t, "Sha256", Sha256, msg, exp)
}

func TestRipemd160(t *testing.T) {
	msg := []byte("abc")
	exp, _ := hex.DecodeString("8eb208f7e05d987a9b044a8e98c6b087f15a0bfc")
	checkhash(t, "Ripemd160", Ripemd160, msg, exp)
}

func checkhash(t *testing.T, name string, f func([]byte) []byte, msg, exp []byte) {
	sum := f(msg)
	if bytes.Compare(exp, sum) != 0 {
		t.Errorf("hash %s returned wrong result.\ngot:  %x\nwant: %x", name, sum, exp)
	}
}

func BenchmarkSha3(b *testing.B) {
	a := []byte("hello world")
	amount := 1000000
	start := time.Now()
	for i := 0; i < amount; i++ {
		Sha3(a)
	}

	fmt.Println(amount, ":", time.Since(start))
}

func Test0Key(t *testing.T) {
	t.Skip()
	key := common.Hex2Bytes("1111111111111111111111111111111111111111111111111111111111111111")

	p, err := secp256k1.GeneratePubKey(key)
	addr := Sha3(p[1:])[12:]
	fmt.Printf("%x\n", p)
	fmt.Printf("%v %x\n", err, addr)
}

func TestInvalidSign(t *testing.T) {
	_, err := Sign(make([]byte, 1), nil)
	if err == nil {
		t.Errorf("expected sign with hash 1 byte to error")
	}

	_, err = Sign(make([]byte, 33), nil)
	if err == nil {
		t.Errorf("expected sign with hash 33 byte to error")
	}
}
