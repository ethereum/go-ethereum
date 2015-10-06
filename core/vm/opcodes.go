// Copyright 2014 The go-ethereum Authors
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
)

// OpCode is an EVM opcode
type OpCode byte

func (op OpCode) IsPush() bool {
	switch op {
	case PUSH1, PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8, PUSH9, PUSH10, PUSH11, PUSH12, PUSH13, PUSH14, PUSH15, PUSH16, PUSH17, PUSH18, PUSH19, PUSH20, PUSH21, PUSH22, PUSH23, PUSH24, PUSH25, PUSH26, PUSH27, PUSH28, PUSH29, PUSH30, PUSH31, PUSH32:
		return true
	}
	return false
}

func (op OpCode) IsStaticJump() bool {
	return op == JUMP
}

const (
	// 0x0 range - arithmetic ops
	STOP OpCode = iota
	ADD
	MUL
	SUB
	DIV
	SDIV
	MOD
	SMOD
	ADDMOD
	MULMOD
	EXP
	SIGNEXTEND
)

const (
	LT OpCode = iota + 0x10
	GT
	SLT
	SGT
	EQ
	ISZERO
	AND
	OR
	XOR
	NOT
	BYTE

	SHA3 = 0x20
)

const (
	// 0x30 range - closure state
	ADDRESS OpCode = 0x30 + iota
	BALANCE
	ORIGIN
	CALLER
	CALLVALUE
	CALLDATALOAD
	CALLDATASIZE
	CALLDATACOPY
	CODESIZE
	CODECOPY
	GASPRICE
	EXTCODESIZE
	EXTCODECOPY
)

const (

	// 0x40 range - block operations
	BLOCKHASH OpCode = 0x40 + iota
	COINBASE
	TIMESTAMP
	NUMBER
	DIFFICULTY
	GASLIMIT
)

const (
	// 0x50 range - 'storage' and execution
	POP OpCode = 0x50 + iota
	MLOAD
	MSTORE
	MSTORE8
	SLOAD
	SSTORE
	JUMP
	JUMPI
	PC
	MSIZE
	GAS
	JUMPDEST
)

const (
	// 0x60 range
	PUSH1 OpCode = 0x60 + iota
	PUSH2
	PUSH3
	PUSH4
	PUSH5
	PUSH6
	PUSH7
	PUSH8
	PUSH9
	PUSH10
	PUSH11
	PUSH12
	PUSH13
	PUSH14
	PUSH15
	PUSH16
	PUSH17
	PUSH18
	PUSH19
	PUSH20
	PUSH21
	PUSH22
	PUSH23
	PUSH24
	PUSH25
	PUSH26
	PUSH27
	PUSH28
	PUSH29
	PUSH30
	PUSH31
	PUSH32
	DUP1
	DUP2
	DUP3
	DUP4
	DUP5
	DUP6
	DUP7
	DUP8
	DUP9
	DUP10
	DUP11
	DUP12
	DUP13
	DUP14
	DUP15
	DUP16
	SWAP1
	SWAP2
	SWAP3
	SWAP4
	SWAP5
	SWAP6
	SWAP7
	SWAP8
	SWAP9
	SWAP10
	SWAP11
	SWAP12
	SWAP13
	SWAP14
	SWAP15
	SWAP16
)

const (
	LOG0 OpCode = 0xa0 + iota
	LOG1
	LOG2
	LOG3
	LOG4
)

const (
	// 0xf0 range - closures
	CREATE OpCode = 0xf0 + iota
	CALL
	CALLCODE
	RETURN

	// 0x70 range - other
	SUICIDE = 0xff
)

// Since the opcodes aren't all in order we can't use a regular slice
var opCodeToString = map[OpCode]string{
	// 0x0 range - arithmetic ops
	STOP:       "STOP",
	ADD:        "ADD",
	MUL:        "MUL",
	SUB:        "SUB",
	DIV:        "DIV",
	SDIV:       "SDIV",
	MOD:        "MOD",
	SMOD:       "SMOD",
	EXP:        "EXP",
	NOT:        "NOT",
	LT:         "LT",
	GT:         "GT",
	SLT:        "SLT",
	SGT:        "SGT",
	EQ:         "EQ",
	ISZERO:     "ISZERO",
	SIGNEXTEND: "SIGNEXTEND",

	// 0x10 range - bit ops
	AND:    "AND",
	OR:     "OR",
	XOR:    "XOR",
	BYTE:   "BYTE",
	ADDMOD: "ADDMOD",
	MULMOD: "MULMOD",

	// 0x20 range - crypto
	SHA3: "SHA3",

	// 0x30 range - closure state
	ADDRESS:      "ADDRESS",
	BALANCE:      "BALANCE",
	ORIGIN:       "ORIGIN",
	CALLER:       "CALLER",
	CALLVALUE:    "CALLVALUE",
	CALLDATALOAD: "CALLDATALOAD",
	CALLDATASIZE: "CALLDATASIZE",
	CALLDATACOPY: "CALLDATACOPY",
	CODESIZE:     "CODESIZE",
	CODECOPY:     "CODECOPY",
	GASPRICE:     "TXGASPRICE",

	// 0x40 range - block operations
	BLOCKHASH:   "BLOCKHASH",
	COINBASE:    "COINBASE",
	TIMESTAMP:   "TIMESTAMP",
	NUMBER:      "NUMBER",
	DIFFICULTY:  "DIFFICULTY",
	GASLIMIT:    "GASLIMIT",
	EXTCODESIZE: "EXTCODESIZE",
	EXTCODECOPY: "EXTCODECOPY",

	// 0x50 range - 'storage' and execution
	POP: "POP",
	//DUP:     "DUP",
	//SWAP:    "SWAP",
	MLOAD:    "MLOAD",
	MSTORE:   "MSTORE",
	MSTORE8:  "MSTORE8",
	SLOAD:    "SLOAD",
	SSTORE:   "SSTORE",
	JUMP:     "JUMP",
	JUMPI:    "JUMPI",
	PC:       "PC",
	MSIZE:    "MSIZE",
	GAS:      "GAS",
	JUMPDEST: "JUMPDEST",

	// 0x60 range - push
	PUSH1:  "PUSH1",
	PUSH2:  "PUSH2",
	PUSH3:  "PUSH3",
	PUSH4:  "PUSH4",
	PUSH5:  "PUSH5",
	PUSH6:  "PUSH6",
	PUSH7:  "PUSH7",
	PUSH8:  "PUSH8",
	PUSH9:  "PUSH9",
	PUSH10: "PUSH10",
	PUSH11: "PUSH11",
	PUSH12: "PUSH12",
	PUSH13: "PUSH13",
	PUSH14: "PUSH14",
	PUSH15: "PUSH15",
	PUSH16: "PUSH16",
	PUSH17: "PUSH17",
	PUSH18: "PUSH18",
	PUSH19: "PUSH19",
	PUSH20: "PUSH20",
	PUSH21: "PUSH21",
	PUSH22: "PUSH22",
	PUSH23: "PUSH23",
	PUSH24: "PUSH24",
	PUSH25: "PUSH25",
	PUSH26: "PUSH26",
	PUSH27: "PUSH27",
	PUSH28: "PUSH28",
	PUSH29: "PUSH29",
	PUSH30: "PUSH30",
	PUSH31: "PUSH31",
	PUSH32: "PUSH32",

	DUP1:  "DUP1",
	DUP2:  "DUP2",
	DUP3:  "DUP3",
	DUP4:  "DUP4",
	DUP5:  "DUP5",
	DUP6:  "DUP6",
	DUP7:  "DUP7",
	DUP8:  "DUP8",
	DUP9:  "DUP9",
	DUP10: "DUP10",
	DUP11: "DUP11",
	DUP12: "DUP12",
	DUP13: "DUP13",
	DUP14: "DUP14",
	DUP15: "DUP15",
	DUP16: "DUP16",

	SWAP1:  "SWAP1",
	SWAP2:  "SWAP2",
	SWAP3:  "SWAP3",
	SWAP4:  "SWAP4",
	SWAP5:  "SWAP5",
	SWAP6:  "SWAP6",
	SWAP7:  "SWAP7",
	SWAP8:  "SWAP8",
	SWAP9:  "SWAP9",
	SWAP10: "SWAP10",
	SWAP11: "SWAP11",
	SWAP12: "SWAP12",
	SWAP13: "SWAP13",
	SWAP14: "SWAP14",
	SWAP15: "SWAP15",
	SWAP16: "SWAP16",
	LOG0:   "LOG0",
	LOG1:   "LOG1",
	LOG2:   "LOG2",
	LOG3:   "LOG3",
	LOG4:   "LOG4",

	// 0xf0 range
	CREATE:   "CREATE",
	CALL:     "CALL",
	RETURN:   "RETURN",
	CALLCODE: "CALLCODE",

	// 0x70 range - other
	SUICIDE: "SUICIDE",
}

func (o OpCode) String() string {
	str := opCodeToString[o]
	if len(str) == 0 {
		return fmt.Sprintf("Missing opcode 0x%x", int(o))
	}

	return str
}

var stringToOp = map[string]OpCode{
	"STOP":         STOP,
	"ADD":          ADD,
	"MUL":          MUL,
	"SUB":          SUB,
	"DIV":          DIV,
	"SDIV":         SDIV,
	"MOD":          MOD,
	"SMOD":         SMOD,
	"EXP":          EXP,
	"NOT":          NOT,
	"LT":           LT,
	"GT":           GT,
	"SLT":          SLT,
	"SGT":          SGT,
	"EQ":           EQ,
	"ISZERO":       ISZERO,
	"SIGNEXTEND":   SIGNEXTEND,
	"AND":          AND,
	"OR":           OR,
	"XOR":          XOR,
	"BYTE":         BYTE,
	"ADDMOD":       ADDMOD,
	"MULMOD":       MULMOD,
	"SHA3":         SHA3,
	"ADDRESS":      ADDRESS,
	"BALANCE":      BALANCE,
	"ORIGIN":       ORIGIN,
	"CALLER":       CALLER,
	"CALLVALUE":    CALLVALUE,
	"CALLDATALOAD": CALLDATALOAD,
	"CALLDATASIZE": CALLDATASIZE,
	"CALLDATACOPY": CALLDATACOPY,
	"CODESIZE":     CODESIZE,
	"CODECOPY":     CODECOPY,
	"GASPRICE":     GASPRICE,
	"BLOCKHASH":    BLOCKHASH,
	"COINBASE":     COINBASE,
	"TIMESTAMP":    TIMESTAMP,
	"NUMBER":       NUMBER,
	"DIFFICULTY":   DIFFICULTY,
	"GASLIMIT":     GASLIMIT,
	"EXTCODESIZE":  EXTCODESIZE,
	"EXTCODECOPY":  EXTCODECOPY,
	"POP":          POP,
	"MLOAD":        MLOAD,
	"MSTORE":       MSTORE,
	"MSTORE8":      MSTORE8,
	"SLOAD":        SLOAD,
	"SSTORE":       SSTORE,
	"JUMP":         JUMP,
	"JUMPI":        JUMPI,
	"PC":           PC,
	"MSIZE":        MSIZE,
	"GAS":          GAS,
	"JUMPDEST":     JUMPDEST,
	"PUSH1":        PUSH1,
	"PUSH2":        PUSH2,
	"PUSH3":        PUSH3,
	"PUSH4":        PUSH4,
	"PUSH5":        PUSH5,
	"PUSH6":        PUSH6,
	"PUSH7":        PUSH7,
	"PUSH8":        PUSH8,
	"PUSH9":        PUSH9,
	"PUSH10":       PUSH10,
	"PUSH11":       PUSH11,
	"PUSH12":       PUSH12,
	"PUSH13":       PUSH13,
	"PUSH14":       PUSH14,
	"PUSH15":       PUSH15,
	"PUSH16":       PUSH16,
	"PUSH17":       PUSH17,
	"PUSH18":       PUSH18,
	"PUSH19":       PUSH19,
	"PUSH20":       PUSH20,
	"PUSH21":       PUSH21,
	"PUSH22":       PUSH22,
	"PUSH23":       PUSH23,
	"PUSH24":       PUSH24,
	"PUSH25":       PUSH25,
	"PUSH26":       PUSH26,
	"PUSH27":       PUSH27,
	"PUSH28":       PUSH28,
	"PUSH29":       PUSH29,
	"PUSH30":       PUSH30,
	"PUSH31":       PUSH31,
	"PUSH32":       PUSH32,
	"DUP1":         DUP1,
	"DUP2":         DUP2,
	"DUP3":         DUP3,
	"DUP4":         DUP4,
	"DUP5":         DUP5,
	"DUP6":         DUP6,
	"DUP7":         DUP7,
	"DUP8":         DUP8,
	"DUP9":         DUP9,
	"DUP10":        DUP10,
	"DUP11":        DUP11,
	"DUP12":        DUP12,
	"DUP13":        DUP13,
	"DUP14":        DUP14,
	"DUP15":        DUP15,
	"DUP16":        DUP16,
	"SWAP1":        SWAP1,
	"SWAP2":        SWAP2,
	"SWAP3":        SWAP3,
	"SWAP4":        SWAP4,
	"SWAP5":        SWAP5,
	"SWAP6":        SWAP6,
	"SWAP7":        SWAP7,
	"SWAP8":        SWAP8,
	"SWAP9":        SWAP9,
	"SWAP10":       SWAP10,
	"SWAP11":       SWAP11,
	"SWAP12":       SWAP12,
	"SWAP13":       SWAP13,
	"SWAP14":       SWAP14,
	"SWAP15":       SWAP15,
	"SWAP16":       SWAP16,
	"LOG0":         LOG0,
	"LOG1":         LOG1,
	"LOG2":         LOG2,
	"LOG3":         LOG3,
	"LOG4":         LOG4,
	"CREATE":       CREATE,
	"CALL":         CALL,
	"RETURN":       RETURN,
	"CALLCODE":     CALLCODE,
	"SUICIDE":      SUICIDE,
}

func StringToOp(str string) OpCode {
	return stringToOp[str]
}
