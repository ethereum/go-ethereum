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

var transitionStatusByteCode []byte = []byte{
	0x60, 0x00, // PUSH1 <calldata offset>
	0x35,       // CALLDATALOAD
	0x54,       // SLOAD
	0x60, 0x00, // PUSH1 <mem dest>
	0x52,       // MSTORE
	0x60, 0x20, // PUSH1 <return size>
	0x60, 0x00, // PUSH1 <return offset>
	0xf3, // RETURN
}

func InitializeBinaryTransitionRegistry(statedb *state.StateDB) {
	statedb.SetCode(params.BinaryTransitionRegistryAddress, transitionStatusByteCode, tracing.CodeChangeUnspecified)
	statedb.SetNonce(params.BinaryTransitionRegistryAddress, 1, tracing.NonceChangeUnspecified)
	statedb.SetState(params.BinaryTransitionRegistryAddress, common.Hash{}, common.Hash{1})
}
