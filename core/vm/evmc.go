// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"fmt"
	"math/big"
	"os"
	"strings"
	"sync"

	"github.com/ethereum/evmc/bindings/go/evmc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

type EVMC struct {
	instance *evmc.Instance
	env      *EVM
	readOnly bool // TODO: The readOnly flag should not be here.
}

var (
	createMu     sync.Mutex
	evmcInstance *evmc.Instance
)

func createVM(path string) *evmc.Instance {
	createMu.Lock()
	defer createMu.Unlock()

	if evmcInstance == nil {
		vmPath := os.Getenv("EVMC_PATH")
		if len(vmPath) == 0 {
			vmPath = path
		}
		if len(vmPath) == 0 {
			panic("EVMC VM path not provided, set EVMC_PATH environment variable or --vm.evm option")
		}

		var err error
		evmcInstance, err = evmc.Load(vmPath)
		if err != nil {
			panic(err.Error())
		}
		log.Info("EVMC VM loaded", "name", evmcInstance.Name(), "version", evmcInstance.Version(), "path", vmPath)

		for _, option := range strings.Split(os.Getenv("EVMC_OPTIONS"), " ") {
			if idx := strings.Index(option, "="); idx >= 0 {
				name := option[:idx]
				value := option[idx+1:]
				err := evmcInstance.SetOption(name, value)
				if err == nil {
					log.Info("EVMC VM option set", "name", name, "value", value)
				} else {
					log.Warn("EVMC VM option setting failed", "name", name, "error", err)
				}
			}
		}
	}
	return evmcInstance
}

func NewEVMC(path string, env *EVM) *EVMC {
	return &EVMC{createVM(path), env, false}
}

// Implements evmc.HostContext interface.
type HostContext struct {
	env      *EVM
	contract *Contract
}

func (host *HostContext) AccountExists(addr common.Address) bool {
	env := host.env
	eip158 := env.ChainConfig().IsEIP158(env.BlockNumber)
	if eip158 {
		if !env.StateDB.Empty(addr) {
			return true
		}
	} else if env.StateDB.Exist(addr) {
		return true
	}
	return false
}

func (host *HostContext) GetStorage(addr common.Address, key common.Hash) common.Hash {
	env := host.env
	return env.StateDB.GetState(addr, key)
}

func (host *HostContext) SetStorage(addr common.Address, key common.Hash, value common.Hash) (status evmc.StorageStatus) {
	env := host.env

	oldValue := env.StateDB.GetState(addr, key)
	if oldValue == value {
		return evmc.StorageUnchanged
	}

	env.StateDB.SetState(addr, key, value)

	zero := common.Hash{}
	status = evmc.StorageModified
	if oldValue == zero {
		return evmc.StorageAdded
	} else if value == zero {
		env.StateDB.AddRefund(params.SstoreRefundGas)
		return evmc.StorageDeleted
	}
	return evmc.StorageModified
}

func (host *HostContext) GetBalance(addr common.Address) common.Hash {
	env := host.env
	balance := env.StateDB.GetBalance(addr)
	return common.BigToHash(balance)
}

func (host *HostContext) GetCodeSize(addr common.Address) int {
	env := host.env
	return env.StateDB.GetCodeSize(addr)
}

func (host *HostContext) GetCodeHash(addr common.Address) common.Hash {
	env := host.env
	return env.StateDB.GetCodeHash(addr)
}

func (host *HostContext) GetCode(addr common.Address) []byte {
	env := host.env
	return env.StateDB.GetCode(addr)
}

func (host *HostContext) Selfdestruct(addr common.Address, beneficiary common.Address) {
	env := host.env
	db := env.StateDB
	if !db.HasSuicided(addr) {
		db.AddRefund(params.SuicideRefundGas)
	}
	balance := db.GetBalance(addr)
	db.AddBalance(beneficiary, balance)
	db.Suicide(addr)
}

func (host *HostContext) GetTxContext() (gasPrice common.Hash, origin common.Address, coinbase common.Address,
	number int64, timestamp int64, gasLimit int64, difficulty common.Hash) {

	env := host.env
	gasPrice = common.BigToHash(env.GasPrice)
	origin = env.Origin
	coinbase = env.Coinbase
	number = env.BlockNumber.Int64()
	timestamp = env.Time.Int64()
	gasLimit = int64(env.GasLimit)
	difficulty = common.BigToHash(env.Difficulty)

	return gasPrice, origin, coinbase, number, timestamp, gasLimit, difficulty
}

func (host *HostContext) GetBlockHash(number int64) common.Hash {
	env := host.env
	b := env.BlockNumber.Int64()
	if number >= (b-256) && number < b {
		return env.GetHash(uint64(number))
	}
	return common.Hash{}
}

func (host *HostContext) EmitLog(addr common.Address, topics []common.Hash, data []byte) {
	env := host.env
	env.StateDB.AddLog(&types.Log{
		Address:     addr,
		Topics:      topics,
		Data:        data,
		BlockNumber: env.BlockNumber.Uint64(),
	})
}

func (host *HostContext) Call(kind evmc.CallKind,
	destination common.Address, sender common.Address, value *big.Int, input []byte, gas int64, depth int,
	static bool) (output []byte, gasLeft int64, createAddr common.Address, err error) {

	env := host.env

	gasU := uint64(gas)
	var gasLeftU uint64

	switch kind {
	case evmc.Call:
		if static {
			output, gasLeftU, err = env.StaticCall(host.contract, destination, input, gasU)
		} else {
			output, gasLeftU, err = env.Call(host.contract, destination, input, gasU, value)
		}
	case evmc.DelegateCall:
		output, gasLeftU, err = env.DelegateCall(host.contract, destination, input, gasU)
	case evmc.CallCode:
		output, gasLeftU, err = env.CallCode(host.contract, destination, input, gasU, value)
	case evmc.Create:
		var createOutput []byte
		createOutput, createAddr, gasLeftU, err = env.Create(host.contract, input, gasU, value)
		isHomestead := env.ChainConfig().IsHomestead(env.BlockNumber)
		if !isHomestead && err == ErrCodeStoreOutOfGas {
			err = nil
		}
		if err == errExecutionReverted {
			// Assign return buffer from REVERT.
			// TODO: Bad API design: return data buffer and the code is returned in the same place. In worst case
			//       the code is returned also when there is not enough funds to deploy the code.
			output = createOutput
		}
	}

	// Map errors.
	if err == errExecutionReverted {
		err = evmc.Revert
	} else if err != nil {
		err = evmc.Failure
	}

	gasLeft = int64(gasLeftU)
	return output, gasLeft, createAddr, err
}

func getRevision(env *EVM) evmc.Revision {
	n := env.BlockNumber
	conf := env.ChainConfig()
	if conf.IsConstantinople(n) {
		return evmc.Constantinople
	}
	if conf.IsByzantium(n) {
		return evmc.Byzantium
	}
	if conf.IsEIP158(n) {
		return evmc.SpuriousDragon
	}
	if conf.IsEIP150(n) {
		return evmc.TangerineWhistle
	}
	if conf.IsHomestead(n) {
		return evmc.Homestead
	}
	return evmc.Frontier
}

func (evm *EVMC) Run(contract *Contract, input []byte, readOnly bool) (ret []byte, err error) {
	evm.env.depth++
	defer func() { evm.env.depth-- }()

	// Don't bother with the execution if there's no code.
	if len(contract.Code) == 0 {
		return nil, nil
	}

	kind := evmc.Call
	if evm.env.StateDB.GetCodeSize(contract.Address()) == 0 {
		// Guess if this is a CREATE.
		kind = evmc.Create
	}

	// Make sure the readOnly is only set if we aren't in readOnly yet.
	// This makes also sure that the readOnly flag isn't removed for child calls.
	if readOnly && !evm.readOnly {
		evm.readOnly = true
		defer func() { evm.readOnly = false }()
	}

	output, gasLeft, err := evm.instance.Execute(
		&HostContext{evm.env, contract},
		getRevision(evm.env),
		kind,
		evm.readOnly,
		evm.env.depth-1,
		int64(contract.Gas),
		contract.Address(),
		contract.Caller(),
		input,
		common.BigToHash(contract.value),
		contract.Code,
		common.Hash{})

	contract.Gas = uint64(gasLeft)

	if err == evmc.Revert {
		err = errExecutionReverted
	} else if evmcError, ok := err.(evmc.Error); ok && evmcError.IsInternalError() {
		panic(fmt.Sprintf("EVMC VM internal error: %s", evmcError.Error()))
	}

	return output, err
}

func (evm *EVMC) CanRun([]byte) bool {
	return true
}
