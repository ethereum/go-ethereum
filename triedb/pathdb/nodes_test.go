// Copyright 2025 The go-ethereum Authors
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

package pathdb

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

func TestNodeSetEncode(t *testing.T) {
	nodes := make(map[common.Hash]map[string]*trienode.Node)
	nodes[common.Hash{}] = map[string]*trienode.Node{
		"":  trienode.New(crypto.Keccak256Hash([]byte{0x0}), []byte{0x0}),
		"1": trienode.New(crypto.Keccak256Hash([]byte{0x1}), []byte{0x1}),
		"2": trienode.New(crypto.Keccak256Hash([]byte{0x2}), []byte{0x2}),
	}
	nodes[common.Hash{0x1}] = map[string]*trienode.Node{
		"":  trienode.New(crypto.Keccak256Hash([]byte{0x0}), []byte{0x0}),
		"1": trienode.New(crypto.Keccak256Hash([]byte{0x1}), []byte{0x1}),
		"2": trienode.New(crypto.Keccak256Hash([]byte{0x2}), []byte{0x2}),
	}
	s := newNodeSet(nodes)

	buf := bytes.NewBuffer(nil)
	if err := s.encode(buf); err != nil {
		t.Fatalf("Failed to encode states, %v", err)
	}
	var dec nodeSet
	if err := dec.decode(rlp.NewStream(buf, 0)); err != nil {
		t.Fatalf("Failed to decode states, %v", err)
	}
	if !reflect.DeepEqual(s.accountNodes, dec.accountNodes) {
		t.Fatal("Unexpected account data")
	}
	if !reflect.DeepEqual(s.storageNodes, dec.storageNodes) {
		t.Fatal("Unexpected storage data")
	}
}

func TestNodeSetWithOriginEncode(t *testing.T) {
	nodes := make(map[common.Hash]map[string]*trienode.Node)
	nodes[common.Hash{}] = map[string]*trienode.Node{
		"":  trienode.New(crypto.Keccak256Hash([]byte{0x0}), []byte{0x0}),
		"1": trienode.New(crypto.Keccak256Hash([]byte{0x1}), []byte{0x1}),
		"2": trienode.New(crypto.Keccak256Hash([]byte{0x2}), []byte{0x2}),
	}
	nodes[common.Hash{0x1}] = map[string]*trienode.Node{
		"":  trienode.New(crypto.Keccak256Hash([]byte{0x0}), []byte{0x0}),
		"1": trienode.New(crypto.Keccak256Hash([]byte{0x1}), []byte{0x1}),
		"2": trienode.New(crypto.Keccak256Hash([]byte{0x2}), []byte{0x2}),
	}
	origins := make(map[common.Hash]map[string][]byte)
	origins[common.Hash{}] = map[string][]byte{
		"":  nil,
		"1": {0x1},
		"2": {0x2},
	}
	origins[common.Hash{0x1}] = map[string][]byte{
		"":  nil,
		"1": {0x1},
		"2": {0x2},
	}

	// Encode with origin set
	s := NewNodeSetWithOrigin(nodes, origins)

	buf := bytes.NewBuffer(nil)
	if err := s.encode(buf); err != nil {
		t.Fatalf("Failed to encode states, %v", err)
	}
	var dec nodeSetWithOrigin
	if err := dec.decode(rlp.NewStream(buf, 0)); err != nil {
		t.Fatalf("Failed to decode states, %v", err)
	}
	if !reflect.DeepEqual(s.accountNodes, dec.accountNodes) {
		t.Fatal("Unexpected account data")
	}
	if !reflect.DeepEqual(s.storageNodes, dec.storageNodes) {
		t.Fatal("Unexpected storage data")
	}
	if !reflect.DeepEqual(s.nodeOrigin, dec.nodeOrigin) {
		t.Fatal("Unexpected node origin data")
	}

	// Encode without origin set
	s = NewNodeSetWithOrigin(nodes, nil)

	buf = bytes.NewBuffer(nil)
	if err := s.encode(buf); err != nil {
		t.Fatalf("Failed to encode states, %v", err)
	}
	var dec2 nodeSetWithOrigin
	if err := dec2.decode(rlp.NewStream(buf, 0)); err != nil {
		t.Fatalf("Failed to decode states, %v", err)
	}
	if !reflect.DeepEqual(s.accountNodes, dec2.accountNodes) {
		t.Fatal("Unexpected account data")
	}
	if !reflect.DeepEqual(s.storageNodes, dec2.storageNodes) {
		t.Fatal("Unexpected storage data")
	}
	if len(dec2.nodeOrigin) != 0 {
		t.Fatal("unexpected node origin data")
	}
	if dec2.size != s.size {
		t.Fatalf("Unexpected data size, got: %d, want: %d", dec2.size, s.size)
	}
}
