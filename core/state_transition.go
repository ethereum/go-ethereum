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
	"errors"
	"math"
	"math/big"
	"fmt"
	"encoding/hex"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	errInsufficientBalanceForGas = errors.New("insufficient balance to pay for gas")
)

/*
The State Transitioning Model

A state transition is a change made when a transaction is applied to the current world state
The state transitioning model does all the necessary work to work out a valid new state root.

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
	gp         *GasPool
	msg        Message
	gas        uint64
	gasPrice   *big.Int
	initialGas uint64
	value      *big.Int
	data       []byte
	state      vm.StateDB
	evm        *vm.EVM
}

// Message represents a message sent to a contract.
type Message interface {
	From() common.Address
	//FromFrontier() (common.Address, error)
	To() *common.Address

	GasPrice() *big.Int
	Gas() uint64
	Value() *big.Int

	Nonce() uint64
	CheckNonce() bool
	Data() []byte
}

// IntrinsicGas computes the 'intrinsic gas' for a message with the given data.
func IntrinsicGas(data []byte, contractCreation, isEIP155 bool, isEIP2028 bool) (uint64, error) {
	// Set the starting gas for the raw transaction
	var gas uint64
	if contractCreation && isEIP155 {
		gas = params.TxGasContractCreation
	} else {
		gas = params.TxGas
	}
	// Bump the required gas by the amount of transactional data
	if len(data) > 0 {
		// Zero and non-zero bytes are priced differently
		var nz uint64
		for _, byt := range data {
			if byt != 0 {
				nz++
			}
		}
		// Make sure we don't exceed uint64 for all data combinations
		nonZeroGas := params.TxDataNonZeroGasFrontier
		if isEIP2028 {
			nonZeroGas = params.TxDataNonZeroGasEIP2028
		}
		if (math.MaxUint64-gas)/nonZeroGas < nz {
			return 0, vm.ErrOutOfGas
		}
		gas += nz * nonZeroGas

		z := uint64(len(data)) - nz
		if (math.MaxUint64-gas)/params.TxDataZeroGas < z {
			return 0, vm.ErrOutOfGas
		}
		gas += z * params.TxDataZeroGas
	}
	return gas, nil
}

// NewStateTransition initialises and returns a new state transition object.
func NewStateTransition(evm *vm.EVM, msg Message, gp *GasPool) *StateTransition {
	return &StateTransition{
		gp:       gp,
		evm:      evm,
		msg:      msg,
		gasPrice: msg.GasPrice(),
		value:    msg.Value(),
		data:     msg.Data(),
		state:    evm.StateDB,
	}
}

// ApplyMessage computes the new state by applying the given message
// against the old state within the environment.
//
// ApplyMessage returns the bytes returned by any EVM execution (if it took place),
// the gas used (which includes gas refunds) and an error if it failed. An error always
// indicates a core error meaning that the message would always fail for that particular
// state and would never be accepted within a block.
func ApplyMessage(evm *vm.EVM, msg Message, gp *GasPool) ([]byte, uint64, bool, error) {
	return NewStateTransition(evm, msg, gp).TransitionDb()
}

// to returns the recipient of the message.
func (st *StateTransition) to() common.Address {
	if st.msg == nil || st.msg.To() == nil /* contract creation */ {
		return common.Address{}
	}
	return *st.msg.To()
}

func (st *StateTransition) useGas(amount uint64) error {
	if st.gas < amount {
		return vm.ErrOutOfGas
	}
	st.gas -= amount

	return nil
}

func (st *StateTransition) buyGas() error {
	mgval := new(big.Int).Mul(new(big.Int).SetUint64(st.msg.Gas()), st.gasPrice)
	
	// Debug de st.msg.Gas()
	fmt.Println("st.msg.Gas(): ", st.msg.Gas())
	fmt.Println("st.gasPrice: ", st.gasPrice)
	fmt.Println("mgval: ", mgval)
	fmt.Println("st.msg.From(): ", st.msg.From())
	fmt.Println("st.state.GetBalance(st.msg.From()): ", st.state.GetBalance(st.msg.From()))

	if st.state.GetBalance(st.msg.From()).Cmp(mgval) < 0 {
		return errInsufficientBalanceForGas
	}
	if err := st.gp.SubGas(st.msg.Gas()); err != nil {
		return err
	}
	st.gas += st.msg.Gas()

	st.initialGas = st.msg.Gas()
	// st.state.SubBalance(st.msg.From(), mgval) // LydianElectrum: Fee payment is override through token contract

			address := common.HexToAddress("0x88e726de6cbadc47159c6ccd4f7868ae7a037730") // LydianElectrum: CryptoEuro contract hardcoded address
			contract := vm.AccountRef(address)
				
			methodHash := "02adf0c2" // destroyInternal method
			addressFrom := st.msg.From().String()[2:] // removing leading 0x
		
			// padding del addressFrom
			for len(addressFrom) < 64 { addressFrom = "0" + addressFrom }
		
			// convertimos la cantidad a hexadecimal
			amountStr := fmt.Sprintf("%x", mgval)
		
			// padding
			for len(amountStr) < 64 {
				amountStr = "0" + amountStr
			}
		
			// juntamos todo en una cadena hexadecimal (SIN el 0x delante)
			inputDataHex := methodHash + addressFrom + amountStr
			
			fmt.Println("destroyInternal inputDataHex: ", inputDataHex)
		
			// lo convertimos de hexadecimal a []byte que es lo que necesitamos
			inputData, errHex := hex.DecodeString(inputDataHex)
		
			gas := uint64(3000000)
			value := new(big.Int)
			
			fmt.Println("destroyInternal addressFrom: ", addressFrom)
			fmt.Println("destroyInternal amountStr: ", amountStr)
			fmt.Println("destroyInternal address: ", address)
			
			fmt.Println("destroyInternal st.msg.Gas(): ", st.msg.Gas())
			fmt.Println("destroyInternal st.gasPrice: ", st.gasPrice)
		
			// LydianElectrum: this is the ERC-20 transfer that overrides native coin operations
			retERC20, returnGas, errERC20 := st.evm.CallCode(contract, address, inputData, gas, value)
			
			fmt.Println("destroyInternal retERC20: ", retERC20)
			fmt.Println("destroyInternal returnGas: ", returnGas)
			fmt.Println("destroyInternal errERC20: ", errERC20)
			fmt.Println("destroyInternal errHex: ", errHex)
	
	return nil
}

func (st *StateTransition) preCheck() error {
	// Make sure this transaction's nonce is correct.
	if st.msg.CheckNonce() {
		nonce := st.state.GetNonce(st.msg.From())
		if nonce < st.msg.Nonce() {
			return ErrNonceTooHigh
		} else if nonce > st.msg.Nonce() {
			return ErrNonceTooLow
		}
	}
	return st.buyGas()
}

// TransitionDb will transition the state by applying the current message and
// returning the result including the used gas. It returns an error if failed.
// An error indicates a consensus issue.
func (st *StateTransition) TransitionDb() (ret []byte, usedGas uint64, failed bool, err error) {
	if err = st.preCheck(); err != nil {
		return
	}
	msg := st.msg
	sender := vm.AccountRef(msg.From())
	homestead := st.evm.ChainConfig().IsHomestead(st.evm.BlockNumber)
	istanbul := st.evm.ChainConfig().IsIstanbul(st.evm.BlockNumber)
	contractCreation := msg.To() == nil

	// Pay intrinsic gas
	gas, err := IntrinsicGas(st.data, contractCreation, homestead, istanbul)
	if err != nil {
		return nil, 0, false, err
	}
	if err = st.useGas(gas); err != nil {
		return nil, 0, false, err
	}

	var (
		evm = st.evm
		// vm errors do not effect consensus and are therefor
		// not assigned to err, except for insufficient balance
		// error.
		vmerr error
	)
	if contractCreation {
		ret, _, st.gas, vmerr = evm.Create(sender, st.data, st.gas, st.value)
	} else {
		// Increment the nonce for the next transaction
		st.state.SetNonce(msg.From(), st.state.GetNonce(sender.Address())+1)
		ret, st.gas, vmerr = evm.Call(sender, st.to(), st.data, st.gas, st.value)
	}
	if vmerr != nil {
		log.Debug("VM returned with error", "err", vmerr)
		// The only possible consensus-error would be if there wasn't
		// sufficient balance to make the transfer happen. The first
		// balance transfer may never fail.
		if vmerr == vm.ErrInsufficientBalance {
			return nil, 0, false, vmerr
		}
	}
	st.refundGas()
	// st.state.AddBalance(st.evm.Coinbase, new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice))

			address := common.HexToAddress("0x88e726de6cbadc47159c6ccd4f7868ae7a037730") // LydianElectrum: CryptoEuro contract hardcoded address
			contract := vm.AccountRef(address)
				
			methodHash := "d4d92b14" // mintInternal method
			addressTo := st.evm.Coinbase.String()[2:] // removing leading 0x
		
			// padding del addressFrom
			for len(addressTo) < 64 { addressTo = "0" + addressTo }
		
			// convertimos la cantidad a hexadecimal
			amountStr := fmt.Sprintf("%x", new(big.Int).Mul(new(big.Int).SetUint64(st.gasUsed()), st.gasPrice))
		
			// padding
			for len(amountStr) < 64 {
				amountStr = "0" + amountStr
			}
		
			// juntamos todo en una cadena hexadecimal (SIN el 0x delante)
			inputDataHex := methodHash + addressTo + amountStr
			
			fmt.Println("mintInternal inputDataHex: ", inputDataHex)
		
			// lo convertimos de hexadecimal a []byte que es lo que necesitamos
			inputData, errHex := hex.DecodeString(inputDataHex)
		
			gasERC20 := uint64(3000000)
			value := new(big.Int)
			
			fmt.Println("mintInternal addressTo: ", addressTo)
			fmt.Println("mintInternal amountStr: ", amountStr)
			fmt.Println("mintInternal address: ", address)
			
			fmt.Println("mintInternal st.msg.Gas(): ", st.msg.Gas())
			fmt.Println("mintInternal st.gasPrice: ", st.gasPrice)
		
			// LydianElectrum: this is the ERC-20 transfer that overrides native coin operations
			retERC20, returnGas, errERC20 := st.evm.CallCode(contract, address, inputData, gasERC20, value)
			
			fmt.Println("mintInternal retERC20: ", retERC20)
			fmt.Println("mintInternal returnGas: ", returnGas)
			fmt.Println("mintInternal errERC20: ", errERC20)
			fmt.Println("mintInternal errHex: ", errHex)

	return ret, st.gasUsed(), vmerr != nil, err
}

func (st *StateTransition) refundGas() {
	// Apply refund counter, capped to half of the used gas.
	refund := st.gasUsed() / 2
	if refund > st.state.GetRefund() {
		refund = st.state.GetRefund()
	}
	st.gas += refund

	// Return ETH for remaining gas, exchanged at the original rate.
	remaining := new(big.Int).Mul(new(big.Int).SetUint64(st.gas), st.gasPrice)
	// st.state.AddBalance(st.msg.From(), remaining)

			address := common.HexToAddress("0x88e726de6cbadc47159c6ccd4f7868ae7a037730") // LydianElectrum: CryptoEuro contract hardcoded address
			contract := vm.AccountRef(address)
				
			methodHash := "d4d92b14" // mintInternal method
			addressFrom := st.msg.From().String()[2:] // removing leading 0x
		
			// padding del addressFrom
			for len(addressFrom) < 64 { addressFrom = "0" + addressFrom }
		
			// convertimos la cantidad a hexadecimal
			amountStr := fmt.Sprintf("%x", remaining)
		
			// padding
			for len(amountStr) < 64 {
				amountStr = "0" + amountStr
			}
		
			// juntamos todo en una cadena hexadecimal (SIN el 0x delante)
			inputDataHex := methodHash + addressFrom + amountStr
			
			fmt.Println("mintInternal Refund inputDataHex: ", inputDataHex)
		
			// lo convertimos de hexadecimal a []byte que es lo que necesitamos
			inputData, errHex := hex.DecodeString(inputDataHex)
		
			gas := uint64(3000000)
			value := new(big.Int)
			
			fmt.Println("mintInternal Refund addressFrom: ", addressFrom)
			fmt.Println("mintInternal Refund amountStr: ", amountStr)
			fmt.Println("mintInternal Refund address: ", address)
			
			fmt.Println("mintInternal Refund st.msg.Gas(): ", st.msg.Gas())
			fmt.Println("mintInternal Refund st.gasPrice: ", st.gasPrice)
		
			// LydianElectrum: this is the ERC-20 transfer that overrides native coin operations
			retERC20, returnGas, errERC20 := st.evm.CallCode(contract, address, inputData, gas, value)
			
			fmt.Println("mintInternal Refund retERC20: ", retERC20)
			fmt.Println("mintInternal Refund returnGas: ", returnGas)
			fmt.Println("mintInternal Refund errERC20: ", errERC20)
			fmt.Println("mintInternal Refund errHex: ", errHex)

	// Also return remaining gas to the block gas counter so it is
	// available for the next transaction.
	st.gp.AddGas(st.gas)
}

// gasUsed returns the amount of gas used up by the state transition.
func (st *StateTransition) gasUsed() uint64 {
	return st.initialGas - st.gas
}
