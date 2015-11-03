// Copyright 2014 The go-ethereum Authors
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

package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
)

/*
The State Transitioning Model

A state transition is a change made when a transaction is applied to the current world state
The state transitioning model does all all the necessary work to work out a valid new state root.

1) Nonce handling
2) Pre pay gas
3) Create a new state object if the recipient is \0*32
4) Value transfer
== If contract creation ==
  4a) Attempt to run transaction data
  4b) If valid, use result as code for the new state object
== end ==
5) Run Script section
6) Derive new state root
*/
type StateTransition struct {
	gp            *GasPool
	msg           Message
	gas, gasPrice *big.Int
	initialGas    *big.Int
	value         *big.Int
	data          []byte
	state         vm.Database

	env vm.Environment
}

// Message represents a message sent to a contract.
type Message interface {
	From() (common.Address, error)
	To() *common.Address

	GasPrice() *big.Int
	Gas() *big.Int
	Value() *big.Int

	Nonce() uint64
	Data() []byte
}

func MessageCreatesContract(msg Message) bool {
	return msg.To() == nil
}

// IntrinsicGas computes the 'intrisic gas' for a message
// with the given data.
func IntrinsicGas(data []byte) *big.Int {
	igas := new(big.Int).Set(params.TxGas)
	if len(data) > 0 {
		var nz int64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		m := big.NewInt(nz)
		m.Mul(m, params.TxDataNonZeroGas)
		igas.Add(igas, m)
		m.SetInt64(int64(len(data)) - nz)
		m.Mul(m, params.TxDataZeroGas)
		igas.Add(igas, m)
	}
	return igas
}

func ApplyMessage(env vm.Environment, msg Message, gp *GasPool) ([]byte, *big.Int, error) {
	var st = StateTransition{
		gp:         gp,
		env:        env,
		msg:        msg,
		gas:        new(big.Int),
		gasPrice:   msg.GasPrice(),
		initialGas: new(big.Int),
		value:      msg.Value(),
		data:       msg.Data(),
		state:      env.Db(),
	}
	return st.transitionDb()
}

func (self *StateTransition) from() (vm.Account, error) {
	f, err := self.msg.From()
	if err != nil {
		return nil, err
	}
	if !self.state.Exist(f) {
		return self.state.CreateAccount(f), nil
	}
	return self.state.GetAccount(f), nil
}
func (self *StateTransition) to() vm.Account {
	if self.msg == nil {
		return nil
	}
	to := self.msg.To()
	if to == nil {
		return nil // contract creation
	}

	if !self.state.Exist(*to) {
		return self.state.CreateAccount(*to)
	}
	return self.state.GetAccount(*to)
}

func (self *StateTransition) useGas(amount *big.Int) error {
	if self.gas.Cmp(amount) < 0 {
		return vm.OutOfGasError
	}
	self.gas.Sub(self.gas, amount)

	return nil
}

func (self *StateTransition) addGas(amount *big.Int) {
	self.gas.Add(self.gas, amount)
}

func (self *StateTransition) buyGas() error {
	mgas := self.msg.Gas()
	mgval := new(big.Int).Mul(mgas, self.gasPrice)

	sender, err := self.from()
	if err != nil {
		return err
	}
	if sender.Balance().Cmp(mgval) < 0 {
		return fmt.Errorf("insufficient ETH for gas (%x). Req %v, has %v", sender.Address().Bytes()[:4], mgval, sender.Balance())
	}
	if err = self.gp.SubGas(mgas); err != nil {
		return err
	}
	self.addGas(mgas)
	self.initialGas.Set(mgas)
	sender.SubBalance(mgval)
	return nil
}

func (self *StateTransition) preCheck() (err error) {
	msg := self.msg
	sender, err := self.from()
	if err != nil {
		return err
	}

	// Make sure this transaction's nonce is correct
	//if sender.Nonce() != msg.Nonce() {
	if n := self.state.GetNonce(sender.Address()); n != msg.Nonce() {
		return NonceError(msg.Nonce(), n)
	}

	// Pre-pay gas
	if err = self.buyGas(); err != nil {
		if IsGasLimitErr(err) {
			return err
		}
		return InvalidTxError(err)
	}

	return nil
}

func (self *StateTransition) transitionDb() (ret []byte, usedGas *big.Int, err error) {
	if err = self.preCheck(); err != nil {
		return
	}

	msg := self.msg
	sender, _ := self.from() // err checked in preCheck

	// Pay intrinsic gas
	if err = self.useGas(IntrinsicGas(self.data)); err != nil {
		return nil, nil, InvalidTxError(err)
	}

	vmenv := self.env
	var addr common.Address
	if MessageCreatesContract(msg) {
		ret, addr, err = vmenv.Create(sender, self.data, self.gas, self.gasPrice, self.value)
		if err == nil {
			dataGas := big.NewInt(int64(len(ret)))
			dataGas.Mul(dataGas, params.CreateDataGas)
			if err := self.useGas(dataGas); err == nil {
				self.state.SetCode(addr, ret)
			} else {
				ret = nil // does not affect consensus but useful for StateTests validations
				glog.V(logger.Core).Infoln("Insufficient gas for creating code. Require", dataGas, "and have", self.gas)
			}
		}
		glog.V(logger.Core).Infoln("VM create err:", err)
	} else {
		// Increment the nonce for the next transaction
		self.state.SetNonce(sender.Address(), self.state.GetNonce(sender.Address())+1)
		ret, err = vmenv.Call(sender, self.to().Address(), self.data, self.gas, self.gasPrice, self.value)
		glog.V(logger.Core).Infoln("VM call err:", err)
	}

	if err != nil && IsValueTransferErr(err) {
		return nil, nil, InvalidTxError(err)
	}

	// We aren't interested in errors here. Errors returned by the VM are non-consensus errors and therefor shouldn't bubble up
	if err != nil {
		err = nil
	}

	if vm.Debug {
		vm.StdErrFormat(vmenv.StructLogs())
	}

	self.refundGas()
	self.state.AddBalance(self.env.Coinbase(), new(big.Int).Mul(self.gasUsed(), self.gasPrice))

	return ret, self.gasUsed(), err
}

func (self *StateTransition) refundGas() {
	// Return eth for remaining gas to the sender account,
	// exchanged at the original rate.
	sender, _ := self.from() // err already checked
	remaining := new(big.Int).Mul(self.gas, self.gasPrice)
	sender.AddBalance(remaining)

	// Apply refund counter, capped to half of the used gas.
	uhalf := remaining.Div(self.gasUsed(), common.Big2)
	refund := common.BigMin(uhalf, self.state.GetRefund())
	self.gas.Add(self.gas, refund)
	self.state.AddBalance(sender.Address(), refund.Mul(refund, self.gasPrice))

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	self.gp.AddGas(self.gas)
}

func (self *StateTransition) gasUsed() *big.Int {
	return new(big.Int).Sub(self.initialGas, self.gas)
}
