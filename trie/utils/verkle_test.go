// Copyright 2022 go-ethereum Authors
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

package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/ethereum/go-verkle"
	"github.com/holiman/uint256"
)

func TestGetTreeKey(t *testing.T) {
	var addr [32]byte
	for i := 0; i < 16; i++ {
		addr[1+2*i] = 0xff
	}
	n := uint256.NewInt(1)
	n = n.Lsh(n, 129)
	n.Add(n, uint256.NewInt(3))
	tk := GetTreeKey(addr[:], n, 1)

	got := hex.EncodeToString(tk)
	exp := "6ede905763d5856cd2d67936541e82aa78f7141bf8cd5ff6c962170f3e9dc201"
	if got != exp {
		t.Fatalf("Generated trie key is incorrect: %s != %s", got, exp)
	}
}

func TestConstantPoint(t *testing.T) {
	var expectedPoly [1]verkle.Fr

	cfg := verkle.GetConfig()
	verkle.FromLEBytes(&expectedPoly[0], []byte{2, 64})
	expected := cfg.CommitToPoly(expectedPoly[:], 1)

	if !expected.Equal(getTreePolyIndex0Point) {
		t.Fatalf("Marshalled constant value is incorrect: %x != %x", expected.Bytes(), getTreePolyIndex0Point.Bytes())
	}
}

func BenchmarkPedersenHash(b *testing.B) {
	var addr, v [32]byte

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rand.Read(v[:])
		rand.Read(addr[:])
		GetTreeKeyCodeSize(addr[:])
	}
}

func sha256GetTreeKeyCodeSize(addr []byte) []byte {
	digest := sha256.New()
	digest.Write(addr)
	treeIndexBytes := new(big.Int).Bytes()
	var payload [32]byte
	copy(payload[:len(treeIndexBytes)], treeIndexBytes)
	digest.Write(payload[:])
	h := digest.Sum(nil)
	h[31] = CodeHashLeafKey
	return h
}

func BenchmarkSha256Hash(b *testing.B) {
	var addr, v [32]byte

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rand.Read(v[:])
		rand.Read(addr[:])
		sha256GetTreeKeyCodeSize(addr[:])
	}
}

func TestCompareGetTreeKeyWithEvaluated(t *testing.T) {
	var addr [32]byte
	rand.Read(addr[:])
	addrpoint := EvaluateAddressPoint(addr[:])
	for i := 0; i < 100; i++ {
		var val [32]byte
		rand.Read(val[:])
		n := uint256.NewInt(0).SetBytes(val[:])
		n.Lsh(n, 8)
		subindex := val[0]
		tk1 := GetTreeKey(addr[:], n, subindex)
		tk2 := GetTreeKeyWithEvaluatedAddess(addrpoint, n, subindex)

		if !bytes.Equal(tk1, tk2) {
			t.Fatalf("differing key: slot=%x, addr=%x", val, addr)
		}
	}
}

func BenchmarkGetTreeKeyWithEvaluatedAddress(b *testing.B) {
	var buf [32]byte
	rand.Read(buf[:])
	addrpoint := EvaluateAddressPoint(buf[:])

	rand.Read(buf[:])
	n := uint256.NewInt(0).SetBytes32(buf[:])

	_ = verkle.GetConfig()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetTreeKeyWithEvaluatedAddess(addrpoint, n, 0)
	}
}
