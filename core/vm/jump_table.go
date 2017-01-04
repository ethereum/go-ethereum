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
	executionFunc       func(pc *uint64, env *EVM, contract *Contract, memory *Memory, stack *Stack) ([]byte, error)
	gasFunc             func(params.GasTable, *EVM, *Contract, *Stack, *Memory, *big.Int) *big.Int
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

var defaultJumpTable = NewJumpTable()

func NewJumpTable() [256]operation {
	return [256]operation{
		ADD: operation{
			execute:       opAdd,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SUB: operation{
			execute:       opSub,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		MUL: operation{
			execute:       opMul,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		DIV: operation{
			execute:       opDiv,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SDIV: operation{
			execute:       opSdiv,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		MOD: operation{
			execute:       opMod,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SMOD: operation{
			execute:       opSmod,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		EXP: operation{
			execute:       opExp,
			gasCost:       gasExp,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SIGNEXTEND: operation{
			execute:       opSignExtend,
			gasCost:       constGasFunc(GasFastStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		NOT: operation{
			execute:       opNot,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		LT: operation{
			execute:       opLt,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		GT: operation{
			execute:       opGt,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SLT: operation{
			execute:       opSlt,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		SGT: operation{
			execute:       opSgt,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		EQ: operation{
			execute:       opEq,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		ISZERO: operation{
			execute:       opIszero,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		AND: operation{
			execute:       opAnd,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		OR: operation{
			execute:       opOr,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		XOR: operation{
			execute:       opXor,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		BYTE: operation{
			execute:       opByte,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		ADDMOD: operation{
			execute:       opAddmod,
			gasCost:       constGasFunc(GasMidStep),
			validateStack: makeStackFunc(3, 1),
			valid:         true,
		},
		MULMOD: operation{
			execute:       opMulmod,
			gasCost:       constGasFunc(GasMidStep),
			validateStack: makeStackFunc(3, 1),
			valid:         true,
		},
		SHA3: operation{
			execute:       opSha3,
			gasCost:       gasSha3,
			validateStack: makeStackFunc(2, 1),
			memorySize:    memorySha3,
			valid:         true,
		},
		ADDRESS: operation{
			execute:       opAddress,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		BALANCE: operation{
			execute:       opBalance,
			gasCost:       gasBalance,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		ORIGIN: operation{
			execute:       opOrigin,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		CALLER: operation{
			execute:       opCaller,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		CALLVALUE: operation{
			execute:       opCallValue,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		CALLDATALOAD: operation{
			execute:       opCalldataLoad,
			gasCost:       constGasFunc(GasFastestStep),
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		CALLDATASIZE: operation{
			execute:       opCalldataSize,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		CALLDATACOPY: operation{
			execute:       opCalldataCopy,
			gasCost:       gasCalldataCopy,
			validateStack: makeStackFunc(3, 1),
			memorySize:    memoryCalldataCopy,
			valid:         true,
		},
		CODESIZE: operation{
			execute:       opCodeSize,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		EXTCODESIZE: operation{
			execute:       opExtCodeSize,
			gasCost:       gasExtCodeSize,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		CODECOPY: operation{
			execute:       opCodeCopy,
			gasCost:       gasCodeCopy,
			validateStack: makeStackFunc(3, 0),
			memorySize:    memoryCodeCopy,
			valid:         true,
		},
		EXTCODECOPY: operation{
			execute:       opExtCodeCopy,
			gasCost:       gasExtCodeCopy,
			validateStack: makeStackFunc(4, 0),
			memorySize:    memoryExtCodeCopy,
			valid:         true,
		},
		GASPRICE: operation{
			execute:       opGasprice,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		BLOCKHASH: operation{
			execute:       opBlockhash,
			gasCost:       constGasFunc(GasExtStep),
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		COINBASE: operation{
			execute:       opCoinbase,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		TIMESTAMP: operation{
			execute:       opTimestamp,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		NUMBER: operation{
			execute:       opNumber,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		DIFFICULTY: operation{
			execute:       opDifficulty,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		GASLIMIT: operation{
			execute:       opGasLimit,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		POP: operation{
			execute:       opPop,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(1, 0),
			valid:         true,
		},
		MLOAD: operation{
			execute:       opMload,
			gasCost:       gasMLoad,
			validateStack: makeStackFunc(1, 1),
			memorySize:    memoryMLoad,
			valid:         true,
		},
		MSTORE: operation{
			execute:       opMstore,
			gasCost:       gasMStore,
			validateStack: makeStackFunc(2, 0),
			memorySize:    memoryMStore,
			valid:         true,
		},
		MSTORE8: operation{
			execute:       opMstore8,
			gasCost:       gasMStore8,
			memorySize:    memoryMStore8,
			validateStack: makeStackFunc(2, 0),

			valid: true,
		},
		SLOAD: operation{
			execute:       opSload,
			gasCost:       gasSLoad,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		SSTORE: operation{
			execute:       opSstore,
			gasCost:       gasSStore,
			validateStack: makeStackFunc(2, 0),
			valid:         true,
		},
		JUMPDEST: operation{
			execute:       opJumpdest,
			gasCost:       constGasFunc(params.JumpdestGas),
			validateStack: makeStackFunc(0, 0),
			valid:         true,
		},
		PC: operation{
			execute:       opPc,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		MSIZE: operation{
			execute:       opMsize,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		GAS: operation{
			execute:       opGas,
			gasCost:       constGasFunc(GasQuickStep),
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		CREATE: operation{
			execute:       opCreate,
			gasCost:       gasCreate,
			validateStack: makeStackFunc(3, 1),
			memorySize:    memoryCreate,
			valid:         true,
		},
		CALL: operation{
			execute:       opCall,
			gasCost:       gasCall,
			validateStack: makeStackFunc(7, 1),
			memorySize:    memoryCall,
			valid:         true,
		},
		CALLCODE: operation{
			execute:       opCallCode,
			gasCost:       gasCallCode,
			validateStack: makeStackFunc(7, 1),
			memorySize:    memoryCall,
			valid:         true,
		},
		DELEGATECALL: operation{
			execute:       opDelegateCall,
			gasCost:       gasDelegateCall,
			validateStack: makeStackFunc(6, 1),
			memorySize:    memoryDelegateCall,
			valid:         true,
		},
		RETURN: operation{
			execute:       opReturn,
			gasCost:       gasReturn,
			validateStack: makeStackFunc(2, 0),
			memorySize:    memoryReturn,
			halts:         true,
			valid:         true,
		},
		SUICIDE: operation{
			execute:       opSuicide,
			gasCost:       gasSuicide,
			validateStack: makeStackFunc(1, 0),
			halts:         true,
			valid:         true,
		},
		JUMP: operation{
			execute:       opJump,
			gasCost:       constGasFunc(GasMidStep),
			validateStack: makeStackFunc(1, 0),
			jumps:         true,
			valid:         true,
		},
		JUMPI: operation{
			execute:       opJumpi,
			gasCost:       constGasFunc(GasSlowStep),
			validateStack: makeStackFunc(2, 0),
			jumps:         true,
			valid:         true,
		},
		STOP: operation{
			execute:       opStop,
			gasCost:       constGasFunc(Zero),
			validateStack: makeStackFunc(0, 0),
			halts:         true,
			valid:         true,
		},
		LOG0: operation{
			execute:       makeLog(0),
			gasCost:       makeGasLog(0),
			validateStack: makeStackFunc(2, 0),
			memorySize:    memoryLog,
			valid:         true,
		},
		LOG1: operation{
			execute:       makeLog(1),
			gasCost:       makeGasLog(1),
			validateStack: makeStackFunc(3, 0),
			memorySize:    memoryLog,
			valid:         true,
		},
		LOG2: operation{
			execute:       makeLog(2),
			gasCost:       makeGasLog(2),
			validateStack: makeStackFunc(4, 0),
			memorySize:    memoryLog,
			valid:         true,
		},
		LOG3: operation{
			execute:       makeLog(3),
			gasCost:       makeGasLog(3),
			validateStack: makeStackFunc(5, 0),
			memorySize:    memoryLog,
			valid:         true,
		},
		LOG4: operation{
			execute:       makeLog(4),
			gasCost:       makeGasLog(4),
			validateStack: makeStackFunc(6, 0),
			memorySize:    memoryLog,
			valid:         true,
		},
		SWAP1: operation{
			execute:       makeSwap(1),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(2, 0),
			valid:         true,
		},
		SWAP2: operation{
			execute:       makeSwap(2),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(3, 0),
			valid:         true,
		},
		SWAP3: operation{
			execute:       makeSwap(3),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(4, 0),
			valid:         true,
		},
		SWAP4: operation{
			execute:       makeSwap(4),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(5, 0),
			valid:         true,
		},
		SWAP5: operation{
			execute:       makeSwap(5),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(6, 0),
			valid:         true,
		},
		SWAP6: operation{
			execute:       makeSwap(6),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(7, 0),
			valid:         true,
		},
		SWAP7: operation{
			execute:       makeSwap(7),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(8, 0),
			valid:         true,
		},
		SWAP8: operation{
			execute:       makeSwap(8),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(9, 0),
			valid:         true,
		},
		SWAP9: operation{
			execute:       makeSwap(9),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(10, 0),
			valid:         true,
		},
		SWAP10: operation{
			execute:       makeSwap(10),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(11, 0),
			valid:         true,
		},
		SWAP11: operation{
			execute:       makeSwap(11),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(12, 0),
			valid:         true,
		},
		SWAP12: operation{
			execute:       makeSwap(12),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(13, 0),
			valid:         true,
		},
		SWAP13: operation{
			execute:       makeSwap(13),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(14, 0),
			valid:         true,
		},
		SWAP14: operation{
			execute:       makeSwap(14),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(15, 0),
			valid:         true,
		},
		SWAP15: operation{
			execute:       makeSwap(15),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(16, 0),
			valid:         true,
		},
		SWAP16: operation{
			execute:       makeSwap(16),
			gasCost:       gasSwap,
			validateStack: makeStackFunc(17, 0),
			valid:         true,
		},
		PUSH1: operation{
			execute:       makePush(1, big.NewInt(1)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH2: operation{
			execute:       makePush(2, big.NewInt(2)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH3: operation{
			execute:       makePush(3, big.NewInt(3)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH4: operation{
			execute:       makePush(4, big.NewInt(4)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH5: operation{
			execute:       makePush(5, big.NewInt(5)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH6: operation{
			execute:       makePush(6, big.NewInt(6)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH7: operation{
			execute:       makePush(7, big.NewInt(7)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH8: operation{
			execute:       makePush(8, big.NewInt(8)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH9: operation{
			execute:       makePush(9, big.NewInt(9)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH10: operation{
			execute:       makePush(10, big.NewInt(10)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH11: operation{
			execute:       makePush(11, big.NewInt(11)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH12: operation{
			execute:       makePush(12, big.NewInt(12)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH13: operation{
			execute:       makePush(13, big.NewInt(13)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH14: operation{
			execute:       makePush(14, big.NewInt(14)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH15: operation{
			execute:       makePush(15, big.NewInt(15)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH16: operation{
			execute:       makePush(16, big.NewInt(16)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH17: operation{
			execute:       makePush(17, big.NewInt(17)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH18: operation{
			execute:       makePush(18, big.NewInt(18)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH19: operation{
			execute:       makePush(19, big.NewInt(19)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH20: operation{
			execute:       makePush(20, big.NewInt(20)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH21: operation{
			execute:       makePush(21, big.NewInt(21)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH22: operation{
			execute:       makePush(22, big.NewInt(22)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH23: operation{
			execute:       makePush(23, big.NewInt(23)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH24: operation{
			execute:       makePush(24, big.NewInt(24)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH25: operation{
			execute:       makePush(25, big.NewInt(25)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH26: operation{
			execute:       makePush(26, big.NewInt(26)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH27: operation{
			execute:       makePush(27, big.NewInt(27)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH28: operation{
			execute:       makePush(28, big.NewInt(28)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH29: operation{
			execute:       makePush(29, big.NewInt(29)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH30: operation{
			execute:       makePush(30, big.NewInt(30)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH31: operation{
			execute:       makePush(31, big.NewInt(31)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		PUSH32: operation{
			execute:       makePush(32, big.NewInt(32)),
			gasCost:       gasPush,
			validateStack: makeStackFunc(0, 1),
			valid:         true,
		},
		DUP1: operation{
			execute:       makeDup(1),
			gasCost:       gasDup,
			validateStack: makeStackFunc(1, 1),
			valid:         true,
		},
		DUP2: operation{
			execute:       makeDup(2),
			gasCost:       gasDup,
			validateStack: makeStackFunc(2, 1),
			valid:         true,
		},
		DUP3: operation{
			execute:       makeDup(3),
			gasCost:       gasDup,
			validateStack: makeStackFunc(3, 1),
			valid:         true,
		},
		DUP4: operation{
			execute:       makeDup(4),
			gasCost:       gasDup,
			validateStack: makeStackFunc(4, 1),
			valid:         true,
		},
		DUP5: operation{
			execute:       makeDup(5),
			gasCost:       gasDup,
			validateStack: makeStackFunc(5, 1),
			valid:         true,
		},
		DUP6: operation{
			execute:       makeDup(6),
			gasCost:       gasDup,
			validateStack: makeStackFunc(6, 1),
			valid:         true,
		},
		DUP7: operation{
			execute:       makeDup(7),
			gasCost:       gasDup,
			validateStack: makeStackFunc(7, 1),
			valid:         true,
		},
		DUP8: operation{
			execute:       makeDup(8),
			gasCost:       gasDup,
			validateStack: makeStackFunc(8, 1),
			valid:         true,
		},
		DUP9: operation{
			execute:       makeDup(9),
			gasCost:       gasDup,
			validateStack: makeStackFunc(9, 1),
			valid:         true,
		},
		DUP10: operation{
			execute:       makeDup(10),
			gasCost:       gasDup,
			validateStack: makeStackFunc(10, 1),
			valid:         true,
		},
		DUP11: operation{
			execute:       makeDup(11),
			gasCost:       gasDup,
			validateStack: makeStackFunc(11, 1),
			valid:         true,
		},
		DUP12: operation{
			execute:       makeDup(12),
			gasCost:       gasDup,
			validateStack: makeStackFunc(12, 1),
			valid:         true,
		},
		DUP13: operation{
			execute:       makeDup(13),
			gasCost:       gasDup,
			validateStack: makeStackFunc(13, 1),
			valid:         true,
		},
		DUP14: operation{
			execute:       makeDup(14),
			gasCost:       gasDup,
			validateStack: makeStackFunc(14, 1),
			valid:         true,
		},
		DUP15: operation{
			execute:       makeDup(15),
			gasCost:       gasDup,
			validateStack: makeStackFunc(15, 1),
			valid:         true,
		},
		DUP16: operation{
			execute:       makeDup(16),
			gasCost:       gasDup,
			validateStack: makeStackFunc(16, 1),
			valid:         true,
		},
	}
}
