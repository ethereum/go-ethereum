// Copyright 2025 go-ethereum Authors
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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type HashedNode common.Hash

func (h HashedNode) Get(_ []byte, _ NodeResolverFn) ([]byte, error) {
	panic("not implemented") // TODO: Implement
}

func (h HashedNode) Insert(key []byte, value []byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	return nil, errors.New("insert not implemented for hashed node")
}

func (h HashedNode) Copy() BinaryNode {
	nh := common.Hash(h)
	return HashedNode(nh)
}

func (h HashedNode) Hash() common.Hash {
	return common.Hash(h)
}

func (h HashedNode) GetValuesAtStem(_ []byte, _ NodeResolverFn) ([][]byte, error) {
	return nil, errors.New("attempted to get values from an unresolved node")
}

func (h HashedNode) InsertValuesAtStem(stem []byte, values [][]byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	// Step 1: Generate the path for this node's position in the tree
	path, err := keyToPath(depth, stem)
	if err != nil {
		return nil, fmt.Errorf("InsertValuesAtStem path generation error: %w", err)
	}

	if resolver == nil {
		return nil, errors.New("InsertValuesAtStem resolve error: resolver is nil")
	}

	// Step 2: Resolve the hashed node to get the actual node data
	data, err := resolver(path, common.Hash(h))
	if err != nil {
		return nil, fmt.Errorf("InsertValuesAtStem resolve error: %w", err)
	}

	// Step 3: Deserialize the resolved data into a concrete node
	node, err := DeserializeNode(data, depth)
	if err != nil {
		return nil, fmt.Errorf("InsertValuesAtStem node deserialization error: %w", err)
	}

	// Step 4: Call InsertValuesAtStem on the resolved concrete node
	return node.InsertValuesAtStem(stem, values, resolver, depth)
}

func (h HashedNode) toDot(parent string, path string) string {
	me := fmt.Sprintf("hash%s", path)
	ret := fmt.Sprintf("%s [label=\"%x\"]\n", me, h)
	ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
	return ret
}

func (h HashedNode) CollectNodes([]byte, NodeFlushFn) error {
	// HashedNodes are already persisted in the database and don't need to be collected.
	return nil
}

func (h HashedNode) GetHeight() int {
	panic("tried to get the height of a hashed node, this is a bug")
}
