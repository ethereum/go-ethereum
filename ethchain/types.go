package ethchain

type OpCode int

// Op codes
const (
	// 0x0 range - arithmetic ops
	oSTOP = 0x00
	oADD  = 0x01
	oMUL  = 0x02
	oSUB  = 0x03
	oDIV  = 0x04
	oSDIV = 0x05
	oMOD  = 0x06
	oSMOD = 0x07
	oEXP  = 0x08
	oNEG  = 0x09
	oLT   = 0x0a
	oGT   = 0x0b
	oEQ   = 0x0c
	oNOT  = 0x0d

	// 0x10 range - bit ops
	oAND  = 0x10
	oOR   = 0x11
	oXOR  = 0x12
	oBYTE = 0x13

	// 0x20 range - crypto
	oSHA3 = 0x20

	// 0x30 range - closure state
	oADDRESS      = 0x30
	oBALANCE      = 0x31
	oORIGIN       = 0x32
	oCALLER       = 0x33
	oCALLVALUE    = 0x34
	oCALLDATALOAD = 0x35
	oCALLDATASIZE = 0x36
	oGASPRICE     = 0x37

	// 0x40 range - block operations
	oPREVHASH   = 0x40
	oCOINBASE   = 0x41
	oTIMESTAMP  = 0x42
	oNUMBER     = 0x43
	oDIFFICULTY = 0x44
	oGASLIMIT   = 0x45

	// 0x50 range - 'storage' and execution
	oPOP     = 0x51
	oDUP     = 0x52
	oSWAP    = 0x53
	oMLOAD   = 0x54
	oMSTORE  = 0x55
	oMSTORE8 = 0x56
	oSLOAD   = 0x57
	oSSTORE  = 0x58
	oJUMP    = 0x59
	oJUMPI   = 0x5a
	oPC      = 0x5b
	oMSIZE   = 0x5c

	// 0x60 range
	oPUSH1  = 0x60
	oPUSH2  = 0x61
	oPUSH3  = 0x62
	oPUSH4  = 0x63
	oPUSH5  = 0x64
	oPUSH6  = 0x65
	oPUSH7  = 0x66
	oPUSH8  = 0x67
	oPUSH9  = 0x68
	oPUSH10 = 0x69
	oPUSH11 = 0x6a
	oPUSH12 = 0x6b
	oPUSH13 = 0x6c
	oPUSH14 = 0x6d
	oPUSH15 = 0x6e
	oPUSH16 = 0x6f
	oPUSH17 = 0x70
	oPUSH18 = 0x71
	oPUSH19 = 0x72
	oPUSH20 = 0x73
	oPUSH21 = 0x74
	oPUSH22 = 0x75
	oPUSH23 = 0x76
	oPUSH24 = 0x77
	oPUSH25 = 0x78
	oPUSH26 = 0x79
	oPUSH27 = 0x7a
	oPUSH28 = 0x7b
	oPUSH29 = 0x7c
	oPUSH30 = 0x7d
	oPUSH31 = 0x7e
	oPUSH32 = 0x7f

	// 0xf0 range - closures
	oCREATE = 0xf0
	oCALL   = 0xf1
	oRETURN = 0xf2

	// 0x70 range - other
	oLOG     = 0xfe // XXX Unofficial
	oSUICIDE = 0xff
)

// Since the opcodes aren't all in order we can't use a regular slice
var opCodeToString = map[OpCode]string{
	// 0x0 range - arithmetic ops
	oSTOP: "STOP",
	oADD:  "ADD",
	oMUL:  "MUL",
	oSUB:  "SUB",
	oDIV:  "DIV",
	oSDIV: "SDIV",
	oMOD:  "MOD",
	oSMOD: "SMOD",
	oEXP:  "EXP",
	oNEG:  "NEG",
	oLT:   "LT",
	oGT:   "GT",
	oEQ:   "EQ",
	oNOT:  "NOT",

	// 0x10 range - bit ops
	oAND:  "AND",
	oOR:   "OR",
	oXOR:  "XOR",
	oBYTE: "BYTE",

	// 0x20 range - crypto
	oSHA3: "SHA3",

	// 0x30 range - closure state
	oADDRESS:      "ADDRESS",
	oBALANCE:      "BALANCE",
	oORIGIN:       "ORIGIN",
	oCALLER:       "CALLER",
	oCALLVALUE:    "CALLVALUE",
	oCALLDATALOAD: "CALLDATALOAD",
	oCALLDATASIZE: "CALLDATASIZE",
	oGASPRICE:     "TXGASPRICE",

	// 0x40 range - block operations
	oPREVHASH:   "PREVHASH",
	oCOINBASE:   "COINBASE",
	oTIMESTAMP:  "TIMESTAMP",
	oNUMBER:     "NUMBER",
	oDIFFICULTY: "DIFFICULTY",
	oGASLIMIT:   "GASLIMIT",

	// 0x50 range - 'storage' and execution
	oDUP:     "DUP",
	oSWAP:    "SWAP",
	oMLOAD:   "MLOAD",
	oMSTORE:  "MSTORE",
	oMSTORE8: "MSTORE8",
	oSLOAD:   "SLOAD",
	oSSTORE:  "SSTORE",
	oJUMP:    "JUMP",
	oJUMPI:   "JUMPI",
	oPC:      "PC",
	oMSIZE:   "MSIZE",

	// 0x60 range - push
	oPUSH1:  "PUSH1",
	oPUSH2:  "PUSH2",
	oPUSH3:  "PUSH3",
	oPUSH4:  "PUSH4",
	oPUSH5:  "PUSH5",
	oPUSH6:  "PUSH6",
	oPUSH7:  "PUSH7",
	oPUSH8:  "PUSH8",
	oPUSH9:  "PUSH9",
	oPUSH10: "PUSH10",
	oPUSH11: "PUSH11",
	oPUSH12: "PUSH12",
	oPUSH13: "PUSH13",
	oPUSH14: "PUSH14",
	oPUSH15: "PUSH15",
	oPUSH16: "PUSH16",
	oPUSH17: "PUSH17",
	oPUSH18: "PUSH18",
	oPUSH19: "PUSH19",
	oPUSH20: "PUSH20",
	oPUSH21: "PUSH21",
	oPUSH22: "PUSH22",
	oPUSH23: "PUSH23",
	oPUSH24: "PUSH24",
	oPUSH25: "PUSH25",
	oPUSH26: "PUSH26",
	oPUSH27: "PUSH27",
	oPUSH28: "PUSH28",
	oPUSH29: "PUSH29",
	oPUSH30: "PUSH30",
	oPUSH31: "PUSH31",
	oPUSH32: "PUSH32",

	// 0xf0 range
	oCREATE: "CREATE",
	oCALL:   "CALL",
	oRETURN: "RETURN",

	// 0x70 range - other
	oLOG:     "LOG",
	oSUICIDE: "SUICIDE",
}

func (o OpCode) String() string {
	return opCodeToString[o]
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

func AppendScript(init, script []byte) []byte {
	s := append(init, byte(oRETURN))
	s = append(s, script...)

	return s
}
