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
	executionFunc func(pc *uint64, interpreter *EVMInterpreter, callContext *ScopeContext) ([]byte, error)
	gasFunc       func(*EVM, *Contract, *Stack, *Memory, uint64) (uint64, error) // last parameter is the requested memory size as a uint64
	// memorySizeFunc returns the required size, and whether the operation overflowed a uint64
	memorySizeFunc func(*Stack) (size uint64, overflow bool)
)

type operation struct {
	// execute is the operation function
	execute     executionFunc
	constantGas uint64
	dynamicGas  gasFunc
	// minStack tells how many stack items are required
	minStack int
	// expandsStack tells whether operation pushes more than pops (it may overflow the stack if true)
	expandsStack bool

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
)

// JumpTable contains the EVM opcodes supported at a given fork.
type JumpTable [256]*operation

func validate(jt JumpTable) JumpTable {
	for i, op := range jt {
		if op == nil {
			panic(fmt.Sprintf("op 0x%x is not set", i))
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

// newLondonInstructionSet returns the frontier, homestead, byzantium,
// contantinople, istanbul, petersburg, berlin and london instructions.
func newLondonInstructionSet() JumpTable {
	instructionSet := newBerlinInstructionSet()
	enable3529(&instructionSet) // EIP-3529: Reduction in refunds https://eips.ethereum.org/EIPS/eip-3529
	enable3198(&instructionSet) // Base fee opcode https://eips.ethereum.org/EIPS/eip-3198
	return validate(instructionSet)
}

// newBerlinInstructionSet returns the frontier, homestead, byzantium,
// contantinople, istanbul, petersburg and berlin instructions.
func newBerlinInstructionSet() JumpTable {
	instructionSet := newIstanbulInstructionSet()
	enable2929(&instructionSet) // Access lists for trie accesses https://eips.ethereum.org/EIPS/eip-2929
	return validate(instructionSet)
}

// newIstanbulInstructionSet returns the frontier, homestead, byzantium,
// contantinople, istanbul and petersburg instructions.
func newIstanbulInstructionSet() JumpTable {
	instructionSet := newConstantinopleInstructionSet()

	enable1344(&instructionSet) // ChainID opcode - https://eips.ethereum.org/EIPS/eip-1344
	enable1884(&instructionSet) // Reprice reader opcodes - https://eips.ethereum.org/EIPS/eip-1884
	enable2200(&instructionSet) // Net metered SSTORE - https://eips.ethereum.org/EIPS/eip-2200

	return validate(instructionSet)
}

// newConstantinopleInstructionSet returns the frontier, homestead,
// byzantium and contantinople instructions.
func newConstantinopleInstructionSet() JumpTable {
	instructionSet := newByzantiumInstructionSet()
	instructionSet[SHL] = &operation{
		execute:     opSHL,
		constantGas: GasFastestStep,
		minStack:    minStack(2, 1),
	}
	instructionSet[SHR] = &operation{
		execute:     opSHR,
		constantGas: GasFastestStep,
		minStack:    minStack(2, 1),
	}
	instructionSet[SAR] = &operation{
		execute:     opSAR,
		constantGas: GasFastestStep,
		minStack:    minStack(2, 1),
	}
	instructionSet[EXTCODEHASH] = &operation{
		execute:     opExtCodeHash,
		constantGas: params.ExtcodeHashGasConstantinople,
		minStack:    minStack(1, 1),
	}
	instructionSet[CREATE2] = &operation{
		execute:     opCreate2,
		constantGas: params.Create2Gas,
		dynamicGas:  gasCreate2,
		minStack:    minStack(4, 1),
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
		constantGas: params.CallGasEIP150,
		dynamicGas:  gasStaticCall,
		minStack:    minStack(6, 1),
		memorySize:  memoryStaticCall,
	}
	instructionSet[RETURNDATASIZE] = &operation{
		execute:      opReturnDataSize,
		constantGas:  GasQuickStep,
		minStack:     minStack(0, 1),
		expandsStack: true,
	}
	instructionSet[RETURNDATACOPY] = &operation{
		execute:     opReturnDataCopy,
		constantGas: GasFastestStep,
		dynamicGas:  gasReturnDataCopy,
		minStack:    minStack(3, 0),
		memorySize:  memoryReturnDataCopy,
	}
	instructionSet[REVERT] = &operation{
		execute:    opRevert,
		dynamicGas: gasRevert,
		minStack:   minStack(2, 0),
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
	instructionSet[BALANCE].constantGas = params.BalanceGasEIP150
	instructionSet[EXTCODESIZE].constantGas = params.ExtcodeSizeGasEIP150
	instructionSet[SLOAD].constantGas = params.SloadGasEIP150
	instructionSet[EXTCODECOPY].constantGas = params.ExtcodeCopyBaseEIP150
	instructionSet[CALL].constantGas = params.CallGasEIP150
	instructionSet[CALLCODE].constantGas = params.CallGasEIP150
	instructionSet[DELEGATECALL].constantGas = params.CallGasEIP150
	return validate(instructionSet)
}

// newHomesteadInstructionSet returns the frontier and homestead
// instructions that can be executed during the homestead phase.
func newHomesteadInstructionSet() JumpTable {
	instructionSet := newFrontierInstructionSet()
	instructionSet[DELEGATECALL] = &operation{
		execute:     opDelegateCall,
		dynamicGas:  gasDelegateCall,
		constantGas: params.CallGasFrontier,
		minStack:    minStack(6, 1),
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
			constantGas: 0,
			minStack:    minStack(0, 0),
		},
		ADD: {
			execute:     opAdd,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		MUL: {
			execute:     opMul,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
		},
		SUB: {
			execute:     opSub,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		DIV: {
			execute:     opDiv,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
		},
		SDIV: {
			execute:     opSdiv,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
		},
		MOD: {
			execute:     opMod,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
		},
		SMOD: {
			execute:     opSmod,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
		},
		ADDMOD: {
			execute:     opAddmod,
			constantGas: GasMidStep,
			minStack:    minStack(3, 1),
		},
		MULMOD: {
			execute:     opMulmod,
			constantGas: GasMidStep,
			minStack:    minStack(3, 1),
		},
		EXP: {
			execute:    opExp,
			dynamicGas: gasExpFrontier,
			minStack:   minStack(2, 1),
		},
		SIGNEXTEND: {
			execute:     opSignExtend,
			constantGas: GasFastStep,
			minStack:    minStack(2, 1),
		},
		LT: {
			execute:     opLt,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		GT: {
			execute:     opGt,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		SLT: {
			execute:     opSlt,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		SGT: {
			execute:     opSgt,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		EQ: {
			execute:     opEq,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		ISZERO: {
			execute:     opIszero,
			constantGas: GasFastestStep,
			minStack:    minStack(1, 1),
		},
		AND: {
			execute:     opAnd,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		XOR: {
			execute:     opXor,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		OR: {
			execute:     opOr,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		NOT: {
			execute:     opNot,
			constantGas: GasFastestStep,
			minStack:    minStack(1, 1),
		},
		BYTE: {
			execute:     opByte,
			constantGas: GasFastestStep,
			minStack:    minStack(2, 1),
		},
		KECCAK256: {
			execute:     opKeccak256,
			constantGas: params.Keccak256Gas,
			dynamicGas:  gasKeccak256,
			minStack:    minStack(2, 1),
			memorySize:  memoryKeccak256,
		},
		ADDRESS: {
			execute:      opAddress,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		BALANCE: {
			execute:     opBalance,
			constantGas: params.BalanceGasFrontier,
			minStack:    minStack(1, 1),
		},
		ORIGIN: {
			execute:      opOrigin,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		CALLER: {
			execute:      opCaller,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		CALLVALUE: {
			execute:      opCallValue,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		CALLDATALOAD: {
			execute:     opCallDataLoad,
			constantGas: GasFastestStep,
			minStack:    minStack(1, 1),
		},
		CALLDATASIZE: {
			execute:      opCallDataSize,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		CALLDATACOPY: {
			execute:     opCallDataCopy,
			constantGas: GasFastestStep,
			dynamicGas:  gasCallDataCopy,
			minStack:    minStack(3, 0),
			memorySize:  memoryCallDataCopy,
		},
		CODESIZE: {
			execute:      opCodeSize,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		CODECOPY: {
			execute:     opCodeCopy,
			constantGas: GasFastestStep,
			dynamicGas:  gasCodeCopy,
			minStack:    minStack(3, 0),
			memorySize:  memoryCodeCopy,
		},
		GASPRICE: {
			execute:      opGasprice,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		EXTCODESIZE: {
			execute:     opExtCodeSize,
			constantGas: params.ExtcodeSizeGasFrontier,
			minStack:    minStack(1, 1),
		},
		EXTCODECOPY: {
			execute:     opExtCodeCopy,
			constantGas: params.ExtcodeCopyBaseFrontier,
			dynamicGas:  gasExtCodeCopy,
			minStack:    minStack(4, 0),
			memorySize:  memoryExtCodeCopy,
		},
		BLOCKHASH: {
			execute:     opBlockhash,
			constantGas: GasExtStep,
			minStack:    minStack(1, 1),
		},
		COINBASE: {
			execute:      opCoinbase,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		TIMESTAMP: {
			execute:      opTimestamp,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		NUMBER: {
			execute:      opNumber,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		DIFFICULTY: {
			execute:      opDifficulty,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		GASLIMIT: {
			execute:      opGasLimit,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		POP: {
			execute:     opPop,
			constantGas: GasQuickStep,
			minStack:    minStack(1, 0),
		},
		MLOAD: {
			execute:     opMload,
			constantGas: GasFastestStep,
			dynamicGas:  gasMLoad,
			minStack:    minStack(1, 1),
			memorySize:  memoryMLoad,
		},
		MSTORE: {
			execute:     opMstore,
			constantGas: GasFastestStep,
			dynamicGas:  gasMStore,
			minStack:    minStack(2, 0),
			memorySize:  memoryMStore,
		},
		MSTORE8: {
			execute:     opMstore8,
			constantGas: GasFastestStep,
			dynamicGas:  gasMStore8,
			memorySize:  memoryMStore8,
			minStack:    minStack(2, 0),
		},
		SLOAD: {
			execute:     opSload,
			constantGas: params.SloadGasFrontier,
			minStack:    minStack(1, 1),
		},
		SSTORE: {
			execute:    opSstore,
			dynamicGas: gasSStore,
			minStack:   minStack(2, 0),
		},
		JUMP: {
			execute:     opJump,
			constantGas: GasMidStep,
			minStack:    minStack(1, 0),
		},
		JUMPI: {
			execute:     opJumpi,
			constantGas: GasSlowStep,
			minStack:    minStack(2, 0),
		},
		PC: {
			execute:      opPc,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		MSIZE: {
			execute:      opMsize,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		GAS: {
			execute:      opGas,
			constantGas:  GasQuickStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		JUMPDEST: {
			execute:     opJumpdest,
			constantGas: params.JumpdestGas,
			minStack:    minStack(0, 0),
		},
		PUSH1: {
			execute:      opPush1,
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH2: {
			execute:      makePush(2, 2),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH3: {
			execute:      makePush(3, 3),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH4: {
			execute:      makePush(4, 4),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH5: {
			execute:      makePush(5, 5),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH6: {
			execute:      makePush(6, 6),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH7: {
			execute:      makePush(7, 7),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH8: {
			execute:      makePush(8, 8),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH9: {
			execute:      makePush(9, 9),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH10: {
			execute:      makePush(10, 10),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH11: {
			execute:      makePush(11, 11),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH12: {
			execute:      makePush(12, 12),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH13: {
			execute:      makePush(13, 13),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH14: {
			execute:      makePush(14, 14),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH15: {
			execute:      makePush(15, 15),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH16: {
			execute:      makePush(16, 16),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH17: {
			execute:      makePush(17, 17),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH18: {
			execute:      makePush(18, 18),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH19: {
			execute:      makePush(19, 19),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH20: {
			execute:      makePush(20, 20),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH21: {
			execute:      makePush(21, 21),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH22: {
			execute:      makePush(22, 22),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH23: {
			execute:      makePush(23, 23),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH24: {
			execute:      makePush(24, 24),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH25: {
			execute:      makePush(25, 25),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH26: {
			execute:      makePush(26, 26),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH27: {
			execute:      makePush(27, 27),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH28: {
			execute:      makePush(28, 28),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH29: {
			execute:      makePush(29, 29),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH30: {
			execute:      makePush(30, 30),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH31: {
			execute:      makePush(31, 31),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		PUSH32: {
			execute:      makePush(32, 32),
			constantGas:  GasFastestStep,
			minStack:     minStack(0, 1),
			expandsStack: true,
		},
		DUP1: {
			execute:      makeDup(1),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(1),
			expandsStack: true,
		},
		DUP2: {
			execute:      makeDup(2),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(2),
			expandsStack: true,
		},
		DUP3: {
			execute:      makeDup(3),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(3),
			expandsStack: true,
		},
		DUP4: {
			execute:      makeDup(4),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(4),
			expandsStack: true,
		},
		DUP5: {
			execute:      makeDup(5),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(5),
			expandsStack: true,
		},
		DUP6: {
			execute:      makeDup(6),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(6),
			expandsStack: true,
		},
		DUP7: {
			execute:      makeDup(7),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(7),
			expandsStack: true,
		},
		DUP8: {
			execute:      makeDup(8),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(8),
			expandsStack: true,
		},
		DUP9: {
			execute:      makeDup(9),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(9),
			expandsStack: true,
		},
		DUP10: {
			execute:      makeDup(10),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(10),
			expandsStack: true,
		},
		DUP11: {
			execute:      makeDup(11),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(11),
			expandsStack: true,
		},
		DUP12: {
			execute:      makeDup(12),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(12),
			expandsStack: true,
		},
		DUP13: {
			execute:      makeDup(13),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(13),
			expandsStack: true,
		},
		DUP14: {
			execute:      makeDup(14),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(14),
			expandsStack: true,
		},
		DUP15: {
			execute:      makeDup(15),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(15),
			expandsStack: true,
		},
		DUP16: {
			execute:      makeDup(16),
			constantGas:  GasFastestStep,
			minStack:     minDupStack(16),
			expandsStack: true,
		},
		SWAP1: {
			execute:     makeSwap(1),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(2),
		},
		SWAP2: {
			execute:     makeSwap(2),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(3),
		},
		SWAP3: {
			execute:     makeSwap(3),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(4),
		},
		SWAP4: {
			execute:     makeSwap(4),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(5),
		},
		SWAP5: {
			execute:     makeSwap(5),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(6),
		},
		SWAP6: {
			execute:     makeSwap(6),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(7),
		},
		SWAP7: {
			execute:     makeSwap(7),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(8),
		},
		SWAP8: {
			execute:     makeSwap(8),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(9),
		},
		SWAP9: {
			execute:     makeSwap(9),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(10),
		},
		SWAP10: {
			execute:     makeSwap(10),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(11),
		},
		SWAP11: {
			execute:     makeSwap(11),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(12),
		},
		SWAP12: {
			execute:     makeSwap(12),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(13),
		},
		SWAP13: {
			execute:     makeSwap(13),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(14),
		},
		SWAP14: {
			execute:     makeSwap(14),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(15),
		},
		SWAP15: {
			execute:     makeSwap(15),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(16),
		},
		SWAP16: {
			execute:     makeSwap(16),
			constantGas: GasFastestStep,
			minStack:    minSwapStack(17),
		},
		LOG0: {
			execute:    makeLog(0),
			dynamicGas: makeGasLog(0),
			minStack:   minStack(2, 0),
			memorySize: memoryLog,
		},
		LOG1: {
			execute:    makeLog(1),
			dynamicGas: makeGasLog(1),
			minStack:   minStack(3, 0),
			memorySize: memoryLog,
		},
		LOG2: {
			execute:    makeLog(2),
			dynamicGas: makeGasLog(2),
			minStack:   minStack(4, 0),
			memorySize: memoryLog,
		},
		LOG3: {
			execute:    makeLog(3),
			dynamicGas: makeGasLog(3),
			minStack:   minStack(5, 0),
			memorySize: memoryLog,
		},
		LOG4: {
			execute:    makeLog(4),
			dynamicGas: makeGasLog(4),
			minStack:   minStack(6, 0),
			memorySize: memoryLog,
		},
		CREATE: {
			execute:     opCreate,
			constantGas: params.CreateGas,
			dynamicGas:  gasCreate,
			minStack:    minStack(3, 1),
			memorySize:  memoryCreate,
		},
		CALL: {
			execute:     opCall,
			constantGas: params.CallGasFrontier,
			dynamicGas:  gasCall,
			minStack:    minStack(7, 1),
			memorySize:  memoryCall,
		},
		CALLCODE: {
			execute:     opCallCode,
			constantGas: params.CallGasFrontier,
			dynamicGas:  gasCallCode,
			minStack:    minStack(7, 1),
			memorySize:  memoryCall,
		},
		RETURN: {
			execute:    opReturn,
			dynamicGas: gasReturn,
			minStack:   minStack(2, 0),
			memorySize: memoryReturn,
		},
		SELFDESTRUCT: {
			execute:    opSelfdestruct,
			dynamicGas: gasSelfdestruct,
			minStack:   minStack(1, 0),
		},
	}

	// Fill all unassigned slots with opUndefined.
	for i, entry := range tbl {
		if entry == nil {
			tbl[i] = &operation{execute: opUndefined}
		}
	}

	return validate(tbl)
}
