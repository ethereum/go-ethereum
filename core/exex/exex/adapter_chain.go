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
	"github.com/ethereum/go-ethereum/core/exex"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
)

// gethChain provides dep-free read access to Geth's internal chain object.
//
// The methods here are not documented as this interface is not a public thing,
// rather it's a dynamic cast to break cross-package dependencies.
type gethChain interface {
	CurrentBlock() *types.Header
	GetHeaderByNumber(number uint64) *types.Header
	GetBlockByNumber(number uint64) *types.Block
	StateAt(root common.Hash) (*state.StateDB, error)
}

// chainAdapter is an adapter to convert Geth's internal blockchain (unstable
// and legacy API) into the exex chain interface (stable API).
type chainAdapter struct {
	chain gethChain
}

// wrapChain wraps a Geth internal chain object into an exex stable API.
func wrapChain(chain gethChain) exex.Chain {
	return &chainAdapter{chain: chain}
}

// Head retrieves the current head block's header from the canonical chain.
func (a *chainAdapter) Head() *types.Header {
	// Headers have public fields, copy to prevent modification
	return types.CopyHeader(a.chain.CurrentBlock())
}

// Header retrieves a block header with the given number from the canonical
// chain. Headers on side-chains are not exposed by the Chain interface.
func (a *chainAdapter) Header(number uint64) *types.Header {
	// Headers have public fields, copy to prevent modification
	if header := a.chain.GetHeaderByNumber(number); header != nil {
		return types.CopyHeader(header)
	}
	return nil
}

// Block retrieves a block header with the given number from the canonical
// chain. Blocks on side-chains are not exposed by the Chain interface.
func (a *chainAdapter) Block(number uint64) *types.Block {
	// Blocks don't have public fields, return live objects directly
	return a.chain.GetBlockByNumber(number)
}

// State retrieves a state accessor at a given root hash.
func (a *chainAdapter) State(root common.Hash) exex.State {
	state, err := a.chain.StateAt(root)
	if err != nil {
		return nil
	}
	return wrapState(state)
}
