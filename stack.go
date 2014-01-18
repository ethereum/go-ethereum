package main

import (
	"fmt"
	"math/big"
)

type OpCode int

// Op codes
const (
	oSTOP OpCode = iota
	oADD
	oMUL
	oSUB
	oDIV
	oSDIV
	oMOD
	oSMOD
	oEXP
	oNEG
	oLT
	oLE
	oGT
	oGE
	oEQ
	oNOT
	oMYADDRESS
	oTXSENDER
	oTXVALUE
	oTXFEE
	oTXDATAN
	oTXDATA
	oBLK_PREVHASH
	oBLK_COINBASE
	oBLK_TIMESTAMP
	oBLK_NUMBER
	oBLK_DIFFICULTY
	oBASEFEE
	oSHA256    OpCode = 32
	oRIPEMD160 OpCode = 33
	oECMUL     OpCode = 34
	oECADD     OpCode = 35
	oECSIGN    OpCode = 36
	oECRECOVER OpCode = 37
	oECVALID   OpCode = 38
	oSHA3      OpCode = 39
	oPUSH      OpCode = 48
	oPOP       OpCode = 49
	oDUP       OpCode = 50
	oSWAP      OpCode = 51
	oMLOAD     OpCode = 52
	oMSTORE    OpCode = 53
	oSLOAD     OpCode = 54
	oSSTORE    OpCode = 55
	oJMP       OpCode = 56
	oJMPI      OpCode = 57
	oIND       OpCode = 58
	oEXTRO     OpCode = 59
	oBALANCE   OpCode = 60
	oMKTX      OpCode = 61
	oSUICIDE   OpCode = 62
)

// Since the opcodes aren't all in order we can't use a regular slice
var opCodeToString = map[OpCode]string{
	oSTOP:           "STOP",
	oADD:            "ADD",
	oMUL:            "MUL",
	oSUB:            "SUB",
	oDIV:            "DIV",
	oSDIV:           "SDIV",
	oMOD:            "MOD",
	oSMOD:           "SMOD",
	oEXP:            "EXP",
	oNEG:            "NEG",
	oLT:             "LT",
	oLE:             "LE",
	oGT:             "GT",
	oGE:             "GE",
	oEQ:             "EQ",
	oNOT:            "NOT",
	oMYADDRESS:      "MYADDRESS",
	oTXSENDER:       "TXSENDER",
	oTXVALUE:        "TXVALUE",
	oTXFEE:          "TXFEE",
	oTXDATAN:        "TXDATAN",
	oTXDATA:         "TXDATA",
	oBLK_PREVHASH:   "BLK_PREVHASH",
	oBLK_COINBASE:   "BLK_COINBASE",
	oBLK_TIMESTAMP:  "BLK_TIMESTAMP",
	oBLK_NUMBER:     "BLK_NUMBER",
	oBLK_DIFFICULTY: "BLK_DIFFICULTY",
	oBASEFEE:        "BASEFEE",
	oSHA256:         "SHA256",
	oRIPEMD160:      "RIPEMD160",
	oECMUL:          "ECMUL",
	oECADD:          "ECADD",
	oECSIGN:         "ECSIGN",
	oECRECOVER:      "ECRECOVER",
	oECVALID:        "ECVALID",
	oSHA3:           "SHA3",
	oPUSH:           "PUSH",
	oPOP:            "POP",
	oDUP:            "DUP",
	oSWAP:           "SWAP",
	oMLOAD:          "MLOAD",
	oMSTORE:         "MSTORE",
	oSLOAD:          "SLOAD",
	oSSTORE:         "SSTORE",
	oJMP:            "JMP",
	oJMPI:           "JMPI",
	oIND:            "IND",
	oEXTRO:          "EXTRO",
	oBALANCE:        "BALANCE",
	oMKTX:           "MKTX",
	oSUICIDE:        "SUICIDE",
}

func (o OpCode) String() string {
	return opCodeToString[o]
}

type OpType int

const (
	tNorm = iota
	tData
	tExtro
	tCrypto
)

type TxCallback func(opType OpType) bool

// Simple push/pop stack mechanism
type Stack struct {
	data []*big.Int
}

func NewStack() *Stack {
	return &Stack{}
}

func (st *Stack) Pop() *big.Int {
	s := len(st.data)

	str := st.data[s-1]
	st.data = st.data[:s-1]

	return str
}

func (st *Stack) Popn() (*big.Int, *big.Int) {
	s := len(st.data)

	ints := st.data[s-2:]
	st.data = st.data[:s-2]

	return ints[0], ints[1]
}

func (st *Stack) Push(d *big.Int) {
	st.data = append(st.data, d)
}
func (st *Stack) Print() {
	fmt.Println(st.data)
}
