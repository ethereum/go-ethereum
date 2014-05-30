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
	ethutil.ReadConfig("", ethutil.LogStd, "")

	db, _ := ethdb.NewMemDatabase()
	state := NewState(ethutil.NewTrie(db, ""))

	callerScript, err := mutan.Compile(strings.NewReader(`
	this.store[this.origin()] = 10**20
	hello := "world"

	return lambda {
		big to = this.data[0]
		big from = this.origin()
		big value = this.data[1]

		if this.store[from] >= value {
			this.store[from] = this.store[from] - value
			this.store[to] = this.store[to] + value
		}
	}
	`), false)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(Disassemble(callerScript))

	callerTx := NewContractCreationTx(ethutil.Big("0"), ethutil.Big("1000"), ethutil.Big("100"), callerScript)
	callerTx.Sign([]byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))

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
	callerClosure := NewClosure(account, c, callerScript, state, gas, gasPrice)

	vm := NewVm(state, nil, RuntimeVars{
		Origin:      account.Address(),
		BlockNumber: 1,
		PrevHash:    ethutil.FromHex("5e20a0453cecd065ea59c37ac63e079ee08998b6045136a8ce6635c7912ec0b6"),
		Coinbase:    ethutil.FromHex("2adc25665018aa1fe0e6bc666dac8fc2697ff9ba"),
		Time:        1,
		Diff:        big.NewInt(256),
	})
	var ret []byte
	ret, _, e = callerClosure.Call(vm, nil, nil)
	if e != nil {
		fmt.Println("error", e)
	}
	fmt.Println(ret)
}
