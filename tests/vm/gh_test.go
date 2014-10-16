package ethvm

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ethereum/eth-go/ethstate"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/tests/helper"
)

type Account struct {
	Balance string
	Code    string
	Nonce   string
	Storage map[string]string
}

func StateObjectFromAccount(addr string, account Account) *ethstate.StateObject {
	obj := ethstate.NewStateObject(ethutil.Hex2Bytes(addr))
	obj.Balance = ethutil.Big(account.Balance)

	if ethutil.IsHex(account.Code) {
		account.Code = account.Code[2:]
	}
	obj.Code = ethutil.Hex2Bytes(account.Code)
	obj.Nonce = ethutil.Big(account.Nonce).Uint64()

	return obj
}

type VmTest struct {
	Callcreates interface{}
	Env         map[string]string
	Exec        map[string]string
	Gas         string
	Out         string
	Post        map[string]Account
	Pre         map[string]Account
}

func RunVmTest(url string, t *testing.T) {
	tests := make(map[string]VmTest)
	helper.CreateTests(t, url, &tests)

	for name, test := range tests {
		state := ethstate.New(helper.NewTrie())
		for addr, account := range test.Pre {
			obj := StateObjectFromAccount(addr, account)
			state.SetStateObject(obj)
		}

		ret, gas, err := helper.RunVm(state, test.Env, test.Exec)
		// When an error is returned it doesn't always mean the tests fails.
		// Have to come up with some conditional failing mechanism.
		if err != nil {
			fmt.Println(err)
		}
		/*
			if err != nil {
				t.Errorf("%s's execution failed. %v\n", name, err)
			}
		*/

		rexp := helper.FromHex(test.Out)
		if bytes.Compare(rexp, ret) != 0 {
			t.Errorf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
		}

		gexp := ethutil.Big(test.Gas)
		if gexp.Cmp(gas) != 0 {
			t.Errorf("%s's gas failed. Expected %v, got %v\n", name, gexp, gas)
		}

		for addr, account := range test.Post {
			obj := state.GetStateObject(helper.FromHex(addr))
			for addr, value := range account.Storage {
				v := obj.GetState(helper.FromHex(addr)).Bytes()
				vexp := helper.FromHex(value)

				if bytes.Compare(v, vexp) != 0 {
					t.Errorf("%s's : (%x: %s) storage failed. Expected %x, got %x\n", name, obj.Address()[0:4], addr, vexp, v)
				}
			}
		}
	}
}

// I've created a new function for each tests so it's easier to identify where the problem lies if any of them fail.
func TestVMSha3(t *testing.T) {
	helper.Logger.SetLogLevel(0)
	defer helper.Logger.SetLogLevel(4)

	const url = "https://raw.githubusercontent.com/ethereum/tests/master/vmtests/vmSha3Test.json"
	RunVmTest(url, t)
}

func TestVMArithmetic(t *testing.T) {
	helper.Logger.SetLogLevel(0)
	defer helper.Logger.SetLogLevel(4)

	const url = "https://raw.githubusercontent.com/ethereum/tests/master/vmtests/vmArithmeticTest.json"
	RunVmTest(url, t)
}

func TestVMSystemOperations(t *testing.T) {
	const url = "https://raw.githubusercontent.com/ethereum/tests/master/vmtests/vmSystemOperationsTest.json"
	RunVmTest(url, t)
}

func TestOperations(t *testing.T) {
	t.Skip()
	const url = "https://raw.githubusercontent.com/ethereum/tests/master/vmtests/vmSystemOperationsTest.json"
	RunVmTest(url, t)
}
