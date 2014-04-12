package ethutil

import (
	_ "fmt"
	"math/big"
	"regexp"
)

// Op codes
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

func CompileInstr(s interface{}) ([]byte, error) {
	switch s.(type) {
	case string:
		str := s.(string)
		isOp := IsOpCode(str)
		if isOp {
			return []byte{OpCodes[str]}, nil
		}

		num := new(big.Int)
		_, success := num.SetString(str, 0)
		// Assume regular bytes during compilation
		if !success {
			num.SetBytes([]byte(str))
		} else {
			// tmp fix for 32 bytes
			n := BigToBytes(num, 256)
			return n, nil
		}

		return num.Bytes(), nil
	case int:
		num := BigToBytes(big.NewInt(int64(s.(int))), 256)
		return num, nil
	case []byte:
		return BigD(s.([]byte)).Bytes(), nil
	}

	return nil, nil
}

// Script compilation functions
// Compiles strings to machine code
func Assemble(instructions ...interface{}) (script []byte) {
	//script = make([]string, len(instructions))

	for _, val := range instructions {
		instr, _ := CompileInstr(val)

		//script[i] = string(instr)
		script = append(script, instr...)
	}

	return
}

/*
Prepocessing function that takes init and main apart:
init() {
	// something
}

main() {
	// main something
}
*/
func PreProcess(data string) (mainInput, initInput string) {
	reg := "\\(\\)\\s*{([\\d\\w\\W\\n\\s]+?)}"
	mainReg := regexp.MustCompile("main" + reg)
	initReg := regexp.MustCompile("init" + reg)

	main := mainReg.FindStringSubmatch(data)
	if len(main) > 0 {
		mainInput = main[1]
	} else {
		mainInput = data
	}

	init := initReg.FindStringSubmatch(data)
	if len(init) > 0 {
		initInput = init[1]
	}

	return
}
