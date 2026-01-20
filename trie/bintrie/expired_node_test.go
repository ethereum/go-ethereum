// Copyright 2026 go-ethereum Authors
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

package bintrie

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/trie/archive"
)

func TestExpiredNodeSerializeDeserialize(t *testing.T) {
	testCases := []struct {
		offset uint64
		size   uint64
	}{
		{0, 0},
		{1, 100},
		{255, 1024},
		{256, 4096},
		{1 << 16, 1 << 20},
		{1 << 32, 1 << 32},
		{1<<64 - 1, 1<<64 - 1},
	}

	for _, tc := range testCases {
		original := &expiredNode{Offset: tc.offset, Size: tc.size, depth: 5}
		serialized := SerializeNode(original)

		deserialized, err := DeserializeNode(serialized, 5)
		if err != nil {
			t.Fatalf("failed to deserialize expired node with offset %d, size %d: %v", tc.offset, tc.size, err)
		}

		expNode, ok := deserialized.(*expiredNode)
		if !ok {
			t.Fatalf("deserialized node is not an expired node, got %T", deserialized)
		}

		if expNode.Offset != original.Offset {
			t.Errorf("offset mismatch: got %d, want %d", expNode.Offset, original.Offset)
		}

		if expNode.Size != original.Size {
			t.Errorf("size mismatch: got %d, want %d", expNode.Size, original.Size)
		}

		if expNode.depth != original.depth {
			t.Errorf("depth mismatch: got %d, want %d", expNode.depth, original.depth)
		}
	}
}

func TestExpiredNodeSerializedFormat(t *testing.T) {
	node := &expiredNode{Offset: 0x0102030405060708, Size: 0x1112131415161718, depth: 0}
	serialized := SerializeNode(node)

	expected := []byte{
		nodeTypeExpired,
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	}
	if !bytes.Equal(serialized, expected) {
		t.Errorf("serialized format mismatch: got %x, want %x", serialized, expected)
	}
}

func TestExpiredNodeSerializedSize(t *testing.T) {
	node := &expiredNode{Offset: 12345, Size: 6789, depth: 0}
	serialized := SerializeNode(node)

	if len(serialized) != NodeTypeBytes+2*archive.OffsetSize {
		t.Errorf("serialized size mismatch: got %d, want %d", len(serialized), NodeTypeBytes+2*archive.OffsetSize)
	}
}

func TestExpiredNodeInvalidLength(t *testing.T) {
	invalidCases := [][]byte{
		{nodeTypeExpired},
		{nodeTypeExpired, 0x01},
		{nodeTypeExpired, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		{nodeTypeExpired, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f},
		{nodeTypeExpired, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11},
	}

	for _, buf := range invalidCases {
		_, err := DeserializeNode(buf, 0)
		if err == nil {
			t.Errorf("expected error for buffer length %d, got nil", len(buf))
		}
	}
}

func TestExpiredNodeHash(t *testing.T) {
	node := &expiredNode{Offset: 100, depth: 5}
	hash := node.Hash()

	if hash != (common.Hash{}) {
		t.Errorf("expected zero hash, got %x", hash)
	}
}

func TestExpiredNodeGetHeight(t *testing.T) {
	node := &expiredNode{Offset: 100, depth: 5}
	height := node.GetHeight()

	if height != 0 {
		t.Errorf("expected height 0, got %d", height)
	}
}

func TestExpiredNodeCollectNodes(t *testing.T) {
	node := &expiredNode{Offset: 100, depth: 5}
	called := false
	err := node.CollectNodes(nil, func(path []byte, n BinaryNode) {
		called = true
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if called {
		t.Error("flush function should not be called for expired nodes")
	}
}

func TestExpiredNodeToDot(t *testing.T) {
	node := &expiredNode{Offset: 12345, depth: 5}
	dot := node.toDot("parent", "path")

	if dot == "" {
		t.Error("toDot should return non-empty string")
	}
}

func TestExpiredNodeCopy(t *testing.T) {
	resolver := func(offset, size uint64) ([]*archive.Record, error) {
		return nil, nil
	}

	original := &expiredNode{
		Offset:          12345,
		Size:            6789,
		depth:           5,
		archiveResolver: resolver,
	}

	copied := original.Copy()
	copiedExp, ok := copied.(*expiredNode)
	if !ok {
		t.Fatalf("copied node is not an expired node, got %T", copied)
	}

	if copiedExp.Offset != original.Offset {
		t.Errorf("offset mismatch: got %d, want %d", copiedExp.Offset, original.Offset)
	}

	if copiedExp.Size != original.Size {
		t.Errorf("size mismatch: got %d, want %d", copiedExp.Size, original.Size)
	}

	if copiedExp.depth != original.depth {
		t.Errorf("depth mismatch: got %d, want %d", copiedExp.depth, original.depth)
	}

	if copiedExp.archiveResolver == nil {
		t.Error("archive resolver was not copied")
	}
}

func TestExpiredNodeNoResolver(t *testing.T) {
	node := &expiredNode{Offset: 100, depth: 5}

	_, err := node.Get(make([]byte, 32), nil)
	if !errors.Is(err, archive.ErrNoResolver) {
		t.Errorf("Get: expected archive.ErrNoResolver, got %v", err)
	}

	_, err = node.Insert(make([]byte, 32), make([]byte, 32), nil, 0)
	if !errors.Is(err, archive.ErrNoResolver) {
		t.Errorf("Insert: expected archive.ErrNoResolver, got %v", err)
	}

	_, err = node.GetValuesAtStem(make([]byte, StemSize), nil)
	if !errors.Is(err, archive.ErrNoResolver) {
		t.Errorf("GetValuesAtStem: expected archive.ErrNoResolver, got %v", err)
	}

	_, err = node.InsertValuesAtStem(make([]byte, StemSize), make([][]byte, StemNodeWidth), nil, 0)
	if !errors.Is(err, archive.ErrNoResolver) {
		t.Errorf("InsertValuesAtStem: expected archive.ErrNoResolver, got %v", err)
	}
}

func TestExpiredNodeWithResolver(t *testing.T) {
	var key [32]byte
	copy(key[:StemSize], make([]byte, StemSize))
	key[StemSize] = 0

	var values [StemNodeWidth][]byte
	values[0] = make([]byte, HashSize)
	copy(values[0], []byte("testvalue"))

	stemNode := &StemNode{
		Stem:   key[:StemSize],
		Values: values[:],
		depth:  5,
	}
	serializedStem := SerializeNode(stemNode)

	resolver := func(offset, size uint64) ([]*archive.Record, error) {
		if offset == 100 {
			return []*archive.Record{{Value: serializedStem}}, nil
		}
		return nil, errors.New("unknown offset")
	}

	node := &expiredNode{
		Offset:          100,
		Size:            uint64(len(serializedStem)),
		depth:           5,
		archiveResolver: resolver,
	}

	vals, err := node.GetValuesAtStem(key[:StemSize], nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if vals == nil {
		t.Fatal("expected non-nil values")
	}

	if !bytes.HasPrefix(vals[0], []byte("testvalue")) {
		t.Errorf("value mismatch: got %q", vals[0])
	}
}

func TestExpiredNodeDepth(t *testing.T) {
	node := &expiredNode{Offset: 100, depth: 42}
	if node.Depth() != 42 {
		t.Errorf("expected depth 42, got %d", node.Depth())
	}
}

func TestExpiredNodeSetArchiveResolver(t *testing.T) {
	node := &expiredNode{Offset: 100, depth: 5}

	if node.archiveResolver != nil {
		t.Error("expected nil archive resolver initially")
	}

	resolver := func(offset, size uint64) ([]*archive.Record, error) {
		return nil, nil
	}
	node.SetArchiveResolver(resolver)

	if node.archiveResolver == nil {
		t.Error("expected non-nil archive resolver after setting")
	}
}
