package tests

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger/glog"
)

func RunVmTestWithReader(r io.Reader) error {
	tests := make(map[string]VmTest)
	err := readJson(r, &tests)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	if err := runVmTests(tests); err != nil {
		return err
	}

	return nil
}

func RunVmTest(p string) error {

	tests := make(map[string]VmTest)
	err := readJsonFile(p, &tests)
	if err != nil {
		return err
	}

	if err := runVmTests(tests); err != nil {
		return err
	}

	return nil
}

func runVmTests(tests map[string]VmTest) error {
	skipTest := make(map[string]bool, len(VmSkipTests))
	for _, name := range VmSkipTests {
		skipTest[name] = true
	}

	for name, test := range tests {
		if skipTest[name] {
			glog.Infoln("Skipping VM test", name)
			return nil
		}

		if err := runVmTest(test); err != nil {
			return fmt.Errorf("%s %s", name, err.Error())
		}

		glog.Infoln("VM test passed: ", name)
		//fmt.Println(string(statedb.Dump()))
	}
	return nil
}

func runVmTest(test VmTest) error {
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

	// Compare expectedand actual return
	rexp := common.FromHex(test.Out)
	if bytes.Compare(rexp, ret) != 0 {
		return fmt.Errorf("return failed. Expected %x, got %x\n", rexp, ret)
	}

	// Check gas usage
	if len(test.Gas) == 0 && err == nil {
		return fmt.Errorf("gas unspecified, indicating an error. VM returned (incorrectly) successfull")
	} else {
		gexp := common.Big(test.Gas)
		if gexp.Cmp(gas) != 0 {
			return fmt.Errorf("gas failed. Expected %v, got %v\n", gexp, gas)
		}
	}

	// check post state
	for addr, account := range test.Post {
		obj := statedb.GetStateObject(common.HexToAddress(addr))
		if obj == nil {
			continue
		}

		for addr, value := range account.Storage {
			v := obj.GetState(common.HexToHash(addr))
			vexp := common.HexToHash(value)

			if v != vexp {
				return fmt.Errorf("(%x: %s) storage failed. Expected %x, got %x (%v %v)\n", obj.Address().Bytes()[0:4], addr, vexp, v, vexp.BigD(vexp), v.Big(v))
			}
		}
	}

	// check logs
	if len(test.Logs) > 0 {
		lerr := checkLogs(test.Logs, logs)
		if lerr != nil {
			return lerr
		}
	}

	return nil
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
