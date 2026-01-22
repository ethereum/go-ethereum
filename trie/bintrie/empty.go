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
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

type Empty struct{}

func (e Empty) Get(_ []byte, _ NodeResolverFn) ([]byte, error) {
	return nil, nil
}

func (e Empty) Insert(key []byte, value []byte, _ NodeResolverFn, depth int) (BinaryNode, error) {
	var values [256][]byte
	values[key[31]] = value
	return &StemNode{
		Stem:   slices.Clone(key[:31]),
		Values: values[:],
		depth:  depth,
	}, nil
}

func (e Empty) Copy() BinaryNode {
	return Empty{}
}

func (e Empty) Hash() common.Hash {
	return common.Hash{}
}

func (e Empty) GetValuesAtStem(_ []byte, _ NodeResolverFn) ([][]byte, error) {
	var values [256][]byte
	return values[:], nil
}

func (e Empty) InsertValuesAtStem(key []byte, values [][]byte, _ NodeResolverFn, depth int) (BinaryNode, error) {
	return &StemNode{
		Stem:   slices.Clone(key[:31]),
		Values: values,
		depth:  depth,
	}, nil
}

func (e Empty) CollectNodes(_ []byte, _ NodeFlushFn) error {
	return nil
}

func (e Empty) toDot(parent string, path string) string {
	return ""
}

func (e Empty) GetHeight() int {
	return 0
}
