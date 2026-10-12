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
	"math/rand"
	"strconv"
	"strings"
	"testing"
)

// naiveMerkleizeProgressive is merkleize_progressive(chunks, num_leaves)
// transcribed from EIP-7916, the reference the production fold is checked
// against: the first num_leaves chunks as a classic tree on the left, the
// recursion over the rest with four times the leaves on the right, and a
// zero chunk once the input is exhausted.
func naiveMerkleizeProgressive(chunks [][32]byte, numLeaves uint64) [32]byte {
	if len(chunks) == 0 {
		return [32]byte{}
	}
	split := min(numLeaves, uint64(len(chunks)))
	a := naiveMerkleize(chunks[:split], numLeaves)
	b := naiveMerkleizeProgressive(chunks[split:], numLeaves*4)
	var buf [64]byte
	copy(buf[:32], a[:])
	copy(buf[32:], b[:])
	return sha256.Sum256(buf[:])
}

// progressiveBoundaries are the chunk counts at which the tree gains a
// subtree, with their neighbours: where the shape changes and off-by-one
// bugs live.
var progressiveBoundaries = []int{
	0, 1, 2, 4, 5, 6, 20, 21, 22, 84, 85, 86, 340, 341, 342,
	1364, 1365, 1366, 5460, 5461, 5462, 21844, 21845, 21846,
}

func TestProgressiveBase(t *testing.T) {
	for i, want := range []struct{ base, size uint64 }{
		{0, 1}, {1, 4}, {5, 16}, {21, 64}, {85, 256}, {341, 1024}, {1365, 4096},
	} {
		if base, size := progressiveBase(i); base != want.base || size != want.size {
			t.Errorf("progressiveBase(%d) = (%d, %d), want (%d, %d)", i, base, size, want.base, want.size)
		}
	}
}

func TestMerkleizeProgressive(t *testing.T) {
	rng := rand.New(rand.NewSource(7916))
	for _, n := range progressiveBoundaries {
		chunks := randomChunks(rng, n)
		if got, want := MerkleizeProgressive(chunks), naiveMerkleizeProgressive(chunks, 1); got != want {
			t.Errorf("n=%d: root %x, want %x", n, got, want)
		}
	}
	// Callers keep ownership of their chunks: the fold must not touch them.
	chunks := randomChunks(rng, 23)
	before := append([][32]byte(nil), chunks...)
	MerkleizeProgressive(chunks)
	for i := range chunks {
		if chunks[i] != before[i] {
			t.Fatalf("MerkleizeProgressive modified input chunk %d", i)
		}
	}
}

// TestProgressiveGrowth checks every length of a growing list against the
// recursion, which fixes each chunk's position by its index alone: appending
// never moves an existing chunk, the property the type exists for.
func TestProgressiveGrowth(t *testing.T) {
	rng := rand.New(rand.NewSource(7495))
	chunks := randomChunks(rng, 100)
	for n := 0; n <= len(chunks); n++ {
		if got, want := MerkleizeProgressive(chunks[:n]), naiveMerkleizeProgressive(chunks[:n], 1); got != want {
			t.Fatalf("n=%d: root %x, want %x", n, got, want)
		}
	}
}

// patternChunks returns n chunks with every byte of chunk i set to i+1, the
// input shape of the roots Erigon publishes.
func patternChunks(n int) [][32]byte {
	chunks := make([][32]byte, n)
	for i := range chunks {
		for j := range chunks[i] {
			chunks[i][j] = byte(i + 1)
		}
	}
	return chunks
}

func mustRoot(t *testing.T, s string) [32]byte {
	var root [32]byte
	if _, err := hex.Decode(root[:], []byte(s)); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestProgressiveKnownRoots checks the progressive roots against values
// published by other implementations, so that correctness does not rest on
// this package's own reading of the EIP: Erigon's reference vectors
// (cl/merkle_tree/progressive_test.go, generated with eth-remerkleable
// v0.1.31) and the ethereum/ssz-specs v0.1.0 test vectors
// (fixtures/ssz/ssz/test_progressive_types).
func TestProgressiveKnownRoots(t *testing.T) {
	// Erigon: merkleize_progressive over pattern chunks at every subtree
	// boundary from 1 to 256 leaves.
	for _, c := range []struct {
		n    int
		root string
	}{
		{0, "0000000000000000000000000000000000000000000000000000000000000000"},
		{1, "037d6dfb3a369a41e01100fdd53c35ee3fb69ddec5830d61e1138d066a4c2285"},
		{2, "2dfe47da19ad9ff11afe44dd8de4db8517cefd5a9bddffe6652b26a1b91ea5ac"},
		{5, "3fd53b812118ddea60b9deab5c72d32b0c4dcfd2c94deda753e6e1d548fbc274"},
		{6, "2e2a2abd4d0e28498ec0cdd817c715b246aa15e7b34767061b7632337188429e"},
		{21, "f148f679afbfebfe5616080a45461aee3d1f4ce2cc752ce824c3f067d2707623"},
		{22, "040be60071c540aafc1d44f366239ab6a41bf8740a38f9d52ab0bbd9cd974c45"},
		{85, "24ea21562226364be74fd2696d0824a4347cfac7dd4b2ae28cd0e9cc22bc341d"},
		{86, "b73c4c427974f47c74c2812d353c966f5dadae70c44f6fe9a15e179b86914977"},
	} {
		if got, want := MerkleizeProgressive(patternChunks(c.n)), mustRoot(t, c.root); got != want {
			t.Errorf("erigon: %d chunks: root %x, want %x", c.n, got, want)
		}
	}
	// Erigon: lists of composites, one pattern chunk per element.
	for _, c := range []struct {
		n    int
		root string
	}{
		{0, "f5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b"},
		{1, "a21da97c8a597221c87c9ea5ecdfbd860fcd52fd6fb5b001723f6437856c8df1"},
		{2, "9a4badc45a45e9dd4b131c2c1aaff8a054527d8db40d2a7cd07e8f0f02a8232b"},
		{5, "183886e81b2e887d5960b2fa49b3464eabee62ec55ff5e6ee6f7e0495d8a01d1"},
		{6, "690beb7f075e2dc91699aa3ee9354687772923889ce755458cb405ae95e34055"},
		{21, "93589633f10a1e8fe51bef0481731c7c19d7a87269127b6c6a19720668ee47da"},
		{22, "e79bdcda4e58dd09c4b855964e1f1c01c99e215b6e01602f5302763effaf8637"},
	} {
		if got, want := HashTreeRootProgressiveRoots(patternChunks(c.n)), mustRoot(t, c.root); got != want {
			t.Errorf("erigon: list of %d composites: root %x, want %x", c.n, got, want)
		}
	}
	// Erigon: sixteen uint16 values packed into one pattern chunk. The length
	// mixed in is the element count, not the chunk count.
	packed := MixInLength(MerkleizeProgressive(patternChunks(1)), 16)
	if want := mustRoot(t, "e803cfbc4d0caecd70bff297ea98e07b6c3e4d0057e7321e020a4023fe52a9d7"); packed != want {
		t.Errorf("erigon: packed uint16 list: root %x, want %x", packed, want)
	}
	// Erigon: byte lists with data[i] = i%251 + 1 at the chunk and subtree
	// boundaries.
	for _, c := range []struct {
		n    int
		root string
	}{
		{0, "f5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b"},
		{1, "905efb51c2764c2c7a4efb0548e372569df06db82115c3b1896c186632f3fe5b"},
		{31, "3e12b2d2b507ef7ffe70761d0b0b69af7a26449621227a7a3e06438917f4aebd"},
		{32, "77a8c5b3ec7b888068f0d2f0237b535b7ac6dc38c9ce75ed40a3bb6250537bc9"},
		{33, "bdb0c331db145d1efad9e022c70ab1f1c0896e7fc8bd8a83c6f0cd6ca89e1009"},
		{160, "2927fdb091eebe601a502656169f978a3fb2cf2a641ee97aad0f25458ef0f93a"},
		{161, "14103a66a65d67d16d80e2b914241af6d1a371dc03925e610f2ebf7068d22c05"},
		{672, "03a448071ba4184a8586b78b154880195db0a7a29aef04bd865f77c3469d6d5b"},
		{673, "d411caaeaf48519cd983b24a40115851e63996cbd47927651f686679a23c6c71"},
	} {
		data := make([]byte, c.n)
		for i := range data {
			data[i] = byte(i%251 + 1)
		}
		if got, want := HashTreeRootProgressiveBytes(data), mustRoot(t, c.root); got != want {
			t.Errorf("erigon: byte list of %d: root %x, want %x", c.n, got, want)
		}
	}
	// ssz-specs: ProgressiveList[uint64] holding 0, 1, ..., n-1 at the lengths
	// that fill and open the first four subtrees, plus the list {1}.
	for _, c := range []struct {
		n     int
		first uint64
		root  string
	}{
		{0, 0, "f5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b"},
		{1, 1, "905efb51c2764c2c7a4efb0548e372569df06db82115c3b1896c186632f3fe5b"},
		{4, 0, "fb119bdda96d8ebf59b511db66fc19a40f4f1543a5aa87d8d9b4519655a8eda9"},
		{5, 0, "b52da986d8c44ac58d43d54d5a6f27363363ad09e1249d211c38c21c5221e5f4"},
		{20, 0, "1957d11b2bce3ef0c72872fca6fa4cffacc27e601c91b88ab8e6b28eebc6525c"},
		{21, 0, "86a8ce9749021379ba7af31ac5d6f3b33e0e0791a5c9bb2e00580fe8d6aeb117"},
		{84, 0, "765cd3f85263382f72bcf782dd80038ec9184206fdde9e0872f5ec4025ed992f"},
		{85, 0, "44149f84b9899187d206378ab59b88e7e59ebd3ed84fd95b4b2ba58b08f98024"},
	} {
		data := make([]byte, 8*c.n)
		for i := 0; i < c.n; i++ {
			binary.LittleEndian.PutUint64(data[8*i:], c.first+uint64(i))
		}
		if got, want := MixInLength(MerkleizeProgressive(Pack(data)), uint64(c.n)), mustRoot(t, c.root); got != want {
			t.Errorf("ssz-specs: uint64 list of %d: root %x, want %x", c.n, got, want)
		}
	}
	// ssz-specs: ProgressiveBitlist from its serialized form, delimiter
	// included: empty, [1, 0, 1], 256 set bits filling one chunk, and 257 set
	// bits reaching one bit into the second subtree.
	for _, c := range []struct {
		serialized string
		root       string
	}{
		{"01", "f5a5fd42d16a20302798ef6ed309979b43003d2320d9f0e8ea9831a92759fb4b"},
		{"0d", "45192380e83a4b9ee939ac3836a6dccc51d3451db8886d53668264ea2e2cb877"},
		{strings.Repeat("ff", 32) + "01", "b3327406854ffab96af59832dfa3f690f72c4f898e2ffd4ef3e90cc2fb876b43"},
		{strings.Repeat("ff", 32) + "03", "be707c375a49431fdb06c00f7a4dcc9200d5613ea02999dc5e081913171bb8d0"},
	} {
		ser, err := hex.DecodeString(c.serialized)
		if err != nil {
			t.Fatal(err)
		}
		bits, nbits, err := DecodeBitlist(ser, NoLimit)
		if err != nil {
			t.Fatalf("ssz-specs: bitlist %s: %v", c.serialized, err)
		}
		if got, want := HashTreeRootProgressiveBitlist(bits, nbits), mustRoot(t, c.root); got != want {
			t.Errorf("ssz-specs: bitlist of %d bits: root %x, want %x", nbits, got, want)
		}
	}
	// ssz-specs: a ProgressiveList[Bytes32] of one element of 0xaa bytes.
	var aa [32]byte
	for i := range aa {
		aa[i] = 0xaa
	}
	if got, want := HashTreeRootProgressiveRoots([][32]byte{aa}), mustRoot(t, "bb93b54e640d8bb22570653bf0423df16e94fb3a1652e9ee7b218e5fa8eaaf5c"); got != want {
		t.Errorf("ssz-specs: list of one Bytes32: root %x, want %x", got, want)
	}
}

// TestProgressiveOrientation guards the consensus-critical hash order. The
// published two-chunk root is reproduced only with the subtree on the left
// and the spine on the right; the mirrored order, which every text and
// vector set older than 2026-01-15 describes, gives a different root.
func TestProgressiveOrientation(t *testing.T) {
	chunks := patternChunks(2)
	want := mustRoot(t, "2dfe47da19ad9ff11afe44dd8de4db8517cefd5a9bddffe6652b26a1b91ea5ac")
	if got := MerkleizeProgressive(chunks); got != want {
		t.Fatalf("root %x, want the published %x", got, want)
	}
	// hash(successor, subtree) at both levels: the shape before the flip.
	mirrored := hashPair(hashPair([32]byte{}, Merkleize(chunks[1:], 4)), chunks[0])
	if mirrored == want {
		t.Fatal("the published root matches the pre-2026-01-15 orientation")
	}
}

// FuzzMerkleizeProgressive checks the fold against the recursion on
// arbitrary chunk counts and contents.
func FuzzMerkleizeProgressive(f *testing.F) {
	f.Add([]byte{}, uint16(0))
	f.Add([]byte{1}, uint16(1))
	f.Add([]byte{0xaa, 0xbb}, uint16(21))
	f.Add([]byte("boundary+1"), uint16(1366))
	f.Fuzz(func(t *testing.T, seed []byte, n uint16) {
		// Derive up to 4096 chunks from the seed, so that large shapes are
		// reachable from small inputs.
		chunks := make([][32]byte, int(n)%4096)
		if len(seed) > 0 {
			for i := range chunks {
				for j := range chunks[i] {
					chunks[i][j] = seed[(i*32+j)%len(seed)] ^ byte(i)
				}
			}
		}
		if got, want := MerkleizeProgressive(chunks), naiveMerkleizeProgressive(chunks, 1); got != want {
			t.Fatalf("%d chunks: root %x, want %x", len(chunks), got, want)
		}
	})
}

func BenchmarkMerkleizeProgressive(b *testing.B) {
	rng := rand.New(rand.NewSource(6))
	for _, n := range []int{32, 1024, 65536} {
		chunks := randomChunks(rng, n)
		b.Run("chunks="+strconv.Itoa(n), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				MerkleizeProgressive(chunks)
			}
		})
	}
}
