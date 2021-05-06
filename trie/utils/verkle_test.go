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
	"crypto/sha256"
	"math/big"
	"math/rand"
	"testing"

	"github.com/gballet/go-verkle"
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
	GetTreeKey(addr[:], n, 1)
}

func TestConstantPoint(t *testing.T) {
	cfg, _ := verkle.GetConfig()
	verkle.FromLEBytes(&getTreePolyIndex0Fr[0], []byte{2, 64})
	expected := cfg.CommitToPoly(getTreePolyIndex0Fr[:], 1)

	if !verkle.Equal(expected, getTreePolyIndex0Point) {
		t.Fatal("Marshalled constant value is incorrect")
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
	h[31] = CodeKeccakLeafKey
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
