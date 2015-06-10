package tests

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	// "github.com/ethereum/go-ethereum/logger"
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

type VmEnv struct {
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
	Env           VmEnv
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
	CreateFileTests(t, p, &tests)

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

		ret, logs, gas, err = RunVm(statedb, env, test.Exec)

		rexp := common.FromHex(test.Out)
		if bytes.Compare(rexp, ret) != 0 {
			t.Errorf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
		}

		if len(test.Gas) == 0 && err == nil {
			t.Errorf("%s's gas unspecified, indicating an error. VM returned (incorrectly) successfull", name)
		} else {
			gexp := common.Big(test.Gas)
			if gexp.Cmp(gas) != 0 {
				t.Errorf("%s's gas failed. Expected %v, got %v\n", name, gexp, gas)
			}
		}

		for addr, account := range test.Post {
			obj := statedb.GetStateObject(common.HexToAddress(addr))
			if obj == nil {
				continue
			}

			for addr, value := range account.Storage {
				v := obj.GetState(common.HexToHash(addr))
				vexp := common.HexToHash(value)

				if v != vexp {
					t.Errorf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address().Bytes()[0:4], addr, vexp, v, vexp.Big(), v.Big())
				}
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

					if !bytes.Equal(logs[i].Data, common.FromHex(log.DataF)) {
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

		fmt.Println("VM test passed: ", name)

		//fmt.Println(string(statedb.Dump()))
	}
	// logger.Flush()
}

type Env struct {
	depth        int
	state        *state.StateDB
	skipTransfer bool
	initial      bool
	Gas          *big.Int

	origin common.Address
	//parent   common.Hash
	coinbase common.Address

	number     *big.Int
	time       int64
	difficulty *big.Int
	gasLimit   *big.Int

	logs state.Logs

	vmTest bool
}

func NewEnv(state *state.StateDB) *Env {
	return &Env{
		state: state,
	}
}

func NewEnvFromMap(state *state.StateDB, envValues map[string]string, exeValues map[string]string) *Env {
	env := NewEnv(state)

	env.origin = common.HexToAddress(exeValues["caller"])
	//env.parent = common.Hex2Bytes(envValues["previousHash"])
	env.coinbase = common.HexToAddress(envValues["currentCoinbase"])
	env.number = common.Big(envValues["currentNumber"])
	env.time = common.Big(envValues["currentTimestamp"]).Int64()
	env.difficulty = common.Big(envValues["currentDifficulty"])
	env.gasLimit = common.Big(envValues["currentGasLimit"])
	env.Gas = new(big.Int)

	return env
}

func (self *Env) Origin() common.Address { return self.origin }
func (self *Env) BlockNumber() *big.Int  { return self.number }

//func (self *Env) PrevHash() []byte      { return self.parent }
func (self *Env) Coinbase() common.Address { return self.coinbase }
func (self *Env) Time() int64              { return self.time }
func (self *Env) Difficulty() *big.Int     { return self.difficulty }
func (self *Env) State() *state.StateDB    { return self.state }
func (self *Env) GasLimit() *big.Int       { return self.gasLimit }
func (self *Env) VmType() vm.Type          { return vm.StdVmTy }
func (self *Env) GetHash(n uint64) common.Hash {
	return common.BytesToHash(crypto.Sha3([]byte(big.NewInt(int64(n)).String())))
}
func (self *Env) AddLog(log *state.Log) {
	self.state.AddLog(log)
}
func (self *Env) Depth() int     { return self.depth }
func (self *Env) SetDepth(i int) { self.depth = i }
func (self *Env) Transfer(from, to vm.Account, amount *big.Int) error {
	if self.skipTransfer {
		// ugly hack
		if self.initial {
			self.initial = false
			return nil
		}

		if from.Balance().Cmp(amount) < 0 {
			return errors.New("Insufficient balance in account")
		}

		return nil
	}
	return vm.Transfer(from, to, amount)
}

func (self *Env) vm(addr *common.Address, data []byte, gas, price, value *big.Int) *core.Execution {
	exec := core.NewExecution(self, addr, data, gas, price, value)

	return exec
}

func (self *Env) Call(caller vm.ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	if self.vmTest && self.depth > 0 {
		caller.ReturnGas(gas, price)

		return nil, nil
	}
	exe := self.vm(&addr, data, gas, price, value)
	ret, err := exe.Call(addr, caller)
	self.Gas = exe.Gas

	return ret, err

}
func (self *Env) CallCode(caller vm.ContextRef, addr common.Address, data []byte, gas, price, value *big.Int) ([]byte, error) {
	if self.vmTest && self.depth > 0 {
		caller.ReturnGas(gas, price)

		return nil, nil
	}

	caddr := caller.Address()
	exe := self.vm(&caddr, data, gas, price, value)
	return exe.Call(addr, caller)
}

func (self *Env) Create(caller vm.ContextRef, data []byte, gas, price, value *big.Int) ([]byte, error, vm.ContextRef) {
	exe := self.vm(nil, data, gas, price, value)
	if self.vmTest {
		caller.ReturnGas(gas, price)

		nonce := self.state.GetNonce(caller.Address())
		obj := self.state.GetOrNewStateObject(crypto.CreateAddress(caller.Address(), nonce))

		return nil, nil, obj
	} else {
		return exe.Create(caller)
	}
}

func RunVm(state *state.StateDB, env, exec map[string]string) ([]byte, state.Logs, *big.Int, error) {
	var (
		to    = common.HexToAddress(exec["address"])
		from  = common.HexToAddress(exec["caller"])
		data  = common.FromHex(exec["data"])
		gas   = common.Big(exec["gas"])
		price = common.Big(exec["gasPrice"])
		value = common.Big(exec["value"])
	)
	// Reset the pre-compiled contracts for VM tests.
	vm.Precompiled = make(map[string]*vm.PrecompiledAccount)

	caller := state.GetOrNewStateObject(from)

	vmenv := NewEnvFromMap(state, env, exec)
	vmenv.vmTest = true
	vmenv.skipTransfer = true
	vmenv.initial = true
	ret, err := vmenv.Call(caller, to, data, gas, price, value)

	return ret, vmenv.state.Logs(), vmenv.Gas, err
}

type Message struct {
	from              common.Address
	to                *common.Address
	value, gas, price *big.Int
	data              []byte
	nonce             uint64
}

func NewMessage(from common.Address, to *common.Address, data []byte, value, gas, price *big.Int, nonce uint64) Message {
	return Message{from, to, value, gas, price, data, nonce}
}

func (self Message) Hash() []byte                  { return nil }
func (self Message) From() (common.Address, error) { return self.from, nil }
func (self Message) To() *common.Address           { return self.to }
func (self Message) GasPrice() *big.Int            { return self.price }
func (self Message) Gas() *big.Int                 { return self.gas }
func (self Message) Value() *big.Int               { return self.value }
func (self Message) Nonce() uint64                 { return self.nonce }
func (self Message) Data() []byte                  { return self.data }
