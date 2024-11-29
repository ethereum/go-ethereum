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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"
)

func memoryGasCost(pc uint64, scope *ScopeContext, newMemSize uint64) (uint64, error) {
	return evmmaxMemoryGasCost(pc, scope, newMemSize, scope.modExtState.AllocSize())
}

// evmmaxMemory calculates the quadratic gas for memory expansion. It does so
// only for the memory region that is expanded, not the total memory.  It uses
// the modified EVMMAX memory expansion rule: consider the size of memory to
// include EVM memory and the memory allocated by all active field contexts.
func evmmaxMemoryGasCost(pc uint64, scope *ScopeContext, newMemSize uint64, newEVMMAXMemSize uint64) (uint64, error) {
	if newMemSize == 0 && newEVMMAXMemSize == 0 {
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
	newMemSizePadded := newMemSizeWords * 32

	curEVMMAXMemSize := scope.modExtState.AllocSize()
	curEVMMAXMemSizePadded := toWordSize(curEVMMAXMemSize) * 32
	newEVMMAXMemSizePadded := toWordSize(newEVMMAXMemSize) * 32

	// if newEVMMAXMemSize + newEVMMemSize > curEVMMAXMemSize + curEVMMemSize
	if newMemSizePadded > uint64(scope.Memory.Len()) || newEVMMAXMemSizePadded > curEVMMAXMemSizePadded {
		// if this is called by the invocation of SETUPX, the new evm memory is
		// 0, but we still need it to compute the fee
		if newMemSize <= uint64(scope.Memory.Len()) {
			newMemSize = uint64(scope.Memory.Len())
		}
		// new effective mem size for the purpose of gas charging is the sum of
		// evmmax memory and evm memory padded to a multiple of 32 bytes.
		newEffectiveMemSizeWords := toWordSize(newEVMMAXMemSize + newMemSize)
		square := newEffectiveMemSizeWords * newEffectiveMemSizeWords
		linCoef := newEffectiveMemSizeWords * params.MemoryGas
		quadCoef := square / params.QuadCoeffDiv
		newTotalFee := linCoef + quadCoef

		fee := newTotalFee - scope.Memory.lastGasCost
		scope.Memory.lastGasCost = newTotalFee

		return fee, nil
	}
	return 0, nil
}

// memoryCopierGas creates the gas functions for the following opcodes, and takes
// the stack position of the operand which determines the size of the data to copy
// as argument:
// CALLDATACOPY (stack position 2)
// CODECOPY (stack position 2)
// MCOPY (stack position 2)
// EXTCODECOPY (stack position 3)
// RETURNDATACOPY (stack position 2)
func memoryCopierGas(stackpos int) gasFunc {
	return func(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
		// Gas for expanding the memory
		gas, err := memoryGasCost(pc, scope, memorySize)
		if err != nil {
			return 0, err
		}
		// And gas for copying data, charged per word at param.CopyGas
		words, overflow := scope.Stack.Back(stackpos).Uint64WithOverflow()
		if overflow {
			return 0, ErrGasUintOverflow
		}

		if words, overflow = math.SafeMul(toWordSize(words), params.CopyGas); overflow {
			return 0, ErrGasUintOverflow
		}

		if gas, overflow = math.SafeAdd(gas, words); overflow {
			return 0, ErrGasUintOverflow
		}
		return gas, nil
	}
}

var (
	gasCallDataCopy   = memoryCopierGas(2)
	gasCodeCopy       = memoryCopierGas(2)
	gasMcopy          = memoryCopierGas(2)
	gasExtCodeCopy    = memoryCopierGas(3)
	gasReturnDataCopy = memoryCopierGas(2)
)

func gasSStore(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	var (
		y, x    = scope.Stack.Back(1), scope.Stack.Back(0)
		current = evm.StateDB.GetState(scope.Contract.Address(), x.Bytes32())
	)
	// The legacy gas metering only takes into consideration the current state
	// Legacy rules should be applied if we are in Petersburg (removal of EIP-1283)
	// OR Constantinople is not active
	if evm.chainRules.IsPetersburg || !evm.chainRules.IsConstantinople {
		// This checks for 3 scenarios and calculates gas accordingly:
		//
		// 1. From a zero-value address to a non-zero value         (NEW VALUE)
		// 2. From a non-zero value address to a zero-value address (DELETE)
		// 3. From a non-zero to a non-zero                         (CHANGE)
		switch {
		case current == (common.Hash{}) && y.Sign() != 0: // 0 => non 0
			return params.SstoreSetGas, nil
		case current != (common.Hash{}) && y.Sign() == 0: // non 0 => 0
			evm.StateDB.AddRefund(params.SstoreRefundGas)
			return params.SstoreClearGas, nil
		default: // non 0 => non 0 (or 0 => 0)
			return params.SstoreResetGas, nil
		}
	}

	// The new gas metering is based on net gas costs (EIP-1283):
	//
	// (1.) If current value equals new value (this is a no-op), 200 gas is deducted.
	// (2.) If current value does not equal new value
	//	(2.1.) If original value equals current value (this storage slot has not been changed by the current execution context)
	//		(2.1.1.) If original value is 0, 20000 gas is deducted.
	//		(2.1.2.) Otherwise, 5000 gas is deducted. If new value is 0, add 15000 gas to refund counter.
	//	(2.2.) If original value does not equal current value (this storage slot is dirty), 200 gas is deducted. Apply both of the following clauses.
	//		(2.2.1.) If original value is not 0
	//			(2.2.1.1.) If current value is 0 (also means that new value is not 0), remove 15000 gas from refund counter. We can prove that refund counter will never go below 0.
	//			(2.2.1.2.) If new value is 0 (also means that current value is not 0), add 15000 gas to refund counter.
	//		(2.2.2.) If original value equals new value (this storage slot is reset)
	//			(2.2.2.1.) If original value is 0, add 19800 gas to refund counter.
	//			(2.2.2.2.) Otherwise, add 4800 gas to refund counter.
	value := common.Hash(y.Bytes32())
	if current == value { // noop (1)
		return params.NetSstoreNoopGas, nil
	}
	original := evm.StateDB.GetCommittedState(scope.Contract.Address(), x.Bytes32())
	if original == current {
		if original == (common.Hash{}) { // create slot (2.1.1)
			return params.NetSstoreInitGas, nil
		}
		if value == (common.Hash{}) { // delete slot (2.1.2b)
			evm.StateDB.AddRefund(params.NetSstoreClearRefund)
		}
		return params.NetSstoreCleanGas, nil // write existing slot (2.1.2)
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
	return params.NetSstoreDirtyGas, nil
}

// Here come the EIP2200 rules:
//
//	(0.) If *gasleft* is less than or equal to 2300, fail the current call.
//	(1.) If current value equals new value (this is a no-op), SLOAD_GAS is deducted.
//	(2.) If current value does not equal new value:
//		(2.1.) If original value equals current value (this storage slot has not been changed by the current execution context):
//			(2.1.1.) If original value is 0, SSTORE_SET_GAS (20K) gas is deducted.
//			(2.1.2.) Otherwise, SSTORE_RESET_GAS gas is deducted. If new value is 0, add SSTORE_CLEARS_SCHEDULE to refund counter.
//		(2.2.) If original value does not equal current value (this storage slot is dirty), SLOAD_GAS gas is deducted. Apply both of the following clauses:
//			(2.2.1.) If original value is not 0:
//				(2.2.1.1.) If current value is 0 (also means that new value is not 0), subtract SSTORE_CLEARS_SCHEDULE gas from refund counter.
//				(2.2.1.2.) If new value is 0 (also means that current value is not 0), add SSTORE_CLEARS_SCHEDULE gas to refund counter.
//			(2.2.2.) If original value equals new value (this storage slot is reset):
//				(2.2.2.1.) If original value is 0, add SSTORE_SET_GAS - SLOAD_GAS to refund counter.
//				(2.2.2.2.) Otherwise, add SSTORE_RESET_GAS - SLOAD_GAS gas to refund counter.
func gasSStoreEIP2200(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	// If we fail the minimum gas availability invariant, fail (0)
	if scope.Contract.Gas <= params.SstoreSentryGasEIP2200 {
		return 0, errors.New("not enough gas for reentrancy sentry")
	}
	// Gas sentry honoured, do the actual gas calculation based on the stored value
	var (
		y, x    = scope.Stack.Back(1), scope.Stack.Back(0)
		current = evm.StateDB.GetState(scope.Contract.Address(), x.Bytes32())
	)
	value := common.Hash(y.Bytes32())

	if current == value { // noop (1)
		return params.SloadGasEIP2200, nil
	}
	original := evm.StateDB.GetCommittedState(scope.Contract.Address(), x.Bytes32())
	if original == current {
		if original == (common.Hash{}) { // create slot (2.1.1)
			return params.SstoreSetGasEIP2200, nil
		}
		if value == (common.Hash{}) { // delete slot (2.1.2b)
			evm.StateDB.AddRefund(params.SstoreClearsScheduleRefundEIP2200)
		}
		return params.SstoreResetGasEIP2200, nil // write existing slot (2.1.2)
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
	return params.SloadGasEIP2200, nil // dirty update (2.2)
}

func makeGasLog(n uint64) gasFunc {
	return func(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
		requestedSize, overflow := scope.Stack.Back(1).Uint64WithOverflow()
		if overflow {
			return 0, ErrGasUintOverflow
		}

		gas, err := memoryGasCost(pc, scope, memorySize)
		if err != nil {
			return 0, err
		}

		if gas, overflow = math.SafeAdd(gas, params.LogGas); overflow {
			return 0, ErrGasUintOverflow
		}
		if gas, overflow = math.SafeAdd(gas, n*params.LogTopicGas); overflow {
			return 0, ErrGasUintOverflow
		}

		var memorySizeGas uint64
		if memorySizeGas, overflow = math.SafeMul(requestedSize, params.LogDataGas); overflow {
			return 0, ErrGasUintOverflow
		}
		if gas, overflow = math.SafeAdd(gas, memorySizeGas); overflow {
			return 0, ErrGasUintOverflow
		}
		return gas, nil
	}
}

func gasKeccak256(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(pc, scope, memorySize)
	if err != nil {
		return 0, err
	}
	wordGas, overflow := scope.Stack.Back(1).Uint64WithOverflow()
	if overflow {
		return 0, ErrGasUintOverflow
	}
	if wordGas, overflow = math.SafeMul(toWordSize(wordGas), params.Keccak256WordGas); overflow {
		return 0, ErrGasUintOverflow
	}
	if gas, overflow = math.SafeAdd(gas, wordGas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

// pureMemoryGascost is used by several operations, which aside from their
// static cost have a dynamic cost which is solely based on the memory
// expansion
func pureMemoryGascost(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	return memoryGasCost(pc, scope, memorySize)
}

var (
	gasReturn  = pureMemoryGascost
	gasRevert  = pureMemoryGascost
	gasMLoad   = pureMemoryGascost
	gasMStore8 = pureMemoryGascost
	gasMStore  = pureMemoryGascost
	gasCreate  = pureMemoryGascost
)

func gasCreate2(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(pc, scope, memorySize)
	if err != nil {
		return 0, err
	}
	wordGas, overflow := scope.Stack.Back(2).Uint64WithOverflow()
	if overflow {
		return 0, ErrGasUintOverflow
	}
	if wordGas, overflow = math.SafeMul(toWordSize(wordGas), params.Keccak256WordGas); overflow {
		return 0, ErrGasUintOverflow
	}
	if gas, overflow = math.SafeAdd(gas, wordGas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasCreateEip3860(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(pc, scope, memorySize)
	if err != nil {
		return 0, err
	}
	size, overflow := scope.Stack.Back(2).Uint64WithOverflow()
	if overflow {
		return 0, ErrGasUintOverflow
	}
	if size > params.MaxInitCodeSize {
		return 0, fmt.Errorf("%w: size %d", ErrMaxInitCodeSizeExceeded, size)
	}
	// Since size <= params.MaxInitCodeSize, these multiplication cannot overflow
	moreGas := params.InitCodeWordGas * ((size + 31) / 32)
	if gas, overflow = math.SafeAdd(gas, moreGas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}
func gasCreate2Eip3860(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(pc, scope, memorySize)
	if err != nil {
		return 0, err
	}
	size, overflow := scope.Stack.Back(2).Uint64WithOverflow()
	if overflow {
		return 0, ErrGasUintOverflow
	}
	if size > params.MaxInitCodeSize {
		return 0, fmt.Errorf("%w: size %d", ErrMaxInitCodeSizeExceeded, size)
	}
	// Since size <= params.MaxInitCodeSize, these multiplication cannot overflow
	moreGas := (params.InitCodeWordGas + params.Keccak256WordGas) * ((size + 31) / 32)
	if gas, overflow = math.SafeAdd(gas, moreGas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasExpFrontier(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	expByteLen := uint64((scope.Stack.data[scope.Stack.len()-2].BitLen() + 7) / 8)

	var (
		gas      = expByteLen * params.ExpByteFrontier // no overflow check required. Max is 256 * ExpByte gas
		overflow bool
	)
	if gas, overflow = math.SafeAdd(gas, params.ExpGas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasExpEIP158(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	expByteLen := uint64((scope.Stack.data[scope.Stack.len()-2].BitLen() + 7) / 8)

	var (
		gas      = expByteLen * params.ExpByteEIP158 // no overflow check required. Max is 256 * ExpByte gas
		overflow bool
	)
	if gas, overflow = math.SafeAdd(gas, params.ExpGas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasCall(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	var (
		gas            uint64
		transfersValue = !scope.Stack.Back(2).IsZero()
		address        = common.Address(scope.Stack.Back(1).Bytes20())
	)
	if evm.chainRules.IsEIP158 {
		if transfersValue && evm.StateDB.Empty(address) {
			gas += params.CallNewAccountGas
		}
	} else if !evm.StateDB.Exist(address) {
		gas += params.CallNewAccountGas
	}
	if transfersValue && !evm.chainRules.IsEIP4762 {
		gas += params.CallValueTransferGas
	}
	memoryGas, err := memoryGasCost(pc, scope, memorySize)
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = math.SafeAdd(gas, memoryGas); overflow {
		return 0, ErrGasUintOverflow
	}
	if evm.chainRules.IsEIP4762 {
		if transfersValue {
			gas, overflow = math.SafeAdd(gas, evm.AccessEvents.ValueTransferGas(scope.Contract.Address(), address))
			if overflow {
				return 0, ErrGasUintOverflow
			}
		}
	}
	evm.callGasTemp, err = callGas(evm.chainRules.IsEIP150, scope.Contract.Gas, gas, scope.Stack.Back(0))
	if err != nil {
		return 0, err
	}
	if gas, overflow = math.SafeAdd(gas, evm.callGasTemp); overflow {
		return 0, ErrGasUintOverflow
	}

	return gas, nil
}

func gasCallCode(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	memoryGas, err := memoryGasCost(pc, scope, memorySize)
	if err != nil {
		return 0, err
	}
	var (
		gas      uint64
		overflow bool
	)
	if scope.Stack.Back(2).Sign() != 0 && !evm.chainRules.IsEIP4762 {
		gas += params.CallValueTransferGas
	}
	if gas, overflow = math.SafeAdd(gas, memoryGas); overflow {
		return 0, ErrGasUintOverflow
	}
	if evm.chainRules.IsEIP4762 {
		address := common.Address(scope.Stack.Back(1).Bytes20())
		transfersValue := !scope.Stack.Back(2).IsZero()
		if transfersValue {
			gas, overflow = math.SafeAdd(gas, evm.AccessEvents.ValueTransferGas(scope.Contract.Address(), address))
			if overflow {
				return 0, ErrGasUintOverflow
			}
		}
	}
	evm.callGasTemp, err = callGas(evm.chainRules.IsEIP150, scope.Contract.Gas, gas, scope.Stack.Back(0))
	if err != nil {
		return 0, err
	}
	if gas, overflow = math.SafeAdd(gas, evm.callGasTemp); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasDelegateCall(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(pc, scope, memorySize)
	if err != nil {
		return 0, err
	}
	evm.callGasTemp, err = callGas(evm.chainRules.IsEIP150, scope.Contract.Gas, gas, scope.Stack.Back(0))
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = math.SafeAdd(gas, evm.callGasTemp); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasStaticCall(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(pc, scope, memorySize)
	if err != nil {
		return 0, err
	}
	evm.callGasTemp, err = callGas(evm.chainRules.IsEIP150, scope.Contract.Gas, gas, scope.Stack.Back(0))
	if err != nil {
		return 0, err
	}
	var overflow bool
	if gas, overflow = math.SafeAdd(gas, evm.callGasTemp); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasSelfdestruct(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	var gas uint64
	// EIP150 homestead gas reprice fork:
	if evm.chainRules.IsEIP150 {
		gas = params.SelfdestructGasEIP150
		var address = common.Address(scope.Stack.Back(0).Bytes20())

		if evm.chainRules.IsEIP158 {
			// if empty and transfers value
			if evm.StateDB.Empty(address) && evm.StateDB.GetBalance(scope.Contract.Address()).Sign() != 0 {
				gas += params.CreateBySelfdestructGas
			}
		} else if !evm.StateDB.Exist(address) {
			gas += params.CreateBySelfdestructGas
		}
	}

	if !evm.StateDB.HasSelfDestructed(scope.Contract.Address()) {
		evm.StateDB.AddRefund(params.SelfdestructRefundGas)
	}
	return gas, nil
}

func gasExtCall(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	panic("not implemented")
}

func gasExtDelegateCall(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	panic("not implemented")
}
func gasExtStaticCall(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	panic("not implemented")
}

// gasEOFCreate returns the gas-cost for EOF-Create. Hashing charge needs to be
// deducted in the opcode itself, since it depends on the immediate
func gasEOFCreate(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	panic("not implemented")
}

func gasSetupx(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	if !scope.Stack.Back(0).IsUint64() || !scope.Stack.Back(2).IsUint64() || !scope.Stack.Back(3).IsUint64() {
		return 0, errors.New("one or more parameters overflows 64 bits")
	}

	modId := uint(scope.Stack.Back(0).Uint64())
	if scope.modExtState.alloced[modId] != nil {
		return 0, nil
	}

	modSize := scope.Stack.Back(2).Uint64()
	if modSize > 96 {
		// TODO: ensure returning error here consumes all evm call context gas
		return 0, fmt.Errorf("modulus cannot exceed 768 bits in width")
	}

	feAllocCount := scope.Stack.Back(3).Uint64()
	if feAllocCount > 256 {
		return 0, fmt.Errorf("cannot allocate more than 256 field elements per modulus id")
	}
	paddedModSize := (modSize + 7) / 8
	precompCost := uint64(params.SetupxPrecompCost[paddedModSize])

	// the size in bytes of the field element heap that this call to SETUPX is
	// allocating.
	allocSize := paddedModSize * feAllocCount

	// if the new evmmax memory alloc would exceed the maximum allowed, return an error
	if scope.modExtState.AllocSize()+allocSize > uint64(params.MaxFEAllocSize) {
		return 0, fmt.Errorf("call context evmmax allocation threshold exceeded")
	}

	// overflow error unchecked because we do not expand evm memory here,
	// and the maximum call-context allocatable memory + reasonable evm memory limit
	// will not overflow a uint64.
	memCost, _ := evmmaxMemoryGasCost(pc, scope, memorySize, allocSize)
	return precompCost + memCost, nil
}

func gasStorex(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	if scope.modExtState.active == nil {
		return 0, errors.New("no active mod state")
	}
	dst := scope.Stack.Back(0)
	src := scope.Stack.Back(1)
	count := scope.Stack.Back(2)

	if !src.IsUint64() || int(src.Uint64()) >= scope.Memory.Len() {
		return 0, errors.New("source index is out of bounds")
	}
	if !dst.IsUint64() || dst.Uint64() >= uint64(scope.modExtState.active.NumElems()) {
		return 0, errors.New("destination of copy out of bounds")
	}
	if !count.IsUint64() || count.Uint64() > uint64(scope.modExtState.active.NumElems()) {
		return 0, errors.New("count must be less than number of field elements in the active space")
	}
	storeSize := count.Uint64() * uint64(scope.modExtState.active.NumElems())
	if src.Uint64()+storeSize > uint64(scope.Memory.Len()) {
		return 0, errors.New("source of copy out of bounds of EVM memory")
	}

	if scope.modExtState.active.IsModulusBinary() {
		return toWordSize(storeSize) * params.CopyGas, nil
	} else {
		return count.Uint64() * uint64(params.MulmodxCost[int(scope.modExtState.active.ElemSize()/8)-1]), nil
	}
}

func gasLoadx(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	if scope.modExtState.active == nil {
		return 0, errors.New("no active mod state")
	}
	dst := scope.Stack.Back(0)
	src := scope.Stack.Back(1)
	count := scope.Stack.Back(2)

	if !src.IsUint64() || uint(src.Uint64()) >= scope.modExtState.active.NumElems() {
		return 0, errors.New("out of bounds copy source")
	}
	if !count.IsUint64() || uint(count.Uint64()) > scope.modExtState.active.NumElems() {
		return 0, errors.New("count must be less than number of field elements")
	}
	if last, overflow := math.SafeAdd(src.Uint64(), count.Uint64()); overflow || last > uint64(scope.modExtState.active.NumElems()) {
		return 0, errors.New("out of bounds copy source")
	}
	if !dst.IsUint64() {
		return 0, errors.New("out of bounds destination")
	}

	loadSize := count.Uint64() * uint64(scope.modExtState.active.ElemSize())
	last, overflow := math.SafeAdd(dst.Uint64(), loadSize)
	if overflow || last > uint64(scope.Memory.Len()) {
		return 0, errors.New("out of bounds destination")
	}

	if scope.modExtState.active.IsModulusBinary() {
		return toWordSize(loadSize) * params.CopyGas, nil
	} else {
		return count.Uint64() * uint64(params.MulmodxCost[int(scope.modExtState.active.ElemSize()/8)-1]), nil
	}
}

func gasEVMMAXArithOp(pc uint64, evm *EVM, scope *ScopeContext, memorySize uint64) (uint64, error) {
	if scope.modExtState.active == nil {
		return 0, errors.New("no active mod state")
	}
	_ = scope.Contract.Code[pc+7]
	out := uint(scope.Contract.Code[pc+1])
	out_stride := uint(scope.Contract.Code[pc+2])
	x := uint(scope.Contract.Code[pc+3])
	x_stride := uint(scope.Contract.Code[pc+4])
	y := uint(scope.Contract.Code[pc+5])
	y_stride := uint(scope.Contract.Code[pc+6])
	count := uint(scope.Contract.Code[pc+7])

	maxOffset := max(x+x_stride*count, y+y_stride*count, out+out_stride*count)
	// TODO: might not need to assert count == 0 ?
	if count == 0 || out_stride == 0 || maxOffset > scope.modExtState.active.NumElems() {
		return 0, errors.New("bad parameters")
	}
	// TODO: fill in gas costs with table lookup multiplied by count...
	return 1, nil
}
