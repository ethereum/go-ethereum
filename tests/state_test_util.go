package tests

import (
	"bytes"
	// "errors"
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

func RunStateTest(p string, t *testing.T) {

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
				obj.SetState(common.HexToHash(a), common.NewValue(common.FromHex(v)))
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

		switch name {
		// the memory required for these tests (4294967297 bytes) would take too much time.
		// on 19 May 2015 decided to skip these tests their output.
		case "mload32bitBound_return", "mload32bitBound_return2":
		default:
			rexp := common.FromHex(test.Out)
			if bytes.Compare(rexp, ret) != 0 {
				t.Errorf("%s's return failed. Expected %x, got %x\n", name, rexp, ret)
			}
		}

		// check post state
		for addr, account := range test.Post {
			obj := statedb.GetStateObject(common.HexToAddress(addr))
			if obj == nil {
				continue
			}

			if obj.Balance().Cmp(common.Big(account.Balance)) != 0 {
				t.Errorf("%s's : (%x) balance failed. Expected %v, got %v => %v\n", name, obj.Address().Bytes()[:4], account.Balance, obj.Balance(), new(big.Int).Sub(common.Big(account.Balance), obj.Balance()))
			}

			if obj.Nonce() != common.String2Big(account.Nonce).Uint64() {
				t.Errorf("%s's : (%x) nonce failed. Expected %v, got %v\n", name, obj.Address().Bytes()[:4], account.Nonce, obj.Nonce())
			}

			for addr, value := range account.Storage {
				v := obj.GetState(common.HexToHash(addr)).Bytes()
				vexp := common.FromHex(value)

				if bytes.Compare(v, vexp) != 0 {
					t.Errorf("%s's : (%x: %s) storage failed. Expected %x, got %x (%v %v)\n", name, obj.Address().Bytes()[0:4], addr, vexp, v, common.BigD(vexp), common.BigD(v))
				}
			}
		}

		statedb.Sync()
		//if !bytes.Equal(common.Hex2Bytes(test.PostStateRoot), statedb.Root()) {
		if common.HexToHash(test.PostStateRoot) != statedb.Root() {
			t.Errorf("%s's : Post state root error. Expected %s, got %x", name, test.PostStateRoot, statedb.Root())
		}

		// check logs
		if len(test.Logs) > 0 {
			lerr := CheckLogs(test.Logs, logs, t)
			if lerr != nil {
				t.Errorf("'%s' ", name, lerr.Error())
			}
		}

		fmt.Println("State test passed: ", name)
		//fmt.Println(string(statedb.Dump()))
	}
	// logger.Flush()
}

func CheckLogs(tlog []Log, logs state.Logs, t *testing.T) error {

	if len(tlog) != len(logs) {
		return fmt.Errorf("log length mismatch. Expected %d, got %d", len(tlog), len(logs))
	} else {
		for i, log := range tlog {
			if common.HexToAddress(log.AddressF) != logs[i].Address {
				return fmt.Errorf("log address expected %v got %x", log.AddressF, logs[i].Address)
			}

			if !bytes.Equal(logs[i].Data, common.FromHex(log.DataF)) {
				return fmt.Errorf("log data expected %v got %x", log.DataF, logs[i].Data)
			}

			if len(log.TopicsF) != len(logs[i].Topics) {
				return fmt.Errorf("log topics length expected %d got %d", len(log.TopicsF), logs[i].Topics)
			} else {
				for j, topic := range log.TopicsF {
					if common.HexToHash(topic) != logs[i].Topics[j] {
						return fmt.Errorf("log topic[%d] expected %v got %x", j, topic, logs[i].Topics[j])
					}
				}
			}
			genBloom := common.LeftPadBytes(types.LogsBloom(state.Logs{logs[i]}).Bytes(), 256)

			if !bytes.Equal(genBloom, common.Hex2Bytes(log.BloomF)) {
				return fmt.Errorf("bloom mismatch")
			}
		}
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
