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

package statediff

import (
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/statediff/types"
)

// Subscription struct holds our subscription channels
type Subscription struct {
	PayloadChan chan<- types.Payload
	QuitChan    chan<- bool
}

// AccountMap is a mapping of hex encoded path => account wrapper
type AccountMap map[string]accountWrapper

// accountWrapper is used to temporary associate the unpacked node with its raw values
type accountWrapper struct {
	Account   *state.Account
	NodeType  types.NodeType
	Path      []byte
	NodeValue []byte
	LeafKey   []byte
}
