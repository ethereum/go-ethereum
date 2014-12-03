package vm

<<<<<<< HEAD
// import (
// 	"bytes"
// 	"testing"

// 	"github.com/ethereum/go-ethereum/ethutil"
// 	"github.com/ethereum/go-ethereum/state"
// 	"github.com/ethereum/go-ethereum/tests/helper"
// )

// type Account struct {
// 	Balance string
// 	Code    string
// 	Nonce   string
// 	Storage map[string]string
// }

// func StateObjectFromAccount(addr string, account Account) *state.StateObject {
// 	obj := state.NewStateObject(ethutil.Hex2Bytes(addr))
// 	obj.SetBalance(ethutil.Big(account.Balance))

// 	if ethutil.IsHex(account.Code) {
// 		account.Code = account.Code[2:]
// 	}
// 	obj.Code = ethutil.Hex2Bytes(account.Code)
// 	obj.Nonce = ethutil.Big(account.Nonce).Uint64()

// 	return obj
// }

// type VmTest struct {
// 	Callcreates interface{}
// 	Env         map[string]string
// 	Exec        map[string]string
// 	Gas         string
// 	Out         string
// 	Post        map[string]Account
// 	Pre         map[string]Account
// }

// func RunVmTest(p string, t *testing.T) {
// 	tests := make(map[string]VmTest)
// 	helper.CreateFileTests(t, p, &tests)

// 	for name, test := range tests {
// 		state := state.New(helper.NewTrie())
// 		for addr, account := range test.Pre {
// 			obj := StateObjectFromAccount(addr, account)
// 			state.SetStateObject(obj)
// 		}

// 		ret, gas, err := helper.RunVm(state, test.Env, test.Exec)
// 		// When an error is returned it doesn't always mean the tests fails.
// 		// Have to come up with some conditional failing mechanism.
// 		if err != nil {
// 			t.Errorf("%s", err)
// 			helper.Log.Infoln(err)
// 		}

// 		rexp := helper.FromHex(test.Out)
// 		if bytes.Compare(rexp, ret) != 0 {
// 			t.Errorf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
// 		}

// 		gexp := ethutil.Big(test.Gas)
// 		if gexp.Cmp(gas) != 0 {
// 			t.Errorf("%s's gas failed. Expected %v, got %v\n", name, gexp, gas)
// 		}

// 		for addr, account := range test.Post {
// 			obj := state.GetStateObject(helper.FromHex(addr))
// 			for addr, value := range account.Storage {
// 				v := obj.GetState(helper.FromHex(addr)).Bytes()
// 				vexp := helper.FromHex(value)

// 				if bytes.Compare(v, vexp) != 0 {
// 					t.Errorf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address()[0:4], addr, vexp, v, ethutil.BigD(vexp), ethutil.BigD(v))
// 				}
// 			}
// 		}
// 	}
// }

// // I've created a new function for each tests so it's easier to identify where the problem lies if any of them fail.
// func TestVMArithmetic(t *testing.T) {
// 	//helper.Logger.SetLogLevel(5)
// 	const fn = "../files/vmtests/vmArithmeticTest.json"
// 	RunVmTest(fn, t)
// }

// /*
// deleted?
// func TestVMSystemOperation(t *testing.T) {
// 	helper.Logger.SetLogLevel(5)
// 	const fn = "../files/vmtests/vmSystemOperationsTest.json"
// 	RunVmTest(fn, t)
// }
// */

// func TestBitwiseLogicOperation(t *testing.T) {
// 	const fn = "../files/vmtests/vmBitwiseLogicOperationTest.json"
// 	RunVmTest(fn, t)
// }

// func TestBlockInfo(t *testing.T) {
// 	const fn = "../files/vmtests/vmBlockInfoTest.json"
// 	RunVmTest(fn, t)
// }

// func TestEnvironmentalInfo(t *testing.T) {
// 	const fn = "../files/vmtests/vmEnvironmentalInfoTest.json"
// 	RunVmTest(fn, t)
// }

// func TestFlowOperation(t *testing.T) {
// 	helper.Logger.SetLogLevel(5)
// 	const fn = "../files/vmtests/vmIOandFlowOperationsTest.json"
// 	RunVmTest(fn, t)
// }

// func TestPushDupSwap(t *testing.T) {
// 	const fn = "../files/vmtests/vmPushDupSwapTest.json"
// 	RunVmTest(fn, t)
// }

// func TestVMSha3(t *testing.T) {
// 	const fn = "../files/vmtests/vmSha3Test.json"
// 	RunVmTest(fn, t)
// }

// func TestVm(t *testing.T) {
// 	const fn = "../files/vmtests/vmtests.json"
// 	RunVmTest(fn, t)
// }
=======
import (
	"bytes"
	"math/big"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/chain"
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

type Log struct {
	Address string
	Data    string
	Topics  []string
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
	Logs        map[string]Log
	Gas         string
	Out         string
	Post        map[string]Account
	Pre         map[string]Account
}

func RunVmTest(p string, t *testing.T) {
	tests := make(map[string]VmTest)
	helper.CreateFileTests(t, p, &tests)

	for name, test := range tests {
		statedb := state.New(helper.NewTrie())
		for addr, account := range test.Pre {
			obj := StateObjectFromAccount(addr, account)
			statedb.SetStateObject(obj)
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

		if len(test.Exec) > 0 {
			ret, logs, gas, err = helper.RunVm(statedb, env, test.Exec)
		} else {
			ret, logs, gas, err = helper.RunState(statedb, env, test.Transaction)
		}

		// When an error is returned it doesn't always mean the tests fails.
		// Have to come up with some conditional failing mechanism.
		if err != nil {
			helper.Log.Infoln(err)
		}

		rexp := helper.FromHex(test.Out)
		if bytes.Compare(rexp, ret) != 0 {
			t.Errorf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
		}

		if len(test.Gas) > 0 {
			gexp := ethutil.Big(test.Gas)
			if gexp.Cmp(gas) != 0 {
				t.Errorf("%s's gas failed. Expected %v, got %v\n", name, gexp, gas)
			}
		}

		for addr, account := range test.Post {
			obj := statedb.GetStateObject(helper.FromHex(addr))
			for addr, value := range account.Storage {
				v := obj.GetState(helper.FromHex(addr)).Bytes()
				vexp := helper.FromHex(value)

				if bytes.Compare(v, vexp) != 0 {
					t.Errorf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address()[0:4], addr, vexp, v, ethutil.BigD(vexp), ethutil.BigD(v))
				}
			}
		}

		if len(test.Logs) > 0 {
			genBloom := ethutil.LeftPadBytes(chain.LogsBloom(logs).Bytes(), 64)
			// Logs within the test itself aren't correct, missing empty fields (32 0s)
			for bloom /*logs*/, _ := range test.Logs {
				if !bytes.Equal(genBloom, ethutil.Hex2Bytes(bloom)) {
					t.Errorf("bloom mismatch")
				}
			}
		}
	}
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

func TestStateSpecialTest(t *testing.T) {
	const fn = "../files/StateTests/stSpecialTest.json"
	RunVmTest(fn, t)
}
>>>>>>> develop
