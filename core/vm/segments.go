package vm

import "math/big"

type jumpSeg struct {
	pos uint64
	err error
	gas *big.Int
}

func (j jumpSeg) do(program *Program, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
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

func (s pushSeg) do(program *Program, pc *uint64, env Environment, contract *Contract, memory *Memory, stack *stack) ([]byte, error) {
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
