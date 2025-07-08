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

type executionFunc func(pc *uint64, interpreter *EVMInterpreter, callContext *ScopeContext) ([]byte, error)

type operation struct {
	// execute is the operation function
	execute     executionFunc
	constantGas uint64
	// undefined denotes if the instruction is not officially defined in the jump table
	undefined bool
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
	mergeInstructionSet            = newMergeInstructionSet()
	shanghaiInstructionSet         = newShanghaiInstructionSet()
	cancunInstructionSet           = newCancunInstructionSet()
	verkleInstructionSet           = newVerkleInstructionSet()
	pragueInstructionSet           = newPragueInstructionSet()
	osakaInstructionSet            = newOsakaInstructionSet()
)

// JumpTable contains the EVM opcodes supported at a given fork.
type JumpTable [256]*operation

func validate(jt JumpTable) JumpTable {
	for i, op := range jt {
		if op == nil {
			panic(fmt.Sprintf("op %#x is not set", i))
		}
	}
	return jt
}

func newVerkleInstructionSet() JumpTable {
	instructionSet := newShanghaiInstructionSet()
	enable4762(&instructionSet)
	return validate(instructionSet)
}

func newOsakaInstructionSet() JumpTable {
	instructionSet := newPragueInstructionSet()
	enable7939(&instructionSet) // EIP-7939 (CLZ opcode)
	return validate(instructionSet)
}

func newPragueInstructionSet() JumpTable {
	instructionSet := newCancunInstructionSet()
	enable7702(&instructionSet) // EIP-7702 Setcode transaction type
	return validate(instructionSet)
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
	instructionSet := newMergeInstructionSet()
	enable3855(&instructionSet) // PUSH0 instruction
	enable3860(&instructionSet) // Limit and meter initcode

	return validate(instructionSet)
}

func newMergeInstructionSet() JumpTable {
	instructionSet := newLondonInstructionSet()
	instructionSet[PREVRANDAO] = &operation{
		execute:     opRandom,
		constantGas: GasQuickStep,
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
	enable2929(&instructionSet) // Gas cost increases for state access opcodes https://eips.ethereum.org/EIPS/eip-2929
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
		execute:     opSHL,
		constantGas: GasFastestStep,
	}
	instructionSet[SHR] = &operation{
		execute:     opSHR,
		constantGas: GasFastestStep,
	}
	instructionSet[SAR] = &operation{
		execute:     opSAR,
		constantGas: GasFastestStep,
	}
	instructionSet[EXTCODEHASH] = &operation{
		execute:     opExtCodeHashConstantinople,
		constantGas: params.ExtcodeHashGasConstantinople,
	}
	instructionSet[CREATE2] = &operation{
		execute:     opCreate2Constantinople,
		constantGas: params.Create2Gas,
	}
	return validate(instructionSet)
}

// newByzantiumInstructionSet returns the frontier, homestead and
// byzantium instructions.
func newByzantiumInstructionSet() JumpTable {
	instructionSet := newSpuriousDragonInstructionSet()
	instructionSet[STATICCALL] = &operation{
		execute:     opStaticCallByzantium,
		constantGas: params.CallGasEIP150,
	}
	instructionSet[RETURNDATASIZE] = &operation{
		execute:     opReturnDataSize,
		constantGas: GasQuickStep,
	}
	instructionSet[RETURNDATACOPY] = &operation{
		execute:     opReturnDataCopy,
		constantGas: GasFastestStep,
	}
	instructionSet[REVERT] = &operation{
		execute: opRevert,
	}
	return validate(instructionSet)
}

// EIP 158 a.k.a Spurious Dragon
func newSpuriousDragonInstructionSet() JumpTable {
	instructionSet := newTangerineWhistleInstructionSet()
	instructionSet[EXP].execute = opExpEIP158
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
		execute:     opDelegateCallHomestead,
		constantGas: params.CallGasFrontier,
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
		},
		ADD: {
			execute:     opAdd,
			constantGas: GasFastestStep,
		},
		MUL: {
			execute:     opMul,
			constantGas: GasFastStep,
		},
		SUB: {
			execute:     opSub,
			constantGas: GasFastestStep,
		},
		DIV: {
			execute:     opDiv,
			constantGas: GasFastStep,
		},
		SDIV: {
			execute:     opSdiv,
			constantGas: GasFastStep,
		},
		MOD: {
			execute:     opMod,
			constantGas: GasFastStep,
		},
		SMOD: {
			execute:     opSmod,
			constantGas: GasFastStep,
		},
		ADDMOD: {
			execute:     opAddmod,
			constantGas: GasMidStep,
		},
		MULMOD: {
			execute:     opMulmod,
			constantGas: GasMidStep,
		},
		EXP: {
			execute: opExpFrontier,
		},
		SIGNEXTEND: {
			execute:     opSignExtend,
			constantGas: GasFastStep,
		},
		LT: {
			execute:     opLt,
			constantGas: GasFastestStep,
		},
		GT: {
			execute:     opGt,
			constantGas: GasFastestStep,
		},
		SLT: {
			execute:     opSlt,
			constantGas: GasFastestStep,
		},
		SGT: {
			execute:     opSgt,
			constantGas: GasFastestStep,
		},
		EQ: {
			execute:     opEq,
			constantGas: GasFastestStep,
		},
		ISZERO: {
			execute:     opIszero,
			constantGas: GasFastestStep,
		},
		AND: {
			execute:     opAnd,
			constantGas: GasFastestStep,
		},
		XOR: {
			execute:     opXor,
			constantGas: GasFastestStep,
		},
		OR: {
			execute:     opOr,
			constantGas: GasFastestStep,
		},
		NOT: {
			execute:     opNot,
			constantGas: GasFastestStep,
		},
		BYTE: {
			execute:     opByte,
			constantGas: GasFastestStep,
		},
		KECCAK256: {
			execute:     opKeccak256,
			constantGas: params.Keccak256Gas,
		},
		ADDRESS: {
			execute:     opAddress,
			constantGas: GasQuickStep,
		},
		BALANCE: {
			execute:     opBalanceFrontier,
			constantGas: params.BalanceGasFrontier,
		},
		ORIGIN: {
			execute:     opOrigin,
			constantGas: GasQuickStep,
		},
		CALLER: {
			execute:     opCaller,
			constantGas: GasQuickStep,
		},
		CALLVALUE: {
			execute:     opCallValue,
			constantGas: GasQuickStep,
		},
		CALLDATALOAD: {
			execute:     opCallDataLoad,
			constantGas: GasFastestStep,
		},
		CALLDATASIZE: {
			execute:     opCallDataSize,
			constantGas: GasQuickStep,
		},
		CALLDATACOPY: {
			execute:     opCallDataCopy,
			constantGas: GasFastestStep,
		},
		CODESIZE: {
			execute:     opCodeSize,
			constantGas: GasQuickStep,
		},
		CODECOPY: {
			execute:     opCodeCopyFrontier,
			constantGas: GasFastestStep,
		},
		GASPRICE: {
			execute:     opGasprice,
			constantGas: GasQuickStep,
		},
		EXTCODESIZE: {
			execute:     opExtCodeSizeFrontier,
			constantGas: params.ExtcodeSizeGasFrontier,
		},
		EXTCODECOPY: {
			execute:     opExtCodeCopyFrontier,
			constantGas: params.ExtcodeCopyBaseFrontier,
		},
		BLOCKHASH: {
			execute:     opBlockhash,
			constantGas: GasExtStep,
		},
		COINBASE: {
			execute:     opCoinbase,
			constantGas: GasQuickStep,
		},
		TIMESTAMP: {
			execute:     opTimestamp,
			constantGas: GasQuickStep,
		},
		NUMBER: {
			execute:     opNumber,
			constantGas: GasQuickStep,
		},
		DIFFICULTY: {
			execute:     opDifficulty,
			constantGas: GasQuickStep,
		},
		GASLIMIT: {
			execute:     opGasLimit,
			constantGas: GasQuickStep,
		},
		POP: {
			execute:     opPop,
			constantGas: GasQuickStep,
		},
		MLOAD: {
			execute:     opMload,
			constantGas: GasFastestStep,
		},
		MSTORE: {
			execute:     opMstore,
			constantGas: GasFastestStep,
		},
		MSTORE8: {
			execute:     opMstore8,
			constantGas: GasFastestStep,
		},
		SLOAD: {
			execute:     opSLoadFrontier,
			constantGas: params.SloadGasFrontier,
		},
		SSTORE: {
			execute: opSstoreFrontier,
		},
		JUMP: {
			execute:     opJump,
			constantGas: GasMidStep,
		},
		JUMPI: {
			execute:     opJumpi,
			constantGas: GasSlowStep,
		},
		PC: {
			execute:     opPc,
			constantGas: GasQuickStep,
		},
		MSIZE: {
			execute:     opMsize,
			constantGas: GasQuickStep,
		},
		GAS: {
			execute:     opGas,
			constantGas: GasQuickStep,
		},
		JUMPDEST: {
			execute:     opJumpdest,
			constantGas: params.JumpdestGas,
		},
		PUSH1: {
			execute:     opPush1,
			constantGas: GasFastestStep,
		},
		PUSH2: {
			execute:     opPush2,
			constantGas: GasFastestStep,
		},
		PUSH3: {
			execute:     makePush(3, 3),
			constantGas: GasFastestStep,
		},
		PUSH4: {
			execute:     makePush(4, 4),
			constantGas: GasFastestStep,
		},
		PUSH5: {
			execute:     makePush(5, 5),
			constantGas: GasFastestStep,
		},
		PUSH6: {
			execute:     makePush(6, 6),
			constantGas: GasFastestStep,
		},
		PUSH7: {
			execute:     makePush(7, 7),
			constantGas: GasFastestStep,
		},
		PUSH8: {
			execute:     makePush(8, 8),
			constantGas: GasFastestStep,
		},
		PUSH9: {
			execute:     makePush(9, 9),
			constantGas: GasFastestStep,
		},
		PUSH10: {
			execute:     makePush(10, 10),
			constantGas: GasFastestStep,
		},
		PUSH11: {
			execute:     makePush(11, 11),
			constantGas: GasFastestStep,
		},
		PUSH12: {
			execute:     makePush(12, 12),
			constantGas: GasFastestStep,
		},
		PUSH13: {
			execute:     makePush(13, 13),
			constantGas: GasFastestStep,
		},
		PUSH14: {
			execute:     makePush(14, 14),
			constantGas: GasFastestStep,
		},
		PUSH15: {
			execute:     makePush(15, 15),
			constantGas: GasFastestStep,
		},
		PUSH16: {
			execute:     makePush(16, 16),
			constantGas: GasFastestStep,
		},
		PUSH17: {
			execute:     makePush(17, 17),
			constantGas: GasFastestStep,
		},
		PUSH18: {
			execute:     makePush(18, 18),
			constantGas: GasFastestStep,
		},
		PUSH19: {
			execute:     makePush(19, 19),
			constantGas: GasFastestStep,
		},
		PUSH20: {
			execute:     makePush(20, 20),
			constantGas: GasFastestStep,
		},
		PUSH21: {
			execute:     makePush(21, 21),
			constantGas: GasFastestStep,
		},
		PUSH22: {
			execute:     makePush(22, 22),
			constantGas: GasFastestStep,
		},
		PUSH23: {
			execute:     makePush(23, 23),
			constantGas: GasFastestStep,
		},
		PUSH24: {
			execute:     makePush(24, 24),
			constantGas: GasFastestStep,
		},
		PUSH25: {
			execute:     makePush(25, 25),
			constantGas: GasFastestStep,
		},
		PUSH26: {
			execute:     makePush(26, 26),
			constantGas: GasFastestStep,
		},
		PUSH27: {
			execute:     makePush(27, 27),
			constantGas: GasFastestStep,
		},
		PUSH28: {
			execute:     makePush(28, 28),
			constantGas: GasFastestStep,
		},
		PUSH29: {
			execute:     makePush(29, 29),
			constantGas: GasFastestStep,
		},
		PUSH30: {
			execute:     makePush(30, 30),
			constantGas: GasFastestStep,
		},
		PUSH31: {
			execute:     makePush(31, 31),
			constantGas: GasFastestStep,
		},
		PUSH32: {
			execute:     makePush(32, 32),
			constantGas: GasFastestStep,
		},
		DUP1: {
			execute:     opDup1,
			constantGas: GasFastestStep,
		},
		DUP2: {
			execute:     opDup2,
			constantGas: GasFastestStep,
		},
		DUP3: {
			execute:     opDup3,
			constantGas: GasFastestStep,
		},
		DUP4: {
			execute:     opDup4,
			constantGas: GasFastestStep,
		},
		DUP5: {
			execute:     opDup5,
			constantGas: GasFastestStep,
		},
		DUP6: {
			execute:     opDup6,
			constantGas: GasFastestStep,
		},
		DUP7: {
			execute:     opDup7,
			constantGas: GasFastestStep,
		},
		DUP8: {
			execute:     opDup8,
			constantGas: GasFastestStep,
		},
		DUP9: {
			execute:     opDup9,
			constantGas: GasFastestStep,
		},
		DUP10: {
			execute:     opDup10,
			constantGas: GasFastestStep,
		},
		DUP11: {
			execute:     opDup11,
			constantGas: GasFastestStep,
		},
		DUP12: {
			execute:     opDup12,
			constantGas: GasFastestStep,
		},
		DUP13: {
			execute:     opDup13,
			constantGas: GasFastestStep,
		},
		DUP14: {
			execute:     opDup14,
			constantGas: GasFastestStep,
		},
		DUP15: {
			execute:     opDup15,
			constantGas: GasFastestStep,
		},
		DUP16: {
			execute:     opDup16,
			constantGas: GasFastestStep,
		},
		SWAP1: {
			execute:     opSwap1,
			constantGas: GasFastestStep,
		},
		SWAP2: {
			execute:     opSwap2,
			constantGas: GasFastestStep,
		},
		SWAP3: {
			execute:     opSwap3,
			constantGas: GasFastestStep,
		},
		SWAP4: {
			execute:     opSwap4,
			constantGas: GasFastestStep,
		},
		SWAP5: {
			execute:     opSwap5,
			constantGas: GasFastestStep,
		},
		SWAP6: {
			execute:     opSwap6,
			constantGas: GasFastestStep,
		},
		SWAP7: {
			execute:     opSwap7,
			constantGas: GasFastestStep,
		},
		SWAP8: {
			execute:     opSwap8,
			constantGas: GasFastestStep,
		},
		SWAP9: {
			execute:     opSwap9,
			constantGas: GasFastestStep,
		},
		SWAP10: {
			execute:     opSwap10,
			constantGas: GasFastestStep,
		},
		SWAP11: {
			execute:     opSwap11,
			constantGas: GasFastestStep,
		},
		SWAP12: {
			execute:     opSwap12,
			constantGas: GasFastestStep,
		},
		SWAP13: {
			execute:     opSwap13,
			constantGas: GasFastestStep,
		},
		SWAP14: {
			execute:     opSwap14,
			constantGas: GasFastestStep,
		},
		SWAP15: {
			execute:     opSwap15,
			constantGas: GasFastestStep,
		},
		SWAP16: {
			execute:     opSwap16,
			constantGas: GasFastestStep,
		},
		LOG0: {
			execute: makeLog(0),
		},
		LOG1: {
			execute: makeLog(1),
		},
		LOG2: {
			execute: makeLog(2),
		},
		LOG3: {
			execute: makeLog(3),
		},
		LOG4: {
			execute: makeLog(4),
		},
		CREATE: {
			execute:     opCreateFrontier,
			constantGas: params.CreateGas,
		},
		CALL: {
			execute:     opCallFrontier,
			constantGas: params.CallGasFrontier,
		},
		CALLCODE: {
			execute:     opCallCodeFrontier,
			constantGas: params.CallGasFrontier,
		},
		RETURN: {
			execute: opReturn,
		},
		SELFDESTRUCT: {
			execute: opSelfdestructFrontier,
		},
		INVALID: {
			execute: opUndefined,
		},
	}

	// Fill all unassigned slots with opUndefined.
	for i, entry := range tbl {
		if entry == nil {
			tbl[i] = &operation{execute: opUndefined, undefined: true}
		}
	}

	return validate(tbl)
}

func copyJumpTable(source *JumpTable) *JumpTable {
	dest := *source
	for i, op := range source {
		if op != nil {
			opCopy := *op
			dest[i] = &opCopy
		}
	}
	return &dest
}
