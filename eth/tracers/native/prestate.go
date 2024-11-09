// Copyright 2022 The go-ethereum Authors
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

package native

import (
	"bytes"
	"encoding/json"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/internal"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

//go:generate go run github.com/fjl/gencodec -type account -field-override accountMarshaling -out gen_account_json.go

func init() {
	tracers.DefaultDirectory.Register("prestateTracer", newPrestateTracer, false)
}

type stateMap = map[common.Address]*account

type account struct {
	Balance *big.Int                    `json:"balance,omitempty"`
	Code    []byte                      `json:"code,omitempty"`
	Nonce   uint64                      `json:"nonce,omitempty"`
	Storage map[common.Hash]common.Hash `json:"storage,omitempty"`
	empty   bool
}

func (a *account) exists() bool {
	return a.Nonce > 0 || len(a.Code) > 0 || len(a.Storage) > 0 || (a.Balance != nil && a.Balance.Sign() != 0)
}

type accountMarshaling struct {
	Balance *hexutil.Big
	Code    hexutil.Bytes
}

type prestateTracer struct {
	env       *tracing.VMContext
	pre       stateMap
	post      stateMap
	to        common.Address
	config    prestateTracerConfig
	interrupt atomic.Bool // Atomic flag to signal execution interruption
	reason    error       // Textual reason for the interruption
	created   map[common.Address]bool
	deleted   map[common.Address]bool
}

type prestateTracerConfig struct {
	DiffMode       bool `json:"diffMode"`       // If true, this tracer will return state modifications
	DisableCode    bool `json:"disableCode"`    // If true, this tracer will not return the contract code
	DisableStorage bool `json:"disableStorage"` // If true, this tracer will not return the contract storage
}

func newPrestateTracer(ctx *tracers.Context, cfg json.RawMessage, chainConfig *params.ChainConfig) (*tracers.Tracer, error) {
	var config prestateTracerConfig
	if err := json.Unmarshal(cfg, &config); err != nil {
		return nil, err
	}
	t := &prestateTracer{
		pre:     stateMap{},
		post:    stateMap{},
		config:  config,
		created: make(map[common.Address]bool),
		deleted: make(map[common.Address]bool),
	}
	return &tracers.Tracer{
		Hooks: &tracing.Hooks{
			OnTxStart: t.OnTxStart,
			OnTxEnd:   t.OnTxEnd,
			OnOpcode:  t.OnOpcode,
		},
		GetResult: t.GetResult,
		Stop:      t.Stop,
	}, nil
}

// OnOpcode implements the EVMLogger interface to trace a single step of VM execution.
func (t *prestateTracer) OnOpcode(pc uint64, opcode byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	if err != nil {
		return
	}
	// Skip if tracing was interrupted
	if t.interrupt.Load() {
		return
	}
	op := vm.OpCode(opcode)
	stackData := scope.StackData()
	stackLen := len(stackData)
	caller := scope.Address()
	switch {
	case stackLen >= 1 && (op == vm.SLOAD || op == vm.SSTORE):
		slot := common.Hash(stackData[stackLen-1].Bytes32())
		t.lookupStorage(caller, slot)
	case stackLen >= 1 && (op == vm.EXTCODECOPY || op == vm.EXTCODEHASH || op == vm.EXTCODESIZE || op == vm.BALANCE || op == vm.SELFDESTRUCT):
		addr := common.Address(stackData[stackLen-1].Bytes20())
		t.lookupAccount(addr)
		if op == vm.SELFDESTRUCT {
			t.deleted[caller] = true
		}
	case stackLen >= 5 && (op == vm.DELEGATECALL || op == vm.CALL || op == vm.STATICCALL || op == vm.CALLCODE):
		addr := common.Address(stackData[stackLen-2].Bytes20())
		t.lookupAccount(addr)
	case op == vm.CREATE:
		nonce := t.env.StateDB.GetNonce(caller)
		addr := crypto.CreateAddress(caller, nonce)
		t.lookupAccount(addr)
		t.created[addr] = true
	case stackLen >= 4 && op == vm.CREATE2:
		offset := stackData[stackLen-2]
		size := stackData[stackLen-3]
		init, err := internal.GetMemoryCopyPadded(scope.MemoryData(), int64(offset.Uint64()), int64(size.Uint64()))
		if err != nil {
			log.Warn("failed to copy CREATE2 input", "err", err, "tracer", "prestateTracer", "offset", offset, "size", size)
			return
		}
		inithash := crypto.Keccak256(init)
		salt := stackData[stackLen-4]
		addr := crypto.CreateAddress2(caller, salt.Bytes32(), inithash)
		t.lookupAccount(addr)
		t.created[addr] = true
	}
}

func (t *prestateTracer) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.env = env
	if tx.To() == nil {
		t.to = crypto.CreateAddress(from, env.StateDB.GetNonce(from))
		t.created[t.to] = true
	} else {
		t.to = *tx.To()
	}

	t.lookupAccount(from)
	t.lookupAccount(t.to)
	t.lookupAccount(env.Coinbase)
}

func (t *prestateTracer) OnTxEnd(receipt *types.Receipt, err error) {
	if err != nil {
		return
	}
	if t.config.DiffMode {
		t.processDiffState()
	}
	// the new created contracts' prestate were empty, so delete them
	for a := range t.created {
		// the created contract maybe exists in statedb before the creating tx
		if s := t.pre[a]; s != nil && s.empty {
			delete(t.pre, a)
		}
	}
}

// GetResult returns the json-encoded nested list of call traces, and any
// error arising from the encoding or forceful termination (via `Stop`).
func (t *prestateTracer) GetResult() (json.RawMessage, error) {
	var res []byte
	var err error
	if t.config.DiffMode {
		res, err = json.Marshal(struct {
			Post stateMap `json:"post"`
			Pre  stateMap `json:"pre"`
		}{t.post, t.pre})
	} else {
		res, err = json.Marshal(t.pre)
	}
	if err != nil {
		return nil, err
	}
	return json.RawMessage(res), t.reason
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *prestateTracer) Stop(err error) {
	t.reason = err
	t.interrupt.Store(true)
}

func (t *prestateTracer) processDiffState() {
	for addr, state := range t.pre {
		// The deleted account's state is pruned from `post` but kept in `pre`
		if _, ok := t.deleted[addr]; ok {
			continue
		}
		modified := false
		postAccount := &account{Storage: make(map[common.Hash]common.Hash)}
		newBalance := t.env.StateDB.GetBalance(addr).ToBig()
		newNonce := t.env.StateDB.GetNonce(addr)

		if newBalance.Cmp(t.pre[addr].Balance) != 0 {
			modified = true
			postAccount.Balance = newBalance
		}
		if newNonce != t.pre[addr].Nonce {
			modified = true
			postAccount.Nonce = newNonce
		}
		if !t.config.DisableCode {
			newCode := t.env.StateDB.GetCode(addr)
			if !bytes.Equal(newCode, t.pre[addr].Code) {
				modified = true
				postAccount.Code = newCode
			}
		}

		if !t.config.DisableStorage {
			for key, val := range state.Storage {
				// don't include the empty slot
				if val == (common.Hash{}) {
					delete(t.pre[addr].Storage, key)
				}

				newVal := t.env.StateDB.GetState(addr, key)
				if val == newVal {
					// Omit unchanged slots
					delete(t.pre[addr].Storage, key)
				} else {
					modified = true
					if newVal != (common.Hash{}) {
						postAccount.Storage[key] = newVal
					}
				}
			}
		}

		if modified {
			t.post[addr] = postAccount
		} else {
			// if state is not modified, then no need to include into the pre state
			delete(t.pre, addr)
		}
	}
}

// lookupAccount fetches details of an account and adds it to the prestate
// if it doesn't exist there.
func (t *prestateTracer) lookupAccount(addr common.Address) {
	if _, ok := t.pre[addr]; ok {
		return
	}

	acc := &account{
		Balance: t.env.StateDB.GetBalance(addr).ToBig(),
		Nonce:   t.env.StateDB.GetNonce(addr),
		Code:    t.env.StateDB.GetCode(addr),
	}
	if !acc.exists() {
		acc.empty = true
	}
	// The code must be fetched first for the emptiness check.
	if t.config.DisableCode {
		acc.Code = nil
	}
	if !t.config.DisableStorage {
		acc.Storage = make(map[common.Hash]common.Hash)
	}
	t.pre[addr] = acc
}

// lookupStorage fetches the requested storage slot and adds
// it to the prestate of the given contract. It assumes `lookupAccount`
// has been performed on the contract before.
func (t *prestateTracer) lookupStorage(addr common.Address, key common.Hash) {
	if t.config.DisableStorage {
		return
	}
	if _, ok := t.pre[addr].Storage[key]; ok {
		return
	}
	t.pre[addr].Storage[key] = t.env.StateDB.GetState(addr, key)
}
