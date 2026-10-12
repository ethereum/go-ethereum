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

package logger

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

func TestAccessListTracerSetCodeFrom(t *testing.T) {
	source := common.HexToAddress("0x1111111111111111111111111111111111111111")
	tracer := NewAccessListTracer(nil, nil)
	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	evm := vm.NewEVM(vm.BlockContext{BlockNumber: new(big.Int), Random: new(common.Hash)}, statedb, params.AllDevChainProtocolChanges, vm.Config{Tracer: tracer.Hooks()})
	contract := vm.NewContract(common.Address{}, common.Address{}, new(uint256.Int), vm.NewGasBudget(100000, 0), nil)
	contract.Code = append(append([]byte{byte(vm.PUSH20)}, source.Bytes()...), byte(vm.SETCODEFROM), byte(vm.STOP))
	_, err := evm.Run(contract, nil, false)
	if got := tracer.AccessList(); err != nil || len(got) != 1 || got[0].Address != source {
		t.Fatalf("execution error %v, access list %+v, want only %x", err, got, source)
	}
}

func TestNewAccessListTracerExcludedAddress(t *testing.T) {
	excluded := common.HexToAddress("0x2222222222222222222222222222222222222222")
	slot := common.HexToHash("0x01")
	prelude := types.AccessList{{
		Address:     excluded,
		StorageKeys: []common.Hash{slot},
	}}
	excl := map[common.Address]struct{}{excluded: {}}
	tracer := NewAccessListTracer(prelude, excl)
	got := tracer.AccessList()
	if len(got) != 0 {
		t.Fatalf("excluded prelude address must not contribute tuples, got %+v", got)
	}
}
