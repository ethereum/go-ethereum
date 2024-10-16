// Copyright 2024 The go-ethereum Authors
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

package exex

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Chain provides read access to Geth's internal chain object.
type Chain interface {
	// TODO(karalabe): Wrap chain config into an exex interface
	// Config() *params.ChainConfig

	// Head retrieves the current head block's header from the canonical chain.
	Head() *types.Header

	// Header retrieves a block header with the given number from the canonical
	// chain. Headers on side-chains are not exposed by the Chain interface.
	Header(number uint64) *types.Header

	// Block retrieves an entire block with the given number from the canonical
	// chain. Blocks on side-chains are not exposed by the Chain interface.
	Block(number uint64) *types.Block

	// State retrieves a state accessor at a given root hash.
	State(root common.Hash) State

	// Receipts retrieves a set of receipts belonging to all transactions within
	// a block from the canonical chain. Receipts on side-chains are not exposed
	// by the Chain interface.
	Receipts(number uint64) []*types.Receipt
}
