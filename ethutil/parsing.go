package ethutil

import (
	"math/big"
	"strconv"
)

// Op codes
var OpCodes = map[string]byte{
	"STOP":           0,
	"ADD":            1,
	"MUL":            2,
	"SUB":            3,
	"DIV":            4,
	"SDIV":           5,
	"MOD":            6,
	"SMOD":           7,
	"EXP":            8,
	"NEG":            9,
	"LT":             10,
	"LE":             11,
	"GT":             12,
	"GE":             13,
	"EQ":             14,
	"NOT":            15,
	"MYADDRESS":      16,
	"TXSENDER":       17,
	"TXVALUE":        18,
	"TXFEE":          19,
	"TXDATAN":        20,
	"TXDATA":         21,
	"BLK_PREVHASH":   22,
	"BLK_COINBASE":   23,
	"BLK_TIMESTAMP":  24,
	"BLK_NUMBER":     25,
	"BLK_DIFFICULTY": 26,
	"BASEFEE":        27,
	"SHA256":         32,
	"RIPEMD160":      33,
	"ECMUL":          34,
	"ECADD":          35,
	"ECSIGN":         36,
	"ECRECOVER":      37,
	"ECVALID":        38,
	"SHA3":           39,
	"PUSH":           48,
	"POP":            49,
	"DUP":            50,
	"SWAP":           51,
	"MLOAD":          52,
	"MSTORE":         53,
	"SLOAD":          54,
	"SSTORE":         55,
	"JMP":            56,
	"JMPI":           57,
	"IND":            58,
	"EXTRO":          59,
	"BALANCE":        60,
	"MKTX":           61,
	"SUICIDE":        62,
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
	num.SetString(s, 0)

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
