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

package misc

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// ApplyEIP7997 inserts the deterministic deployment factory into the state as an
// irregular state transition, as specified by EIP-7997. The factory is a keyless
// CREATE2 factory that, once present at the canonical address on every EVM chain,
// allows contracts to be deployed at identical addresses across chains.
func ApplyEIP7997(statedb vm.StateDB) {
	contractHash := statedb.GetCodeHash(params.DeterministicFactoryAddress)
	nonce := statedb.GetNonce(params.DeterministicFactoryAddress)

	// Reject the irregular state transition if the destination doesn't
	// satisfy the deployment condition.
	if nonce != 0 || (contractHash != (common.Hash{}) && contractHash != types.EmptyCodeHash) {
		return
	}
	if !statedb.Exist(params.DeterministicFactoryAddress) {
		statedb.CreateAccount(params.DeterministicFactoryAddress)
	}
	statedb.CreateContract(params.DeterministicFactoryAddress)
	statedb.SetCode(params.DeterministicFactoryAddress, params.DeterministicFactoryCode, tracing.CodeChangeUnspecified)
	statedb.SetNonce(params.DeterministicFactoryAddress, 1, tracing.NonceChangeNewContract)
}
