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
	AND  = 0x10
	OR   = 0x11
	XOR  = 0x12
	BYTE = 0x13

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

	// 0x40 range - block operations
	PREVHASH   = 0x40
	COINBASE   = 0x41
	TIMESTAMP  = 0x42
	NUMBER     = 0x43
	DIFFICULTY = 0x44
	GASLIMIT   = 0x45

	// 0x50 range - 'storage' and execution
	POP     = 0x50
	DUP     = 0x51
	SWAP    = 0x52
	MLOAD   = 0x53
	MSTORE  = 0x54
	MSTORE8 = 0x55
	SLOAD   = 0x56
	SSTORE  = 0x57
	JUMP    = 0x58
	JUMPI   = 0x59
	PC      = 0x5a
	MSIZE   = 0x5b
	GAS     = 0x5c

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

	// 0xf0 range - closures
	CREATE = 0xf0
	CALL   = 0xf1
	RETURN = 0xf2

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
	AND:  "AND",
	OR:   "OR",
	XOR:  "XOR",
	BYTE: "BYTE",

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
	PREVHASH:   "PREVHASH",
	COINBASE:   "COINBASE",
	TIMESTAMP:  "TIMESTAMP",
	NUMBER:     "NUMBER",
	DIFFICULTY: "DIFFICULTY",
	GASLIMIT:   "GASLIMIT",

	// 0x50 range - 'storage' and execution
	POP:     "POP",
	DUP:     "DUP",
	SWAP:    "SWAP",
	MLOAD:   "MLOAD",
	MSTORE:  "MSTORE",
	MSTORE8: "MSTORE8",
	SLOAD:   "SLOAD",
	SSTORE:  "SSTORE",
	JUMP:    "JUMP",
	JUMPI:   "JUMPI",
	PC:      "PC",
	MSIZE:   "MSIZE",
	GAS:     "GAS",

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

	// 0xf0 range
	CREATE: "CREATE",
	CALL:   "CALL",
	RETURN: "RETURN",

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

// Op codes for assembling
var OpCodes = map[string]byte{
	// 0x0 range - arithmetic ops
	"STOP": 0x00,
	"ADD":  0x01,
	"MUL":  0x02,
	"SUB":  0x03,
	"DIV":  0x04,
	"SDIV": 0x05,
	"MOD":  0x06,
	"SMOD": 0x07,
	"EXP":  0x08,
	"NEG":  0x09,
	"LT":   0x0a,
	"GT":   0x0b,
	"EQ":   0x0c,
	"NOT":  0x0d,

	// 0x10 range - bit ops
	"AND":  0x10,
	"OR":   0x11,
	"XOR":  0x12,
	"BYTE": 0x13,

	// 0x20 range - crypto
	"SHA3": 0x20,

	// 0x30 range - closure state
	"ADDRESS":      0x30,
	"BALANCE":      0x31,
	"ORIGIN":       0x32,
	"CALLER":       0x33,
	"CALLVALUE":    0x34,
	"CALLDATALOAD": 0x35,
	"CALLDATASIZE": 0x36,
	"GASPRICE":     0x38,

	// 0x40 range - block operations
	"PREVHASH":   0x40,
	"COINBASE":   0x41,
	"TIMESTAMP":  0x42,
	"NUMBER":     0x43,
	"DIFFICULTY": 0x44,
	"GASLIMIT":   0x45,

	// 0x50 range - 'storage' and execution
	"POP":     0x51,
	"DUP":     0x52,
	"SWAP":    0x53,
	"MLOAD":   0x54,
	"MSTORE":  0x55,
	"MSTORE8": 0x56,
	"SLOAD":   0x57,
	"SSTORE":  0x58,
	"JUMP":    0x59,
	"JUMPI":   0x5a,
	"PC":      0x5b,
	"MSIZE":   0x5c,

	// 0x70 range - 'push'
	"PUSH1":  0x60,
	"PUSH2":  0x61,
	"PUSH3":  0x62,
	"PUSH4":  0x63,
	"PUSH5":  0x64,
	"PUSH6":  0x65,
	"PUSH7":  0x66,
	"PUSH8":  0x67,
	"PUSH9":  0x68,
	"PUSH10": 0x69,
	"PUSH11": 0x6a,
	"PUSH12": 0x6b,
	"PUSH13": 0x6c,
	"PUSH14": 0x6d,
	"PUSH15": 0x6e,
	"PUSH16": 0x6f,
	"PUSH17": 0x70,
	"PUSH18": 0x71,
	"PUSH19": 0x72,
	"PUSH20": 0x73,
	"PUSH21": 0x74,
	"PUSH22": 0x75,
	"PUSH23": 0x76,
	"PUSH24": 0x77,
	"PUSH25": 0x78,
	"PUSH26": 0x70,
	"PUSH27": 0x7a,
	"PUSH28": 0x7b,
	"PUSH29": 0x7c,
	"PUSH30": 0x7d,
	"PUSH31": 0x7e,
	"PUSH32": 0x7f,

	// 0xf0 range - closures
	"CREATE": 0xf0,
	"CALL":   0xf1,
	"RETURN": 0xf2,

	// 0x70 range - other
	"LOG":     0xfe,
	"SUICIDE": 0x7f,
}

func IsOpCode(s string) bool {
	for key, _ := range OpCodes {
		if key == s {
			return true
		}
	}
	return false
}
