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
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
	"github.com/ethereum/go-ethereum/tests/helper"
)

type Log struct {
	AddressF string   `json:"address"`
	DataF    string   `json:"data"`
	TopicsF  []string `json:"topics"`
	BloomF   string   `json:"bloom"`
}

func (self Log) Address() []byte      { return ethutil.Hex2Bytes(self.AddressF) }
func (self Log) Data() []byte         { return ethutil.Hex2Bytes(self.DataF) }
func (self Log) RlpData() interface{} { return nil }
func (self Log) Topics() [][]byte {
	t := make([][]byte, len(self.TopicsF))
	for i, topic := range self.TopicsF {
		t[i] = ethutil.Hex2Bytes(topic)
	}
	return t
}

type Account struct {
	Balance string
	Code    string
	Nonce   string
	Storage map[string]string
}

func StateObjectFromAccount(db ethutil.Database, addr string, account Account) *state.StateObject {
	obj := state.NewStateObject(ethutil.Hex2Bytes(addr), db)
	obj.SetBalance(ethutil.Big(account.Balance))

	if ethutil.IsHex(account.Code) {
		account.Code = account.Code[2:]
	}
	obj.SetCode(ethutil.Hex2Bytes(account.Code))
	obj.SetNonce(ethutil.Big(account.Nonce).Uint64())

	return obj
}

type VmTest struct {
	Callcreates interface{}
	//Env         map[string]string
	Env           Env
	Exec          map[string]string
	Transaction   map[string]string
	Logs          []Log
	Gas           string
	Out           string
	Post          map[string]Account
	Pre           map[string]Account
	PostStateRoot string
}

type Env struct {
	CurrentCoinbase   string
	CurrentDifficulty string
	CurrentGasLimit   string
	CurrentNumber     string
	CurrentTimestamp  interface{}
	PreviousHash      string
}

func RunVmTest(r io.Reader) (failed int) {
	tests := make(map[string]VmTest)

	data, _ := ioutil.ReadAll(r)
	err := json.Unmarshal(data, &tests)
	if err != nil {
		log.Fatalln(err)
	}

	for name, test := range tests {
		db, _ := ethdb.NewMemDatabase()
		statedb := state.New(nil, db)
		for addr, account := range test.Pre {
			obj := StateObjectFromAccount(db, addr, account)
			statedb.SetStateObject(obj)
		}

		env := make(map[string]string)
		env["currentCoinbase"] = test.Env.CurrentCoinbase
		env["currentDifficulty"] = test.Env.CurrentDifficulty
		env["currentGasLimit"] = test.Env.CurrentGasLimit
		env["currentNumber"] = test.Env.CurrentNumber
		env["previousHash"] = test.Env.PreviousHash
		if n, ok := test.Env.CurrentTimestamp.(float64); ok {
			env["currentTimestamp"] = strconv.Itoa(int(n))
		} else {
			env["currentTimestamp"] = test.Env.CurrentTimestamp.(string)
		}

		ret, logs, _, _ := helper.RunState(statedb, env, test.Transaction)
		statedb.Sync()

		rexp := helper.FromHex(test.Out)
		if bytes.Compare(rexp, ret) != 0 {
			fmt.Printf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
			failed = 1
		}

		for addr, account := range test.Post {
			obj := statedb.GetStateObject(helper.FromHex(addr))
			if obj == nil {
				continue
			}

			if len(test.Exec) == 0 {
				if obj.Balance().Cmp(ethutil.Big(account.Balance)) != 0 {
					fmt.Printf("%s's : (%x) balance failed. Expected %v, got %v => %v\n", name, obj.Address()[:4], account.Balance, obj.Balance(), new(big.Int).Sub(ethutil.Big(account.Balance), obj.Balance()))
					failed = 1
				}
			}

			for addr, value := range account.Storage {
				v := obj.GetState(helper.FromHex(addr)).Bytes()
				vexp := helper.FromHex(value)

				if bytes.Compare(v, vexp) != 0 {
					fmt.Printf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address()[0:4], addr, vexp, v, ethutil.BigD(vexp), ethutil.BigD(v))
					failed = 1
				}
			}
		}

		if !bytes.Equal(ethutil.Hex2Bytes(test.PostStateRoot), statedb.Root()) {
			fmt.Printf("%s's : Post state root error. Expected %s, got %x", name, test.PostStateRoot, statedb.Root())
			failed = 1
		}

		if len(test.Logs) > 0 {
			if len(test.Logs) != len(logs) {
				fmt.Printf("log length mismatch. Expected %d, got %d", len(test.Logs), len(logs))
				failed = 1
			} else {
				/*
					fmt.Println("A", test.Logs)
					fmt.Println("B", logs)
						for i, log := range test.Logs {
							genBloom := ethutil.LeftPadBytes(types.LogsBloom(state.Logs{logs[i]}).Bytes(), 256)
							if !bytes.Equal(genBloom, ethutil.Hex2Bytes(log.BloomF)) {
								t.Errorf("bloom mismatch")
							}
						}
				*/
			}
		}

		logger.Flush()
	}

	return
}

func main() {
	helper.Logger.SetLogLevel(5)

	if len(os.Args) > 1 {
		os.Exit(RunVmTest(strings.NewReader(os.Args[1])))
	} else {
		os.Exit(RunVmTest(os.Stdin))
	}
}
