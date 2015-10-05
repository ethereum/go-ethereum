package vm

import "math/big"

type jumpPtr struct {
	fn    instrFn
	valid bool
}

var jumpTable [256]jumpPtr

func init() {
	jumpTable[ADD] = jumpPtr{opAdd, true}
	jumpTable[SUB] = jumpPtr{opSub, true}
	jumpTable[MUL] = jumpPtr{opMul, true}
	jumpTable[DIV] = jumpPtr{opDiv, true}
	jumpTable[SDIV] = jumpPtr{opSdiv, true}
	jumpTable[MOD] = jumpPtr{opMod, true}
	jumpTable[SMOD] = jumpPtr{opSmod, true}
	jumpTable[EXP] = jumpPtr{opExp, true}
	jumpTable[SIGNEXTEND] = jumpPtr{opSignExtend, true}
	jumpTable[NOT] = jumpPtr{opNot, true}
	jumpTable[LT] = jumpPtr{opLt, true}
	jumpTable[GT] = jumpPtr{opGt, true}
	jumpTable[SLT] = jumpPtr{opSlt, true}
	jumpTable[SGT] = jumpPtr{opSgt, true}
	jumpTable[EQ] = jumpPtr{opEq, true}
	jumpTable[ISZERO] = jumpPtr{opIszero, true}
	jumpTable[AND] = jumpPtr{opAnd, true}
	jumpTable[OR] = jumpPtr{opOr, true}
	jumpTable[XOR] = jumpPtr{opXor, true}
	jumpTable[BYTE] = jumpPtr{opByte, true}
	jumpTable[ADDMOD] = jumpPtr{opAddmod, true}
	jumpTable[MULMOD] = jumpPtr{opMulmod, true}
	jumpTable[SHA3] = jumpPtr{opSha3, true}
	jumpTable[ADDRESS] = jumpPtr{opAddress, true}
	jumpTable[BALANCE] = jumpPtr{opBalance, true}
	jumpTable[ORIGIN] = jumpPtr{opOrigin, true}
	jumpTable[CALLER] = jumpPtr{opCaller, true}
	jumpTable[CALLVALUE] = jumpPtr{opCallValue, true}
	jumpTable[CALLDATALOAD] = jumpPtr{opCalldataLoad, true}
	jumpTable[CALLDATASIZE] = jumpPtr{opCalldataSize, true}
	jumpTable[CALLDATACOPY] = jumpPtr{opCalldataCopy, true}
	jumpTable[CODESIZE] = jumpPtr{opCodeSize, true}
	jumpTable[EXTCODESIZE] = jumpPtr{opExtCodeSize, true}
	jumpTable[CODECOPY] = jumpPtr{opCodeCopy, true}
	jumpTable[EXTCODECOPY] = jumpPtr{opExtCodeCopy, true}
	jumpTable[GASPRICE] = jumpPtr{opGasprice, true}
	jumpTable[BLOCKHASH] = jumpPtr{opBlockhash, true}
	jumpTable[COINBASE] = jumpPtr{opCoinbase, true}
	jumpTable[TIMESTAMP] = jumpPtr{opTimestamp, true}
	jumpTable[NUMBER] = jumpPtr{opNumber, true}
	jumpTable[DIFFICULTY] = jumpPtr{opDifficulty, true}
	jumpTable[GASLIMIT] = jumpPtr{opGasLimit, true}
	jumpTable[POP] = jumpPtr{opPop, true}
	jumpTable[MLOAD] = jumpPtr{opMload, true}
	jumpTable[MSTORE] = jumpPtr{opMstore, true}
	jumpTable[MSTORE8] = jumpPtr{opMstore8, true}
	jumpTable[SLOAD] = jumpPtr{opSload, true}
	jumpTable[SSTORE] = jumpPtr{opSstore, true}
	jumpTable[JUMPDEST] = jumpPtr{opJumpdest, true}
	jumpTable[PC] = jumpPtr{nil, true}
	jumpTable[MSIZE] = jumpPtr{opMsize, true}
	jumpTable[GAS] = jumpPtr{opGas, true}
	jumpTable[CREATE] = jumpPtr{opCreate, true}
	jumpTable[CALL] = jumpPtr{opCall, true}
	jumpTable[CALLCODE] = jumpPtr{opCallCode, true}
	jumpTable[LOG0] = jumpPtr{makeLog(0), true}
	jumpTable[LOG1] = jumpPtr{makeLog(1), true}
	jumpTable[LOG2] = jumpPtr{makeLog(2), true}
	jumpTable[LOG3] = jumpPtr{makeLog(3), true}
	jumpTable[LOG4] = jumpPtr{makeLog(4), true}
	jumpTable[SWAP1] = jumpPtr{makeSwap(1), true}
	jumpTable[SWAP2] = jumpPtr{makeSwap(2), true}
	jumpTable[SWAP3] = jumpPtr{makeSwap(3), true}
	jumpTable[SWAP4] = jumpPtr{makeSwap(4), true}
	jumpTable[SWAP5] = jumpPtr{makeSwap(5), true}
	jumpTable[SWAP6] = jumpPtr{makeSwap(6), true}
	jumpTable[SWAP7] = jumpPtr{makeSwap(7), true}
	jumpTable[SWAP8] = jumpPtr{makeSwap(8), true}
	jumpTable[SWAP9] = jumpPtr{makeSwap(9), true}
	jumpTable[SWAP10] = jumpPtr{makeSwap(10), true}
	jumpTable[SWAP11] = jumpPtr{makeSwap(11), true}
	jumpTable[SWAP12] = jumpPtr{makeSwap(12), true}
	jumpTable[SWAP13] = jumpPtr{makeSwap(13), true}
	jumpTable[SWAP14] = jumpPtr{makeSwap(14), true}
	jumpTable[SWAP15] = jumpPtr{makeSwap(15), true}
	jumpTable[SWAP16] = jumpPtr{makeSwap(16), true}
	jumpTable[PUSH1] = jumpPtr{makePush(1, big.NewInt(1)), true}
	jumpTable[PUSH2] = jumpPtr{makePush(2, big.NewInt(2)), true}
	jumpTable[PUSH3] = jumpPtr{makePush(3, big.NewInt(3)), true}
	jumpTable[PUSH4] = jumpPtr{makePush(4, big.NewInt(4)), true}
	jumpTable[PUSH5] = jumpPtr{makePush(5, big.NewInt(5)), true}
	jumpTable[PUSH6] = jumpPtr{makePush(6, big.NewInt(6)), true}
	jumpTable[PUSH7] = jumpPtr{makePush(7, big.NewInt(7)), true}
	jumpTable[PUSH8] = jumpPtr{makePush(8, big.NewInt(8)), true}
	jumpTable[PUSH9] = jumpPtr{makePush(9, big.NewInt(9)), true}
	jumpTable[PUSH10] = jumpPtr{makePush(10, big.NewInt(10)), true}
	jumpTable[PUSH11] = jumpPtr{makePush(11, big.NewInt(11)), true}
	jumpTable[PUSH12] = jumpPtr{makePush(12, big.NewInt(12)), true}
	jumpTable[PUSH13] = jumpPtr{makePush(13, big.NewInt(13)), true}
	jumpTable[PUSH14] = jumpPtr{makePush(14, big.NewInt(14)), true}
	jumpTable[PUSH15] = jumpPtr{makePush(15, big.NewInt(15)), true}
	jumpTable[PUSH16] = jumpPtr{makePush(16, big.NewInt(16)), true}
	jumpTable[PUSH17] = jumpPtr{makePush(17, big.NewInt(17)), true}
	jumpTable[PUSH18] = jumpPtr{makePush(18, big.NewInt(18)), true}
	jumpTable[PUSH19] = jumpPtr{makePush(19, big.NewInt(19)), true}
	jumpTable[PUSH20] = jumpPtr{makePush(20, big.NewInt(20)), true}
	jumpTable[PUSH21] = jumpPtr{makePush(21, big.NewInt(21)), true}
	jumpTable[PUSH22] = jumpPtr{makePush(22, big.NewInt(22)), true}
	jumpTable[PUSH23] = jumpPtr{makePush(23, big.NewInt(23)), true}
	jumpTable[PUSH24] = jumpPtr{makePush(24, big.NewInt(24)), true}
	jumpTable[PUSH25] = jumpPtr{makePush(25, big.NewInt(25)), true}
	jumpTable[PUSH26] = jumpPtr{makePush(26, big.NewInt(26)), true}
	jumpTable[PUSH27] = jumpPtr{makePush(27, big.NewInt(27)), true}
	jumpTable[PUSH28] = jumpPtr{makePush(28, big.NewInt(28)), true}
	jumpTable[PUSH29] = jumpPtr{makePush(29, big.NewInt(29)), true}
	jumpTable[PUSH30] = jumpPtr{makePush(30, big.NewInt(30)), true}
	jumpTable[PUSH31] = jumpPtr{makePush(31, big.NewInt(31)), true}
	jumpTable[PUSH32] = jumpPtr{makePush(32, big.NewInt(32)), true}
	jumpTable[DUP1] = jumpPtr{makeDup(1), true}
	jumpTable[DUP2] = jumpPtr{makeDup(2), true}
	jumpTable[DUP3] = jumpPtr{makeDup(3), true}
	jumpTable[DUP4] = jumpPtr{makeDup(4), true}
	jumpTable[DUP5] = jumpPtr{makeDup(5), true}
	jumpTable[DUP6] = jumpPtr{makeDup(6), true}
	jumpTable[DUP7] = jumpPtr{makeDup(7), true}
	jumpTable[DUP8] = jumpPtr{makeDup(8), true}
	jumpTable[DUP9] = jumpPtr{makeDup(9), true}
	jumpTable[DUP10] = jumpPtr{makeDup(10), true}
	jumpTable[DUP11] = jumpPtr{makeDup(11), true}
	jumpTable[DUP12] = jumpPtr{makeDup(12), true}
	jumpTable[DUP13] = jumpPtr{makeDup(13), true}
	jumpTable[DUP14] = jumpPtr{makeDup(14), true}
	jumpTable[DUP15] = jumpPtr{makeDup(15), true}
	jumpTable[DUP16] = jumpPtr{makeDup(16), true}

	jumpTable[RETURN] = jumpPtr{nil, true}
	jumpTable[SUICIDE] = jumpPtr{nil, true}
	jumpTable[JUMP] = jumpPtr{nil, true}
	jumpTable[JUMPI] = jumpPtr{nil, true}
	jumpTable[STOP] = jumpPtr{nil, true}
}
