package ethchain

import (
	"fmt"
	"math/big"
)

type OpCode int

// Op codes
const (
	oSTOP           = 0x00
	oADD            = 0x01
	oMUL            = 0x02
	oSUB            = 0x03
	oDIV            = 0x04
	oSDIV           = 0x05
	oMOD            = 0x06
	oSMOD           = 0x07
	oEXP            = 0x08
	oNEG            = 0x09
	oLT             = 0x0a
	oLE             = 0x0b
	oGT             = 0x0c
	oGE             = 0x0d
	oEQ             = 0x0e
	oNOT            = 0x0f
	oMYADDRESS      = 0x10
	oTXSENDER       = 0x11
	oTXVALUE        = 0x12
	oTXDATAN        = 0x13
	oTXDATA         = 0x14
	oBLK_PREVHASH   = 0x15
	oBLK_COINBASE   = 0x16
	oBLK_TIMESTAMP  = 0x17
	oBLK_NUMBER     = 0x18
	oBLK_DIFFICULTY = 0x19
	oBLK_NONCE      = 0x1a
	oBASEFEE        = 0x1b
	oSHA256         = 0x20
	oRIPEMD160      = 0x21
	oECMUL          = 0x22
	oECADD          = 0x23
	oECSIGN         = 0x24
	oECRECOVER      = 0x25
	oECVALID        = 0x26
	oSHA3           = 0x27
	oPUSH           = 0x30
	oPOP            = 0x31
	oDUP            = 0x32
	oSWAP           = 0x33
	oMLOAD          = 0x34
	oMSTORE         = 0x35
	oSLOAD          = 0x36
	oSSTORE         = 0x37
	oJMP            = 0x38
	oJMPI           = 0x39
	oIND            = 0x3a
	oEXTRO          = 0x3b
	oBALANCE        = 0x3c
	oMKTX           = 0x3d
	oSUICIDE        = 0x3f
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

func (st *Stack) Peek() *big.Int {
	s := len(st.data)

	str := st.data[s-1]

	return str
}

func (st *Stack) Peekn() (*big.Int, *big.Int) {
	s := len(st.data)

	ints := st.data[s-2:]

	return ints[0], ints[1]
}

func (st *Stack) Push(d *big.Int) {
	st.data = append(st.data, d)
}
func (st *Stack) Print() {
	fmt.Println("### STACK ###")
	if len(st.data) > 0 {
		for i, val := range st.data {
			fmt.Printf("%-3d  %v\n", i, val)
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("#############")
}
