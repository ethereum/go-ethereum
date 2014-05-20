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

func TestRun4(t *testing.T) {
	ethutil.ReadConfig("", ethutil.LogStd)

	db, _ := ethdb.NewMemDatabase()
	state := NewState(ethutil.NewTrie(db, ""))

	script, err := mutan.Compile(strings.NewReader(`
		int32 a = 10
		int32 b = 20
		if a > b {
			int32 c = this.caller()
		}
		Exit()
	`), false)
	tx := NewContractCreationTx(ethutil.Big("0"), ethutil.Big("1000"), ethutil.Big("100"), script, nil)
	addr := tx.Hash()[12:]
	contract := MakeContract(tx, state)
	state.UpdateStateObject(contract)
	fmt.Printf("%x\n", addr)

	callerScript, err := mutan.Compile(strings.NewReader(`
		// Check if there's any cash in the initial store
		if this.store[1000] == 0 {
			this.store[1000] = 10^20
		}


		this.store[1001] = this.value() * 20
		this.store[this.origin()] = this.store[this.origin()] + 1000

		if this.store[1001] > 20 {
			this.store[1001] = 10^50
		}

		int8 ret = 0
		int8 arg = 10
		call(0xe6a12555fad1fb6eaaaed69001a87313d1fd7b54, 0, 100, arg, ret)

		big t
		for int8 i = 0; i < 10; i++ {
			t = i
		}

		if 10 > 20 {
			int8 shouldnt = 2
		} else {
			int8 should = 1
		}
	`), false)
	if err != nil {
		fmt.Println(err)
	}

	callerTx := NewContractCreationTx(ethutil.Big("0"), ethutil.Big("1000"), ethutil.Big("100"), callerScript, nil)

	// Contract addr as test address
	gas := big.NewInt(1000)
	gasPrice := big.NewInt(10)
	account := NewAccount(ContractAddr, big.NewInt(10000000))
	fmt.Println("account.Amount =", account.Amount)
	c := MakeContract(callerTx, state)
	e := account.ConvertGas(gas, gasPrice)
	if e != nil {
		fmt.Println(err)
	}
	fmt.Println("account.Amount =", account.Amount)
	callerClosure := NewClosure(account, c, c.script, state, gas, gasPrice)

	vm := NewVm(state, nil, RuntimeVars{
		Origin:      account.Address(),
		BlockNumber: 1,
		PrevHash:    ethutil.FromHex("5e20a0453cecd065ea59c37ac63e079ee08998b6045136a8ce6635c7912ec0b6"),
		Coinbase:    ethutil.FromHex("2adc25665018aa1fe0e6bc666dac8fc2697ff9ba"),
		Time:        1,
		Diff:        big.NewInt(256),
	})
	_, e = callerClosure.Call(vm, nil, nil)
	if e != nil {
		fmt.Println("error", e)
	}
	fmt.Println("account.Amount =", account.Amount)
}
