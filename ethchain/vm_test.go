package ethchain

import (
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
		"PUSH", "300",
		"PUSH", "31",
		"MSTORE",
		"PUSH", "62",
		"PUSH", "0",
		"RETURN",
	})
	tx := NewTransaction(ContractAddr, ethutil.Big("100000000000000000000000000000000000000000000000000"), script)
	addr := tx.Hash()[12:]
	fmt.Printf("addr contract %x\n", addr)
	contract := MakeContract(tx, state)
	state.UpdateContract(addr, contract)

	callerScript := Compile([]string{
		"PUSH", "62", // REND
		"PUSH", "0", // RSTART
		"PUSH", "22", // MEND
		"PUSH", "15", // MSTART
		"PUSH", "1000", /// Gas
		"PUSH", "0", /// value
		"PUSH", string(addr), // Sender
		"CALL",
	})
	callerTx := NewTransaction(ContractAddr, ethutil.Big("100000000000000000000000000000000000000000000000000"), callerScript)
	callerAddr := callerTx.Hash()[12:]
	executer := NewTransaction(ContractAddr, ethutil.Big("10000"), nil)

	executer.Sign([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	callerClosure := NewClosure(executer, MakeContract(callerTx, state), state, big.NewInt(1000000000), new(big.Int))

	vm := NewVm(state, RuntimeVars{
		address:     callerAddr,
		blockNumber: 1,
		sender:      ethutil.FromHex("cd1722f3947def4cf144679da39c4c32bdc35681"),
		prevHash:    ethutil.FromHex("5e20a0453cecd065ea59c37ac63e079ee08998b6045136a8ce6635c7912ec0b6"),
		coinbase:    ethutil.FromHex("2adc25665018aa1fe0e6bc666dac8fc2697ff9ba"),
		time:        1,
		diff:        big.NewInt(256),
		txValue:     big.NewInt(10000),
		txData:      nil,
	})
	callerClosure.Call(vm, nil)
}
