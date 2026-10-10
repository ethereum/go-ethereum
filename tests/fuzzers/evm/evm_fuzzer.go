// Copyright 2026 The go-ethereum Authors
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

package evm

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var (
	contractAddr = common.HexToAddress("0x000000000000000000000000000000000000c0de")
	calleeAddr   = common.BytesToAddress([]byte{0x20}) // low, so a PUSH1 reaches it, past the precompiles
	callerAddr   = common.HexToAddress("0x000000000000000000000000000000000000face")

	// calleeCode stores, logs and returns 32 bytes, as a target of the call opcodes.
	calleeCode = []byte{
		byte(vm.PUSH1), 0x2a, byte(vm.PUSH1), 0x07, byte(vm.SSTORE),
		byte(vm.PUSH1), 0xbb, byte(vm.PUSH1), 0x00, byte(vm.MSTORE),
		byte(vm.PUSH1), 0x20, byte(vm.PUSH1), 0x00, byte(vm.LOG0),
		byte(vm.PUSH1), 0x20, byte(vm.PUSH1), 0x00, byte(vm.RETURN),
	}
)

// fork is a chain configuration the interpreters are compared under.
type fork struct {
	name   string
	config *params.ChainConfig
	merged bool
}

// forks spans the fork gates of the generated dispatch (SHL from Constantinople,
// PUSH0 from Shanghai) in both states, and the state gas of Amsterdam.
var forks = func() []fork {
	byzantium := *params.TestChainConfig
	byzantium.ConstantinopleBlock = nil
	byzantium.PetersburgBlock = nil
	byzantium.IstanbulBlock = nil
	byzantium.MuirGlacierBlock = nil
	byzantium.BerlinBlock = nil
	byzantium.LondonBlock = nil
	byzantium.ArrowGlacierBlock = nil
	byzantium.GrayGlacierBlock = nil

	amsterdam := *params.MergedTestChainConfig
	amsterdam.AmsterdamTime = new(uint64)

	return []fork{
		{"frontier", params.NonActivatedConfig, false},
		{"byzantium", &byzantium, false},
		{"london", params.TestChainConfig, false},
		{"merged", params.MergedTestChainConfig, true},
		{"amsterdam", &amsterdam, true},
	}
}()

// result is the observable outcome of an execution.
type result struct {
	ret    []byte
	gas    vm.GasBudget
	err    string
	refund uint64
	root   common.Hash
	logs   []*types.Log
}

func (r *result) String() string {
	return fmt.Sprintf("ret=%x gas=%+v err=%q refund=%d root=%x logs=%d", r.ret, r.gas, r.err, r.refund, r.root, len(r.logs))
}

func (r *result) equal(o *result) bool {
	if !bytes.Equal(r.ret, o.ret) || r.gas != o.gas || r.err != o.err || r.refund != o.refund || r.root != o.root {
		return false
	}
	if len(r.logs) != len(o.logs) {
		return false
	}
	for i := range r.logs {
		if r.logs[i].Address != o.logs[i].Address || !bytes.Equal(r.logs[i].Data, o.logs[i].Data) || len(r.logs[i].Topics) != len(o.logs[i].Topics) {
			return false
		}
		for j := range r.logs[i].Topics {
			if r.logs[i].Topics[j] != o.logs[i].Topics[j] {
				return false
			}
		}
	}
	return true
}

// execute runs the code on fresh state. A tracer makes EVM.Run take the table
// loop instead of the generated dispatch, its hooks being empty changes nothing
// else.
func execute(f fork, tableLoop bool, code, input []byte, gas uint64) *result {
	statedb, err := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	if err != nil {
		panic(err)
	}
	statedb.CreateAccount(contractAddr)
	statedb.SetBalance(contractAddr, uint256.NewInt(1000), tracing.BalanceChangeUnspecified)
	statedb.SetState(contractAddr, common.Hash{31: 0x07}, common.Hash{31: 0x07})
	statedb.SetCode(contractAddr, code, tracing.CodeChangeUnspecified)
	statedb.CreateAccount(calleeAddr)
	statedb.SetBalance(calleeAddr, uint256.NewInt(500), tracing.BalanceChangeUnspecified)
	statedb.SetCode(calleeAddr, calleeCode, tracing.CodeChangeUnspecified)
	statedb.CreateAccount(callerAddr)
	statedb.SetBalance(callerAddr, uint256.NewInt(1<<62), tracing.BalanceChangeUnspecified)
	statedb.Finalise(params.Rules{IsEIP158: true})

	blockCtx := vm.BlockContext{
		CanTransfer:      core.CanTransfer,
		Transfer:         core.Transfer,
		GetHash:          func(uint64) common.Hash { return common.Hash{0xde, 0xad} },
		Coinbase:         common.HexToAddress("0xc01ba5e"),
		BlockNumber:      big.NewInt(8),
		Time:             1234,
		Difficulty:       big.NewInt(0x20000),
		GasLimit:         30_000_000,
		BaseFee:          big.NewInt(7),
		BlobBaseFee:      big.NewInt(3),
		CostPerStateByte: params.CostPerStateByte,
	}
	if f.merged {
		blockCtx.Random = &common.Hash{0x01, 0x02}
	}
	var config vm.Config
	if tableLoop {
		config.Tracer = new(tracing.Hooks)
	}
	evm := vm.NewEVM(blockCtx, statedb, f.config, config)
	evm.SetTxContext(vm.TxContext{
		Origin:     callerAddr,
		GasPrice:   uint256.NewInt(1),
		BlobHashes: []common.Hash{{0xb1, 0x0b}},
	})
	ret, left, err := evm.Call(callerAddr, contractAddr, input, vm.NewGasBudget(gas, 0), new(uint256.Int))

	res := &result{
		ret:    ret,
		gas:    left,
		refund: statedb.GetRefund(),
		root:   statedb.IntermediateRoot(params.Rules{IsEIP158: true}),
		logs:   statedb.Logs(),
	}
	if err != nil {
		res.err = err.Error()
	}
	return res
}

// compare runs the code through both interpreters under every fork and panics
// on any observable difference.
func compare(code, input []byte, gas uint64) {
	for _, f := range forks {
		gen := execute(f, false, code, input, gas)
		table := execute(f, true, code, input, gas)
		if !gen.equal(table) {
			panic(fmt.Sprintf("interpreters diverge on %s: code=%x input=%x gas=%d\n  generated: %v\n  table:     %v", f.name, code, input, gas, gen, table))
		}
	}
}

// splitInput takes the gas and calldata from the head of the fuzzer input and
// returns the rest as code: 3 bytes of gas, a length byte, then the calldata.
func splitInput(data []byte) (gas uint64, input, rest []byte) {
	if len(data) < 4 {
		return 0, nil, nil
	}
	gas = uint64(data[0])<<16 | uint64(data[1])<<8 | uint64(data[2])
	n := min(int(data[3]), len(data)-4)
	return gas, data[4 : 4+n], data[4+n:]
}

// fuzzCode executes arbitrary bytecode.
func fuzzCode(data []byte) int {
	gas, input, code := splitInput(data)
	if len(code) == 0 || len(code) > params.MaxCodeSize {
		return 0
	}
	compare(code, input, gas)
	return 1
}

// fuzzDeepStack fills the stack to a fuzzed depth before the fuzzed code, which
// random bytecode rarely gets close to, to hit the stack limit and the gas
// boundaries of the opcodes that run there.
func fuzzDeepStack(data []byte) int {
	gas, input, rest := splitInput(data)
	if len(rest) < 3 {
		return 0
	}
	depth := int(binary.BigEndian.Uint16(rest)) % (int(params.StackLimit) + 1)
	tail := rest[2:]
	if len(tail) > 1024 {
		return 0
	}
	code := make([]byte, 0, 2*depth+len(tail))
	for i := range depth {
		code = append(code, byte(vm.PUSH1), byte(i))
	}
	code = append(code, tail...)

	// Pushing costs 3 gas each, give the tail the fuzzed gas on top.
	compare(code, input, 3*uint64(depth)+gas)
	return 1
}
