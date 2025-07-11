// Copyright 2020 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func gasSStoreWithClearingRefund(evm *EVM, contract *Contract, slot256, value256 *uint256.Int, clearingRefund uint64) (uint64, error) {
	// If we fail the minimum gas availability invariant, fail (0)
	if contract.Gas <= params.SstoreSentryGasEIP2200 {
		return 0, errors.New("not enough gas for reentrancy sentry")
	}
	// Gas sentry honoured, do the actual gas calculation based on the stored value
	var (
		slot              = common.Hash(slot256.Bytes32())
		current, original = evm.StateDB.GetStateAndCommittedState(contract.Address(), slot)
		cost              = uint64(0)
	)
	// Check slot presence in the access list
	if _, slotPresent := evm.StateDB.SlotInAccessList(contract.Address(), slot); !slotPresent {
		cost = params.ColdSloadCostEIP2929
		// If the caller cannot afford the cost, this change will be rolled back
		evm.StateDB.AddSlotToAccessList(contract.Address(), slot)
	}
	value := common.Hash(value256.Bytes32())

	if current == value { // noop (1)
		// EIP 2200 original clause:
		//		return params.SloadGasEIP2200, nil
		return cost + params.WarmStorageReadCostEIP2929, nil // SLOAD_GAS
	}
	if original == current {
		if original == (common.Hash{}) { // create slot (2.1.1)
			return cost + params.SstoreSetGasEIP2200, nil
		}
		if value == (common.Hash{}) { // delete slot (2.1.2b)
			evm.StateDB.AddRefund(clearingRefund)
		}
		// EIP-2200 original clause:
		//		return params.SstoreResetGasEIP2200, nil // write existing slot (2.1.2)
		return cost + (params.SstoreResetGasEIP2200 - params.ColdSloadCostEIP2929), nil // write existing slot (2.1.2)
	}
	if original != (common.Hash{}) {
		if current == (common.Hash{}) { // recreate slot (2.2.1.1)
			evm.StateDB.SubRefund(clearingRefund)
		} else if value == (common.Hash{}) { // delete slot (2.2.1.2)
			evm.StateDB.AddRefund(clearingRefund)
		}
	}
	if original == value {
		if original == (common.Hash{}) { // reset to original inexistent slot (2.2.2.1)
			// EIP 2200 Original clause:
			//evm.StateDB.AddRefund(params.SstoreSetGasEIP2200 - params.SloadGasEIP2200)
			evm.StateDB.AddRefund(params.SstoreSetGasEIP2200 - params.WarmStorageReadCostEIP2929)
		} else { // reset to original existing slot (2.2.2.2)
			// EIP 2200 Original clause:
			//	evm.StateDB.AddRefund(params.SstoreResetGasEIP2200 - params.SloadGasEIP2200)
			// - SSTORE_RESET_GAS redefined as (5000 - COLD_SLOAD_COST)
			// - SLOAD_GAS redefined as WARM_STORAGE_READ_COST
			// Final: (5000 - COLD_SLOAD_COST) - WARM_STORAGE_READ_COST
			evm.StateDB.AddRefund((params.SstoreResetGasEIP2200 - params.ColdSloadCostEIP2929) - params.WarmStorageReadCostEIP2929)
		}
	}
	// EIP-2200 original clause:
	//return params.SloadGasEIP2200, nil // dirty update (2.2)
	return cost + params.WarmStorageReadCostEIP2929, nil // dirty update (2.2)
}

// gasSLoadEIP2929 calculates dynamic gas for SLOAD according to EIP-2929
// For SLOAD, if the (address, storage_key) pair (where address is the address of the contract
// whose storage is being read) is not yet in accessed_storage_keys,
// charge 2100 gas and add the pair to accessed_storage_keys.
// If the pair is already in accessed_storage_keys, charge 100 gas.
func gasSLoadEIP2929(evm *EVM, contract *Contract, loc *uint256.Int) (uint64, error) {
	slot := common.Hash(loc.Bytes32())
	// Check slot presence in the access list
	if _, slotPresent := evm.StateDB.SlotInAccessList(contract.Address(), slot); !slotPresent {
		// If the caller cannot afford the cost, this change will be rolled back
		// If he does afford it, we can skip checking the same thing later on, during execution
		evm.StateDB.AddSlotToAccessList(contract.Address(), slot)
		return params.ColdSloadCostEIP2929, nil
	}
	return params.WarmStorageReadCostEIP2929, nil
}

// gasExtCodeCopyEIP2929 implements extcodecopy according to EIP-2929
// EIP spec:
// > If the target is not in accessed_addresses,
// > charge COLD_ACCOUNT_ACCESS_COST gas, and add the address to accessed_addresses.
// > Otherwise, charge WARM_STORAGE_READ_COST gas.
func gasExtCodeCopyEIP2929(evm *EVM, mem *Memory, memorySize uint64, a, length *uint256.Int) (uint64, error) {
	// memory expansion first (dynamic part of pre-2929 implementation)
	gas, err := memoryCopierGas(mem, memorySize, length)
	if err != nil {
		return 0, err
	}
	addr := common.Address(a.Bytes20())
	// Check slot presence in the access list
	if !evm.StateDB.AddressInAccessList(addr) {
		evm.StateDB.AddAddressToAccessList(addr)
		var overflow bool
		// We charge (cold-warm), since 'warm' is already charged as constantGas
		if gas, overflow = math.SafeAdd(gas, params.ColdAccountAccessCostEIP2929-params.WarmStorageReadCostEIP2929); overflow {
			return 0, ErrGasUintOverflow
		}
		return gas, nil
	}
	return gas, nil
}

// gasEip2929AccountCheck checks whether the first stack item (as address) is present in the access list.
// If it is, this method returns '0', otherwise 'cold-warm' gas, presuming that the opcode using it
// is also using 'warm' as constant factor.
// This method is used by:
// - extcodehash,
// - extcodesize,
// - (ext) balance
func gasEip2929AccountCheck(evm *EVM, addr *uint256.Int) (uint64, error) {
	address := common.Address(addr.Bytes20())
	// Check slot presence in the access list
	if !evm.StateDB.AddressInAccessList(address) {
		// If the caller cannot afford the cost, this change will be rolled back
		evm.StateDB.AddAddressToAccessList(address)
		// The warm storage read cost is already charged as constantGas
		return params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929, nil
	}
	return 0, nil
}

func gasCallEIP2929(evm *EVM, contract *Contract, address *uint256.Int, oldCalculator func() (uint64, error)) (uint64, error) {
	addr := common.Address(address.Bytes20())
	// Check slot presence in the access list
	warmAccess := evm.StateDB.AddressInAccessList(addr)
	// The WarmStorageReadCostEIP2929 (100) is already deducted in the form of a constant cost, so
	// the cost to charge for cold access, if any, is Cold - Warm
	coldCost := params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929
	if !warmAccess {
		evm.StateDB.AddAddressToAccessList(addr)
		// Charge the remaining difference here already, to correctly calculate available
		// gas for call
		if !contract.UseGas(coldCost, evm.Config.Tracer.OnGasChange, tracing.GasChangeCallStorageColdAccess) {
			return 0, ErrOutOfGas
		}
	}
	// Now call the old calculator, which takes into account
	// - create new account
	// - transfer value
	// - memory expansion
	// - 63/64ths rule
	gas, err := oldCalculator()
	if warmAccess || err != nil {
		return gas, err
	}
	// In case of a cold access, we temporarily add the cold charge back, and also
	// add it to the returned gas. By adding it to the return, it will be charged
	// outside of this function, as part of the dynamic gas, and that will make it
	// also become correctly reported to tracers.
	contract.Gas += coldCost

	var overflow bool
	if gas, overflow = math.SafeAdd(gas, coldCost); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasSelfdestructEIP(evm *EVM, contract *Contract, addr *uint256.Int, refundsEnabled bool) (uint64, error) {
	var (
		gas     uint64
		address = common.Address(addr.Bytes20())
	)
	if !evm.StateDB.AddressInAccessList(address) {
		// If the caller cannot afford the cost, this change will be rolled back
		evm.StateDB.AddAddressToAccessList(address)
		gas = params.ColdAccountAccessCostEIP2929
	}
	// if empty and transfers value
	if evm.StateDB.Empty(address) && evm.StateDB.GetBalance(contract.Address()).Sign() != 0 {
		gas += params.CreateBySelfdestructGas
	}
	if refundsEnabled && !evm.StateDB.HasSelfDestructed(contract.Address()) {
		evm.StateDB.AddRefund(params.SelfdestructRefundGas)
	}
	return gas, nil
}

func gasCallEIP7702(evm *EVM, contract *Contract, address *uint256.Int, oldCalculator func() (uint64, error)) (uint64, error) {
	var (
		total uint64 // total dynamic gas used
		addr  = common.Address(address.Bytes20())
	)

	// Check slot presence in the access list
	if !evm.StateDB.AddressInAccessList(addr) {
		evm.StateDB.AddAddressToAccessList(addr)
		// The WarmStorageReadCostEIP2929 (100) is already deducted in the form of a constant cost, so
		// the cost to charge for cold access, if any, is Cold - Warm
		coldCost := params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929
		// Charge the remaining difference here already, to correctly calculate available
		// gas for call
		if !contract.UseGas(coldCost, evm.Config.Tracer.OnGasChange, tracing.GasChangeCallStorageColdAccess) {
			return 0, ErrOutOfGas
		}
		total += coldCost
	}

	// Check if code is a delegation and if so, charge for resolution.
	if target, ok := types.ParseDelegation(evm.StateDB.GetCode(addr)); ok {
		var cost uint64
		if evm.StateDB.AddressInAccessList(target) {
			cost = params.WarmStorageReadCostEIP2929
		} else {
			evm.StateDB.AddAddressToAccessList(target)
			cost = params.ColdAccountAccessCostEIP2929
		}
		if !contract.UseGas(cost, evm.Config.Tracer.OnGasChange, tracing.GasChangeCallStorageColdAccess) {
			return 0, ErrOutOfGas
		}
		total += cost
	}

	// Now call the old calculator, which takes into account
	// - create new account
	// - transfer value
	// - memory expansion
	// - 63/64ths rule
	old, err := oldCalculator()
	if err != nil {
		return old, err
	}

	// Temporarily add the gas charge back to the contract and return value. By
	// adding it to the return, it will be charged outside of this function, as
	// part of the dynamic gas. This will ensure it is correctly reported to
	// tracers.
	contract.Gas += total

	var overflow bool
	if total, overflow = math.SafeAdd(old, total); overflow {
		return 0, ErrGasUintOverflow
	}
	return total, nil
}
