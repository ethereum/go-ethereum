package tests

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
)

func RunVmTest(p string) error {

	tests := make(map[string]VmTest)
	err := CreateFileTests(p, &tests)
	if err != nil {
		return err
	}

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

		// Compare expectedand actual return
		rexp := common.FromHex(test.Out)
		if bytes.Compare(rexp, ret) != 0 {
			return fmt.Errorf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
		}

		// Check gas usage
		if len(test.Gas) == 0 && err == nil {
			return fmt.Errorf("%s's gas unspecified, indicating an error. VM returned (incorrectly) successfull", name)
		} else {
			gexp := common.Big(test.Gas)
			if gexp.Cmp(gas) != 0 {
				return fmt.Errorf("%s's gas failed. Expected %v, got %v\n", name, gexp, gas)
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
					return t.Errorf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address().Bytes()[0:4], addr, vexp, v, vexp.Big(), v.Big())
				}
			}
		}

		// check logs
		if len(test.Logs) > 0 {
			lerr := checkLogs(test.Logs, logs)
			if lerr != nil {
				return fmt.Errorf("'%s' ", name, lerr.Error())
			}
		}

		fmt.Println("VM test passed: ", name)

		//fmt.Println(string(statedb.Dump()))
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
