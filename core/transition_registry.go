// Copyright 2026 The go-ethereum Authors
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

package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/params"
)

// transitionStatusByteCode is a minimal contract that returns a single 32-byte
// storage slot to the caller. CALLDATALOAD picks the slot index from the call
// input and SLOAD reads the value, which is then returned as a 32-byte word.
var transitionStatusByteCode = []byte{
	0x60, 0x00, // PUSH1 0
	0x35,       // CALLDATALOAD (slot index)
	0x54,       // SLOAD
	0x60, 0x00, // PUSH1 0
	0x52,       // MSTORE
	0x60, 0x20, // PUSH1 32
	0x60, 0x00, // PUSH1 0
	0xf3, // RETURN
}

// transitionRegistryBaseRootSlot is slot 5 of the transition registry, where
// the frozen MPT base root is stored. The slot indices match those decoded by
// overlay.LoadTransitionState; the layout is intentionally kept private to
// the core package so external callers go through these helpers.
var transitionRegistryBaseRootSlot = common.BytesToHash([]byte{5})

// InitializeBinaryTransitionRegistry deploys the binary transition registry
// system contract and marks the transition as started by writing 1 into slot
// 0. It must be called exactly once, on the first block after the UBT
// activation.
func InitializeBinaryTransitionRegistry(statedb *state.StateDB) {
	statedb.SetCode(params.BinaryTransitionRegistryAddress, transitionStatusByteCode, tracing.CodeChangeUnspecified)
	statedb.SetNonce(params.BinaryTransitionRegistryAddress, 1, tracing.NonceChangeUnspecified)
	statedb.SetState(params.BinaryTransitionRegistryAddress, common.Hash{}, common.Hash{1})
}

// WriteBinaryTransitionBaseRoot records the frozen MPT base root in slot 5 of
// the transition registry. This must be called on the first UBT block, right
// after InitializeBinaryTransitionRegistry, with the parent block's state
// root.
func WriteBinaryTransitionBaseRoot(statedb *state.StateDB, baseRoot common.Hash) {
	statedb.SetState(params.BinaryTransitionRegistryAddress, transitionRegistryBaseRootSlot, baseRoot)
}
