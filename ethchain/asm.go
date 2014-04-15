package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
	"regexp"
)

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
			n := ethutil.BigToBytes(num, 256)
			return n, nil
		}

		return num.Bytes(), nil
	case int:
		num := ethutil.BigToBytes(big.NewInt(int64(s.(int))), 256)
		return num, nil
	case []byte:
		return ethutil.BigD(s.([]byte)).Bytes(), nil
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

func Disassemble(script []byte) (asm []string) {
	pc := new(big.Int)
	for {
		if pc.Cmp(big.NewInt(int64(len(script)))) >= 0 {
			return
		}

		// Get the memory location of pc
		val := script[pc.Int64()]
		// Get the opcode (it must be an opcode!)
		op := OpCode(val)

		asm = append(asm, fmt.Sprintf("%v", op))

		switch op {
		case oPUSH: // Push PC+1 on to the stack
			pc.Add(pc, ethutil.Big1)
			data := script[pc.Int64() : pc.Int64()+32]
			val := ethutil.BigD(data)

			var b []byte
			if val.Int64() == 0 {
				b = []byte{0}
			} else {
				b = val.Bytes()
			}

			asm = append(asm, fmt.Sprintf("0x%x", b))

			pc.Add(pc, big.NewInt(31))
		case oPUSH20:
			pc.Add(pc, ethutil.Big1)
			data := script[pc.Int64() : pc.Int64()+20]
			val := ethutil.BigD(data)
			var b []byte
			if val.Int64() == 0 {
				b = []byte{0}
			} else {
				b = val.Bytes()
			}

			asm = append(asm, fmt.Sprintf("0x%x", b))

			pc.Add(pc, big.NewInt(19))
		}

		pc.Add(pc, ethutil.Big1)
	}

	return
}

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
