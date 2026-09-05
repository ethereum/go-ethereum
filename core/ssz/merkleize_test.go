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

package ssz

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// naiveMerkleize is merkleize(chunks, limit) transcribed from the spec: pad
// with zero chunks to next_pow_of_two(limit) leaves and hash the whole tree,
// without the zero-hash shortcut.
func naiveMerkleize(chunks [][32]byte, limit uint64) [32]byte {
	size := uint64(1)
	for size < limit {
		size *= 2
	}
	layer := make([][32]byte, size)
	copy(layer, chunks)
	for len(layer) > 1 {
		next := make([][32]byte, len(layer)/2)
		for i := range next {
			var buf [64]byte
			copy(buf[:32], layer[2*i][:])
			copy(buf[32:], layer[2*i+1][:])
			next[i] = sha256.Sum256(buf[:])
		}
		layer = next
	}
	return layer[0]
}

func TestDepthOf(t *testing.T) {
	for _, c := range []struct {
		limit uint64
		depth int
	}{
		{0, 0}, {1, 0}, {2, 1}, {3, 2}, {4, 2}, {5, 3}, {8, 3}, {9, 4},
		{1 << 10, 10}, {1<<10 + 1, 11}, {1 << 63, 63}, {math.MaxUint64, 64},
	} {
		if got := depthOf(c.limit); got != c.depth {
			t.Errorf("depthOf(%d) = %d, want %d", c.limit, got, c.depth)
		}
	}
}

// TestMerkleizeEmpty checks that an empty input merkleized to any limit is
// the root of an all-zero tree of the corresponding depth.
func TestMerkleizeEmpty(t *testing.T) {
	for d := 0; d < 64; d++ {
		if got := Merkleize(nil, 1<<d); got != zeroHashes[d] {
			t.Errorf("Merkleize(nil, 1<<%d) != zeroHashes[%d]", d, d)
		}
	}
}

func TestMerkleize(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	for n := 0; n <= 70; n++ {
		chunks := randomChunks(rng, n)
		limits := []uint64{uint64(n), 1 << depthOf(uint64(n)), 2 * uint64(n), 64 * uint64(n), 1 << 12}
		if n == 0 {
			limits = append(limits, 1)
		}
		for _, limit := range limits {
			if limit < uint64(n) {
				continue
			}
			if got, want := Merkleize(chunks, limit), naiveMerkleize(chunks, limit); got != want {
				t.Errorf("n=%d limit=%d: root %x, want %x", n, limit, got, want)
			}
		}
	}
	// Callers keep ownership of their chunks: the fold must not touch them.
	chunks := randomChunks(rng, 9)
	before := append([][32]byte(nil), chunks...)
	Merkleize(chunks, 16)
	for i := range chunks {
		if chunks[i] != before[i] {
			t.Fatalf("Merkleize modified input chunk %d", i)
		}
	}
}

func FuzzMerkleize(f *testing.F) {
	f.Add([]byte{}, uint8(0))
	f.Add([]byte{1}, uint8(0))
	f.Add(make([]byte, 32), uint8(1))
	f.Add(make([]byte, 33), uint8(2))
	f.Add(make([]byte, 32*5), uint8(3))
	f.Add(make([]byte, 32*64), uint8(200))
	f.Fuzz(func(t *testing.T, data []byte, extra uint8) {
		chunks := Pack(data)
		limit := uint64(len(chunks)) + uint64(extra)
		if got, want := Merkleize(chunks, limit), naiveMerkleize(chunks, limit); got != want {
			t.Fatalf("%d chunks, limit %d: root %x, want %x", len(chunks), limit, got, want)
		}
	})
}

func TestMerkleizeOverLimitPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Merkleize with more chunks than the limit did not panic")
		}
	}()
	Merkleize(make([][32]byte, 3), 2)
}

func TestMixIns(t *testing.T) {
	rng := rand.New(rand.NewSource(4))
	var root [32]byte
	rng.Read(root[:])
	for _, length := range []uint64{0, 1, 255, 256, 1 << 32, math.MaxUint64} {
		var buf [64]byte
		copy(buf[:32], root[:])
		binary.LittleEndian.PutUint64(buf[32:40], length)
		if got, want := MixInLength(root, length), sha256.Sum256(buf[:]); got != want {
			t.Errorf("MixInLength(%d) = %x, want %x", length, got, want)
		}
	}
	for _, selector := range []uint8{0, 1, 127, 255} {
		var buf [64]byte
		copy(buf[:32], root[:])
		buf[32] = selector
		if got, want := MixInSelector(root, selector), sha256.Sum256(buf[:]); got != want {
			t.Errorf("MixInSelector(%d) = %x, want %x", selector, got, want)
		}
	}
}

// TestKnownRoots checks hash_tree_root values taken from the ssz_generic
// suite of consensus-spec-tests v1.7.0-alpha.12, embedded here so that root
// correctness is verified even where the full vector set of TestSpecVectors
// is unavailable (-short CI runs do not download it).
func TestKnownRoots(t *testing.T) {
	for _, c := range []struct {
		name       string
		serialized string
		limit      uint64
		root       string
	}{
		{
			name:       "boolean/true",
			serialized: "01",
			limit:      1,
			root:       "0100000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:       "uints/uint_64_max",
			serialized: "ffffffffffffffff",
			limit:      1,
			root:       "ffffffffffffffff000000000000000000000000000000000000000000000000",
		},
		{
			name:       "uints/uint_256_random_0",
			serialized: "82a63075aa80de1ebc8684e5b5dedd9b60dd8f2436827c8d35f8bf4b5956270c",
			limit:      1,
			root:       "82a63075aa80de1ebc8684e5b5dedd9b60dd8f2436827c8d35f8bf4b5956270c",
		},
		{
			name:       "bitvector/bitvec_15_max",
			serialized: "ff7f",
			limit:      1,
			root:       "ff7f000000000000000000000000000000000000000000000000000000000000",
		},
		{
			name:       "bitvector/bitvec_512_max",
			serialized: strings.Repeat("ff", 64),
			limit:      2,
			root:       "8667e718294e9e0df1d30600ba3eeb201f764aad2dad72748643e4a285e1d1f7",
		},
		{
			name:       "basic_vector/vec_uint64_5_random",
			serialized: "70d8d69b0eaaa6db734d0fcabf5c28c611d2c1d0649882aeb39cafd74f45da4ff17e056a81bef220",
			limit:      2,
			root:       "8d4d9cc64aa0b6d53b8966df74f3338efc8aa3eeb058c5df69df6ed9cbbc3453",
		},
		{
			name:       "basic_vector/vec_uint16_31_random",
			serialized: "5c06985ecce4105c8fa783700914617c9b40f3bbebaff38d7f7e26d7c03e7a2b2b70e56983dae3e133b927d8825372d2da6c0f34acbab64dd45be282db87",
			limit:      2,
			root:       "27032d0989b867669fcafe33373a1b98ad20d1603a4d9998e3e22879192ea8dd",
		},
		{
			name:       "basic_vector/vec_uint256_512_max",
			serialized: strings.Repeat("ff", 16384),
			limit:      512,
			root:       "a278cf32ca74f920b67a7b3d02447453d8883fecb4a7aa1ba4327079fa3d5162",
		},
	} {
		data, err := hex.DecodeString(c.serialized)
		if err != nil {
			t.Fatal(err)
		}
		var want [32]byte
		if _, err := hex.Decode(want[:], []byte(c.root)); err != nil {
			t.Fatal(err)
		}
		if got := Merkleize(Pack(data), c.limit); got != want {
			t.Errorf("%s: root %x, want %x", c.name, got, want)
		}
	}
	// The depth-1 zero hash, H(0^32 || 0^32), quoted in the SSZ annotated
	// spec and hardcoded by consensus clients.
	want := "f5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b"
	if got := hex.EncodeToString(zeroHashes[1][:]); got != want {
		t.Errorf("zeroHashes[1] = %s, want %s", got, want)
	}
}

func BenchmarkMerkleize(b *testing.B) {
	rng := rand.New(rand.NewSource(5))
	for _, n := range []int{32, 1024, 65536} {
		chunks := randomChunks(rng, n)
		b.Run("chunks="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				Merkleize(chunks, uint64(n))
			}
		})
	}
}
