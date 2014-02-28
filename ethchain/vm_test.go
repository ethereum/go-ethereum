package ethchain

import (
	"fmt"
	"github.com/ethereum/eth-go/ethdb"
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
	"testing"
)

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
