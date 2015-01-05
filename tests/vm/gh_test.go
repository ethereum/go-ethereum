package vm

import (
	"bytes"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
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
	Env         Env
	Exec        map[string]string
	Transaction map[string]string
	Logs        []Log
	Gas         string
	Out         string
	Post        map[string]Account
	Pre         map[string]Account
}

func RunVmTest(p string, t *testing.T) {
	tests := make(map[string]VmTest)
	helper.CreateFileTests(t, p, &tests)

	for name, test := range tests {
		/*
			helper.Logger.SetLogLevel(5)
			if name != "jump0_jumpdest2" {
				continue
			}
		*/
		statedb := state.New(helper.NewTrie())
		for addr, account := range test.Pre {
			obj := StateObjectFromAccount(addr, account)
			statedb.SetStateObject(obj)
			for a, v := range account.Storage {
				obj.SetState(helper.FromHex(a), ethutil.NewValue(helper.FromHex(v)))
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

		// Log the error if there is one. Error does not mean failing test.
		// A test fails if err != nil and post params are specified in the test.
		if err != nil {
			helper.Log.Infof("%s's: %v\n", name, err)
		}

		rexp := helper.FromHex(test.Out)
		if bytes.Compare(rexp, ret) != 0 {
			t.Errorf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
		}

		if isVmTest {
			if len(test.Gas) == 0 && err == nil {
				t.Errorf("%s's gas unspecified, indicating an error. VM returned (incorrectly) successfull", name)
			} else {
				gexp := ethutil.Big(test.Gas)
				if gexp.Cmp(gas) != 0 {
					t.Errorf("%s's gas failed. Expected %v, got %v\n", name, gexp, gas)
				}
			}
		}

		for addr, account := range test.Post {
			obj := statedb.GetStateObject(helper.FromHex(addr))
			if obj == nil {
				continue
			}

			if len(test.Exec) == 0 {
				if obj.Balance().Cmp(ethutil.Big(account.Balance)) != 0 {
					t.Errorf("%s's : (%x) balance failed. Expected %v, got %v => %v\n", name, obj.Address()[:4], account.Balance, obj.Balance(), new(big.Int).Sub(ethutil.Big(account.Balance), obj.Balance()))
				}
			}

			for addr, value := range account.Storage {
				v := obj.GetState(helper.FromHex(addr)).Bytes()
				vexp := helper.FromHex(value)

				if bytes.Compare(v, vexp) != 0 {
					t.Errorf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address()[0:4], addr, vexp, v, ethutil.BigD(vexp), ethutil.BigD(v))
				}
			}
		}

		if len(test.Logs) > 0 {
			for i, log := range test.Logs {
				genBloom := ethutil.LeftPadBytes(types.LogsBloom(state.Logs{logs[i]}).Bytes(), 64)
				if !bytes.Equal(genBloom, ethutil.Hex2Bytes(log.BloomF)) {
					t.Errorf("bloom mismatch")
				}
			}
		}
	}
	logger.Flush()
}

// I've created a new function for each tests so it's easier to identify where the problem lies if any of them fail.
func TestVMArithmetic(t *testing.T) {
	const fn = "../files/vmtests/vmArithmeticTest.json"
	RunVmTest(fn, t)
}

func TestBitwiseLogicOperation(t *testing.T) {
	const fn = "../files/vmtests/vmBitwiseLogicOperationTest.json"
	RunVmTest(fn, t)
}

func TestBlockInfo(t *testing.T) {
	const fn = "../files/vmtests/vmBlockInfoTest.json"
	RunVmTest(fn, t)
}

func TestEnvironmentalInfo(t *testing.T) {
	const fn = "../files/vmtests/vmEnvironmentalInfoTest.json"
	RunVmTest(fn, t)
}

func TestFlowOperation(t *testing.T) {
	const fn = "../files/vmtests/vmIOandFlowOperationsTest.json"
	RunVmTest(fn, t)
}

func TestPushDupSwap(t *testing.T) {
	const fn = "../files/vmtests/vmPushDupSwapTest.json"
	RunVmTest(fn, t)
}

func TestVMSha3(t *testing.T) {
	const fn = "../files/vmtests/vmSha3Test.json"
	RunVmTest(fn, t)
}

func TestVm(t *testing.T) {
	const fn = "../files/vmtests/vmtests.json"
	RunVmTest(fn, t)
}

func TestVmLog(t *testing.T) {
	const fn = "../files/vmtests/vmLogTest.json"
	RunVmTest(fn, t)
}

func TestStateSystemOperations(t *testing.T) {
	const fn = "../files/StateTests/stSystemOperationsTest.json"
	RunVmTest(fn, t)
}

func TestStatePreCompiledContracts(t *testing.T) {
	const fn = "../files/StateTests/stPreCompiledContracts.json"
	RunVmTest(fn, t)
}

func TestStateRecursiveCreate(t *testing.T) {
	const fn = "../files/StateTests/stRecursiveCreate.json"
	RunVmTest(fn, t)
}

func TestStateSpecial(t *testing.T) {
	const fn = "../files/StateTests/stSpecialTest.json"
	RunVmTest(fn, t)
}

func TestStateRefund(t *testing.T) {
	const fn = "../files/StateTests/stRefundTest.json"
	RunVmTest(fn, t)
}
