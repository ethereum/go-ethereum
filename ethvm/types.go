package ethvm

import (
	"fmt"
)

type OpCode int

// Op codes
const (
	// 0x0 range - arithmetic ops
	STOP = 0x00
	ADD  = 0x01
	MUL  = 0x02
	SUB  = 0x03
	DIV  = 0x04
	SDIV = 0x05
	MOD  = 0x06
	SMOD = 0x07
	EXP  = 0x08
	NEG  = 0x09
	LT   = 0x0a
	GT   = 0x0b
	SLT  = 0x0c
	SGT  = 0x0d
	EQ   = 0x0e
	NOT  = 0x0f

	// 0x10 range - bit ops
	AND    = 0x10
	OR     = 0x11
	XOR    = 0x12
	BYTE   = 0x13
	ADDMOD = 0x14
	MULMOD = 0x15

	// 0x20 range - crypto
	SHA3 = 0x20

	// 0x30 range - closure state
	ADDRESS      = 0x30
	BALANCE      = 0x31
	ORIGIN       = 0x32
	CALLER       = 0x33
	CALLVALUE    = 0x34
	CALLDATALOAD = 0x35
	CALLDATASIZE = 0x36
	CALLDATACOPY = 0x37
	CODESIZE     = 0x38
	CODECOPY     = 0x39
	GASPRICE     = 0x3a
	EXTCODECOPY  = 0x3b
	EXTCODESIZE  = 0x3c

	// 0x40 range - block operations
	PREVHASH   = 0x40
	COINBASE   = 0x41
	TIMESTAMP  = 0x42
	NUMBER     = 0x43
	DIFFICULTY = 0x44
	GASLIMIT   = 0x45

	// 0x50 range - 'storage' and execution
	POP = 0x50
	//DUP     = 0x51
	//SWAP    = 0x52
	MLOAD    = 0x53
	MSTORE   = 0x54
	MSTORE8  = 0x55
	SLOAD    = 0x56
	SSTORE   = 0x57
	JUMP     = 0x58
	JUMPI    = 0x59
	PC       = 0x5a
	MSIZE    = 0x5b
	GAS      = 0x5c
	JUMPDEST = 0x5d

	// 0x60 range
	PUSH1  = 0x60
	PUSH2  = 0x61
	PUSH3  = 0x62
	PUSH4  = 0x63
	PUSH5  = 0x64
	PUSH6  = 0x65
	PUSH7  = 0x66
	PUSH8  = 0x67
	PUSH9  = 0x68
	PUSH10 = 0x69
	PUSH11 = 0x6a
	PUSH12 = 0x6b
	PUSH13 = 0x6c
	PUSH14 = 0x6d
	PUSH15 = 0x6e
	PUSH16 = 0x6f
	PUSH17 = 0x70
	PUSH18 = 0x71
	PUSH19 = 0x72
	PUSH20 = 0x73
	PUSH21 = 0x74
	PUSH22 = 0x75
	PUSH23 = 0x76
	PUSH24 = 0x77
	PUSH25 = 0x78
	PUSH26 = 0x79
	PUSH27 = 0x7a
	PUSH28 = 0x7b
	PUSH29 = 0x7c
	PUSH30 = 0x7d
	PUSH31 = 0x7e
	PUSH32 = 0x7f

	DUP1  = 0x80
	DUP2  = 0x81
	DUP3  = 0x82
	DUP4  = 0x83
	DUP5  = 0x84
	DUP6  = 0x85
	DUP7  = 0x86
	DUP8  = 0x87
	DUP9  = 0x88
	DUP10 = 0x89
	DUP11 = 0x8a
	DUP12 = 0x8b
	DUP13 = 0x8c
	DUP14 = 0x8d
	DUP15 = 0x8e
	DUP16 = 0x8f

	SWAP1  = 0x90
	SWAP2  = 0x91
	SWAP3  = 0x92
	SWAP4  = 0x93
	SWAP5  = 0x94
	SWAP6  = 0x95
	SWAP7  = 0x96
	SWAP8  = 0x97
	SWAP9  = 0x98
	SWAP10 = 0x99
	SWAP11 = 0x9a
	SWAP12 = 0x9b
	SWAP13 = 0x9c
	SWAP14 = 0x9d
	SWAP15 = 0x9e
	SWAP16 = 0x9f

	// 0xf0 range - closures
	CREATE   = 0xf0
	CALL     = 0xf1
	RETURN   = 0xf2
	CALLCODE = 0xf3

	// 0x70 range - other
	LOG     = 0xfe // XXX Unofficial
	SUICIDE = 0xff
)

// Since the opcodes aren't all in order we can't use a regular slice
var opCodeToString = map[OpCode]string{
	// 0x0 range - arithmetic ops
	STOP: "STOP",
	ADD:  "ADD",
	MUL:  "MUL",
	SUB:  "SUB",
	DIV:  "DIV",
	SDIV: "SDIV",
	MOD:  "MOD",
	SMOD: "SMOD",
	EXP:  "EXP",
	NEG:  "NEG",
	LT:   "LT",
	GT:   "GT",
	SLT:  "SLT",
	SGT:  "SGT",
	EQ:   "EQ",
	NOT:  "NOT",

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
	PREVHASH:    "PREVHASH",
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

	// 0xf0 range
	CREATE:   "CREATE",
	CALL:     "CALL",
	RETURN:   "RETURN",
	CALLCODE: "CALLCODE",

	// 0x70 range - other
	LOG:     "LOG",
	SUICIDE: "SUICIDE",
}

func (o OpCode) String() string {
	str := opCodeToString[o]
	if len(str) == 0 {
		return fmt.Sprintf("Missing opcode 0x%x", int(o))
	}

	return str
}
