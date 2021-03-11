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

package vm

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"sort"

	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

var activators = map[int]func(*JumpTable){
	3074: enable3074,
	2929: enable2929,
	2200: enable2200,
	1884: enable1884,
	1344: enable1344,
}

// EnableEIP enables the given EIP on the config.
// This operation writes in-place, and callers need to ensure that the globally
// defined jump tables are not polluted.
func EnableEIP(eipNum int, jt *JumpTable) error {
	enablerFn, ok := activators[eipNum]
	if !ok {
		return fmt.Errorf("undefined eip %d", eipNum)
	}
	enablerFn(jt)
	return nil
}

func ValidEip(eipNum int) bool {
	_, ok := activators[eipNum]
	return ok
}
func ActivateableEips() []string {
	var nums []string
	for k := range activators {
		nums = append(nums, fmt.Sprintf("%d", k))
	}
	sort.Strings(nums)
	return nums
}

// enable1884 applies EIP-1884 to the given jump table:
// - Increase cost of BALANCE to 700
// - Increase cost of EXTCODEHASH to 700
// - Increase cost of SLOAD to 800
// - Define SELFBALANCE, with cost GasFastStep (5)
func enable1884(jt *JumpTable) {
	// Gas cost changes
	jt[SLOAD].constantGas = params.SloadGasEIP1884
	jt[BALANCE].constantGas = params.BalanceGasEIP1884
	jt[EXTCODEHASH].constantGas = params.ExtcodeHashGasEIP1884

	// New opcode
	jt[SELFBALANCE] = &operation{
		execute:     opSelfBalance,
		constantGas: GasFastStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

func opSelfBalance(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	balance, _ := uint256.FromBig(interpreter.evm.StateDB.GetBalance(callContext.contract.Address()))
	callContext.stack.push(balance)
	return nil, nil
}

// enable1344 applies EIP-1344 (ChainID Opcode)
// - Adds an opcode that returns the current chain’s EIP-155 unique identifier
func enable1344(jt *JumpTable) {
	// New opcode
	jt[CHAINID] = &operation{
		execute:     opChainID,
		constantGas: GasQuickStep,
		minStack:    minStack(0, 1),
		maxStack:    maxStack(0, 1),
	}
}

// opChainID implements CHAINID opcode
func opChainID(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	chainId, _ := uint256.FromBig(interpreter.evm.chainConfig.ChainID)
	callContext.stack.push(chainId)
	return nil, nil
}

// enable2200 applies EIP-2200 (Rebalance net-metered SSTORE)
func enable2200(jt *JumpTable) {
	jt[SLOAD].constantGas = params.SloadGasEIP2200
	jt[SSTORE].dynamicGas = gasSStoreEIP2200
}

// enable2929 enables "EIP-2929: Gas cost increases for state access opcodes"
// https://eips.ethereum.org/EIPS/eip-2929
func enable2929(jt *JumpTable) {
	jt[SSTORE].dynamicGas = gasSStoreEIP2929

	jt[SLOAD].constantGas = 0
	jt[SLOAD].dynamicGas = gasSLoadEIP2929

	jt[EXTCODECOPY].constantGas = WarmStorageReadCostEIP2929
	jt[EXTCODECOPY].dynamicGas = gasExtCodeCopyEIP2929

	jt[EXTCODESIZE].constantGas = WarmStorageReadCostEIP2929
	jt[EXTCODESIZE].dynamicGas = gasEip2929AccountCheck

	jt[EXTCODEHASH].constantGas = WarmStorageReadCostEIP2929
	jt[EXTCODEHASH].dynamicGas = gasEip2929AccountCheck

	jt[BALANCE].constantGas = WarmStorageReadCostEIP2929
	jt[BALANCE].dynamicGas = gasEip2929AccountCheck

	jt[CALL].constantGas = WarmStorageReadCostEIP2929
	jt[CALL].dynamicGas = gasCallEIP2929

	jt[CALLCODE].constantGas = WarmStorageReadCostEIP2929
	jt[CALLCODE].dynamicGas = gasCallCodeEIP2929

	jt[STATICCALL].constantGas = WarmStorageReadCostEIP2929
	jt[STATICCALL].dynamicGas = gasStaticCallEIP2929

	jt[DELEGATECALL].constantGas = WarmStorageReadCostEIP2929
	jt[DELEGATECALL].dynamicGas = gasDelegateCallEIP2929

	// This was previously part of the dynamic cost, but we're using it as a constantGas
	// factor here
	jt[SELFDESTRUCT].constantGas = params.SelfdestructGasEIP150
	jt[SELFDESTRUCT].dynamicGas = gasSelfdestructEIP2929
}

func enable3074(jt *JumpTable) {
	jt[AUTHORIZE] = &operation{
		execute:     opAuthorize,
		constantGas: params.EcrecoverGas,
		minStack:    minStack(4, 1),
		maxStack:    maxStack(4, 1),
	}

	jt[AUTHORIZEDCALL] = &operation{
		execute:     opAuthorizedCall,
		constantGas: jt[CALL].constantGas,
		dynamicGas:  jt[CALL].dynamicGas,
		minStack:    minStack(7, 1),
		maxStack:    maxStack(7, 1),
		memorySize:  jt[CALL].memorySize,
		returns:     true,
	}
}

func opAuthorize(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	stack := callContext.stack
	commit, sigV, sigR, sigS := stack.pop(), stack.pop(), stack.pop(), stack.pop()

	r := sigR.ToBig()
	s := sigS.ToBig()
	v := uint8(sigV[0]) - 27

	callContext.sponsee = nil

	// tighter sig s values input homestead only apply to tx sigs
	if val, of := sigV.Uint64WithOverflow(); !of && val <= 0xff && crypto.ValidateSignatureValues(v, r, s, false) {
		input := make([]byte, 65)
		input[0] = 0x03
		copy(input[13:33], callContext.contract.Address().Bytes())
		commit.WriteToSlice(input[33:65])
		hash := crypto.Keccak256(input)

		sig := make([]byte, 65)
		sigR.WriteToSlice(sig[0:32])
		sigS.WriteToSlice(sig[32:64])
		sig[64] = v
		// v needs to be at the end for libsecp256k1
		pubKey, err := crypto.Ecrecover(hash[:], sig)
		// make sure the public key is a valid one
		// the first byte of pubkey is bitcoin heritage
		if err == nil {
			from := common.BytesToAddress(crypto.Keccak256(pubKey[1:])[12:])
			if from != interpreter.evm.Origin {
				callContext.sponsee = &from
			}
		}
	}

	if callContext.sponsee != nil {
		commit.SetBytes20(callContext.sponsee.Bytes())
	} else {
		commit.Clear()
	}
	stack.push(&commit)
	return nil, nil
}

func opAuthorizedCall(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	stack := callContext.stack
	// Pop gas. The actual gas in interpreter.evm.callGasTemp.
	// We can use this as a temporary value
	temp := stack.pop()
	gas := interpreter.evm.callGasTemp
	// Pop other call parameters.
	addr, value, inOffset, inSize, retOffset, retSize := stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop(), stack.pop()
	toAddr := common.Address(addr.Bytes20())
	// Get the arguments from the memory.
	args := callContext.memory.GetPtr(int64(inOffset.Uint64()), int64(inSize.Uint64()))

	var bigVal = big0
	//TODO: use uint256.Int instead of converting with toBig()
	// By using big0 here, we save an alloc for the most common case (non-ether-transferring contract calls),
	// but it would make more sense to extend the usage of uint256.Int
	if !value.IsZero() {
		gas += params.CallStipend
		bigVal = value.ToBig()
	}

	ret, returnGas, err := []byte(nil), gas, ErrNoAuthorizedSponsee
	if callContext.sponsee != nil {
		ret, returnGas, err = interpreter.evm.Call(callContext.contract, *callContext.sponsee, toAddr, args, gas, bigVal)
	}

	if err != nil {
		temp.Clear()
	} else {
		temp.SetOne()
	}
	stack.push(&temp)
	if err == nil || err == ErrExecutionReverted {
		callContext.memory.Set(retOffset.Uint64(), retSize.Uint64(), ret)
	}
	callContext.contract.Gas += returnGas

	return ret, nil
}
