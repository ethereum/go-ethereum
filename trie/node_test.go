// Copyright 2016 The go-ethereum Authors
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
	"bytes"
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/rlp"
)

func newTestFullNode(v []byte) []interface{} {
	fullNodeData := []interface{}{}
	for i := 0; i < 16; i++ {
		k := bytes.Repeat([]byte{byte(i + 1)}, 32)
		fullNodeData = append(fullNodeData, k)
	}
	fullNodeData = append(fullNodeData, v)
	return fullNodeData
}

func TestDecodeNestedNode(t *testing.T) {
	fullNodeData := newTestFullNode([]byte("fullnode"))

	data := [][]byte{}
	for i := 0; i < 16; i++ {
		data = append(data, nil)
	}
	data = append(data, []byte("subnode"))
	fullNodeData[15] = data

	buf := bytes.NewBuffer([]byte{})
	rlp.Encode(buf, fullNodeData)

	if _, err := decodeNode([]byte("testdecode"), buf.Bytes()); err != nil {
		t.Fatalf("decode nested full node err: %v", err)
	}
}

func TestDecodeFullNodeWrongSizeChild(t *testing.T) {
	fullNodeData := newTestFullNode([]byte("wrongsizechild"))
	fullNodeData[0] = []byte("00")
	buf := bytes.NewBuffer([]byte{})
	rlp.Encode(buf, fullNodeData)

	_, err := decodeNode([]byte("testdecode"), buf.Bytes())
	if _, ok := err.(*decodeError); !ok {
		t.Fatalf("decodeNode returned wrong err: %v", err)
	}
}

func TestDecodeFullNodeWrongNestedFullNode(t *testing.T) {
	fullNodeData := newTestFullNode([]byte("fullnode"))

	data := [][]byte{}
	for i := 0; i < 16; i++ {
		data = append(data, []byte("123456"))
	}
	data = append(data, []byte("subnode"))
	fullNodeData[15] = data

	buf := bytes.NewBuffer([]byte{})
	rlp.Encode(buf, fullNodeData)

	_, err := decodeNode([]byte("testdecode"), buf.Bytes())
	if _, ok := err.(*decodeError); !ok {
		t.Fatalf("decodeNode returned wrong err: %v", err)
	}
}

func TestDecodeFullNode(t *testing.T) {
	fullNodeData := newTestFullNode([]byte("decodefullnode"))
	buf := bytes.NewBuffer([]byte{})
	rlp.Encode(buf, fullNodeData)

	_, err := decodeNode([]byte("testdecode"), buf.Bytes())
	if err != nil {
		t.Fatalf("decode full node err: %v", err)
	}
}

func makeTestLeafNode(small bool) []byte {
	l := leafNodeEncoder{}
	l.Key = hexToCompact(keybytesToHex(testrand.Bytes(10)))
	if small {
		l.Val = testrand.Bytes(10)
	} else {
		l.Val = testrand.Bytes(32)
	}
	buf := rlp.NewEncoderBuffer(nil)
	l.encode(buf)
	return buf.ToBytes()
}

func makeTestFullNode(small bool) []byte {
	n := fullnodeEncoder{}
	for i := 0; i < 16; i++ {
		switch rand.Intn(3) {
		case 0:
			// write nil
		case 1:
			// write hash
			n.Children[i] = testrand.Bytes(32)
		case 2:
			// write embedded node
			n.Children[i] = makeTestLeafNode(small)
		}
	}
	n.Children[16] = testrand.Bytes(32) // value
	buf := rlp.NewEncoderBuffer(nil)
	n.encode(buf)
	return buf.ToBytes()
}

func TestEncodeDecodeNodeElements(t *testing.T) {
	var nodes [][]byte
	nodes = append(nodes, makeTestFullNode(true))
	nodes = append(nodes, makeTestFullNode(false))
	nodes = append(nodes, makeTestLeafNode(true))
	nodes = append(nodes, makeTestLeafNode(false))

	for _, blob := range nodes {
		elements, err := decodeNodeElements(blob)
		if err != nil {
			t.Fatalf("Failed to decode node elements: %v", err)
		}
		enc, err := encodeNodeElements(elements)
		if err != nil {
			t.Fatalf("Failed to encode node elements: %v", err)
		}
		if !bytes.Equal(enc, blob) {
			t.Fatalf("Unexpected encoded node element, want: %v, got: %v", blob, enc)
		}
	}
}

func makeTestLeafNodePair() ([]byte, []byte, [][]byte, []int) {
	var (
		na = leafNodeEncoder{}
		nb = leafNodeEncoder{}
	)
	key := keybytesToHex(testrand.Bytes(10))
	na.Key = hexToCompact(key)
	nb.Key = hexToCompact(key)

	valA := testrand.Bytes(32)
	valB := testrand.Bytes(32)
	na.Val = valA
	nb.Val = valB

	bufa, bufb := rlp.NewEncoderBuffer(nil), rlp.NewEncoderBuffer(nil)
	na.encode(bufa)
	nb.encode(bufb)
	diff, _ := rlp.EncodeToBytes(valA)
	return bufa.ToBytes(), bufb.ToBytes(), [][]byte{diff}, []int{1}
}

func makeTestFullNodePair() ([]byte, []byte, [][]byte, []int) {
	var (
		na      = fullnodeEncoder{}
		nb      = fullnodeEncoder{}
		indices []int
		values  [][]byte
	)
	for i := 0; i < 16; i++ {
		switch rand.Intn(3) {
		case 0:
			// write nil
		case 1:
			// write same
			var child []byte
			if rand.Intn(2) == 0 {
				child = testrand.Bytes(32) // hashnode
			} else {
				child = makeTestLeafNode(true) // embedded node
			}
			na.Children[i] = child
			nb.Children[i] = child
		case 2:
			// write different
			var (
				va   []byte
				diff []byte
			)
			rnd := rand.Intn(3)
			if rnd == 0 {
				va = testrand.Bytes(32) // hashnode
				diff, _ = rlp.EncodeToBytes(va)
			} else if rnd == 1 {
				va = makeTestLeafNode(true) // embedded node
				diff = va
			} else {
				va = nil
				diff = rlp.EmptyString
			}
			vb := testrand.Bytes(32) // hashnode
			na.Children[i] = va
			nb.Children[i] = vb

			indices = append(indices, i)
			values = append(values, diff)
		}
	}
	na.Children[16] = nil
	nb.Children[16] = nil

	bufa, bufb := rlp.NewEncoderBuffer(nil), rlp.NewEncoderBuffer(nil)
	na.encode(bufa)
	nb.encode(bufb)
	return bufa.ToBytes(), bufb.ToBytes(), values, indices
}

func TestNodeDifference(t *testing.T) {
	type testsuite struct {
		old        []byte
		new        []byte
		expErr     bool
		expIndices []int
		expValues  [][]byte
	}
	var tests = []testsuite{
		// Invalid node data
		{
			old: nil, new: nil, expErr: true,
		},
		{
			old: testrand.Bytes(32), new: nil, expErr: true,
		},
		{
			old: nil, new: testrand.Bytes(32), expErr: true,
		},
		{
			old: bytes.Repeat([]byte{0x1}, 32), new: bytes.Repeat([]byte{0x2}, 32), expErr: true,
		},
		// Different node type
		{
			old: makeTestLeafNode(true), new: makeTestFullNode(true), expErr: true,
		},
	}
	for range 10 {
		va, vb, elements, indices := makeTestLeafNodePair()
		tests = append(tests, testsuite{
			old:        va,
			new:        vb,
			expErr:     false,
			expIndices: indices,
			expValues:  elements,
		})
	}
	for range 10 {
		va, vb, elements, indices := makeTestFullNodePair()
		tests = append(tests, testsuite{
			old:        va,
			new:        vb,
			expErr:     false,
			expIndices: indices,
			expValues:  elements,
		})
	}

	for i, test := range tests {
		_, indices, values, err := NodeDifference(test.old, test.new)
		if test.expErr && err == nil {
			t.Fatalf("Expect error, got nil %d", i)
		}
		if !test.expErr && err != nil {
			t.Fatalf("Unexpect error, %v", err)
		}
		if err == nil {
			if !slices.Equal(indices, test.expIndices) {
				t.Fatalf("Unexpected indices, want: %v, got: %v", test.expIndices, indices)
			}
			if !slices.EqualFunc(values, test.expValues, bytes.Equal) {
				t.Fatalf("Unexpected values, want: %v, got: %v", test.expValues, values)
			}
		}
	}
}

func TestReassembleFullNode(t *testing.T) {
	var fn fullnodeEncoder
	for i := 0; i < 16; i++ {
		if rand.Intn(2) == 0 {
			fn.Children[i] = testrand.Bytes(32)
		}
	}
	buf := rlp.NewEncoderBuffer(nil)
	fn.encode(buf)
	enc := buf.ToBytes()

	// Generate a list of diffs
	var (
		values  [][][]byte
		indices [][]int
	)
	for i := 0; i < 10; i++ {
		var (
			pos       = make(map[int]struct{})
			poslist   []int
			valuelist [][]byte
		)
		for j := 0; j < 3; j++ {
			p := rand.Intn(16)
			if _, ok := pos[p]; ok {
				continue
			}
			pos[p] = struct{}{}

			nh := testrand.Bytes(32)
			diff, _ := rlp.EncodeToBytes(nh)
			poslist = append(poslist, p)
			valuelist = append(valuelist, diff)
			fn.Children[p] = nh
		}
		values = append(values, valuelist)
		indices = append(indices, poslist)
	}
	reassembled, err := ReassembleNode(enc, values, indices)
	if err != nil {
		t.Fatalf("Failed to re-assemble full node %v", err)
	}
	buf2 := rlp.NewEncoderBuffer(nil)
	fn.encode(buf2)
	enc2 := buf2.ToBytes()
	if !reflect.DeepEqual(enc2, reassembled) {
		t.Fatalf("Unexpeted reassembled node")
	}
}

func TestReassembleShortNode(t *testing.T) {
	var ln leafNodeEncoder
	ln.Key = hexToCompact(keybytesToHex(testrand.Bytes(10)))
	ln.Val = testrand.Bytes(10)
	buf := rlp.NewEncoderBuffer(nil)
	ln.encode(buf)
	enc := buf.ToBytes()

	// Generate a list of diffs
	var (
		values  [][][]byte
		indices [][]int
	)
	for i := 0; i < 10; i++ {
		val := testrand.Bytes(10)
		ln.Val = val
		diff, _ := rlp.EncodeToBytes(val)
		values = append(values, [][]byte{diff})
		indices = append(indices, []int{1})
	}
	reassembled, err := ReassembleNode(enc, values, indices)
	if err != nil {
		t.Fatalf("Failed to re-assemble full node %v", err)
	}
	buf2 := rlp.NewEncoderBuffer(nil)
	ln.encode(buf2)
	enc2 := buf2.ToBytes()
	if !reflect.DeepEqual(enc2, reassembled) {
		t.Fatalf("Unexpeted reassembled node")
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkEncodeShortNode
// BenchmarkEncodeShortNode-8   	16878850	        70.81 ns/op	      48 B/op	       1 allocs/op
func BenchmarkEncodeShortNode(b *testing.B) {
	node := &shortNode{
		Key: []byte{0x1, 0x2},
		Val: hashNode(randBytes(32)),
	}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		nodeToBytes(node)
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkEncodeFullNode
// BenchmarkEncodeFullNode-8   	 4323273	       284.4 ns/op	     576 B/op	       1 allocs/op
func BenchmarkEncodeFullNode(b *testing.B) {
	node := &fullNode{}
	for i := 0; i < 16; i++ {
		node.Children[i] = hashNode(randBytes(32))
	}
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		nodeToBytes(node)
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkDecodeShortNode
// BenchmarkDecodeShortNode-8   	 7925638	       151.0 ns/op	     157 B/op	       4 allocs/op
func BenchmarkDecodeShortNode(b *testing.B) {
	node := &shortNode{
		Key: []byte{0x1, 0x2},
		Val: hashNode(randBytes(32)),
	}
	blob := nodeToBytes(node)
	hash := crypto.Keccak256(blob)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mustDecodeNode(hash, blob)
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkDecodeShortNodeUnsafe
// BenchmarkDecodeShortNodeUnsafe-8   	 9027476	       128.6 ns/op	     109 B/op	       3 allocs/op
func BenchmarkDecodeShortNodeUnsafe(b *testing.B) {
	node := &shortNode{
		Key: []byte{0x1, 0x2},
		Val: hashNode(randBytes(32)),
	}
	blob := nodeToBytes(node)
	hash := crypto.Keccak256(blob)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mustDecodeNodeUnsafe(hash, blob)
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkDecodeFullNode
// BenchmarkDecodeFullNode-8   	 1597462	       761.9 ns/op	    1280 B/op	      18 allocs/op
func BenchmarkDecodeFullNode(b *testing.B) {
	node := &fullNode{}
	for i := 0; i < 16; i++ {
		node.Children[i] = hashNode(randBytes(32))
	}
	blob := nodeToBytes(node)
	hash := crypto.Keccak256(blob)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mustDecodeNode(hash, blob)
	}
}

// goos: darwin
// goarch: arm64
// pkg: github.com/ethereum/go-ethereum/trie
// BenchmarkDecodeFullNodeUnsafe
// BenchmarkDecodeFullNodeUnsafe-8   	 1789070	       687.1 ns/op	     704 B/op	      17 allocs/op
func BenchmarkDecodeFullNodeUnsafe(b *testing.B) {
	node := &fullNode{}
	for i := 0; i < 16; i++ {
		node.Children[i] = hashNode(randBytes(32))
	}
	blob := nodeToBytes(node)
	hash := crypto.Keccak256(blob)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mustDecodeNodeUnsafe(hash, blob)
	}
}
