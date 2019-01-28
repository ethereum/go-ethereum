// Copyright 2019 The go-ethereum Authors
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

// Contains a batch of utility type declarations used by the tests. As the node
// operates on unique types, a lot of them are needed to check various features.

package types

import "github.com/ethereum/go-ethereum/common"

// NodeType for explicitly setting type of node
type NodeType string

const (
	Unknown   NodeType = "Unknown"
	Leaf      NodeType = "Leaf"
	Extension NodeType = "Extension"
	Branch    NodeType = "Branch"
	Removed   NodeType = "Removed" // used to represent pathes which have been emptied
)

// StateNode holds the data for a single state diff node
type StateNode struct {
	NodeType     NodeType      `json:"nodeType"        gencodec:"required"`
	Path         []byte        `json:"path"            gencodec:"required"`
	NodeValue    []byte        `json:"value"           gencodec:"required"`
	StorageNodes []StorageNode `json:"storage"`
	LeafKey      []byte        `json:"leafKey"`
}

// StorageNode holds the data for a single storage diff node
type StorageNode struct {
	NodeType  NodeType `json:"nodeType"        gencodec:"required"`
	Path      []byte   `json:"path"            gencodec:"required"`
	NodeValue []byte   `json:"value"           gencodec:"required"`
	LeafKey   []byte   `json:"leafKey"`
}

// CodeAndCodeHash struct for holding codehash => code mappings
// we can't use an actual map because they are not rlp serializable
type CodeAndCodeHash struct {
	Hash common.Hash `json:"codeHash"`
	Code []byte      `json:"code"`
}

type StateNodeSink func(StateNode) error
type StorageNodeSink func(StorageNode) error
type CodeSink func(CodeAndCodeHash) error
