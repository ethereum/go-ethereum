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

package vm

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/params"
)

// ErrSystemContractCodeChange is returned when SETCODEFROM executes in the
// context of a system contract.
var ErrSystemContractCodeChange = errors.New("code change in system contract")

// isSystemContract reports whether addr is called by the protocol from the
// system address during block processing. The code of these accounts cannot be
// replaced by SETCODEFROM. The deposit contract, whose logs are read but which
// is never called, is not one of them.
func isSystemContract(addr common.Address) bool {
	switch addr {
	case params.BeaconRootsAddress,
		params.HistoryStorageAddress,
		params.WithdrawalQueueAddress,
		params.ConsolidationQueueAddress,
		params.BuilderDepositAddress,
		params.BuilderExitAddress:
		return true
	}
	return false
}

// enable8298 enables the SETCODEFROM instruction (EIP-8298).
func enable8298(jt *JumpTable) {
	jt[SETCODEFROM] = &operation{
		execute:    opSetCodeFrom,
		dynamicGas: gasSetCodeFrom,
		minStack:   minStack(1, 1),
		maxStack:   maxStack(1, 1),
	}
}

// gasSetCodeFrom charges SETCODEFROM_SOURCE_GAS, the source account access
// plus WARM_ACCESS for inspecting its code, and warms the source. A static
// context or a system contract as the current account halt before the source
// is accessed.
func gasSetCodeFrom(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	if isSystemContract(contract.Address()) {
		return GasCosts{}, ErrSystemContractCodeChange
	}
	source := common.Address(stack.peek().Bytes20())
	cost := params.WarmAccountAccessAmsterdam
	if !evm.StateDB.AddressInAccessList(source) {
		// If the caller cannot afford the cost, this change will be rolled back
		evm.StateDB.AddAddressToAccessList(source)
		cost = params.ColdAccountAccessAmsterdam
	}
	return GasCosts{ExecutionGas: cost + params.WarmAccountAccessAmsterdam}, nil
}

// opSetCodeFrom sets the code hash of the current account to the code hash of
// the source account, if the source holds regular deployed code and was not
// created in the current transaction. The running frame keeps executing the
// code it loaded.
func opSetCodeFrom(pc *uint64, evm *EVM, scope *ScopeContext) ([]byte, error) {
	var (
		slot   = scope.Stack.peek()
		source = common.Address(slot.Bytes20())
		code   = evm.StateDB.GetCode(source)
	)
	// A non-existent account has no code, so the emptiness check also covers
	// the existence requirement.
	if len(code) == 0 || code[0] == 0xEF || evm.StateDB.IsNewContract(source) {
		slot.Clear()
		return nil, nil
	}
	// A valid source pays for comparing the current account's code hash with
	// its own, and for writing it when the two differ.
	if !scope.Contract.chargeExecution(params.WarmAccountAccessAmsterdam, evm.Config.Tracer, tracing.GasChangeUnspecified) {
		return nil, ErrOutOfGas
	}
	var (
		self     = scope.Contract.Address()
		codeHash = evm.StateDB.GetCodeHash(source)
	)
	if evm.StateDB.GetCodeHash(self) != codeHash {
		if !scope.Contract.chargeExecution(params.AccountWriteAmsterdam, evm.Config.Tracer, tracing.GasChangeUnspecified) {
			return nil, ErrOutOfGas
		}
		evm.StateDB.AdoptCode(self, codeHash, code)
	}
	slot.SetOne()
	return nil, nil
}
