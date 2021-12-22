// Copyright 2017 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/params"
)

// memorySizeFunc returns the required size, and whether the operation overflowed a uint64
type memorySizeFunc func(*Stack) (size uint64, overflow bool)

// memoryGasCost calculates the quadratic gas for memory expansion. It does so
// only for the memory region that is expanded, not the total memory.
func memoryGasCost(mem *Memory, newMemSize uint64) (uint64, error) {
	if newMemSize == 0 {
		return 0, nil
	}
	// The maximum that will fit in a uint64 is max_word_count - 1. Anything above
	// that will result in an overflow. Additionally, a newMemSize which results in
	// a newMemSizeWords larger than 0xFFFFFFFF will cause the square operation to
	// overflow. The constant 0x1FFFFFFFE0 is the highest number that can be used
	// without overflowing the gas calculation.
	if newMemSize > 0x1FFFFFFFE0 {
		return 0, ErrGasUintOverflow
	}
	newMemSizeWords := toWordSize(newMemSize)
	newMemSize = newMemSizeWords * 32

	if newMemSize > uint64(mem.Len()) {
		square := newMemSizeWords * newMemSizeWords
		linCoef := newMemSizeWords * params.MemoryGas
		quadCoef := square / params.QuadCoeffDiv
		newTotalFee := linCoef + quadCoef

		fee := newTotalFee - mem.lastGasCost
		mem.lastGasCost = newTotalFee

		return fee, nil
	}
	return 0, nil
}

// memoryGasCost finds memory size required for instruction, by calling provided memorySizeFunc
// then finds corresponding gas charge for memory expansion
func memoryGasCostAndSize(stack *Stack, mem *Memory, sizeFunc memorySizeFunc) (uint64, uint64, error) {
	memSize, overflow := sizeFunc(stack)
	if overflow {
		return 0, 0, ErrGasUintOverflow
	}
	// memory is expanded in words of 32 bytes. Gas is also calculated in words.
	memorySize, overflow := math.SafeMul(toWordSize(memSize), 32)
	if overflow {
		return 0, 0, ErrGasUintOverflow
	}
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, 0, err
	}
	return gas, memorySize, nil
}

// memoryCopierGas creates the gas functions for the following opcodes, and takes
// the stack positions of the operand which determine the memory offsset and size of
// the data to copy as argument:
// CALLDATACOPY (offset stack position 0, len stack position 2)
// CODECOPY (offset stack position 0, len stack position 2)
// EXTCODECOPY (offset stack position 1, len stack position 3)
// RETURNDATACOPY (offset stack position 0, len stack position 2)
func memoryCopierGas(offsetStackPos, lenStackPos int) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
		// Gas for expanding the memory
		memSize, overflow := calcMemSize64(stack.Back(offsetStackPos), stack.Back(lenStackPos))
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}
		// memory is expanded in words of 32 bytes. Gas is also calculated in words.
		memorySize, overflow := math.SafeMul(toWordSize(memSize), 32)
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}
		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return 0, 0, err
		}

		// And gas for copying data, charged per word at param.CopyGas
		words, overflow := stack.Back(lenStackPos).Uint64WithOverflow()
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}

		if words, overflow = math.SafeMul(toWordSize(words), params.CopyGas); overflow {
			return 0, 0, ErrGasUintOverflow
		}

		if gas, overflow = math.SafeAdd(gas, words); overflow {
			return 0, 0, ErrGasUintOverflow
		}
		return gas, memorySize, nil
	}
}

var (
	gasCallDataCopy   = memoryCopierGas(0, 2)
	gasCodeCopy       = memoryCopierGas(0, 2)
	gasExtCodeCopy    = memoryCopierGas(1, 3)
	gasReturnDataCopy = memoryCopierGas(0, 2)
)

func gasSStore(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	var (
		y, x    = stack.Back(1), stack.Back(0)
		current = evm.StateDB.GetState(contract.Address(), x.Bytes32())
	)
	// The legacy gas metering only takes into consideration the current state
	// Legacy rules should be applied if we are in Petersburg (removal of EIP-1283)
	// OR Constantinople is not active
	if evm.chainRules.IsPetersburg || !evm.chainRules.IsConstantinople {
		// This checks for 3 scenario's and calculates gas accordingly:
		//
		// 1. From a zero-value address to a non-zero value         (NEW VALUE)
		// 2. From a non-zero value address to a zero-value address (DELETE)
		// 3. From a non-zero to a non-zero                         (CHANGE)
		switch {
		case current == (common.Hash{}) && y.Sign() != 0: // 0 => non 0
			return params.SstoreSetGas, 0, nil
		case current != (common.Hash{}) && y.Sign() == 0: // non 0 => 0
			evm.StateDB.AddRefund(params.SstoreRefundGas)
			return params.SstoreClearGas, 0, nil
		default: // non 0 => non 0 (or 0 => 0)
			return params.SstoreResetGas, 0, nil
		}
	}
	// The new gas metering is based on net gas costs (EIP-1283):
	//
	// 1. If current value equals new value (this is a no-op), 200 gas is deducted.
	// 2. If current value does not equal new value
	//   2.1. If original value equals current value (this storage slot has not been changed by the current execution context)
	//     2.1.1. If original value is 0, 20000 gas is deducted.
	// 	   2.1.2. Otherwise, 5000 gas is deducted. If new value is 0, add 15000 gas to refund counter.
	// 	2.2. If original value does not equal current value (this storage slot is dirty), 200 gas is deducted. Apply both of the following clauses.
	// 	  2.2.1. If original value is not 0
	//       2.2.1.1. If current value is 0 (also means that new value is not 0), remove 15000 gas from refund counter. We can prove that refund counter will never go below 0.
	//       2.2.1.2. If new value is 0 (also means that current value is not 0), add 15000 gas to refund counter.
	// 	  2.2.2. If original value equals new value (this storage slot is reset)
	//       2.2.2.1. If original value is 0, add 19800 gas to refund counter.
	// 	     2.2.2.2. Otherwise, add 4800 gas to refund counter.
	value := common.Hash(y.Bytes32())
	if current == value { // noop (1)
		return params.NetSstoreNoopGas, 0, nil
	}
	original := evm.StateDB.GetCommittedState(contract.Address(), x.Bytes32())
	if original == current {
		if original == (common.Hash{}) { // create slot (2.1.1)
			return params.NetSstoreInitGas, 0, nil
		}
		if value == (common.Hash{}) { // delete slot (2.1.2b)
			evm.StateDB.AddRefund(params.NetSstoreClearRefund)
		}
		return params.NetSstoreCleanGas, 0, nil // write existing slot (2.1.2)
	}
	if original != (common.Hash{}) {
		if current == (common.Hash{}) { // recreate slot (2.2.1.1)
			evm.StateDB.SubRefund(params.NetSstoreClearRefund)
		} else if value == (common.Hash{}) { // delete slot (2.2.1.2)
			evm.StateDB.AddRefund(params.NetSstoreClearRefund)
		}
	}
	if original == value {
		if original == (common.Hash{}) { // reset to original inexistent slot (2.2.2.1)
			evm.StateDB.AddRefund(params.NetSstoreResetClearRefund)
		} else { // reset to original existing slot (2.2.2.2)
			evm.StateDB.AddRefund(params.NetSstoreResetRefund)
		}
	}
	return params.NetSstoreDirtyGas, 0, nil
}

// 0. If *gasleft* is less than or equal to 2300, fail the current call.
// 1. If current value equals new value (this is a no-op), SLOAD_GAS is deducted.
// 2. If current value does not equal new value:
//   2.1. If original value equals current value (this storage slot has not been changed by the current execution context):
//     2.1.1. If original value is 0, SSTORE_SET_GAS (20K) gas is deducted.
//     2.1.2. Otherwise, SSTORE_RESET_GAS gas is deducted. If new value is 0, add SSTORE_CLEARS_SCHEDULE to refund counter.
//   2.2. If original value does not equal current value (this storage slot is dirty), SLOAD_GAS gas is deducted. Apply both of the following clauses:
//     2.2.1. If original value is not 0:
//       2.2.1.1. If current value is 0 (also means that new value is not 0), subtract SSTORE_CLEARS_SCHEDULE gas from refund counter.
//       2.2.1.2. If new value is 0 (also means that current value is not 0), add SSTORE_CLEARS_SCHEDULE gas to refund counter.
//     2.2.2. If original value equals new value (this storage slot is reset):
//       2.2.2.1. If original value is 0, add SSTORE_SET_GAS - SLOAD_GAS to refund counter.
//       2.2.2.2. Otherwise, add SSTORE_RESET_GAS - SLOAD_GAS gas to refund counter.
func gasSStoreEIP2200(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	// If we fail the minimum gas availability invariant, fail (0)
	if contract.Gas <= params.SstoreSentryGasEIP2200 {
		return 0, 0, errors.New("not enough gas for reentrancy sentry")
	}
	// Gas sentry honoured, do the actual gas calculation based on the stored value
	var (
		y, x    = stack.Back(1), stack.Back(0)
		current = evm.StateDB.GetState(contract.Address(), x.Bytes32())
	)
	value := common.Hash(y.Bytes32())

	if current == value { // noop (1)
		return params.SloadGasEIP2200, 0, nil
	}
	original := evm.StateDB.GetCommittedState(contract.Address(), x.Bytes32())
	if original == current {
		if original == (common.Hash{}) { // create slot (2.1.1)
			return params.SstoreSetGasEIP2200, 0, nil
		}
		if value == (common.Hash{}) { // delete slot (2.1.2b)
			evm.StateDB.AddRefund(params.SstoreClearsScheduleRefundEIP2200)
		}
		return params.SstoreResetGasEIP2200, 0, nil // write existing slot (2.1.2)
	}
	if original != (common.Hash{}) {
		if current == (common.Hash{}) { // recreate slot (2.2.1.1)
			evm.StateDB.SubRefund(params.SstoreClearsScheduleRefundEIP2200)
		} else if value == (common.Hash{}) { // delete slot (2.2.1.2)
			evm.StateDB.AddRefund(params.SstoreClearsScheduleRefundEIP2200)
		}
	}
	if original == value {
		if original == (common.Hash{}) { // reset to original inexistent slot (2.2.2.1)
			evm.StateDB.AddRefund(params.SstoreSetGasEIP2200 - params.SloadGasEIP2200)
		} else { // reset to original existing slot (2.2.2.2)
			evm.StateDB.AddRefund(params.SstoreResetGasEIP2200 - params.SloadGasEIP2200)
		}
	}
	return params.SloadGasEIP2200, 0, nil // dirty update (2.2)
}

func makeGasLog(n uint64) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
		requestedSize, overflow := stack.Back(1).Uint64WithOverflow()
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}

		memSize, overflow := calcMemSize64(stack.Back(0), stack.Back(1))
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}
		// memory is expanded in words of 32 bytes. Gas is also calculated in words.
		memorySize, overflow := math.SafeMul(toWordSize(memSize), 32)
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}
		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return 0, 0, err
		}

		if gas, overflow = math.SafeAdd(gas, params.LogGas); overflow {
			return 0, 0, ErrGasUintOverflow
		}
		if gas, overflow = math.SafeAdd(gas, n*params.LogTopicGas); overflow {
			return 0, 0, ErrGasUintOverflow
		}

		var memorySizeGas uint64
		if memorySizeGas, overflow = math.SafeMul(requestedSize, params.LogDataGas); overflow {
			return 0, 0, ErrGasUintOverflow
		}
		if gas, overflow = math.SafeAdd(gas, memorySizeGas); overflow {
			return 0, 0, ErrGasUintOverflow
		}
		return gas, memorySize, nil
	}
}

func gasKeccak256(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	gas, memorySize, err := memoryGasCostAndSize(stack, mem, memoryKeccak256)
	if err != nil {
		return 0, 0, err
	}
	wordGas, overflow := stack.Back(1).Uint64WithOverflow()
	if overflow {
		return 0, 0, ErrGasUintOverflow
	}
	if wordGas, overflow = math.SafeMul(toWordSize(wordGas), params.Keccak256WordGas); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	if gas, overflow = math.SafeAdd(gas, wordGas); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	return gas, memorySize, nil
}

// makePureMemoryGasCost is used by several operations, which aside from their
// static cost have a dynamic cost which is solely based on the memory
// expansion
func makePureMemoryGasCost(offsetStackPos, lenStackPos int) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
		memSize, overflow := calcMemSize64(stack.Back(offsetStackPos), stack.Back(lenStackPos))
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}
		// memory is expanded in words of 32 bytes. Gas is also calculated in words.
		memorySize, overflow := math.SafeMul(toWordSize(memSize), 32)
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}
		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return 0, 0, err
		}
		return gas, memorySize, nil
	}
}

var (
	gasReturn = makePureMemoryGasCost(0, 1)
	gasRevert = makePureMemoryGasCost(0, 1)
	gasCreate = makePureMemoryGasCost(1, 2)
)

func makePureMemoryGasCostWithUint(offsetStackPos int, length64 uint64) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
		memSize, overflow := calcMemSize64WithUint(stack.Back(offsetStackPos), length64)
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}
		// memory is expanded in words of 32 bytes. Gas is also calculated in words.
		memorySize, overflow := math.SafeMul(toWordSize(memSize), 32)
		if overflow {
			return 0, 0, ErrGasUintOverflow
		}
		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return 0, 0, err
		}
		return gas, memorySize, nil
	}
}

var (
	gasMLoad   = makePureMemoryGasCostWithUint(0, 32)
	gasMStore8 = makePureMemoryGasCostWithUint(0, 1)
	gasMStore  = makePureMemoryGasCostWithUint(0, 32)
)

func gasCreate2(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	gas, memorySize, err := memoryGasCostAndSize(stack, mem, memoryCreate2)
	if err != nil {
		return 0, 0, err
	}
	wordGas, overflow := stack.Back(2).Uint64WithOverflow()
	if overflow {
		return 0, 0, ErrGasUintOverflow
	}
	if wordGas, overflow = math.SafeMul(toWordSize(wordGas), params.Keccak256WordGas); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	if gas, overflow = math.SafeAdd(gas, wordGas); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	return gas, memorySize, nil
}

func gasExpFrontier(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	expByteLen := uint64((stack.data[stack.len()-2].BitLen() + 7) / 8)

	var (
		gas      = expByteLen * params.ExpByteFrontier // no overflow check required. Max is 256 * ExpByte gas
		overflow bool
	)
	if gas, overflow = math.SafeAdd(gas, params.ExpGas); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	return gas, 0, nil
}

func gasExpEIP158(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	expByteLen := uint64((stack.data[stack.len()-2].BitLen() + 7) / 8)

	var (
		gas      = expByteLen * params.ExpByteEIP158 // no overflow check required. Max is 256 * ExpByte gas
		overflow bool
	)
	if gas, overflow = math.SafeAdd(gas, params.ExpGas); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	return gas, 0, nil
}

func gasCall(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	var (
		gas            uint64
		transfersValue = !stack.Back(2).IsZero()
		address        = common.Address(stack.Back(1).Bytes20())
	)
	if evm.chainRules.IsEIP158 {
		if transfersValue && evm.StateDB.Empty(address) {
			gas += params.CallNewAccountGas
		}
	} else if !evm.StateDB.Exist(address) {
		gas += params.CallNewAccountGas
	}
	if transfersValue {
		gas += params.CallValueTransferGas
	}
	memoryGas, memorySize, err := memoryGasCostAndSize(stack, mem, memoryCall)
	if err != nil {
		return 0, 0, err
	}
	var overflow bool
	if gas, overflow = math.SafeAdd(gas, memoryGas); overflow {
		return 0, 0, ErrGasUintOverflow
	}

	evm.callGasTemp, err = callGas(evm.chainRules.IsEIP150, contract.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, 0, err
	}
	if gas, overflow = math.SafeAdd(gas, evm.callGasTemp); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	return gas, memorySize, nil
}

func gasCallCode(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	memoryGas, memorySize, err := memoryGasCostAndSize(stack, mem, memoryCall)
	if err != nil {
		return 0, 0, err
	}
	var (
		gas      uint64
		overflow bool
	)
	if stack.Back(2).Sign() != 0 {
		gas += params.CallValueTransferGas
	}
	if gas, overflow = math.SafeAdd(gas, memoryGas); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	evm.callGasTemp, err = callGas(evm.chainRules.IsEIP150, contract.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, 0, err
	}
	if gas, overflow = math.SafeAdd(gas, evm.callGasTemp); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	return gas, memorySize, nil
}

func gasDelegateCall(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	gas, memorySize, err := memoryGasCostAndSize(stack, mem, memoryDelegateCall)
	if err != nil {
		return 0, 0, err
	}
	evm.callGasTemp, err = callGas(evm.chainRules.IsEIP150, contract.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, 0, err
	}
	var overflow bool
	if gas, overflow = math.SafeAdd(gas, evm.callGasTemp); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	return gas, memorySize, nil
}

func gasStaticCall(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	gas, memorySize, err := memoryGasCostAndSize(stack, mem, memoryStaticCall)
	if err != nil {
		return 0, 0, err
	}
	evm.callGasTemp, err = callGas(evm.chainRules.IsEIP150, contract.Gas, gas, stack.Back(0))
	if err != nil {
		return 0, 0, err
	}
	var overflow bool
	if gas, overflow = math.SafeAdd(gas, evm.callGasTemp); overflow {
		return 0, 0, ErrGasUintOverflow
	}
	return gas, memorySize, nil
}

func gasSelfdestruct(evm *EVM, contract *Contract, stack *Stack, mem *Memory) (uint64, uint64, error) {
	var gas uint64
	// EIP150 homestead gas reprice fork:
	if evm.chainRules.IsEIP150 {
		gas = params.SelfdestructGasEIP150
		var address = common.Address(stack.Back(0).Bytes20())

		if evm.chainRules.IsEIP158 {
			// if empty and transfers value
			if evm.StateDB.Empty(address) && evm.StateDB.GetBalance(contract.Address()).Sign() != 0 {
				gas += params.CreateBySelfdestructGas
			}
		} else if !evm.StateDB.Exist(address) {
			gas += params.CreateBySelfdestructGas
		}
	}

	if !evm.StateDB.HasSuicided(contract.Address()) {
		evm.StateDB.AddRefund(params.SelfdestructRefundGas)
	}
	return gas, 0, nil
}
