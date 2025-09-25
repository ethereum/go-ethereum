// Copyright 2021 The go-ethereum Authors
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
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var loopInterruptTests = []string{
	// infinite loop using JUMP: push(2) jumpdest dup1 jump
	"60025b8056",
	// infinite loop using JUMPI: push(1) push(4) jumpdest dup2 dup2 jumpi
	"600160045b818157",
}

func TestLoopInterrupt(t *testing.T) {
	address := common.BytesToAddress([]byte("contract"))
	vmctx := BlockContext{
		Transfer: func(StateDB, common.Address, common.Address, *uint256.Int) {},
	}

	for i, tt := range loopInterruptTests {
		statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		statedb.CreateAccount(address)
		statedb.SetCode(address, common.Hex2Bytes(tt), tracing.CodeChangeUnspecified)
		statedb.Finalise(true)

		evm := NewEVM(vmctx, statedb, params.AllEthashProtocolChanges, Config{})

		errChannel := make(chan error)
		timeout := make(chan bool)

		go func(evm *EVM) {
			_, _, err := evm.Call(common.Address{}, address, nil, math.MaxUint64, new(uint256.Int))
			errChannel <- err
		}(evm)

		go func() {
			<-time.After(time.Second)
			timeout <- true
		}()

		evm.Cancel()

		select {
		case <-timeout:
			t.Errorf("test %d timed out", i)
		case err := <-errChannel:
			if err != nil {
				t.Errorf("test %d failure: %v", i, err)
			}
		}
	}
}

func BenchmarkInterpreter(b *testing.B) {
	var (
		statedb, _        = state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
		evm               = NewEVM(BlockContext{BlockNumber: big.NewInt(1), Time: 1, Random: &common.Hash{}}, statedb, params.MergedTestChainConfig, Config{})
		startGas   uint64 = 100_000_000
		value             = uint256.NewInt(0)
		stack             = newstack()
		mem               = NewMemory()
		contract          = NewContract(common.Address{}, common.Address{}, value, startGas, nil)
	)
	stack.push(uint256.NewInt(123))
	stack.push(uint256.NewInt(123))
	gasSStoreEIP3529 = makeGasSStoreFunc(params.SstoreClearsScheduleRefundEIP3529)
	for b.Loop() {
		gasSStoreEIP3529(evm, contract, stack, mem, 1234)
	}
}
