package main

import (
	"fmt"
	"github.com/ethereum/ethutil-go"
	"math/big"
)

type OpCode byte

// Op codes
const (
	oSTOP           OpCode = 0x00
	oADD            OpCode = 0x01
	oMUL            OpCode = 0x02
	oSUB            OpCode = 0x03
	oDIV            OpCode = 0x04
	oSDIV           OpCode = 0x05
	oMOD            OpCode = 0x06
	oSMOD           OpCode = 0x07
	oEXP            OpCode = 0x08
	oNEG            OpCode = 0x09
	oLT             OpCode = 0x0a
	oLE             OpCode = 0x0b
	oGT             OpCode = 0x0c
	oGE             OpCode = 0x0d
	oEQ             OpCode = 0x0e
	oNOT            OpCode = 0x0f
	oMYADDRESS      OpCode = 0x10
	oTXSENDER       OpCode = 0x11
	oTXVALUE        OpCode = 0x12
	oTXFEE          OpCode = 0x13
	oTXDATAN        OpCode = 0x14
	oTXDATA         OpCode = 0x15
	oBLK_PREVHASH   OpCode = 0x16
	oBLK_COINBASE   OpCode = 0x17
	oBLK_TIMESTAMP  OpCode = 0x18
	oBLK_NUMBER     OpCode = 0x19
	oBLK_DIFFICULTY OpCode = 0x1a
	oSHA256         OpCode = 0x20
	oRIPEMD160      OpCode = 0x21
	oECMUL          OpCode = 0x22
	oECADD          OpCode = 0x23
	oECSIGN         OpCode = 0x24
	oECRECOVER      OpCode = 0x25
	oECVALID        OpCode = 0x26
	oPUSH           OpCode = 0x30
	oPOP            OpCode = 0x31
	oDUP            OpCode = 0x32
	oDUPN           OpCode = 0x33
	oSWAP           OpCode = 0x34
	oSWAPN          OpCode = 0x35
	oLOAD           OpCode = 0x36
	oSTORE          OpCode = 0x37
	oJMP            OpCode = 0x40
	oJMPI           OpCode = 0x41
	oIND            OpCode = 0x42
	oEXTRO          OpCode = 0x50
	oBALANCE        OpCode = 0x51
	oMKTX           OpCode = 0x60
	oSUICIDE        OpCode = 0xff
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
	oBLK_DIFFICULTY: "BLK_DIFFIFULTY",
	oSHA256:         "SHA256",
	oRIPEMD160:      "RIPEMD160",
	oECMUL:          "ECMUL",
	oECADD:          "ECADD",
	oECSIGN:         "ECSIGN",
	oECRECOVER:      "ECRECOVER",
	oECVALID:        "ECVALID",
	oPUSH:           "PUSH",
	oPOP:            "POP",
	oDUP:            "DUP",
	oDUPN:           "DUPN",
	oSWAP:           "SWAP",
	oSWAPN:          "SWAPN",
	oLOAD:           "LOAD",
	oSTORE:          "STORE",
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
	data []string
}

func NewStack() *Stack {
	return &Stack{}
}

func (st *Stack) Pop() string {
	s := len(st.data)

	str := st.data[s-1]
	st.data = st.data[:s-1]

	return str
}

func (st *Stack) Popn() (*big.Int, *big.Int) {
	s := len(st.data)

	strs := st.data[s-2:]
	st.data = st.data[:s-2]

	return ethutil.Big(strs[0]), ethutil.Big(strs[1])
}

func (st *Stack) Push(d string) {
	st.data = append(st.data, d)
}
func (st *Stack) Print() {
	fmt.Println(st.data)
}
