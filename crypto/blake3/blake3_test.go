// Copyright 2026 The go-ethereum Authors
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

package blake3

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	luke "lukechampine.com/blake3"
)

// vectorFile mirrors testdata/vectors.json, generated from the official
// BLAKE3 python bindings over the reference input pattern (byte i = i % 251).
type vectorFile struct {
	Meta    map[string]string `json:"meta"`
	Vectors []struct {
		Len    int    `json:"len"`
		Digest string `json:"digest"`
	} `json:"vectors"`
}

func loadVectors(t *testing.T) vectorFile {
	t.Helper()
	blob, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vf vectorFile
	if err := json.Unmarshal(blob, &vf); err != nil {
		t.Fatal(err)
	}
	return vf
}

func patternInput(n int) []byte {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i % 251)
	}
	return data
}

// TestVectors pins the shipped wrapper against the official BLAKE3 vectors.
func TestVectors(t *testing.T) {
	for _, v := range loadVectors(t).Vectors {
		want, _ := hex.DecodeString(v.Digest)
		got := Sum256(patternInput(v.Len))
		if !bytes.Equal(got[:], want) {
			t.Fatalf("len %d: got %x want %x", v.Len, got, want)
		}
	}
}

// TestVectorsLuke cross-checks the alternate candidate library against the
// same vectors, keeping the library swap trivially safe.
func TestVectorsLuke(t *testing.T) {
	for _, v := range loadVectors(t).Vectors {
		want, _ := hex.DecodeString(v.Digest)
		got := luke.Sum256(patternInput(v.Len))
		if !bytes.Equal(got[:], want) {
			t.Fatalf("len %d: got %x want %x", v.Len, got, want)
		}
	}
}

// The benchmark sizes match the EIP-8297 engine's hash workload: 67/99B leaf
// preimages, 133B max branch preimages, plus larger blobs for context.
var benchSizes = []struct {
	name string
	n    int
}{
	{"64", 64}, {"67", 67}, {"99", 99}, {"133", 133}, {"1k", 1024}, {"8k", 8192},
}

func BenchmarkBlake3(b *testing.B) {
	for _, s := range benchSizes {
		data := patternInput(s.n)
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(s.n))
			for b.Loop() {
				Sum256(data)
			}
		})
	}
}

func BenchmarkBlake3Luke(b *testing.B) {
	for _, s := range benchSizes {
		data := patternInput(s.n)
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(s.n))
			for b.Loop() {
				luke.Sum256(data)
			}
		})
	}
}

func BenchmarkSha256(b *testing.B) {
	for _, s := range benchSizes {
		data := patternInput(s.n)
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(s.n))
			for b.Loop() {
				sha256.Sum256(data)
			}
		})
	}
}

func BenchmarkKeccak256(b *testing.B) {
	for _, s := range benchSizes {
		data := patternInput(s.n)
		b.Run(s.name, func(b *testing.B) {
			b.SetBytes(int64(s.n))
			for b.Loop() {
				crypto.Keccak256(data)
			}
		})
	}
}
