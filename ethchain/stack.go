package ethchain

import (
	"fmt"
	_ "github.com/ethereum/eth-go/ethutil"
	"math/big"
)

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
	str := st.data[len(st.data)-1]

	copy(st.data[:len(st.data)-1], st.data[:len(st.data)-1])
	st.data = st.data[:len(st.data)-1]

	return str
}

func (st *Stack) Popn() (*big.Int, *big.Int) {
	ints := st.data[len(st.data)-2:]

	copy(st.data[:len(st.data)-2], st.data[:len(st.data)-2])
	st.data = st.data[:len(st.data)-2]

	return ints[0], ints[1]
}

func (st *Stack) Peek() *big.Int {
	str := st.data[len(st.data)-1]

	return str
}

func (st *Stack) Peekn() (*big.Int, *big.Int) {
	ints := st.data[:2]

	return ints[0], ints[1]
}

func (st *Stack) Push(d *big.Int) {
	st.data = append(st.data, d)
}
func (st *Stack) Print() {
	fmt.Println("### stack ###")
	if len(st.data) > 0 {
		for i, val := range st.data {
			fmt.Printf("%-3d  %v\n", i, val)
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("#############")
}

type Memory struct {
	store []byte
}

func (m *Memory) Set(offset, size int64, value []byte) {
	totSize := offset + size
	lenSize := int64(len(m.store) - 1)
	if totSize > lenSize {
		// Calculate the diff between the sizes
		diff := totSize - lenSize
		if diff > 0 {
			// Create a new empty slice and append it
			newSlice := make([]byte, diff-1)
			// Resize slice
			m.store = append(m.store, newSlice...)
		}
	}
	copy(m.store[offset:offset+size], value)
}

func (m *Memory) Get(offset, size int64) []byte {
	return m.store[offset : offset+size]
}

func (m *Memory) Len() int {
	return len(m.store)
}

func (m *Memory) Print() {
	fmt.Printf("### mem %d bytes ###\n", len(m.store))
	if len(m.store) > 0 {
		addr := 0
		for i := 0; i+32 <= len(m.store); i += 32 {
			fmt.Printf("%03d %v\n", addr, m.store[i:i+32])
			addr++
		}
	} else {
		fmt.Println("-- empty --")
	}
	fmt.Println("####################")
}
