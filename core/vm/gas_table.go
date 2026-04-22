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

// memoryCopierGas creates the gas functions for the following opcodes, and takes
// the stack position of the operand which determines the size of the data to copy
// as argument:
// CALLDATACOPY (stack position 2)
// CODECOPY (stack position 2)
// MCOPY (stack position 2)
// EXTCODECOPY (stack position 3)
// RETURNDATACOPY (stack position 2)
func memoryCopierGas(stackpos int) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
		// Gas for expanding the memory
		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return GasCosts{}, err
		}
		// And gas for copying data, charged per word at param.CopyGas
		words, overflow := stack.Back(stackpos).Uint64WithOverflow()
		if overflow {
			return GasCosts{}, ErrGasUintOverflow
		}

		if words, overflow = math.SafeMul(toWordSize(words), params.CopyGas); overflow {
			return GasCosts{}, ErrGasUintOverflow
		}

		if gas, overflow = math.SafeAdd(gas, words); overflow {
			return GasCosts{}, ErrGasUintOverflow
		}
		return GasCosts{RegularGas: gas}, nil
	}
}

var (
	gasCallDataCopy   = memoryCopierGas(2)
	gasCodeCopy       = memoryCopierGas(2)
	gasMcopy          = memoryCopierGas(2)
	gasExtCodeCopy    = memoryCopierGas(3)
	gasReturnDataCopy = memoryCopierGas(2)
)

func gasSStore(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	var (
		y, x              = stack.Back(1), stack.Back(0)
		current, original = evm.StateDB.GetStateAndCommittedState(contract.Address(), x.Bytes32())
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
			return GasCosts{RegularGas: params.SstoreSetGas}, nil
		case current != (common.Hash{}) && y.Sign() == 0: // non 0 => 0
			evm.StateDB.AddRefund(params.SstoreRefundGas)
			return GasCosts{RegularGas: params.SstoreClearGas}, nil
		default: // non 0 => non 0 (or 0 => 0)
			return GasCosts{RegularGas: params.SstoreResetGas}, nil
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
		return GasCosts{RegularGas: params.NetSstoreNoopGas}, nil
	}
	if original == current {
		if original == (common.Hash{}) { // create slot (2.1.1)
			return GasCosts{RegularGas: params.NetSstoreInitGas}, nil
		}
		if value == (common.Hash{}) { // delete slot (2.1.2b)
			evm.StateDB.AddRefund(params.NetSstoreClearRefund)
		}
		return GasCosts{RegularGas: params.NetSstoreCleanGas}, nil // write existing slot (2.1.2)
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
	return GasCosts{RegularGas: params.NetSstoreDirtyGas}, nil
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
func gasSStoreEIP2200(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	// If we fail the minimum gas availability invariant, fail (0)
	if contract.Gas.RegularGas <= params.SstoreSentryGasEIP2200 {
		return GasCosts{}, errors.New("not enough gas for reentrancy sentry")
	}
	// Gas sentry honoured, do the actual gas calculation based on the stored value
	var (
		y, x              = stack.Back(1), stack.Back(0)
		current, original = evm.StateDB.GetStateAndCommittedState(contract.Address(), x.Bytes32())
	)
	value := common.Hash(y.Bytes32())

	if current == value { // noop (1)
		return GasCosts{RegularGas: params.SloadGasEIP2200}, nil
	}
	if original == current {
		if original == (common.Hash{}) { // create slot (2.1.1)
			return GasCosts{RegularGas: params.SstoreSetGasEIP2200}, nil
		}
		if value == (common.Hash{}) { // delete slot (2.1.2b)
			evm.StateDB.AddRefund(params.SstoreClearsScheduleRefundEIP2200)
		}
		return GasCosts{RegularGas: params.SstoreResetGasEIP2200}, nil // write existing slot (2.1.2)
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
	return GasCosts{RegularGas: params.SloadGasEIP2200}, nil // dirty update (2.2)
}

func makeGasLog(n uint64) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
		requestedSize, overflow := stack.Back(1).Uint64WithOverflow()
		if overflow {
			return GasCosts{}, ErrGasUintOverflow
		}

		gas, err := memoryGasCost(mem, memorySize)
		if err != nil {
			return GasCosts{}, err
		}

		if gas, overflow = math.SafeAdd(gas, params.LogGas); overflow {
			return GasCosts{}, ErrGasUintOverflow
		}
		if gas, overflow = math.SafeAdd(gas, n*params.LogTopicGas); overflow {
			return GasCosts{}, ErrGasUintOverflow
		}

		var memorySizeGas uint64
		if memorySizeGas, overflow = math.SafeMul(requestedSize, params.LogDataGas); overflow {
			return GasCosts{}, ErrGasUintOverflow
		}
		if gas, overflow = math.SafeAdd(gas, memorySizeGas); overflow {
			return GasCosts{}, ErrGasUintOverflow
		}
		return GasCosts{RegularGas: gas}, nil
	}
}

func gasKeccak256(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return GasCosts{}, err
	}
	wordGas, overflow := stack.Back(1).Uint64WithOverflow()
	if overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	if wordGas, overflow = math.SafeMul(toWordSize(wordGas), params.Keccak256WordGas); overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	if gas, overflow = math.SafeAdd(gas, wordGas); overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	return GasCosts{RegularGas: gas}, nil
}

// pureMemoryGascost is used by several operations, which aside from their
// static cost have a dynamic cost which is solely based on the memory
// expansion
func pureMemoryGascost(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return GasCosts{}, err
	}
	return GasCosts{RegularGas: gas}, nil
}

var (
	gasReturn  = pureMemoryGascost
	gasRevert  = pureMemoryGascost
	gasMLoad   = pureMemoryGascost
	gasMStore8 = pureMemoryGascost
	gasMStore  = pureMemoryGascost
)

func gasCreate(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	return pureMemoryGascost(evm, contract, stack, mem, memorySize)
}

func gasCreate2(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return GasCosts{}, err
	}
	wordGas, overflow := stack.Back(2).Uint64WithOverflow()
	if overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	if wordGas, overflow = math.SafeMul(toWordSize(wordGas), params.Keccak256WordGas); overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	if gas, overflow = math.SafeAdd(gas, wordGas); overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	return GasCosts{RegularGas: gas}, nil
}

func gasCreateEip3860(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return GasCosts{}, err
	}
	size, overflow := stack.Back(2).Uint64WithOverflow()
	if overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	if err := CheckMaxInitCodeSize(&evm.chainRules, size); err != nil {
		return GasCosts{}, err
	}
	// Since size <= the protocol-defined maximum initcode size limit, these multiplication cannot overflow
	moreGas := params.InitCodeWordGas * ((size + 31) / 32)
	if gas, overflow = math.SafeAdd(gas, moreGas); overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	return GasCosts{RegularGas: gas}, nil
}

func gasCreate2Eip3860(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return GasCosts{}, err
	}
	size, overflow := stack.Back(2).Uint64WithOverflow()
	if overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	if err := CheckMaxInitCodeSize(&evm.chainRules, size); err != nil {
		return GasCosts{}, err
	}
	// Since size <= the protocol-defined maximum initcode size limit, these multiplication cannot overflow
	moreGas := (params.InitCodeWordGas + params.Keccak256WordGas) * ((size + 31) / 32)
	if gas, overflow = math.SafeAdd(gas, moreGas); overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	return GasCosts{RegularGas: gas}, nil
}

func gasExpFrontier(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	expByteLen := uint64((stack.data[stack.len()-2].BitLen() + 7) / 8)

	var (
		gas      = expByteLen * params.ExpByteFrontier // no overflow check required. Max is 256 * ExpByte gas
		overflow bool
	)
	if gas, overflow = math.SafeAdd(gas, params.ExpGas); overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	return GasCosts{RegularGas: gas}, nil
}

func gasExpEIP158(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	expByteLen := uint64((stack.data[stack.len()-2].BitLen() + 7) / 8)

	var (
		gas      = expByteLen * params.ExpByteEIP158 // no overflow check required. Max is 256 * ExpByte gas
		overflow bool
	)
	if gas, overflow = math.SafeAdd(gas, params.ExpGas); overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	return GasCosts{RegularGas: gas}, nil
}

var (
	gasCall         = makeCallVariantGasCost(gasCallIntrinsic)
	gasCallCode     = makeCallVariantGasCost(gasCallCodeIntrinsic)
	gasDelegateCall = makeCallVariantGasCost(gasDelegateCallIntrinsic)
	gasStaticCall   = makeCallVariantGasCost(gasStaticCallIntrinsic)
)

func makeCallVariantGasCost(intrinsicFunc intrinsicGasFunc) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
		intrinsic, err := intrinsicFunc(evm, contract, stack, mem, memorySize)
		if err != nil {
			return GasCosts{}, err
		}
		evm.callGasTemp, err = callGas(evm.chainRules.IsEIP150, contract.Gas.RegularGas, intrinsic, stack.Back(0))
		if err != nil {
			return GasCosts{}, err
		}
		gas, overflow := math.SafeAdd(intrinsic, evm.callGasTemp)
		if overflow {
			return GasCosts{}, ErrGasUintOverflow
		}
		return GasCosts{RegularGas: gas}, nil
	}
}

func gasCallIntrinsic(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	var (
		gas            uint64
		transfersValue = !stack.Back(2).IsZero()
		address        = common.Address(stack.Back(1).Bytes20())
	)
	if evm.readOnly && transfersValue {
		return 0, ErrWriteProtection
	}
	// Stateless check
	memoryGas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var transferGas uint64
	if transfersValue && !evm.chainRules.IsEIP4762 {
		transferGas = params.CallValueTransferGas
	}
	var overflow bool
	if gas, overflow = math.SafeAdd(memoryGas, transferGas); overflow {
		return 0, ErrGasUintOverflow
	}
	// Terminate the gas measurement if the leftover gas is not sufficient,
	// it can effectively prevent accessing the states in the following steps.
	if contract.Gas.RegularGas < gas {
		return 0, ErrOutOfGas
	}
	// Stateful check
	var stateGas uint64
	if evm.chainRules.IsEIP158 {
		if transfersValue && evm.StateDB.Empty(address) {
			stateGas += params.CallNewAccountGas
		}
	} else if !evm.StateDB.Exist(address) {
		stateGas += params.CallNewAccountGas
	}
	if gas, overflow = math.SafeAdd(gas, stateGas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

// gasCallIntrinsic8037 is the intrinsic gas calculator for CALL in Amsterdam.
// It computes memory expansion + value transfer gas but excludes new account
// creation, which is handled as state gas by the wrapper.
func gasCallIntrinsic8037(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	var (
		gas            uint64
		transfersValue = !stack.Back(2).IsZero()
	)
	if evm.readOnly && transfersValue {
		return 0, ErrWriteProtection
	}
	memoryGas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var transferGas uint64
	if transfersValue && !evm.chainRules.IsEIP4762 {
		transferGas = params.CallValueTransferGas
	}
	var overflow bool
	if gas, overflow = math.SafeAdd(memoryGas, transferGas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasCallCodeIntrinsic(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	memoryGas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	var (
		gas            uint64
		overflow       bool
		transfersValue = !stack.Back(2).IsZero()
	)
	if transfersValue {
		if !evm.chainRules.IsEIP4762 {
			gas += params.CallValueTransferGas
		}
	}
	if gas, overflow = math.SafeAdd(gas, memoryGas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}

func gasDelegateCallIntrinsic(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	return gas, nil
}

func gasStaticCallIntrinsic(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return 0, err
	}
	return gas, nil
}

func gasSelfdestruct(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	var gas uint64
	// EIP150 homestead gas reprice fork:
	if evm.chainRules.IsEIP150 {
		gas = params.SelfdestructGasEIP150
		if gas > contract.Gas.RegularGas {
			return GasCosts{RegularGas: gas}, nil
		}

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

	if !evm.StateDB.HasSelfDestructed(contract.Address()) {
		evm.StateDB.AddRefund(params.SelfdestructRefundGas)
	}
	return GasCosts{RegularGas: gas}, nil
}

func gasCreateEip8037(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return GasCosts{}, err
	}
	size, overflow := stack.Back(2).Uint64WithOverflow()
	if overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	if err := CheckMaxInitCodeSize(&evm.chainRules, size); err != nil {
		return GasCosts{}, err
	}
	// Since size <= MaxInitCodeSizeAmsterdam, these multiplications cannot overflow
	words := (size + 31) / 32
	wordGas := params.InitCodeWordGas * words
	stateGas := params.AccountCreationSize * evm.Context.CostPerGasByte
	return GasCosts{RegularGas: gas + wordGas, StateGas: stateGas}, nil
}

func gasCreate2Eip8037(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	gas, err := memoryGasCost(mem, memorySize)
	if err != nil {
		return GasCosts{}, err
	}
	size, overflow := stack.Back(2).Uint64WithOverflow()
	if overflow {
		return GasCosts{}, ErrGasUintOverflow
	}
	if err := CheckMaxInitCodeSize(&evm.chainRules, size); err != nil {
		return GasCosts{}, err
	}
	// Since size <= MaxInitCodeSizeAmsterdam, these multiplications cannot overflow
	words := (size + 31) / 32
	// CREATE2 charges both InitCodeWordGas (EIP-3860) and Keccak256WordGas (for address hashing).
	wordGas := (params.InitCodeWordGas + params.Keccak256WordGas) * words
	stateGas := params.AccountCreationSize * evm.Context.CostPerGasByte
	return GasCosts{RegularGas: gas + wordGas, StateGas: stateGas}, nil
}

// gasCall8037 is the stateful gas calculator for CALL in Amsterdam (EIP-8037).
// It only returns the state-dependent gas (account creation as state gas).
// Memory gas, transfer gas, and callGas are handled by gasCallStateless and
// makeCallVariantGasCall.
func gasCall8037(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	var (
		gas            GasCosts
		transfersValue = !stack.Back(2).IsZero()
		address        = common.Address(stack.Back(1).Bytes20())
	)
	if evm.chainRules.IsEIP158 {
		if transfersValue && evm.StateDB.Empty(address) {
			gas.StateGas += params.AccountCreationSize * evm.Context.CostPerGasByte
		}
	} else if !evm.StateDB.Exist(address) {
		gas.StateGas += params.AccountCreationSize * evm.Context.CostPerGasByte
	}
	return gas, nil
}

func gasSelfdestruct8037(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	var (
		gas     GasCosts
		address = common.Address(stack.peek().Bytes20())
	)
	if !evm.StateDB.AddressInAccessList(address) {
		// If the caller cannot afford the cost, this change will be rolled back
		evm.StateDB.AddAddressToAccessList(address)
		gas.RegularGas = params.ColdAccountAccessCostEIP2929
	}
	// Check we have enough regular gas before we add the address to the BAL
	if contract.Gas.RegularGas < gas.RegularGas {
		return gas, nil
	}
	// if empty and transfers value
	if evm.StateDB.Empty(address) && evm.StateDB.GetBalance(contract.Address()).Sign() != 0 {
		gas.StateGas += params.AccountCreationSize * evm.Context.CostPerGasByte
	}
	return gas, nil
}

func gasSStore8037(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (GasCosts, error) {
	if evm.readOnly {
		return GasCosts{}, ErrWriteProtection
	}
	// If we fail the minimum gas availability invariant, fail (0)
	if contract.Gas.RegularGas <= params.SstoreSentryGasEIP2200 {
		return GasCosts{}, errors.New("not enough gas for reentrancy sentry")
	}
	// Gas sentry honoured, do the actual gas calculation based on the stored value
	var (
		y, x              = stack.Back(1), stack.peek()
		slot              = common.Hash(x.Bytes32())
		current, original = evm.StateDB.GetStateAndCommittedState(contract.Address(), slot)
		cost              GasCosts
	)
	// Check slot presence in the access list
	if _, slotPresent := evm.StateDB.SlotInAccessList(contract.Address(), slot); !slotPresent {
		cost = GasCosts{RegularGas: params.ColdSloadCostEIP2929}
		// If the caller cannot afford the cost, this change will be rolled back
		evm.StateDB.AddSlotToAccessList(contract.Address(), slot)
	}
	value := common.Hash(y.Bytes32())

	if current == value { // noop (1)
		// EIP 2200 original clause:
		//		return params.SloadGasEIP2200, nil
		return GasCosts{RegularGas: cost.RegularGas + params.WarmStorageReadCostEIP2929}, nil // SLOAD_GAS
	}
	if original == current {
		if original == (common.Hash{}) { // create slot (2.1.1)
			// EIP-8037: Return both regular and state gas. System calls do not charge state gas.
			var stateGas uint64
			if !contract.IsSystemCall {
				stateGas = params.StorageCreationSize * evm.Context.CostPerGasByte
			}
			return GasCosts{
				RegularGas: cost.RegularGas + params.SstoreResetGasEIP2200 - params.ColdSloadCostEIP2929,
				StateGas:   stateGas,
			}, nil
		}
		if value == (common.Hash{}) { // delete slot (2.1.2b)
			evm.StateDB.AddRefund(params.SstoreClearsScheduleRefundEIP3529)
		}
		// EIP-2200 original clause:
		//		return params.SstoreResetGasEIP2200, nil // write existing slot (2.1.2)
		return GasCosts{RegularGas: cost.RegularGas + params.SstoreResetGasEIP2200 - params.ColdSloadCostEIP2929}, nil // write existing slot (2.1.2)
	}
	if original != (common.Hash{}) {
		if current == (common.Hash{}) { // recreate slot (2.2.1.1)
			evm.StateDB.SubRefund(params.SstoreClearsScheduleRefundEIP3529)
		} else if value == (common.Hash{}) { // delete slot (2.2.1.2)
			evm.StateDB.AddRefund(params.SstoreClearsScheduleRefundEIP3529)
		}
	}
	if original == value {
		if original == (common.Hash{}) { // reset to original inexistent slot (2.2.2.1)
			// EIP-8037 point (2): refund state gas directly to the reservoir
			// at the SSTORE restoration point (0→x→0 in same tx); not to the
			// refund counter, which is capped at gas_used/5.
			stateRefund := params.StorageCreationSize * evm.Context.CostPerGasByte
			contract.Gas.StateGas += stateRefund
			contract.GasUsed.StateGas -= int64(stateRefund)
			// Regular portion of the refund still goes through the refund counter.
			evm.StateDB.AddRefund(params.SstoreResetGasEIP2200 - params.ColdSloadCostEIP2929 - params.WarmStorageReadCostEIP2929)
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
	return GasCosts{RegularGas: cost.RegularGas + params.WarmStorageReadCostEIP2929}, nil // dirty update (2.2)
}
