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
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/tests/helper"
)

type Log struct {
	AddressF string   `json:"address"`
	DataF    string   `json:"data"`
	TopicsF  []string `json:"topics"`
	BloomF   string   `json:"bloom"`
}

func (self Log) Address() []byte      { return common.Hex2Bytes(self.AddressF) }
func (self Log) Data() []byte         { return common.Hex2Bytes(self.DataF) }
func (self Log) RlpData() interface{} { return nil }
func (self Log) Topics() [][]byte {
	t := make([][]byte, len(self.TopicsF))
	for i, topic := range self.TopicsF {
		t[i] = common.Hex2Bytes(topic)
	}
	return t
}

type Account struct {
	Balance string
	Code    string
	Nonce   string
	Storage map[string]string
}

func StateObjectFromAccount(db common.Database, addr string, account Account) *state.StateObject {
	obj := state.NewStateObject(common.HexToAddress(addr), db)
	obj.SetBalance(common.Big(account.Balance))

	if common.IsHex(account.Code) {
		account.Code = account.Code[2:]
	}
	obj.SetCode(common.Hex2Bytes(account.Code))
	obj.SetNonce(common.Big(account.Nonce).Uint64())

	return obj
}

type VmTest struct {
	Callcreates   interface{}
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

	vm.Debug = true
	glog.SetV(4)
	glog.SetToStderr(true)
	for name, test := range tests {
		db, _ := ethdb.NewMemDatabase()
		statedb := state.New(common.Hash{}, db)
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
			glog.V(logger.Info).Infof("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
			failed = 1
		}

		for addr, account := range test.Post {
			obj := statedb.GetStateObject(common.HexToAddress(addr))
			if obj == nil {
				continue
			}

			if len(test.Exec) == 0 {
				if obj.Balance().Cmp(common.Big(account.Balance)) != 0 {
					glog.V(logger.Info).Infof("%s's : (%x) balance failed. Expected %v, got %v => %v\n", name, obj.Address().Bytes()[:4], account.Balance, obj.Balance(), new(big.Int).Sub(common.Big(account.Balance), obj.Balance()))
					failed = 1
				}
			}

			for addr, value := range account.Storage {
				v := obj.GetState(common.HexToHash(addr)).Bytes()
				vexp := helper.FromHex(value)

				if bytes.Compare(v, vexp) != 0 {
					glog.V(logger.Info).Infof("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address().Bytes()[0:4], addr, vexp, v, common.BigD(vexp), common.BigD(v))
					failed = 1
				}
			}
		}

		statedb.Sync()
		//if !bytes.Equal(common.Hex2Bytes(test.PostStateRoot), statedb.Root()) {
		if common.HexToHash(test.PostStateRoot) != statedb.Root() {
			glog.V(logger.Info).Infof("%s's : Post state root failed. Expected %s, got %x", name, test.PostStateRoot, statedb.Root())
			failed = 1
		}

		if len(test.Logs) > 0 {
			if len(test.Logs) != len(logs) {
				glog.V(logger.Info).Infof("log length failed. Expected %d, got %d", len(test.Logs), len(logs))
				failed = 1
			} else {
				for i, log := range test.Logs {
					if common.HexToAddress(log.AddressF) != logs[i].Address {
						glog.V(logger.Info).Infof("'%s' log address failed. Expected %v got %x", name, log.AddressF, logs[i].Address)
						failed = 1
					}

					if !bytes.Equal(logs[i].Data, helper.FromHex(log.DataF)) {
						glog.V(logger.Info).Infof("'%s' log data failed. Expected %v got %x", name, log.DataF, logs[i].Data)
						failed = 1
					}

					if len(log.TopicsF) != len(logs[i].Topics) {
						glog.V(logger.Info).Infof("'%s' log topics length failed. Expected %d got %d", name, len(log.TopicsF), logs[i].Topics)
						failed = 1
					} else {
						for j, topic := range log.TopicsF {
							if common.HexToHash(topic) != logs[i].Topics[j] {
								glog.V(logger.Info).Infof("'%s' log topic[%d] failed. Expected %v got %x", name, j, topic, logs[i].Topics[j])
								failed = 1
							}
						}
					}
					genBloom := common.LeftPadBytes(types.LogsBloom(state.Logs{logs[i]}).Bytes(), 256)

					if !bytes.Equal(genBloom, common.Hex2Bytes(log.BloomF)) {
						glog.V(logger.Info).Infof("'%s' bloom failed.", name)
						failed = 1
					}
				}
			}
		}

		if failed == 1 {
			glog.V(logger.Info).Infoln(string(statedb.Dump()))
		}

		logger.Flush()
	}

	return
}

func main() {
	helper.Logger.SetLogLevel(5)
	vm.Debug = true

	if len(os.Args) < 2 {
		glog.Exit("Must specify test type")
	}

	test := os.Args[1]

	var code int
	switch test {
	case "vm", "VMTests":
		glog.Exit("VMTests not yet implemented")
	case "state", "StateTest":
		if len(os.Args) > 2 {
			code = RunVmTest(strings.NewReader(os.Args[2]))
		} else {
			code = RunVmTest(os.Stdin)
		}
	case "tx", "TransactionTests":
		glog.Exit("TransactionTests not yet implemented")
	case "bc", "BlockChainTest":
		glog.Exit("BlockChainTest not yet implemented")
	default:
		glog.Exit("Invalid test type specified")
	}

	os.Exit(code)
}
