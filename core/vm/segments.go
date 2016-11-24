// Copyright 2015 The go-ethereum Authors
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

import "math/big"

type jumpSeg struct {
	pos uint64
	err error
	gas *big.Int
}

func (j jumpSeg) do(program *Program, pc *uint64, env *Environment, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	if !contract.UseGas(j.gas) {
		return nil, OutOfGasError
	}
	if j.err != nil {
		return nil, j.err
	}
	*pc = j.pos
	return nil, nil
}
func (s jumpSeg) halts() bool { return false }
func (s jumpSeg) Op() OpCode  { return 0 }

type pushSeg struct {
	data []*big.Int
	gas  *big.Int
}

func (s pushSeg) do(program *Program, pc *uint64, env *Environment, contract *Contract, memory *Memory, stack *Stack) ([]byte, error) {
	// Use the calculated gas. When insufficient gas is present, use all gas and return an
	// Out Of Gas error
	if !contract.UseGas(s.gas) {
		return nil, OutOfGasError
	}

	for _, d := range s.data {
		stack.push(new(big.Int).Set(d))
	}
	*pc += uint64(len(s.data))
	return nil, nil
}

func (s pushSeg) halts() bool { return false }
func (s pushSeg) Op() OpCode  { return 0 }
