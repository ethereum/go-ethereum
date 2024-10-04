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

package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/params"
)

func gasSStore4762(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	gas := evm.AccessEvents.SlotGas(contract.Address(), stack.peek().Bytes32(), true)
	if gas == 0 {
		gas = params.WarmStorageReadCostEIP2929
	}
	return gas, nil
}

func gasSLoad4762(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	gas := evm.AccessEvents.SlotGas(contract.Address(), stack.peek().Bytes32(), false)
	if gas == 0 {
		gas = params.WarmStorageReadCostEIP2929
	}
	return gas, nil
}

func gasBalance4762(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	address := stack.peek().Bytes20()
	gas := evm.AccessEvents.BasicDataGas(address, false)
	if gas == 0 {
		gas = params.WarmStorageReadCostEIP2929
	}
	return gas, nil
}

func gasExtCodeSize4762(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	address := stack.peek().Bytes20()
	if _, isPrecompile := evm.precompile(address); isPrecompile {
		return 0, nil
	}
	gas := evm.AccessEvents.BasicDataGas(address, false)
	if gas == 0 {
		gas = params.WarmStorageReadCostEIP2929
	}
	return gas, nil
}

func gasExtCodeHash4762(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	address := stack.peek().Bytes20()
	if _, isPrecompile := evm.precompile(address); isPrecompile {
		return 0, nil
	}
	gas := evm.AccessEvents.CodeHashGas(address, false)
	if gas == 0 {
		gas = params.WarmStorageReadCostEIP2929
	}
	return gas, nil
}

func makeCallVariantGasEIP4762(oldCalculator gasFunc) gasFunc {
	return func(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
		gas, err := oldCalculator(evm, contract, stack, mem, memorySize)
		if err != nil {
			return 0, err
		}
		if _, isPrecompile := evm.precompile(contract.Address()); isPrecompile {
			return gas, nil
		}
		witnessGas := evm.AccessEvents.MessageCallGas(contract.Address())
		if witnessGas == 0 {
			witnessGas = params.WarmStorageReadCostEIP2929
		}
		return witnessGas + gas, nil
	}
}

var (
	gasCallEIP4762         = makeCallVariantGasEIP4762(gasCall)
	gasCallCodeEIP4762     = makeCallVariantGasEIP4762(gasCallCode)
	gasStaticCallEIP4762   = makeCallVariantGasEIP4762(gasStaticCall)
	gasDelegateCallEIP4762 = makeCallVariantGasEIP4762(gasDelegateCall)
)

func gasSelfdestructEIP4762(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	beneficiaryAddr := common.Address(stack.peek().Bytes20())
	if _, isPrecompile := evm.precompile(beneficiaryAddr); isPrecompile {
		return 0, nil
	}
	contractAddr := contract.Address()
	statelessGas := evm.AccessEvents.BasicDataGas(contractAddr, false)
	if contractAddr != beneficiaryAddr {
		statelessGas += evm.AccessEvents.BasicDataGas(beneficiaryAddr, false)
	}
	// Charge write costs if it transfers value
	if evm.StateDB.GetBalance(contractAddr).Sign() != 0 {
		statelessGas += evm.AccessEvents.BasicDataGas(contractAddr, true)
		if contractAddr != beneficiaryAddr {
			statelessGas += evm.AccessEvents.BasicDataGas(beneficiaryAddr, true)
		}
	}
	return statelessGas, nil
}

func gasCodeCopyEip4762(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	gas, err := gasCodeCopy(evm, contract, stack, mem, memorySize)
	if err != nil {
		return 0, err
	}
	var (
		codeOffset = stack.Back(1)
		length     = stack.Back(2)
	)
	uint64CodeOffset, overflow := codeOffset.Uint64WithOverflow()
	if overflow {
		uint64CodeOffset = math.MaxUint64
	}
	_, copyOffset, nonPaddedCopyLength := getDataAndAdjustedBounds(contract.Code, uint64CodeOffset, length.Uint64())
	if !contract.IsDeployment {
		gas += evm.AccessEvents.CodeChunksRangeGas(contract.Address(), copyOffset, nonPaddedCopyLength, uint64(len(contract.Code)), false)
	}
	return gas, nil
}

func gasExtCodeCopyEIP4762(evm *EVM, contract *Contract, stack *Stack, mem *Memory, memorySize uint64) (uint64, error) {
	// memory expansion first (dynamic part of pre-2929 implementation)
	gas, err := gasExtCodeCopy(evm, contract, stack, mem, memorySize)
	if err != nil {
		return 0, err
	}
	addr := common.Address(stack.peek().Bytes20())
	wgas := evm.AccessEvents.BasicDataGas(addr, false)
	if wgas == 0 {
		wgas = params.WarmStorageReadCostEIP2929
	}
	var overflow bool
	// We charge (cold-warm), since 'warm' is already charged as constantGas
	if gas, overflow = math.SafeAdd(gas, wgas); overflow {
		return 0, ErrGasUintOverflow
	}
	return gas, nil
}
