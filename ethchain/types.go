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
	oCALLDATA     = 0x35
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
	oPUSH    = 0x50
	oPUSH20  = 0x80
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

	// 0x60 range - closures
	oCREATE = 0x60
	oCALL   = 0x61
	oRETURN = 0x62

	// 0x70 range - other
	oLOG     = 0x70 // XXX Unofficial
	oSUICIDE = 0x7f
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
	oCALLDATA:     "CALLDATA",
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
	oPUSH:    "PUSH",
	oPOP:     "POP",
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

	// 0x60 range - closures
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
	"CALLDATA":     0x35,
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
	"PUSH": 0x50,

	"PUSH20": 0x80,

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

	// 0x60 range - closures
	"CREATE": 0x60,
	"CALL":   0x61,
	"RETURN": 0x62,

	// 0x70 range - other
	"LOG":     0x70,
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
