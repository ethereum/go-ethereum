// Copyright 2019 The go-ethereum Authors
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

import "fmt"

// EnableEIP enables the given EIP on the config.
// This operation write in-place, and callers need to ensure that the globally
// defined jumptables are not polluted
func EnableEIP(eipNum int, jt *JumpTable) error {
	switch eipNum {
	case 1884:
		enable1884(jt)
	default:
		return fmt.Errorf("undefined eip %d", eipNum)
	}
	return nil
}

// Enable1884 applies EIP-1884 to the given jumptable
// - Increase cost of BALANCE to 700
// - Increase cost of EXTCODEHASH to 700
// - Increase cost of SLOAD to 800
// - Define SELFBALANCE, with cost GasFastStep (5)
func enable1884(jt *JumpTable) {
	// Gas cost changes
	jt[BALANCE].constantGas = 700
	jt[EXTCODEHASH].constantGas = 700
	jt[SLOAD].constantGas = 800
	// New opcode
	jt[SELFBALANCE] = operation{
		execute:     opSelfBalance,
		constantGas: GasFastStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
		valid:       true,
	}
}

func opSelfBalance(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	balance := interpreter.intPool.get().Set(interpreter.evm.StateDB.GetBalance(contract.Address()))
	stack.push(balance)
	return nil, nil
}
