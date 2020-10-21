package vm

import "github.com/holiman/uint256"

// auto-generated, do not edit
func opPush1(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 1)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes1(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 1
	return nil, nil
}

// auto-generated, do not edit
func opPush2(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 2)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes2(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 2
	return nil, nil
}

// auto-generated, do not edit
func opPush3(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 3)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes3(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 3
	return nil, nil
}

// auto-generated, do not edit
func opPush4(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 4)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes4(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 4
	return nil, nil
}

// auto-generated, do not edit
func opPush5(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 5)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes5(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 5
	return nil, nil
}

// auto-generated, do not edit
func opPush6(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 6)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes6(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 6
	return nil, nil
}

// auto-generated, do not edit
func opPush7(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 7)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes7(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 7
	return nil, nil
}

// auto-generated, do not edit
func opPush8(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 8)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes8(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 8
	return nil, nil
}

// auto-generated, do not edit
func opPush9(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 9)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes9(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 9
	return nil, nil
}

// auto-generated, do not edit
func opPush10(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 10)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes10(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 10
	return nil, nil
}

// auto-generated, do not edit
func opPush11(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 11)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes11(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 11
	return nil, nil
}

// auto-generated, do not edit
func opPush12(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 12)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes12(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 12
	return nil, nil
}

// auto-generated, do not edit
func opPush13(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 13)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes13(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 13
	return nil, nil
}

// auto-generated, do not edit
func opPush14(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 14)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes14(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 14
	return nil, nil
}

// auto-generated, do not edit
func opPush15(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 15)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes15(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 15
	return nil, nil
}

// auto-generated, do not edit
func opPush16(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 16)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes16(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 16
	return nil, nil
}

// auto-generated, do not edit
func opPush17(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 17)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes17(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 17
	return nil, nil
}

// auto-generated, do not edit
func opPush18(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 18)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes18(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 18
	return nil, nil
}

// auto-generated, do not edit
func opPush19(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 19)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes19(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 19
	return nil, nil
}

// auto-generated, do not edit
func opPush20(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 20)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes20(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 20
	return nil, nil
}

// auto-generated, do not edit
func opPush21(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 21)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes21(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 21
	return nil, nil
}

// auto-generated, do not edit
func opPush22(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 22)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes22(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 22
	return nil, nil
}

// auto-generated, do not edit
func opPush23(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 23)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes23(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 23
	return nil, nil
}

// auto-generated, do not edit
func opPush24(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 24)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes24(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 24
	return nil, nil
}

// auto-generated, do not edit
func opPush25(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 25)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes25(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 25
	return nil, nil
}

// auto-generated, do not edit
func opPush26(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 26)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes26(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 26
	return nil, nil
}

// auto-generated, do not edit
func opPush27(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 27)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes27(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 27
	return nil, nil
}

// auto-generated, do not edit
func opPush28(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 28)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes28(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 28
	return nil, nil
}

// auto-generated, do not edit
func opPush29(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 29)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes29(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 29
	return nil, nil
}

// auto-generated, do not edit
func opPush30(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 30)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes30(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 30
	return nil, nil
}

// auto-generated, do not edit
func opPush31(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 31)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes31(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 31
	return nil, nil
}

// auto-generated, do not edit
func opPush32(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	end := int(*pc + 1 + 32)
	integer := new(uint256.Int)
	if code := callContext.contract.Code; end < len(code) {
		integer.SetBytes32(code[int(*pc+1):end])
	}
	callContext.stack.push(integer)
	*pc += 32
	return nil, nil
}

// auto-generated, do not edit
func opSwap1(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(1 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap2(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(2 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap3(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(3 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap4(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(4 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap5(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(5 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap6(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(6 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap7(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(7 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap8(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(8 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap9(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(9 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap10(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(10 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap11(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(11 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap12(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(12 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap13(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(13 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap14(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(14 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap15(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(15 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opSwap16(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.swap(16 + 1)
	return nil, nil
}

// auto-generated, do not edit
func opDup1(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(1)
	return nil, nil
}

// auto-generated, do not edit
func opDup2(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(2)
	return nil, nil
}

// auto-generated, do not edit
func opDup3(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(3)
	return nil, nil
}

// auto-generated, do not edit
func opDup4(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(4)
	return nil, nil
}

// auto-generated, do not edit
func opDup5(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(5)
	return nil, nil
}

// auto-generated, do not edit
func opDup6(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(6)
	return nil, nil
}

// auto-generated, do not edit
func opDup7(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(7)
	return nil, nil
}

// auto-generated, do not edit
func opDup8(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(8)
	return nil, nil
}

// auto-generated, do not edit
func opDup9(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(9)
	return nil, nil
}

// auto-generated, do not edit
func opDup10(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(10)
	return nil, nil
}

// auto-generated, do not edit
func opDup11(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(11)
	return nil, nil
}

// auto-generated, do not edit
func opDup12(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(12)
	return nil, nil
}

// auto-generated, do not edit
func opDup13(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(13)
	return nil, nil
}

// auto-generated, do not edit
func opDup14(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(14)
	return nil, nil
}

// auto-generated, do not edit
func opDup15(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(15)
	return nil, nil
}

// auto-generated, do not edit
func opDup16(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	callContext.stack.dup(16)
	return nil, nil
}
