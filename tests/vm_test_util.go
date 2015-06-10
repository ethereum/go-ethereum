package tests

import (
	"bytes"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/tests/helper"
)

type Account struct {
	Balance string
	Code    string
	Nonce   string
	Storage map[string]string
}

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

type Env struct {
	CurrentCoinbase   string
	CurrentDifficulty string
	CurrentGasLimit   string
	CurrentNumber     string
	CurrentTimestamp  interface{}
	PreviousHash      string
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

func RunVmTest(p string, t *testing.T) {

	tests := make(map[string]VmTest)
	helper.CreateFileTests(t, p, &tests)

	for name, test := range tests {
		/*
		   vm.Debug = true
		   glog.SetV(4)
		   glog.SetToStderr(true)
		   if name != "Call50000_sha256" {
		     continue
		   }
		*/
		db, _ := ethdb.NewMemDatabase()
		statedb := state.New(common.Hash{}, db)
		for addr, account := range test.Pre {
			obj := StateObjectFromAccount(db, addr, account)
			statedb.SetStateObject(obj)
			for a, v := range account.Storage {
				obj.SetState(common.HexToHash(a), common.HexToHash(v))
			}
		}

		// XXX Yeah, yeah...
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

		var (
			ret  []byte
			gas  *big.Int
			err  error
			logs state.Logs
		)

		isVmTest := len(test.Exec) > 0
		if isVmTest {
			ret, logs, gas, err = helper.RunVm(statedb, env, test.Exec)
		} else {
			ret, logs, gas, err = helper.RunState(statedb, env, test.Transaction)
		}

		switch name {
		// the memory required for these tests (4294967297 bytes) would take too much time.
		// on 19 May 2015 decided to skip these tests their output.
		case "mload32bitBound_return", "mload32bitBound_return2":
		default:
			rexp := helper.FromHex(test.Out)
			if bytes.Compare(rexp, ret) != 0 {
				t.Errorf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
			}
		}

		if isVmTest {
			if len(test.Gas) == 0 && err == nil {
				t.Errorf("%s's gas unspecified, indicating an error. VM returned (incorrectly) successfull", name)
			} else {
				gexp := common.Big(test.Gas)
				if gexp.Cmp(gas) != 0 {
					t.Errorf("%s's gas failed. Expected %v, got %v\n", name, gexp, gas)
				}
			}
		}

		for addr, account := range test.Post {
			obj := statedb.GetStateObject(common.HexToAddress(addr))
			if obj == nil {
				continue
			}

			if len(test.Exec) == 0 {
				if obj.Balance().Cmp(common.Big(account.Balance)) != 0 {
					t.Errorf("%s's : (%x) balance failed. Expected %v, got %v => %v\n", name, obj.Address().Bytes()[:4], account.Balance, obj.Balance(), new(big.Int).Sub(common.Big(account.Balance), obj.Balance()))
				}

				if obj.Nonce() != common.String2Big(account.Nonce).Uint64() {
					t.Errorf("%s's : (%x) nonce failed. Expected %v, got %v\n", name, obj.Address().Bytes()[:4], account.Nonce, obj.Nonce())
				}

			}

			for addr, value := range account.Storage {
				v := obj.GetState(common.HexToHash(addr))
				vexp := common.HexToHash(value)

				if v != vexp {
					t.Errorf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address().Bytes()[0:4], addr, vexp, v, vexp.Big(), v.Big())
				}
			}
		}

		if !isVmTest {
			statedb.Sync()
			//if !bytes.Equal(common.Hex2Bytes(test.PostStateRoot), statedb.Root()) {
			if common.HexToHash(test.PostStateRoot) != statedb.Root() {
				t.Errorf("%s's : Post state root error. Expected %s, got %x", name, test.PostStateRoot, statedb.Root())
			}
		}

		if len(test.Logs) > 0 {
			if len(test.Logs) != len(logs) {
				t.Errorf("log length mismatch. Expected %d, got %d", len(test.Logs), len(logs))
			} else {
				for i, log := range test.Logs {
					if common.HexToAddress(log.AddressF) != logs[i].Address {
						t.Errorf("'%s' log address expected %v got %x", name, log.AddressF, logs[i].Address)
					}

					if !bytes.Equal(logs[i].Data, helper.FromHex(log.DataF)) {
						t.Errorf("'%s' log data expected %v got %x", name, log.DataF, logs[i].Data)
					}

					if len(log.TopicsF) != len(logs[i].Topics) {
						t.Errorf("'%s' log topics length expected %d got %d", name, len(log.TopicsF), logs[i].Topics)
					} else {
						for j, topic := range log.TopicsF {
							if common.HexToHash(topic) != logs[i].Topics[j] {
								t.Errorf("'%s' log topic[%d] expected %v got %x", name, j, topic, logs[i].Topics[j])
							}
						}
					}
					genBloom := common.LeftPadBytes(types.LogsBloom(state.Logs{logs[i]}).Bytes(), 256)

					if !bytes.Equal(genBloom, common.Hex2Bytes(log.BloomF)) {
						t.Errorf("'%s' bloom mismatch", name)
					}
				}
			}
		}
		//fmt.Println(string(statedb.Dump()))
	}
	logger.Flush()
}
