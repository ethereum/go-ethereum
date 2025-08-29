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

func (h HashedNode) InsertValuesAtStem(key []byte, values [][]byte, resolver NodeResolverFn, depth int) (BinaryNode, error) {
	return nil, errors.New("insertValuesAtStem not implemented for hashed node")
}

func (h HashedNode) toDot(parent string, path string) string {
	me := fmt.Sprintf("hash%s", path)
	ret := fmt.Sprintf("%s [label=\"%x\"]\n", me, h)
	ret = fmt.Sprintf("%s %s -> %s\n", ret, parent, me)
	return ret
}

func (h HashedNode) CollectNodes([]byte, NodeFlushFn) error {
	return errors.New("collectNodes not implemented for hashed node")
}

func (h HashedNode) GetHeight() int {
	panic("tried to get the height of a hashed node, this is a bug")
}
