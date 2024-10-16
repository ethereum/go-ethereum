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

import "github.com/ethereum/go-ethereum/core/types"

// InitHook is called when the chain gets initialized within Geth.
type InitHook = func(chain Chain)

// CloseHook is called when the chain gets torn down within Geth.
type CloseHook = func()

// HeadHook is called when the chain head block is updated.
//
//   - During full sync, this will be called for each block
//   - During snap sync, this will be called from pivot onwards
//   - In sync, this will be called on fork-choice updates
type HeadHook = func(head *types.Header)

// TODO(karalabe): This is interesting. We need to keep events in sync with the
// user's chain access capabilities. We either need to provide side-chain access,
// which gets nasty fast; or we need to reorg in lockstep; or we need two events
// one to start a reorg (going back) and one having finished (going forward).
// type ReorgHook = func(old, new *types.Header) error
