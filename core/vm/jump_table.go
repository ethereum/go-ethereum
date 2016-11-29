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
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

type (
	executionFunc       func(pc *uint64, env *Environment, contract *Contract, memory *Memory, stack *Stack) ([]byte, error)
	gasFunc             func(params.GasTable, *Environment, *Contract, *Stack, *Memory, *big.Int) *big.Int
	stackValidationFunc func(*Stack) error
	memorySizeFunc      func(*Stack) *big.Int
)

type operation struct {
	// op is the operation function
	execute executionFunc
	// gasCost is the gas function and returns the gas required for execution
	gasCost gasFunc
	// validateStack validates the stack (size) for the operation
	validateStack stackValidationFunc
	// memorySize returns the memory size required for the operation
	memorySize memorySizeFunc
	// halts indicates whether the operation shoult halt further execution
	// and return
	halts bool
	// jumps indicates whether operation made a jump. This prevents the program
	// counter from further incrementing.
	jumps bool
	// valid is used to check whether the retrieved operation is valid and known
	valid bool
}

type vmJumpTable [256]operation

func newJumpTable(ruleset *params.ChainConfig, blockNumber *big.Int) vmJumpTable {
	var jumpTable vmJumpTable

	jumpTable[ADD] = operation{
		execute:       opAdd,
		gasCost:       makeGenericGasFunc(ADD),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[SUB] = operation{
		execute:       opSub,
		gasCost:       makeGenericGasFunc(SUB),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[MUL] = operation{
		execute:       opMul,
		gasCost:       makeGenericGasFunc(MUL),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[DIV] = operation{
		execute:       opDiv,
		gasCost:       makeGenericGasFunc(DIV),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[SDIV] = operation{
		execute:       opSdiv,
		gasCost:       makeGenericGasFunc(SDIV),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[MOD] = operation{
		execute:       opMod,
		gasCost:       makeGenericGasFunc(MOD),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[SMOD] = operation{
		execute:       opSmod,
		gasCost:       makeGenericGasFunc(SMOD),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[EXP] = operation{
		execute:       opExp,
		gasCost:       gasExp,
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[SIGNEXTEND] = operation{
		execute:       opSignExtend,
		gasCost:       makeGenericGasFunc(SIGNEXTEND),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[NOT] = operation{
		execute:       opNot,
		gasCost:       makeGenericGasFunc(NOT),
		validateStack: makeStackFunc(1, 1),
		valid:         true,
	}
	jumpTable[LT] = operation{
		execute:       opLt,
		gasCost:       makeGenericGasFunc(LT),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[GT] = operation{
		execute:       opGt,
		gasCost:       makeGenericGasFunc(GT),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[SLT] = operation{
		execute:       opSlt,
		gasCost:       makeGenericGasFunc(SLT),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[SGT] = operation{
		execute:       opSgt,
		gasCost:       makeGenericGasFunc(SGT),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[EQ] = operation{
		execute:       opEq,
		gasCost:       makeGenericGasFunc(EQ),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[ISZERO] = operation{
		execute:       opIszero,
		gasCost:       makeGenericGasFunc(ISZERO),
		validateStack: makeStackFunc(1, 1),
		valid:         true,
	}
	jumpTable[AND] = operation{
		execute:       opAnd,
		gasCost:       makeGenericGasFunc(AND),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[OR] = operation{
		execute:       opOr,
		gasCost:       makeGenericGasFunc(OR),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[XOR] = operation{
		execute:       opXor,
		gasCost:       makeGenericGasFunc(XOR),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[BYTE] = operation{
		execute:       opByte,
		gasCost:       makeGenericGasFunc(BYTE),
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[ADDMOD] = operation{
		execute:       opAddmod,
		gasCost:       makeGenericGasFunc(ADDMOD),
		validateStack: makeStackFunc(3, 1),
		valid:         true,
	}
	jumpTable[MULMOD] = operation{
		execute:       opMulmod,
		gasCost:       makeGenericGasFunc(MULMOD),
		validateStack: makeStackFunc(3, 1),
		valid:         true,
	}
	jumpTable[SHA3] = operation{
		execute:       opSha3,
		gasCost:       gasSha3,
		validateStack: makeStackFunc(2, 1),
		memorySize:    memorySha3,
		valid:         true,
	}
	jumpTable[ADDRESS] = operation{
		execute:       opAddress,
		gasCost:       makeGenericGasFunc(ADDRESS),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[BALANCE] = operation{
		execute:       opBalance,
		gasCost:       gasBalance,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[ORIGIN] = operation{
		execute:       opOrigin,
		gasCost:       makeGenericGasFunc(ORIGIN),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[CALLER] = operation{
		execute:       opCaller,
		gasCost:       makeGenericGasFunc(CALLER),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[CALLVALUE] = operation{
		execute:       opCallValue,
		gasCost:       makeGenericGasFunc(CALLVALUE),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[CALLDATALOAD] = operation{
		execute:       opCalldataLoad,
		gasCost:       makeGenericGasFunc(CALLDATALOAD),
		validateStack: makeStackFunc(1, 1),
		valid:         true,
	}
	jumpTable[CALLDATASIZE] = operation{
		execute:       opCalldataSize,
		gasCost:       makeGenericGasFunc(CALLDATASIZE),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[CALLDATACOPY] = operation{
		execute:       opCalldataCopy,
		gasCost:       gasCalldataCopy,
		validateStack: makeStackFunc(3, 1),
		memorySize:    memoryCalldataCopy,
		valid:         true,
	}
	jumpTable[CODESIZE] = operation{
		execute:       opCodeSize,
		gasCost:       makeGenericGasFunc(CODESIZE),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[EXTCODESIZE] = operation{
		execute:       opExtCodeSize,
		gasCost:       gasExtCodeSize,
		validateStack: makeStackFunc(1, 1),
		valid:         true,
	}
	jumpTable[CODECOPY] = operation{
		execute:       opCodeCopy,
		gasCost:       gasCodeCopy,
		validateStack: makeStackFunc(3, 0),
		memorySize:    memoryCodeCopy,
		valid:         true,
	}
	jumpTable[EXTCODECOPY] = operation{
		execute:       opExtCodeCopy,
		gasCost:       gasExtCodeCopy,
		validateStack: makeStackFunc(4, 0),
		memorySize:    memoryExtCodeCopy,
		valid:         true,
	}
	jumpTable[GASPRICE] = operation{
		execute:       opGasprice,
		gasCost:       makeGenericGasFunc(GASPRICE),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[BLOCKHASH] = operation{
		execute:       opBlockhash,
		gasCost:       makeGenericGasFunc(BLOCKHASH),
		validateStack: makeStackFunc(1, 1),
		valid:         true,
	}
	jumpTable[COINBASE] = operation{
		execute:       opCoinbase,
		gasCost:       makeGenericGasFunc(COINBASE),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[TIMESTAMP] = operation{
		execute:       opTimestamp,
		gasCost:       makeGenericGasFunc(TIMESTAMP),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[NUMBER] = operation{
		execute:       opNumber,
		gasCost:       makeGenericGasFunc(NUMBER),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[DIFFICULTY] = operation{
		execute:       opDifficulty,
		gasCost:       makeGenericGasFunc(DIFFICULTY),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[GASLIMIT] = operation{
		execute:       opGasLimit,
		gasCost:       makeGenericGasFunc(GASLIMIT),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[POP] = operation{
		execute:       opPop,
		gasCost:       makeGenericGasFunc(TIMESTAMP),
		validateStack: makeStackFunc(1, 0),
		valid:         true,
	}
	jumpTable[MLOAD] = operation{
		execute:       opMload,
		gasCost:       gasMLoad,
		validateStack: makeStackFunc(1, 1),
		memorySize:    memoryMLoad,
		valid:         true,
	}
	jumpTable[MSTORE] = operation{
		execute:       opMstore,
		gasCost:       gasMStore,
		validateStack: makeStackFunc(2, 0),
		memorySize:    memoryMStore,
		valid:         true,
	}
	jumpTable[MSTORE8] = operation{
		execute:       opMstore8,
		gasCost:       gasMStore8,
		memorySize:    memoryMStore8,
		validateStack: makeStackFunc(2, 0),

		valid: true,
	}
	jumpTable[SLOAD] = operation{
		execute:       opSload,
		gasCost:       gasSLoad,
		validateStack: makeStackFunc(1, 1),
		valid:         true,
	}
	jumpTable[SSTORE] = operation{
		execute:       opSstore,
		gasCost:       gasSStore,
		validateStack: makeStackFunc(2, 0),
		valid:         true,
	}
	jumpTable[JUMPDEST] = operation{
		execute:       opJumpdest,
		gasCost:       makeGenericGasFunc(JUMPDEST),
		validateStack: makeStackFunc(0, 0),
		valid:         true,
	}
	jumpTable[PC] = operation{
		execute:       opPc,
		gasCost:       makeGenericGasFunc(PC),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[MSIZE] = operation{
		execute:       opMsize,
		gasCost:       makeGenericGasFunc(MSIZE),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[GAS] = operation{
		execute:       opGas,
		gasCost:       makeGenericGasFunc(GAS),
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[CREATE] = operation{
		execute:       opCreate,
		gasCost:       gasCreate,
		validateStack: makeStackFunc(3, 1),
		memorySize:    memoryCreate,
		valid:         true,
	}
	jumpTable[CALL] = operation{
		execute:       opCall,
		gasCost:       gasCall,
		validateStack: makeStackFunc(7, 1),
		memorySize:    memoryCall,
		valid:         true,
	}
	jumpTable[CALLCODE] = operation{
		execute:       opCallCode,
		gasCost:       gasCallCode,
		validateStack: makeStackFunc(7, 1),
		memorySize:    memoryCall,
		valid:         true,
	}
	if ruleset.IsHomestead(blockNumber) {
		jumpTable[DELEGATECALL] = operation{
			execute:       opDelegateCall,
			gasCost:       gasDelegateCall,
			validateStack: makeStackFunc(6, 1),
			memorySize:    memoryDelegateCall,
			valid:         true,
		}
	}

	jumpTable[RETURN] = operation{
		execute:       opReturn,
		gasCost:       gasReturn,
		validateStack: makeStackFunc(2, 0),
		memorySize:    memoryReturn,
		halts:         true,
		valid:         true,
	}
	jumpTable[SUICIDE] = operation{
		execute:       opSuicide,
		gasCost:       gasSuicide,
		validateStack: makeStackFunc(1, 0),
		halts:         true,
		valid:         true,
	}
	jumpTable[JUMP] = operation{
		execute:       opJump,
		gasCost:       makeGenericGasFunc(JUMP),
		validateStack: makeStackFunc(1, 0),
		jumps:         true,
		valid:         true,
	}
	jumpTable[JUMPI] = operation{
		execute:       opJumpi,
		gasCost:       makeGenericGasFunc(JUMPI),
		validateStack: makeStackFunc(2, 0),
		jumps:         true,
		valid:         true,
	}
	jumpTable[STOP] = operation{
		execute:       opStop,
		gasCost:       makeGenericGasFunc(STOP),
		validateStack: makeStackFunc(0, 0),
		halts:         true,
		valid:         true,
	}
	jumpTable[LOG0] = operation{
		execute:       makeLog(0),
		gasCost:       makeGasLog(0),
		validateStack: makeStackFunc(2, 0),
		memorySize:    memoryLog,
		valid:         true,
	}
	jumpTable[LOG1] = operation{
		execute:       makeLog(1),
		gasCost:       makeGasLog(1),
		validateStack: makeStackFunc(3, 0),
		memorySize:    memoryLog,
		valid:         true,
	}
	jumpTable[LOG2] = operation{
		execute:       makeLog(2),
		gasCost:       makeGasLog(2),
		validateStack: makeStackFunc(4, 0),
		memorySize:    memoryLog,
		valid:         true,
	}
	jumpTable[LOG3] = operation{
		execute:       makeLog(3),
		gasCost:       makeGasLog(3),
		validateStack: makeStackFunc(5, 0),
		memorySize:    memoryLog,
		valid:         true,
	}
	jumpTable[LOG4] = operation{
		execute:       makeLog(4),
		gasCost:       makeGasLog(4),
		validateStack: makeStackFunc(6, 0),
		memorySize:    memoryLog,
		valid:         true,
	}
	jumpTable[SWAP1] = operation{
		execute:       makeSwap(1),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(2, 0),
		valid:         true,
	}
	jumpTable[SWAP2] = operation{
		execute:       makeSwap(2),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(3, 0),
		valid:         true,
	}
	jumpTable[SWAP3] = operation{
		execute:       makeSwap(3),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(4, 0),
		valid:         true,
	}
	jumpTable[SWAP4] = operation{
		execute:       makeSwap(4),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(5, 0),
		valid:         true,
	}
	jumpTable[SWAP5] = operation{
		execute:       makeSwap(5),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(6, 0),
		valid:         true,
	}
	jumpTable[SWAP6] = operation{
		execute:       makeSwap(6),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(7, 0),
		valid:         true,
	}
	jumpTable[SWAP7] = operation{
		execute:       makeSwap(7),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(8, 0),
		valid:         true,
	}
	jumpTable[SWAP8] = operation{
		execute:       makeSwap(8),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(9, 0),
		valid:         true,
	}
	jumpTable[SWAP9] = operation{
		execute:       makeSwap(9),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(10, 0),
		valid:         true,
	}
	jumpTable[SWAP10] = operation{
		execute:       makeSwap(10),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(11, 0),
		valid:         true,
	}
	jumpTable[SWAP11] = operation{
		execute:       makeSwap(11),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(12, 0),
		valid:         true,
	}
	jumpTable[SWAP12] = operation{
		execute:       makeSwap(12),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(13, 0),
		valid:         true,
	}
	jumpTable[SWAP13] = operation{
		execute:       makeSwap(13),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(14, 0),
		valid:         true,
	}
	jumpTable[SWAP14] = operation{
		execute:       makeSwap(14),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(15, 0),
		valid:         true,
	}
	jumpTable[SWAP15] = operation{
		execute:       makeSwap(15),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(16, 0),
		valid:         true,
	}
	jumpTable[SWAP16] = operation{
		execute:       makeSwap(16),
		gasCost:       gasSwap,
		validateStack: makeStackFunc(17, 0),
		valid:         true,
	}
	jumpTable[PUSH1] = operation{
		execute:       makePush(1, big.NewInt(1)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH2] = operation{
		execute:       makePush(2, big.NewInt(2)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH3] = operation{
		execute:       makePush(3, big.NewInt(3)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH4] = operation{
		execute:       makePush(4, big.NewInt(4)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH5] = operation{
		execute:       makePush(5, big.NewInt(5)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH6] = operation{
		execute:       makePush(6, big.NewInt(6)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH7] = operation{
		execute:       makePush(7, big.NewInt(7)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH8] = operation{
		execute:       makePush(8, big.NewInt(8)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH9] = operation{
		execute:       makePush(9, big.NewInt(9)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH10] = operation{
		execute:       makePush(10, big.NewInt(10)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH11] = operation{
		execute:       makePush(11, big.NewInt(11)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH12] = operation{
		execute:       makePush(12, big.NewInt(12)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH13] = operation{
		execute:       makePush(13, big.NewInt(13)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH14] = operation{
		execute:       makePush(14, big.NewInt(14)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH15] = operation{
		execute:       makePush(15, big.NewInt(15)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH16] = operation{
		execute:       makePush(16, big.NewInt(16)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH17] = operation{
		execute:       makePush(17, big.NewInt(17)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH18] = operation{
		execute:       makePush(18, big.NewInt(18)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH19] = operation{
		execute:       makePush(19, big.NewInt(19)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH20] = operation{
		execute:       makePush(20, big.NewInt(20)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH21] = operation{
		execute:       makePush(21, big.NewInt(21)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH22] = operation{
		execute:       makePush(22, big.NewInt(22)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH23] = operation{
		execute:       makePush(23, big.NewInt(23)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH24] = operation{
		execute:       makePush(24, big.NewInt(24)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH25] = operation{
		execute:       makePush(25, big.NewInt(25)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH26] = operation{
		execute:       makePush(26, big.NewInt(26)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH27] = operation{
		execute:       makePush(27, big.NewInt(27)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH28] = operation{
		execute:       makePush(28, big.NewInt(28)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH29] = operation{
		execute:       makePush(29, big.NewInt(29)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH30] = operation{
		execute:       makePush(30, big.NewInt(30)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH31] = operation{
		execute:       makePush(31, big.NewInt(31)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[PUSH32] = operation{
		execute:       makePush(32, big.NewInt(32)),
		gasCost:       gasPush,
		validateStack: makeStackFunc(0, 1),
		valid:         true,
	}
	jumpTable[DUP1] = operation{
		execute:       makeDup(1),
		gasCost:       gasDup,
		validateStack: makeStackFunc(1, 1),
		valid:         true,
	}
	jumpTable[DUP2] = operation{
		execute:       makeDup(2),
		gasCost:       gasDup,
		validateStack: makeStackFunc(2, 1),
		valid:         true,
	}
	jumpTable[DUP3] = operation{
		execute:       makeDup(3),
		gasCost:       gasDup,
		validateStack: makeStackFunc(3, 1),
		valid:         true,
	}
	jumpTable[DUP4] = operation{
		execute:       makeDup(4),
		gasCost:       gasDup,
		validateStack: makeStackFunc(4, 1),
		valid:         true,
	}
	jumpTable[DUP5] = operation{
		execute:       makeDup(5),
		gasCost:       gasDup,
		validateStack: makeStackFunc(5, 1),
		valid:         true,
	}
	jumpTable[DUP6] = operation{
		execute:       makeDup(6),
		gasCost:       gasDup,
		validateStack: makeStackFunc(6, 1),
		valid:         true,
	}
	jumpTable[DUP7] = operation{
		execute:       makeDup(7),
		gasCost:       gasDup,
		validateStack: makeStackFunc(7, 1),
		valid:         true,
	}
	jumpTable[DUP8] = operation{
		execute:       makeDup(8),
		gasCost:       gasDup,
		validateStack: makeStackFunc(8, 1),
		valid:         true,
	}
	jumpTable[DUP9] = operation{
		execute:       makeDup(9),
		gasCost:       gasDup,
		validateStack: makeStackFunc(9, 1),
		valid:         true,
	}
	jumpTable[DUP10] = operation{
		execute:       makeDup(10),
		gasCost:       gasDup,
		validateStack: makeStackFunc(10, 1),
		valid:         true,
	}
	jumpTable[DUP11] = operation{
		execute:       makeDup(11),
		gasCost:       gasDup,
		validateStack: makeStackFunc(11, 1),
		valid:         true,
	}
	jumpTable[DUP12] = operation{
		execute:       makeDup(12),
		gasCost:       gasDup,
		validateStack: makeStackFunc(12, 1),
		valid:         true,
	}
	jumpTable[DUP13] = operation{
		execute:       makeDup(13),
		gasCost:       gasDup,
		validateStack: makeStackFunc(13, 1),
		valid:         true,
	}
	jumpTable[DUP14] = operation{
		execute:       makeDup(14),
		gasCost:       gasDup,
		validateStack: makeStackFunc(14, 1),
		valid:         true,
	}
	jumpTable[DUP15] = operation{
		execute:       makeDup(15),
		gasCost:       gasDup,
		validateStack: makeStackFunc(15, 1),
		valid:         true,
	}
	jumpTable[DUP16] = operation{
		execute:       makeDup(16),
		gasCost:       gasDup,
		validateStack: makeStackFunc(16, 1),
		valid:         true,
	}

	return jumpTable
}
