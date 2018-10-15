// Copyright 2016 The go-ethereum Authors
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
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

type dummyContractRef struct {
	calledForEach bool
}

func (dummyContractRef) ReturnGas(*big.Int)          {}
func (dummyContractRef) Address() common.Address     { return common.Address{} }
func (dummyContractRef) Value() *big.Int             { return new(big.Int) }
func (dummyContractRef) SetCode(common.Hash, []byte) {}
func (d *dummyContractRef) ForEachStorage(callback func(key, value common.Hash) bool) {
	d.calledForEach = true
}
func (d *dummyContractRef) SubBalance(amount *big.Int) {}
func (d *dummyContractRef) AddBalance(amount *big.Int) {}
func (d *dummyContractRef) SetBalance(*big.Int)        {}
func (d *dummyContractRef) SetNonce(uint64)            {}
func (d *dummyContractRef) Balance() *big.Int          { return new(big.Int) }

type dummyStatedb struct {
}

func (dummyStatedb) CreateAccount(common.Address)                              { panic("implement me") }
func (dummyStatedb) SubBalance(common.Address, *big.Int)                       { panic("implement me") }
func (dummyStatedb) AddBalance(common.Address, *big.Int)                       { panic("implement me") }
func (dummyStatedb) GetBalance(common.Address) *big.Int                        { panic("implement me") }
func (dummyStatedb) GetNonce(common.Address) uint64                            { panic("implement me") }
func (dummyStatedb) SetNonce(common.Address, uint64)                           { panic("implement me") }
func (dummyStatedb) GetCodeHash(common.Address) common.Hash                    { panic("implement me") }
func (dummyStatedb) GetCode(common.Address) []byte                             { panic("implement me") }
func (dummyStatedb) SetCode(common.Address, []byte)                            { panic("implement me") }
func (dummyStatedb) GetCodeSize(common.Address) int                            { panic("implement me") }
func (dummyStatedb) AddRefund(uint64)                                          { panic("implement me") }
func (dummyStatedb) SubRefund(uint64)                                          { panic("implement me") }
func (dummyStatedb) GetRefund() uint64                                         { return 1337 }
func (dummyStatedb) GetCommittedState(common.Address, common.Hash) common.Hash { panic("implement me") }
func (dummyStatedb) GetState(common.Address, common.Hash) common.Hash          { panic("implement me") }
func (dummyStatedb) SetState(common.Address, common.Hash, common.Hash)         { panic("implement me") }
func (dummyStatedb) Suicide(common.Address) bool                               { panic("implement me") }
func (dummyStatedb) HasSuicided(common.Address) bool                           { panic("implement me") }
func (dummyStatedb) Exist(common.Address) bool                                 { panic("implement me") }
func (dummyStatedb) Empty(common.Address) bool                                 { panic("implement me") }
func (dummyStatedb) RevertToSnapshot(int)                                      { panic("implement me") }
func (dummyStatedb) Snapshot() int                                             { panic("implement me") }
func (dummyStatedb) AddLog(*types.Log)                                         { panic("implement me") }
func (dummyStatedb) AddPreimage(common.Hash, []byte)                           { panic("implement me") }
func (dummyStatedb) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) {
	panic("implement me")
}

func TestStoreCapture(t *testing.T) {
	var (
		env      = NewEVM(Context{}, dummyStatedb{}, params.TestChainConfig, Config{})
		logger   = NewStructLogger(nil)
		mem      = NewMemory()
		stack    = newstack()
		contract = NewContract(&dummyContractRef{}, &dummyContractRef{}, new(big.Int), 0)
	)
	stack.push(big.NewInt(1))
	stack.push(big.NewInt(0))

	var index common.Hash

	logger.CaptureState(env, 0, SSTORE, 0, 0, mem, stack, contract, 0, nil)
	if len(logger.changedValues[contract.Address()]) == 0 {
		t.Fatalf("expected exactly 1 changed value on address %x, got %d", contract.Address(), len(logger.changedValues[contract.Address()]))
	}
	exp := common.BigToHash(big.NewInt(1))
	if logger.changedValues[contract.Address()][index] != exp {
		t.Errorf("expected %x, got %x", exp, logger.changedValues[contract.Address()][index])
	}
}
