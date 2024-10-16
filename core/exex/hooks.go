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

// ReorgHook is called when the chain head is updated to a different parent than
// the previous head. In this case previously applied state changes need to be
// rolled back, and state changes from a sidechain need to be applied.
//
// This method is called with a set of header being operated on and the direction
// of the operation, usually both directions being called one after the other:
//
//   - If revert == true, the given headers are being rolled back, they are in
//     reverse order, headers[0] being the previous chain head, and the last item
//     being the olders block getting undone.
//   - If revert == false, the given headers are being applied after the rollback,
//     they are in forward order, headers[0] being the oldest block being applied
//     and the last item being the newest getting applied. Note, the chain head
//     that triggered the reorg will arrive in the HeadHook.
//
// The reason the reorg event it "emitted" in two parts is for both operations to
// have access to a unified singletone view of the chain. An alternative would be
// to pass in both the reverted and applied headers at the same time, but that
// would require chain accessorts to support sidechains, which complicate APIs.
type ReorgHook = func(headers []*types.Header, revert bool)

// FinalHook is called when the chain finalizes a past block.
type FinalHook = func(header *types.Header)
