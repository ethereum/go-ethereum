package ethchain

import (
	_ "bytes"
	"fmt"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/obscuren/mutan"
	"math/big"
	"strings"
	"testing"
)

/*
func TestRun3(t *testing.T) {
	ethutil.ReadConfig("")

	db, _ := ethdb.NewMemDatabase()
	state := NewState(ethutil.NewTrie(db, ""))

	script := Compile([]string{
		"PUSH", "300",
		"PUSH", "0",
		"MSTORE",

		"PUSH", "32",
		"CALLDATA",

		"PUSH", "64",
		"PUSH", "0",
		"RETURN",
	})
	tx := NewContractCreationTx(ethutil.Big("0"), ethutil.Big("1000"), script)
	addr := tx.Hash()[12:]
	contract := MakeContract(tx, state)
	state.UpdateContract(contract)

	callerScript := ethutil.Assemble(
		"PUSH", 1337, // Argument
		"PUSH", 65, // argument mem offset
		"MSTORE",
		"PUSH", 64, // ret size
		"PUSH", 0, // ret offset

		"PUSH", 32, // arg size
		"PUSH", 65, // arg offset
		"PUSH", 1000, /// Gas
		"PUSH", 0, /// value
		"PUSH", addr, // Sender
		"CALL",
		"PUSH", 64,
		"PUSH", 0,
		"RETURN",
	)
	callerTx := NewContractCreationTx(ethutil.Big("0"), ethutil.Big("1000"), callerScript)

	// Contract addr as test address
	account := NewAccount(ContractAddr, big.NewInt(10000000))
	callerClosure := NewClosure(account, MakeContract(callerTx, state), state, big.NewInt(1000000000), new(big.Int))

	vm := NewVm(state, RuntimeVars{
		origin:      account.Address(),
		blockNumber: 1,
		prevHash:    ethutil.FromHex("5e20a0453cecd065ea59c37ac63e079ee08998b6045136a8ce6635c7912ec0b6"),
		coinbase:    ethutil.FromHex("2adc25665018aa1fe0e6bc666dac8fc2697ff9ba"),
		time:        1,
		diff:        big.NewInt(256),
		// XXX Tx data? Could be just an argument to the closure instead
		txData: nil,
	})
	ret := callerClosure.Call(vm, nil)

	exp := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 44, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 5, 57}
	if bytes.Compare(ret, exp) != 0 {
		t.Errorf("expected return value to be %v, got %v", exp, ret)
	}
}*/

func TestRun4(t *testing.T) {
	ethutil.ReadConfig("")

	db, _ := ethdb.NewMemDatabase()
	state := NewState(ethutil.NewTrie(db, ""))

	asm, err := mutan.Compile(strings.NewReader(`
		int32 a = 10
		int32 b = 20
		if a > b {
			int32 c = this.caller()
		}
		exit()
	`), false)
	script := ethutil.Assemble(asm...)
	tx := NewContractCreationTx(ethutil.Big("0"), ethutil.Big("1000"), script)
	addr := tx.Hash()[12:]
	contract := MakeContract(tx, state)
	state.UpdateContract(contract)
	fmt.Printf("%x\n", addr)

	asm, err = mutan.Compile(strings.NewReader(`
		int32 a = 10
		int32 b = 10
		if a == b {
			int32 c = 10
			if c == 10 {
				int32 d = 1000
				int32 e = 10
			}
		}

		store[0] = 20
		store[a] = 20
		store[b] = this.caller()

		int8 ret = 0
		int8 arg = 10
		call(938726394128221156290138488023434115948430767407, 0, 100000000, arg, ret)
	`), false)
	if err != nil {
		fmt.Println(err)
	}
	//asm = append(asm, "LOG")
	fmt.Println(asm)

	callerScript := ethutil.Assemble(asm...)
	callerTx := NewContractCreationTx(ethutil.Big("0"), ethutil.Big("1000"), callerScript)

	// Contract addr as test address
	account := NewAccount(ContractAddr, big.NewInt(10000000))
	c := MakeContract(callerTx, state)
	//fmt.Println(c.script[230:240])
	//fmt.Println(c.script)
	callerClosure := NewClosure(account, c, c.script, state, big.NewInt(1000000000), new(big.Int))

	vm := NewVm(state, RuntimeVars{
		origin:      account.Address(),
		blockNumber: 1,
		prevHash:    ethutil.FromHex("5e20a0453cecd065ea59c37ac63e079ee08998b6045136a8ce6635c7912ec0b6"),
		coinbase:    ethutil.FromHex("2adc25665018aa1fe0e6bc666dac8fc2697ff9ba"),
		time:        1,
		diff:        big.NewInt(256),
		// XXX Tx data? Could be just an argument to the closure instead
		txData: nil,
	})
	callerClosure.Call(vm, nil)
}
