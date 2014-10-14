package ethvm

import (
	"bytes"
	"log"
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

func TestRemote(t *testing.T) {
	tests := make(map[string]VmTest)
	err := helper.CreateTests("https://raw.githubusercontent.com/ethereum/tests/master/vmtests/vmSha3Test.json", &tests)
	if err != nil {
		log.Fatal(err)
	}

	for name, test := range tests {
		state := ethstate.New(helper.NewTrie())
		for addr, account := range test.Pre {
			obj := StateObjectFromAccount(addr, account)
			state.SetStateObject(obj)
		}

		ret, gas := helper.RunVm(state, test.Env, test.Exec)

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
				v := obj.GetStorage(ethutil.BigD(helper.FromHex(addr))).Bytes()
				vexp := helper.FromHex(value)

				if bytes.Compare(v, vexp) != 0 {
					t.Errorf("%s's : %s storage failed. Expected %x, get %x\n", name, addr, vexp, v)
				}
			}
		}
	}
}
