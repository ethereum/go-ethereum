package vm

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
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

// I've created a new function for each tests so it's easier to identify where the problem lies if any of them fail.
func TestVMArithmetic(t *testing.T) {
	const fn = "../files/VMTests/vmArithmeticTest.json"
	RunVmTest(fn, t)
}

func TestBitwiseLogicOperation(t *testing.T) {
	const fn = "../files/VMTests/vmBitwiseLogicOperationTest.json"
	RunVmTest(fn, t)
}

func TestBlockInfo(t *testing.T) {
	const fn = "../files/VMTests/vmBlockInfoTest.json"
	RunVmTest(fn, t)
}

func TestEnvironmentalInfo(t *testing.T) {
	const fn = "../files/VMTests/vmEnvironmentalInfoTest.json"
	RunVmTest(fn, t)
}

func TestFlowOperation(t *testing.T) {
	const fn = "../files/VMTests/vmIOandFlowOperationsTest.json"
	RunVmTest(fn, t)
}

func TestLogTest(t *testing.T) {
	const fn = "../files/VMTests/vmLogTest.json"
	RunVmTest(fn, t)
}

func TestPerformance(t *testing.T) {
	const fn = "../files/VMTests/vmPerformanceTest.json"
	RunVmTest(fn, t)
}

func TestPushDupSwap(t *testing.T) {
	const fn = "../files/VMTests/vmPushDupSwapTest.json"
	RunVmTest(fn, t)
}

func TestVMSha3(t *testing.T) {
	const fn = "../files/VMTests/vmSha3Test.json"
	RunVmTest(fn, t)
}

func TestVm(t *testing.T) {
	const fn = "../files/VMTests/vmtests.json"
	RunVmTest(fn, t)
}

func TestVmLog(t *testing.T) {
	const fn = "../files/VMTests/vmLogTest.json"
	RunVmTest(fn, t)
}

func TestInputLimits(t *testing.T) {
	const fn = "../files/VMTests/vmInputLimits.json"
	RunVmTest(fn, t)
}

func TestInputLimitsLight(t *testing.T) {
	const fn = "../files/VMTests/vmInputLimitsLight.json"
	RunVmTest(fn, t)
}

func TestStateSystemOperations(t *testing.T) {
	const fn = "../files/StateTests/stSystemOperationsTest.json"
	RunVmTest(fn, t)
}

func TestStateExample(t *testing.T) {
	const fn = "../files/StateTests/stExample.json"
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

func TestStateBlockHash(t *testing.T) {
	const fn = "../files/StateTests/stBlockHashTest.json"
	RunVmTest(fn, t)
}

func TestStateInitCode(t *testing.T) {
	const fn = "../files/StateTests/stInitCodeTest.json"
	RunVmTest(fn, t)
}

func TestStateLog(t *testing.T) {
	const fn = "../files/StateTests/stLogTests.json"
	RunVmTest(fn, t)
}

func TestStateTransaction(t *testing.T) {
	const fn = "../files/StateTests/stTransactionTest.json"
	RunVmTest(fn, t)
}

func TestCallCreateCallCode(t *testing.T) {
	const fn = "../files/StateTests/stCallCreateCallCodeTest.json"
	RunVmTest(fn, t)
}

func TestMemory(t *testing.T) {
	const fn = "../files/StateTests/stMemoryTest.json"
	RunVmTest(fn, t)
}

func TestMemoryStress(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	const fn = "../files/StateTests/stMemoryStressTest.json"
	RunVmTest(fn, t)
}

func TestQuadraticComplexity(t *testing.T) {
	if os.Getenv("TEST_VM_COMPLEX") == "" {
		t.Skip()
	}
	const fn = "../files/StateTests/stQuadraticComplexityTest.json"
	RunVmTest(fn, t)
}

func TestSolidity(t *testing.T) {
	const fn = "../files/StateTests/stSolidityTest.json"
	RunVmTest(fn, t)
}

func TestWallet(t *testing.T) {
	const fn = "../files/StateTests/stWalletTest.json"
	RunVmTest(fn, t)
}

func TestStateTestsRandom(t *testing.T) {
	fns, _ := filepath.Glob("../files/StateTests/RandomTests/*")
	for _, fn := range fns {
		RunVmTest(fn, t)
	}
}

func TestVMRandom(t *testing.T) {
	t.Skip() // fucked as of 2015-06-09. unskip once unfucked /Gustav
	fns, _ := filepath.Glob("../files/VMTests/RandomTests/*")
	for _, fn := range fns {
		RunVmTest(fn, t)
	}
}
