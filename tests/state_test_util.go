package tests

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

func RunStateTest(p string) error {
	skipTest := make(map[string]bool, len(stateSkipTests))
	for _, name := range stateSkipTests {
		skipTest[name] = true
	}

	tests := make(map[string]VmTest)
	CreateFileTests(p, &tests)

	for name, test := range tests {
		if skipTest[name] {
			fmt.Println("Skipping state test", name)
			return nil
		}
		db, _ := ethdb.NewMemDatabase()
		statedb := state.New(common.Hash{}, db)
		for addr, account := range test.Pre {
			obj := StateObjectFromAccount(db, addr, account)
			statedb.SetStateObject(obj)
			for a, v := range account.Storage {
				obj.SetState(common.HexToHash(a), common.HexToHash(s))
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
			ret []byte
			// gas  *big.Int
			// err  error
			logs state.Logs
		)

		ret, logs, _, _ = RunState(statedb, env, test.Transaction)

		// // Compare expected  and actual return
		// switch name {
		// // the memory required for these tests (4294967297 bytes) would take too much time.
		// // on 19 May 2015 decided to skip these tests their output.
		// case "mload32bitBound_return", "mload32bitBound_return2":
		// default:
		rexp := common.FromHex(test.Out)
		if bytes.Compare(rexp, ret) != 0 {
			return fmt.Errorf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
		}
		// }

		// check post state
		for addr, account := range test.Post {
			obj := statedb.GetStateObject(common.HexToAddress(addr))
			if obj == nil {
				continue
			}

			if obj.Balance().Cmp(common.Big(account.Balance)) != 0 {
				return fmt.Errorf("%s's : (%x) balance failed. Expected %v, got %v => %v\n", name, obj.Address().Bytes()[:4], account.Balance, obj.Balance(), new(big.Int).Sub(common.Big(account.Balance), obj.Balance()))
			}

			if obj.Nonce() != common.String2Big(account.Nonce).Uint64() {
				return fmt.Errorf("%s's : (%x) nonce failed. Expected %v, got %v\n", name, obj.Address().Bytes()[:4], account.Nonce, obj.Nonce())
			}

			for addr, value := range account.Storage {
				v := obj.GetState(common.HexToHash(addr)).Bytes()
				vexp := common.FromHex(value)

				if bytes.Compare(v, vexp) != 0 {
					return fmt.Errorf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address().Bytes()[0:4], addr, vexp, v, common.BigD(vexp), common.BigD(v))
				}
			}
		}

		statedb.Sync()
		//if !bytes.Equal(common.Hex2Bytes(test.PostStateRoot), statedb.Root()) {
		if common.HexToHash(test.PostStateRoot) != statedb.Root() {
			return fmt.Errorf("%s's : Post state root error. Expected %s, got %x", name, test.PostStateRoot, statedb.Root())
		}

		// check logs
		if len(test.Logs) > 0 {
			lerr := checkLogs(test.Logs, logs)
			if lerr != nil {
				return fmt.Errorf("'%s' ", name, lerr.Error())
			}
		}

		fmt.Println("State test passed: ", name)
		//fmt.Println(string(statedb.Dump()))
	}
	return nil
}

func RunState(statedb *state.StateDB, env, tx map[string]string) ([]byte, state.Logs, *big.Int, error) {
	var (
		keyPair, _ = crypto.NewKeyPairFromSec([]byte(common.Hex2Bytes(tx["secretKey"])))
		data       = common.FromHex(tx["data"])
		gas        = common.Big(tx["gasLimit"])
		price      = common.Big(tx["gasPrice"])
		value      = common.Big(tx["value"])
		nonce      = common.Big(tx["nonce"]).Uint64()
		caddr      = common.HexToAddress(env["currentCoinbase"])
	)

	var to *common.Address
	if len(tx["to"]) > 2 {
		t := common.HexToAddress(tx["to"])
		to = &t
	}
	// Set pre compiled contracts
	vm.Precompiled = vm.PrecompiledContracts()

	snapshot := statedb.Copy()
	coinbase := statedb.GetOrNewStateObject(caddr)
	coinbase.SetGasPool(common.Big(env["currentGasLimit"]))

	message := NewMessage(common.BytesToAddress(keyPair.Address()), to, data, value, gas, price, nonce)
	vmenv := NewEnvFromMap(statedb, env, tx)
	vmenv.origin = common.BytesToAddress(keyPair.Address())
	ret, _, err := core.ApplyMessage(vmenv, message, coinbase)
	if core.IsNonceErr(err) || core.IsInvalidTxErr(err) {
		statedb.Set(snapshot)
	}
	statedb.Update()

	return ret, vmenv.state.Logs(), vmenv.Gas, err
}
