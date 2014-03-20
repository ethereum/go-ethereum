package ethutil

import (
	"math/big"
	"strconv"
)

// Op codes
var OpCodes = map[string]byte{
	"STOP":           0x00,
	"ADD":            0x01,
	"MUL":            0x02,
	"SUB":            0x03,
	"DIV":            0x04,
	"SDIV":           0x05,
	"MOD":            0x06,
	"SMOD":           0x07,
	"EXP":            0x08,
	"NEG":            0x09,
	"LT":             0x0a,
	"LE":             0x0b,
	"GT":             0x0c,
	"GE":             0x0d,
	"EQ":             0x0e,
	"NOT":            0x0f,
	"MYADDRESS":      0x10,
	"TXSENDER":       0x11,
	"TXVALUE":        0x12,
	"TXDATAN":        0x13,
	"TXDATA":         0x14,
	"BLK_PREVHASH":   0x15,
	"BLK_COINBASE":   0x16,
	"BLK_TIMESTAMP":  0x17,
	"BLK_NUMBER":     0x18,
	"BLK_DIFFICULTY": 0x19,
	"BLK_NONCE":      0x1a,
	"BASEFEE":        0x1b,
	"SHA256":         0x20,
	"RIPEMD160":      0x21,
	"ECMUL":          0x22,
	"ECADD":          0x23,
	"ECSIGN":         0x24,
	"ECRECOVER":      0x25,
	"ECVALID":        0x26,
	"SHA3":           0x27,
	"PUSH":           0x30,
	"POP":            0x31,
	"DUP":            0x32,
	"SWAP":           0x33,
	"MLOAD":          0x34,
	"MSTORE":         0x35,
	"SLOAD":          0x36,
	"SSTORE":         0x37,
	"JMP":            0x38,
	"JMPI":           0x39,
	"IND":            0x3a,
	"EXTRO":          0x3b,
	"BALANCE":        0x3c,
	"MKTX":           0x3d,
	"SUICIDE":        0x3f,

	// TODO FIX OPCODES
	"CALL":   0x40,
	"RETURN": 0x41,
}

func IsOpCode(s string) bool {
	for key, _ := range OpCodes {
		if key == s {
			return true
		}
	}
	return false
}

func CompileInstr(s string) ([]byte, error) {
	isOp := IsOpCode(s)
	if isOp {
		return []byte{OpCodes[s]}, nil
	}

	num := new(big.Int)
	_, success := num.SetString(s, 0)
	// Assume regular bytes during compilation
	if !success {
		num.SetBytes([]byte(s))
	}

	return num.Bytes(), nil
}

func Instr(instr string) (int, []string, error) {

	base := new(big.Int)
	base.SetString(instr, 0)

	args := make([]string, 7)
	for i := 0; i < 7; i++ {
		// int(int(val) / int(math.Pow(256,float64(i)))) % 256
		exp := BigPow(256, i)
		num := new(big.Int)
		num.Div(base, exp)

		args[i] = num.Mod(num, big.NewInt(256)).String()
	}
	op, _ := strconv.Atoi(args[0])

	return op, args[1:7], nil
}
