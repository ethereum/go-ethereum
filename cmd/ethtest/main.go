/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors:
 * 	Jeffrey Wilcke <i@jev.io>
 */

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/tests/helper"
)

type Account struct {
	Balance string
	Code    string
	Nonce   string
	Storage map[string]string
}

func StateObjectFromAccount(addr string, account Account) *state.StateObject {
	obj := state.NewStateObject(ethutil.Hex2Bytes(addr))
	obj.SetBalance(ethutil.Big(account.Balance))

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

func RunVmTest(js string) (failed int) {
	tests := make(map[string]VmTest)

	data, _ := ioutil.ReadAll(strings.NewReader(js))
	err := json.Unmarshal(data, &tests)
	if err != nil {
		log.Fatalln(err)
	}

	for name, test := range tests {
		state := state.New(helper.NewTrie())
		for addr, account := range test.Pre {
			obj := StateObjectFromAccount(addr, account)
			state.SetStateObject(obj)
		}

		ret, _, gas, err := helper.RunVm(state, test.Env, test.Exec)
		// When an error is returned it doesn't always mean the tests fails.
		// Have to come up with some conditional failing mechanism.
		if err != nil {
			log.Println(err)
		}

		rexp := helper.FromHex(test.Out)
		if bytes.Compare(rexp, ret) != 0 {
			log.Printf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
			failed = 1
		}

		if len(test.Gas) == 0 && err == nil {
			log.Printf("0 gas indicates error but no error given by VM")
			failed = 1
		} else {
			gexp := ethutil.Big(test.Gas)
			if gexp.Cmp(gas) != 0 {
				log.Printf("%s's gas failed. Expected %v, got %v\n", name, gexp, gas)
				failed = 1
			}
		}

		for addr, account := range test.Post {
			obj := state.GetStateObject(helper.FromHex(addr))
			for addr, value := range account.Storage {
				v := obj.GetState(helper.FromHex(addr)).Bytes()
				vexp := helper.FromHex(value)

				if bytes.Compare(v, vexp) != 0 {
					log.Printf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address()[0:4], addr, vexp, v, ethutil.BigD(vexp), ethutil.BigD(v))
					failed = 1
				}
			}
		}
	}

	return
}

func main() {
	helper.Logger.SetLogLevel(5)
	if len(os.Args) == 1 {
		log.Fatalln("no json supplied")
	}

	os.Exit(RunVmTest(os.Args[1]))
}
