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
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
	jt[AUTH] = &operation{
		execute:     opAuth,
		constantGas: params.EcrecoverGas,
		minStack:    minStack(4, 1),
		maxStack:    maxStack(4, 1),
	}

	jt[AUTHCALL] = &operation{
		execute:     opAuthCall,
		constantGas: jt[CALL].constantGas,
		dynamicGas:  jt[CALL].dynamicGas,
		minStack:    minStack(7, 1),
		maxStack:    maxStack(7, 1),
		memorySize:  jt[CALL].memorySize,
		returns:     true,
	}
}

func opAuth(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	stack := callContext.stack
	commit, v, r, s := stack.pop(), stack.pop(), stack.pop(), stack.pop()

	// Zero out the current authorized account. Only update it if an address
	// is successfully recovered from the signature.
	callContext.authorized = nil

	if v.BitLen() < 8 && crypto.ValidateSignatureValues(byte(v.Uint64()), r.ToBig(), s.ToBig(), true) {
		msg := make([]byte, 65)

		// EIP-3074 messages are of the form
		// keccak256(type ++ invoker ++ commit)
		msg[0] = 0x03
		copy(msg[13:33], callContext.contract.Address().Bytes())
		commit.WriteToSlice(msg[33:65])
		hash := crypto.Keccak256(msg)

		sig := make([]byte, 65)
		r.WriteToSlice(sig[0:32])
		s.WriteToSlice(sig[32:64])
		sig[64] = byte(v.Uint64())

		pub, err := crypto.Ecrecover(hash[:], sig)

		if err == nil {
			var addr common.Address
			copy(addr[:], crypto.Keccak256(pub[1:])[12:])

			// Transaction origin is not allowed to authorize itself.
			// This is to prevent reentering contracts that expect
			// caller == origin only in the first frame of a transaction.
			if addr != interpreter.evm.Origin {
				callContext.authorized = &addr
			}
		}
	}

	// reuse commit to push the result
	temp := commit
	if callContext.authorized != nil {
		temp.SetBytes20(callContext.authorized.Bytes())
	} else {
		temp.Clear()
	}

	stack.push(&temp)
	return nil, nil
}

func opAuthCall(pc *uint64, interpreter *EVMInterpreter, callContext *callCtx) ([]byte, error) {
	// If no authorized account is set, revert.
	if callContext.authorized == nil {
		return nil, ErrNoAuthorizedAccount
	}

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
	if !value.IsZero() {
		gas += params.CallStipend
		bigVal = value.ToBig()
	}

	ret, returnGas, err := interpreter.evm.AuthCall(callContext.contract, *callContext.authorized, toAddr, args, gas, bigVal)

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
