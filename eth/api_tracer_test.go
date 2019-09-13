// Copyright 2019 The go-ethereum Authors
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

package eth

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/params"
)

// dummyMsg implements core.Message to allow passing it to execution
type dummyMsg struct{}

func (m dummyMsg) From() common.Address { return common.Address{} }
func (m dummyMsg) Nonce() uint64        { return 0 }
func (m dummyMsg) CheckNonce() bool     { return false }
func (m dummyMsg) To() *common.Address  { return &common.Address{0xde, 0xad} }
func (m dummyMsg) GasPrice() *big.Int   { return new(big.Int) }
func (m dummyMsg) Gas() uint64          { return 100000 }
func (m dummyMsg) Value() *big.Int      { return new(big.Int) }
func (m dummyMsg) Data() []byte         { return []byte{} }

// TestTraceTx tests some very basic stuff about the tracing
func TestTraceTx(t *testing.T) {
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(rawdb.NewMemoryDatabase()), nil)
	dest := common.Address{0xde, 0xad}
	statedb.SetCode(dest, []byte{
		byte(vm.PUSH1), 0x01, // PUSH1 1
		byte(vm.PUSH1), 0x01, // PUSH1 1
		byte(vm.PUSH1), 0x01, // PUSH1 1
		byte(vm.PUSH1), 0x01, // PUSH1 1
		byte(vm.JUMPDEST), // JUMPDEST
		byte(vm.SSTORE),
		byte(vm.LT), // LT

	})
	statedb.Commit(false)

	vmctx := vm.Context{
		CanTransfer: func(db vm.StateDB, addresses common.Address, i *big.Int) bool {
			return true
		},
		Transfer: func(db vm.StateDB, addresses common.Address, addresses2 common.Address, i *big.Int) {},
		GetHash: func(u uint64) common.Hash {
			return common.Hash{}
		},
		BlockNumber: new(big.Int),
		Time:        new(big.Int),
		GasLimit:    8000000,
		Difficulty:  new(big.Int),
		GasPrice:    new(big.Int),
	}
	tracer := vm.NewStructLogger(nil)
	vmenv := vm.NewEVM(vmctx, statedb, params.MainnetChainConfig, vm.Config{Debug: true, Tracer: tracer})
	core.ApplyMessage(vmenv, dummyMsg{}, new(core.GasPool).AddGas(1000000))
	formatted := ethapi.FormatLogs(tracer.StructLogs())

	if exp, got := 8, len(formatted); exp != got {
		t.Fatalf("trace length wrong, got %d, exp %d", got, exp)
	}
	if exp, got := uint64(3), formatted[6].GasCost; exp != got {
		t.Fatalf("gas cost wrong, got %d, exp %d", got, exp)
	}
	//for i, log := range formatted {
	//	fmt.Printf("%d : %v %v\n", i, log.Op, log.GasCost)
	//}
}
