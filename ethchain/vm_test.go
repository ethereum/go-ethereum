package ethchain

import (
	"bytes"
	"fmt"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
	"testing"
)

/*

func TestRun(t *testing.T) {
	InitFees()

	ethutil.ReadConfig("")

	db, _ := ethdb.NewMemDatabase()
	state := NewState(ethutil.NewTrie(db, ""))

	script := Compile([]string{
		"TXSENDER",
		"SUICIDE",
	})

	tx := NewTransaction(ContractAddr, big.NewInt(1e17), script)
	fmt.Printf("contract addr %x\n", tx.Hash()[12:])
	contract := MakeContract(tx, state)
	vm := &Vm{}

	vm.Process(contract, state, RuntimeVars{
		address:     tx.Hash()[12:],
		blockNumber: 1,
		sender:      ethutil.FromHex("cd1722f3947def4cf144679da39c4c32bdc35681"),
		prevHash:    ethutil.FromHex("5e20a0453cecd065ea59c37ac63e079ee08998b6045136a8ce6635c7912ec0b6"),
		coinbase:    ethutil.FromHex("2adc25665018aa1fe0e6bc666dac8fc2697ff9ba"),
		time:        1,
		diff:        big.NewInt(256),
		txValue:     tx.Value,
		txData:      tx.Data,
	})
}

func TestRun1(t *testing.T) {
	ethutil.ReadConfig("")

	db, _ := ethdb.NewMemDatabase()
	state := NewState(ethutil.NewTrie(db, ""))

	script := Compile([]string{
		"PUSH", "0",
		"PUSH", "0",
		"TXSENDER",
		"PUSH", "10000000",
		"MKTX",
	})
	fmt.Println(ethutil.NewValue(script))

	tx := NewTransaction(ContractAddr, ethutil.Big("100000000000000000000000000000000000000000000000000"), script)
	fmt.Printf("contract addr %x\n", tx.Hash()[12:])
	contract := MakeContract(tx, state)
	vm := &Vm{}

	vm.Process(contract, state, RuntimeVars{
		address:     tx.Hash()[12:],
		blockNumber: 1,
		sender:      ethutil.FromHex("cd1722f3947def4cf144679da39c4c32bdc35681"),
		prevHash:    ethutil.FromHex("5e20a0453cecd065ea59c37ac63e079ee08998b6045136a8ce6635c7912ec0b6"),
		coinbase:    ethutil.FromHex("2adc25665018aa1fe0e6bc666dac8fc2697ff9ba"),
		time:        1,
		diff:        big.NewInt(256),
		txValue:     tx.Value,
		txData:      tx.Data,
	})
}

func TestRun2(t *testing.T) {
	ethutil.ReadConfig("")

	db, _ := ethdb.NewMemDatabase()
	state := NewState(ethutil.NewTrie(db, ""))

	script := Compile([]string{
		"PUSH", "0",
		"PUSH", "0",
		"TXSENDER",
		"PUSH", "10000000",
		"MKTX",
	})
	fmt.Println(ethutil.NewValue(script))

	tx := NewTransaction(ContractAddr, ethutil.Big("100000000000000000000000000000000000000000000000000"), script)
	fmt.Printf("contract addr %x\n", tx.Hash()[12:])
	contract := MakeContract(tx, state)
	vm := &Vm{}

	vm.Process(contract, state, RuntimeVars{
		address:     tx.Hash()[12:],
		blockNumber: 1,
		sender:      ethutil.FromHex("cd1722f3947def4cf144679da39c4c32bdc35681"),
		prevHash:    ethutil.FromHex("5e20a0453cecd065ea59c37ac63e079ee08998b6045136a8ce6635c7912ec0b6"),
		coinbase:    ethutil.FromHex("2adc25665018aa1fe0e6bc666dac8fc2697ff9ba"),
		time:        1,
		diff:        big.NewInt(256),
		txValue:     tx.Value,
		txData:      tx.Data,
	})
}
*/

// XXX Full stack test
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
	tx := NewTransaction(ContractAddr, ethutil.Big("100000000000000000000000000000000000000000000000000"), script)
	addr := tx.Hash()[12:]
	fmt.Printf("addr contract %x\n", addr)
	contract := MakeContract(tx, state)
	state.UpdateContract(contract)

	callerScript := Compile([]string{
		"PUSH", "1337", // Argument
		"PUSH", "65", // argument mem offset
		"MSTORE",
		"PUSH", "64", // ret size
		"PUSH", "0", // ret offset

		"PUSH", "32", // arg size
		"PUSH", "65", // arg offset
		"PUSH", "1000", /// Gas
		"PUSH", "0", /// value
		"PUSH", string(addr), // Sender
		"CALL",
		"PUSH", "64",
		"PUSH", "0",
		"RETURN",
	})
	callerTx := NewTransaction(ContractAddr, ethutil.Big("100000000000000000000000000000000000000000000000000"), callerScript)

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
}
