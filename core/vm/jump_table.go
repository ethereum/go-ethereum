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

import (
	"fmt"

	"github.com/ethereum/go-ethereum/params"
)

type (
	ExecutionFunc func(pc *uint64, interpreter *EVMInterpreter, callContext *ScopeContext) ([]byte, error)
	gasFunc       func(*EVM, *Contract, *Stack, *Memory, uint64) (uint64, error) // last parameter is the requested memory size as a uint64
	// memorySizeFunc returns the required size, and whether the operation overflowed a uint64
	memorySizeFunc func(*Stack) (size uint64, overflow bool)
)

type operation struct {
	// execute is the operation function
	execute     ExecutionFunc
	ConstantGas uint64
	dynamicGas  gasFunc
	// minStack tells how many stack items are required
	minStack int
	// maxStack specifies the max length the stack can have for this operation
	// to not overflow the stack.
	maxStack int

	// memorySize returns the memory size required for the operation
	memorySize memorySizeFunc
}

var (
	frontierInstructionSet         = newFrontierInstructionSet()
	homesteadInstructionSet        = newHomesteadInstructionSet()
	tangerineWhistleInstructionSet = newTangerineWhistleInstructionSet()
	spuriousDragonInstructionSet   = newSpuriousDragonInstructionSet()
	byzantiumInstructionSet        = newByzantiumInstructionSet()
	constantinopleInstructionSet   = newConstantinopleInstructionSet()
	istanbulInstructionSet         = newIstanbulInstructionSet()
	berlinInstructionSet           = newBerlinInstructionSet()
	londonInstructionSet           = newLondonInstructionSet()
	mergeInstructionSet            = NewMergeInstructionSet()
	shanghaiInstructionSet         = newShanghaiInstructionSet()
	cancunInstructionSet           = newCancunInstructionSet()
)

// JumpTable contains the EVM opcodes supported at a given fork.
type JumpTable [256]*operation

func validate(jt JumpTable) JumpTable {
	for i, op := range jt {
		if op == nil {
			panic(fmt.Sprintf("op %#x is not set", i))
		}
		// The interpreter has an assumption that if the memorySize function is
		// set, then the dynamicGas function is also set. This is a somewhat
		// arbitrary assumption, and can be removed if we need to -- but it
		// allows us to avoid a condition check. As long as we have that assumption
		// in there, this little sanity check prevents us from merging in a
		// change which violates it.
		if op.memorySize != nil && op.dynamicGas == nil {
			panic(fmt.Sprintf("op %v has dynamic memory but not dynamic gas", OpCode(i).String()))
		}
	}
	return jt
}

func newCancunInstructionSet() JumpTable {
	instructionSet := newShanghaiInstructionSet()
	enable4844(&instructionSet) // EIP-4844 (BLOBHASH opcode)
	enable7516(&instructionSet) // EIP-7516 (BLOBBASEFEE opcode)
	enable1153(&instructionSet) // EIP-1153 "Transient Storage"
	enable5656(&instructionSet) // EIP-5656 (MCOPY opcode)
	enable6780(&instructionSet) // EIP-6780 SELFDESTRUCT only in same transaction
	return validate(instructionSet)
}

func newShanghaiInstructionSet() JumpTable {
	instructionSet := NewMergeInstructionSet()
	enable3855(&instructionSet) // PUSH0 instruction
	enable3860(&instructionSet) // Limit and meter initcode

	return validate(instructionSet)
}

func NewMergeInstructionSet() JumpTable {
	instructionSet := newLondonInstructionSet()
	instructionSet[PREVRANDAO] = &operation{
		execute:     OpRandom,
		ConstantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
	return validate(instructionSet)
}

// newLondonInstructionSet returns the frontier, homestead, byzantium,
// constantinople, istanbul, petersburg, berlin and london instructions.
func newLondonInstructionSet() JumpTable {
	instructionSet := newBerlinInstructionSet()
	enable3529(&instructionSet) // EIP-3529: Reduction in refunds https://eips.ethereum.org/EIPS/eip-3529
	enable3198(&instructionSet) // Base fee opcode https://eips.ethereum.org/EIPS/eip-3198
	return validate(instructionSet)
}

// newBerlinInstructionSet returns the frontier, homestead, byzantium,
// constantinople, istanbul, petersburg and berlin instructions.
func newBerlinInstructionSet() JumpTable {
	instructionSet := newIstanbulInstructionSet()
	enable2929(&instructionSet) // Access lists for trie accesses https://eips.ethereum.org/EIPS/eip-2929
	return validate(instructionSet)
}

// newIstanbulInstructionSet returns the frontier, homestead, byzantium,
// constantinople, istanbul and petersburg instructions.
func newIstanbulInstructionSet() JumpTable {
	instructionSet := newConstantinopleInstructionSet()

	enable1344(&instructionSet) // ChainID opcode - https://eips.ethereum.org/EIPS/eip-1344
	enable1884(&instructionSet) // Reprice reader opcodes - https://eips.ethereum.org/EIPS/eip-1884
	enable2200(&instructionSet) // Net metered SSTORE - https://eips.ethereum.org/EIPS/eip-2200

	return validate(instructionSet)
}

// newConstantinopleInstructionSet returns the frontier, homestead,
// byzantium and constantinople instructions.
func newConstantinopleInstructionSet() JumpTable {
	instructionSet := newByzantiumInstructionSet()
	instructionSet[SHL] = &operation{
		execute:     OpSHL,
		ConstantGas: GasFastestStep,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}
	instructionSet[SHR] = &operation{
		execute:     OpSHR,
		ConstantGas: GasFastestStep,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}
	instructionSet[SAR] = &operation{
		execute:     OpSAR,
		ConstantGas: GasFastestStep,
		minStack:    minStack(2, 1),
		maxStack:    maxStack(2, 1),
	}
	instructionSet[EXTCODEHASH] = &operation{
		execute:     opExtCodeHash,
		ConstantGas: params.ExtcodeHashGasConstantinople,
		minStack:    minStack(1, 1),
		maxStack:    maxStack(1, 1),
	}
	instructionSet[CREATE2] = &operation{
		execute:     opCreate2,
		ConstantGas: params.Create2Gas,
		dynamicGas:  gasCreate2,
		minStack:    minStack(4, 1),
		maxStack:    maxStack(4, 1),
		memorySize:  memoryCreate2,
	}
	return validate(instructionSet)
}

// newByzantiumInstructionSet returns the frontier, homestead and
// byzantium instructions.
func newByzantiumInstructionSet() JumpTable {
	instructionSet := newSpuriousDragonInstructionSet()
	instructionSet[STATICCALL] = &operation{
		execute:     opStaticCall,
		ConstantGas: params.CallGasEIP150,
		dynamicGas:  gasStaticCall,
		minStack:    minStack(6, 1),
		maxStack:    maxStack(6, 1),
		memorySize:  memoryStaticCall,
	}
	instructionSet[RETURNDATASIZE] = &operation{
		execute:     opReturnDataSize,
		ConstantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
	instructionSet[RETURNDATACOPY] = &operation{
		execute:     opReturnDataCopy,
		ConstantGas: GasFastestStep,
		dynamicGas:  gasReturnDataCopy,
		minStack:    minStack(3, 0),
		maxStack:    maxStack(3, 0),
		memorySize:  memoryReturnDataCopy,
	}
	instructionSet[REVERT] = &operation{
		execute:    opRevert,
		dynamicGas: gasRevert,
		minStack:   minStack(2, 0),
		maxStack:   maxStack(2, 0),
		memorySize: memoryRevert,
	}
	return validate(instructionSet)
}

// EIP 158 a.k.a Spurious Dragon
func newSpuriousDragonInstructionSet() JumpTable {
	instructionSet := newTangerineWhistleInstructionSet()
	instructionSet[EXP].dynamicGas = gasExpEIP158
	return validate(instructionSet)
}

// EIP 150 a.k.a Tangerine Whistle
func newTangerineWhistleInstructionSet() JumpTable {
	instructionSet := newHomesteadInstructionSet()
	instructionSet[BALANCE].ConstantGas = params.BalanceGasEIP150
	instructionSet[EXTCODESIZE].ConstantGas = params.ExtcodeSizeGasEIP150
	instructionSet[SLOAD].ConstantGas = params.SloadGasEIP150
	instructionSet[EXTCODECOPY].ConstantGas = params.ExtcodeCopyBaseEIP150
	instructionSet[CALL].ConstantGas = params.CallGasEIP150
	instructionSet[CALLCODE].ConstantGas = params.CallGasEIP150
	instructionSet[DELEGATECALL].ConstantGas = params.CallGasEIP150
	return validate(instructionSet)
}

// newHomesteadInstructionSet returns the frontier and homestead
// instructions that can be executed during the homestead phase.
func newHomesteadInstructionSet() JumpTable {
	instructionSet := newFrontierInstructionSet()
	instructionSet[DELEGATECALL] = &operation{
		execute:     opDelegateCall,
		dynamicGas:  gasDelegateCall,
		ConstantGas: params.CallGasFrontier,
		minStack:    minStack(6, 1),
		maxStack:    maxStack(6, 1),
		memorySize:  memoryDelegateCall,
	}
	return validate(instructionSet)
}

// newFrontierInstructionSet returns the frontier instructions
// that can be executed during the frontier phase.
func newFrontierInstructionSet() JumpTable {
	tbl := JumpTable{
		STOP: {
			execute:     opStop,
			ConstantGas: 0,
			minStack:    minStack(0, 0),
			maxStack:    maxStack(0, 0),
		},
		ADD: {
			execute:     OpAdd,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		MUL: {
			execute:     OpMul,
			ConstantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SUB: {
			execute:     OpSub,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		DIV: {
			execute:     OpDiv,
			ConstantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SDIV: {
			execute:     OpSdiv,
			ConstantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		MOD: {
			execute:     OpMod,
			ConstantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SMOD: {
			execute:     OpSmod,
			ConstantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		ADDMOD: {
			execute:     OpAddmod,
			ConstantGas: GasMidStep,
			minStack:    minStack(3, 1),
			maxStack:    maxStack(3, 1),
		},
		MULMOD: {
			execute:     OpMulmod,
			ConstantGas: GasMidStep,
			minStack:    minStack(3, 1),
			maxStack:    maxStack(3, 1),
		},
		EXP: {
			execute:    OpExp,
			dynamicGas: gasExpFrontier,
			minStack:   minStack(2, 1),
			maxStack:   maxStack(2, 1),
		},
		SIGNEXTEND: {
			execute:     OpSignExtend,
			ConstantGas: GasFastStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		LT: {
			execute:     OpLt,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		GT: {
			execute:     OpGt,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SLT: {
			execute:     OpSlt,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		SGT: {
			execute:     OpSgt,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		EQ: {
			execute:     OpEq,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		ISZERO: {
			execute:     OpIszero,
			ConstantGas: GasFastestStep,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		AND: {
			execute:     OpAnd,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		XOR: {
			execute:     OpXor,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		OR: {
			execute:     OpOr,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		NOT: {
			execute:     opNot,
			ConstantGas: GasFastestStep,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		BYTE: {
			execute:     OpByte,
			ConstantGas: GasFastestStep,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
		},
		KECCAK256: {
			execute:     OpKeccak256,
			ConstantGas: params.Keccak256Gas,
			dynamicGas:  gasKeccak256,
			minStack:    minStack(2, 1),
			maxStack:    maxStack(2, 1),
			memorySize:  memoryKeccak256,
		},
		ADDRESS: {
			execute:     OpAddress,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		BALANCE: {
			execute:     opBalance,
			ConstantGas: params.BalanceGasFrontier,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		ORIGIN: {
			execute:     OpOrigin,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CALLER: {
			execute:     opCaller,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CALLVALUE: {
			execute:     opCallValue,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CALLDATALOAD: {
			execute:     opCallDataLoad,
			ConstantGas: GasFastestStep,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		CALLDATASIZE: {
			execute:     opCallDataSize,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CALLDATACOPY: {
			execute:     opCallDataCopy,
			ConstantGas: GasFastestStep,
			dynamicGas:  gasCallDataCopy,
			minStack:    minStack(3, 0),
			maxStack:    maxStack(3, 0),
			memorySize:  memoryCallDataCopy,
		},
		CODESIZE: {
			execute:     opCodeSize,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		CODECOPY: {
			execute:     opCodeCopy,
			ConstantGas: GasFastestStep,
			dynamicGas:  gasCodeCopy,
			minStack:    minStack(3, 0),
			maxStack:    maxStack(3, 0),
			memorySize:  memoryCodeCopy,
		},
		GASPRICE: {
			execute:     opGasprice,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		EXTCODESIZE: {
			execute:     opExtCodeSize,
			ConstantGas: params.ExtcodeSizeGasFrontier,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		EXTCODECOPY: {
			execute:     opExtCodeCopy,
			ConstantGas: params.ExtcodeCopyBaseFrontier,
			dynamicGas:  gasExtCodeCopy,
			minStack:    minStack(4, 0),
			maxStack:    maxStack(4, 0),
			memorySize:  memoryExtCodeCopy,
		},
		BLOCKHASH: {
			execute:     opBlockhash,
			ConstantGas: GasExtStep,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		COINBASE: {
			execute:     opCoinbase,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		TIMESTAMP: {
			execute:     opTimestamp,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		NUMBER: {
			execute:     opNumber,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		DIFFICULTY: {
			execute:     opDifficulty,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		GASLIMIT: {
			execute:     opGasLimit,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		POP: {
			execute:     opPop,
			ConstantGas: GasQuickStep,
			minStack:    minStack(1, 0),
			maxStack:    maxStack(1, 0),
		},
		MLOAD: {
			execute:     opMload,
			ConstantGas: GasFastestStep,
			dynamicGas:  gasMLoad,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
			memorySize:  memoryMLoad,
		},
		MSTORE: {
			execute:     OpMstore,
			ConstantGas: GasFastestStep,
			dynamicGas:  gasMStore,
			minStack:    minStack(2, 0),
			maxStack:    maxStack(2, 0),
			memorySize:  memoryMStore,
		},
		MSTORE8: {
			execute:     OpMstore8,
			ConstantGas: GasFastestStep,
			dynamicGas:  gasMStore8,
			memorySize:  memoryMStore8,
			minStack:    minStack(2, 0),
			maxStack:    maxStack(2, 0),
		},
		SLOAD: {
			execute:     opSload,
			ConstantGas: params.SloadGasFrontier,
			minStack:    minStack(1, 1),
			maxStack:    maxStack(1, 1),
		},
		SSTORE: {
			execute:    opSstore,
			dynamicGas: gasSStore,
			minStack:   minStack(2, 0),
			maxStack:   maxStack(2, 0),
		},
		JUMP: {
			execute:     opJump,
			ConstantGas: GasMidStep,
			minStack:    minStack(1, 0),
			maxStack:    maxStack(1, 0),
		},
		JUMPI: {
			execute:     opJumpi,
			ConstantGas: GasSlowStep,
			minStack:    minStack(2, 0),
			maxStack:    maxStack(2, 0),
		},
		PC: {
			execute:     opPc,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		MSIZE: {
			execute:     opMsize,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		GAS: {
			execute:     opGas,
			ConstantGas: GasQuickStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		JUMPDEST: {
			execute:     opJumpdest,
			ConstantGas: params.JumpdestGas,
			minStack:    minStack(0, 0),
			maxStack:    maxStack(0, 0),
		},
		PUSH1: {
			execute:     opPush1,
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH2: {
			execute:     makePush(2, 2),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH3: {
			execute:     makePush(3, 3),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH4: {
			execute:     makePush(4, 4),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH5: {
			execute:     makePush(5, 5),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH6: {
			execute:     makePush(6, 6),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH7: {
			execute:     makePush(7, 7),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH8: {
			execute:     makePush(8, 8),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH9: {
			execute:     makePush(9, 9),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH10: {
			execute:     makePush(10, 10),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH11: {
			execute:     makePush(11, 11),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH12: {
			execute:     makePush(12, 12),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH13: {
			execute:     makePush(13, 13),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH14: {
			execute:     makePush(14, 14),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH15: {
			execute:     makePush(15, 15),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH16: {
			execute:     makePush(16, 16),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH17: {
			execute:     makePush(17, 17),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH18: {
			execute:     makePush(18, 18),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH19: {
			execute:     makePush(19, 19),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH20: {
			execute:     makePush(20, 20),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH21: {
			execute:     makePush(21, 21),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH22: {
			execute:     makePush(22, 22),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH23: {
			execute:     makePush(23, 23),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH24: {
			execute:     makePush(24, 24),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH25: {
			execute:     makePush(25, 25),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH26: {
			execute:     makePush(26, 26),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH27: {
			execute:     makePush(27, 27),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH28: {
			execute:     makePush(28, 28),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH29: {
			execute:     makePush(29, 29),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH30: {
			execute:     makePush(30, 30),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH31: {
			execute:     makePush(31, 31),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		PUSH32: {
			execute:     makePush(32, 32),
			ConstantGas: GasFastestStep,
			minStack:    minStack(0, 1),
			maxStack:    maxStack(0, 1),
		},
		DUP1: {
			execute:     makeDup(1),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(1),
			maxStack:    maxDupStack(1),
		},
		DUP2: {
			execute:     makeDup(2),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(2),
			maxStack:    maxDupStack(2),
		},
		DUP3: {
			execute:     makeDup(3),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(3),
			maxStack:    maxDupStack(3),
		},
		DUP4: {
			execute:     makeDup(4),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(4),
			maxStack:    maxDupStack(4),
		},
		DUP5: {
			execute:     makeDup(5),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(5),
			maxStack:    maxDupStack(5),
		},
		DUP6: {
			execute:     makeDup(6),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(6),
			maxStack:    maxDupStack(6),
		},
		DUP7: {
			execute:     makeDup(7),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(7),
			maxStack:    maxDupStack(7),
		},
		DUP8: {
			execute:     makeDup(8),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(8),
			maxStack:    maxDupStack(8),
		},
		DUP9: {
			execute:     makeDup(9),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(9),
			maxStack:    maxDupStack(9),
		},
		DUP10: {
			execute:     makeDup(10),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(10),
			maxStack:    maxDupStack(10),
		},
		DUP11: {
			execute:     makeDup(11),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(11),
			maxStack:    maxDupStack(11),
		},
		DUP12: {
			execute:     makeDup(12),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(12),
			maxStack:    maxDupStack(12),
		},
		DUP13: {
			execute:     makeDup(13),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(13),
			maxStack:    maxDupStack(13),
		},
		DUP14: {
			execute:     makeDup(14),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(14),
			maxStack:    maxDupStack(14),
		},
		DUP15: {
			execute:     makeDup(15),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(15),
			maxStack:    maxDupStack(15),
		},
		DUP16: {
			execute:     makeDup(16),
			ConstantGas: GasFastestStep,
			minStack:    minDupStack(16),
			maxStack:    maxDupStack(16),
		},
		SWAP1: {
			execute:     makeSwap(1),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(2),
			maxStack:    maxSwapStack(2),
		},
		SWAP2: {
			execute:     makeSwap(2),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(3),
			maxStack:    maxSwapStack(3),
		},
		SWAP3: {
			execute:     makeSwap(3),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(4),
			maxStack:    maxSwapStack(4),
		},
		SWAP4: {
			execute:     makeSwap(4),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(5),
			maxStack:    maxSwapStack(5),
		},
		SWAP5: {
			execute:     makeSwap(5),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(6),
			maxStack:    maxSwapStack(6),
		},
		SWAP6: {
			execute:     makeSwap(6),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(7),
			maxStack:    maxSwapStack(7),
		},
		SWAP7: {
			execute:     makeSwap(7),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(8),
			maxStack:    maxSwapStack(8),
		},
		SWAP8: {
			execute:     makeSwap(8),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(9),
			maxStack:    maxSwapStack(9),
		},
		SWAP9: {
			execute:     makeSwap(9),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(10),
			maxStack:    maxSwapStack(10),
		},
		SWAP10: {
			execute:     makeSwap(10),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(11),
			maxStack:    maxSwapStack(11),
		},
		SWAP11: {
			execute:     makeSwap(11),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(12),
			maxStack:    maxSwapStack(12),
		},
		SWAP12: {
			execute:     makeSwap(12),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(13),
			maxStack:    maxSwapStack(13),
		},
		SWAP13: {
			execute:     makeSwap(13),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(14),
			maxStack:    maxSwapStack(14),
		},
		SWAP14: {
			execute:     makeSwap(14),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(15),
			maxStack:    maxSwapStack(15),
		},
		SWAP15: {
			execute:     makeSwap(15),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(16),
			maxStack:    maxSwapStack(16),
		},
		SWAP16: {
			execute:     makeSwap(16),
			ConstantGas: GasFastestStep,
			minStack:    minSwapStack(17),
			maxStack:    maxSwapStack(17),
		},
		LOG0: {
			execute:    makeLog(0),
			dynamicGas: makeGasLog(0),
			minStack:   minStack(2, 0),
			maxStack:   maxStack(2, 0),
			memorySize: memoryLog,
		},
		LOG1: {
			execute:    makeLog(1),
			dynamicGas: makeGasLog(1),
			minStack:   minStack(3, 0),
			maxStack:   maxStack(3, 0),
			memorySize: memoryLog,
		},
		LOG2: {
			execute:    makeLog(2),
			dynamicGas: makeGasLog(2),
			minStack:   minStack(4, 0),
			maxStack:   maxStack(4, 0),
			memorySize: memoryLog,
		},
		LOG3: {
			execute:    makeLog(3),
			dynamicGas: makeGasLog(3),
			minStack:   minStack(5, 0),
			maxStack:   maxStack(5, 0),
			memorySize: memoryLog,
		},
		LOG4: {
			execute:    makeLog(4),
			dynamicGas: makeGasLog(4),
			minStack:   minStack(6, 0),
			maxStack:   maxStack(6, 0),
			memorySize: memoryLog,
		},
		CREATE: {
			execute:     opCreate,
			ConstantGas: params.CreateGas,
			dynamicGas:  gasCreate,
			minStack:    minStack(3, 1),
			maxStack:    maxStack(3, 1),
			memorySize:  memoryCreate,
		},
		CALL: {
			execute:     opCall,
			ConstantGas: params.CallGasFrontier,
			dynamicGas:  gasCall,
			minStack:    minStack(7, 1),
			maxStack:    maxStack(7, 1),
			memorySize:  memoryCall,
		},
		CALLCODE: {
			execute:     opCallCode,
			ConstantGas: params.CallGasFrontier,
			dynamicGas:  gasCallCode,
			minStack:    minStack(7, 1),
			maxStack:    maxStack(7, 1),
			memorySize:  memoryCall,
		},
		RETURN: {
			execute:    opReturn,
			dynamicGas: gasReturn,
			minStack:   minStack(2, 0),
			maxStack:   maxStack(2, 0),
			memorySize: memoryReturn,
		},
		SELFDESTRUCT: {
			execute:    opSelfdestruct,
			dynamicGas: gasSelfdestruct,
			minStack:   minStack(1, 0),
			maxStack:   maxStack(1, 0),
		},
	}

	// Fill all unassigned slots with opUndefined.
	for i, entry := range tbl {
		if entry == nil {
			tbl[i] = &operation{execute: opUndefined, maxStack: maxStack(0, 0)}
		}
	}

	return validate(tbl)
}

func CopyJumpTable(source *JumpTable) *JumpTable {
	dest := *source
	for i, op := range source {
		if op != nil {
			opCopy := *op
			dest[i] = &opCopy
		}
	}
	return &dest
}
