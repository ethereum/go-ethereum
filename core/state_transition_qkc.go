// Copyright 2026-2027 The QuarkChain Authors
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

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// executeDeposit applies a QuarkChain cross-shard deposit. The source shard
// burnt its value and the cost of its gas, at the gas price of the sending
// transaction, so the deposit:
//
//  1. mints the value and gas cost to the sender,
//  2. buys its gas from the sender, out of the block gas, at its gas price,
//  3. calls the recipient with the value, as a transaction would, except that
//     the nonce of the sender is left alone,
//  4. returns the gas left to the sender, and pays the gas used to the
//     coinbase in full, as there is no base fee to burn.
//
// The consensus layer forces deposits into blocks, so a deposit that cannot
// run still has to be included: one failing to pay its intrinsic gas uses all
// of its gas, and leaves its value to the sender, as a reverted call does. Only
// a deposit no valid source shard could have sent is a consensus error.
func (st *stateTransition) executeDeposit(rules params.Rules) (*ExecutionResult, error) {
	msg := st.msg
	if !rules.IsQuarkChain {
		return nil, fmt.Errorf("%w: deposit tx (sender %v)", ErrTxTypeNotSupported, msg.From)
	}
	mint := new(uint256.Int).SetUint64(msg.GasLimit)
	_, overflow := mint.MulOverflow(mint, msg.GasPrice)
	if !overflow {
		_, overflow = mint.AddOverflow(mint, msg.Value)
	}
	if overflow {
		return nil, fmt.Errorf("%w: deposit value and gas cost exceed 256 bits (sender %v)", ErrInsufficientFunds, msg.From)
	}
	st.state.AddBalance(msg.From, mint, tracing.BalanceIncreaseXShardDeposit)

	// Reserve the block gas and buy the gas with the minted funds.
	if err := st.preCheck(rules); err != nil {
		return nil, err
	}
	intrinsicGas, err := vm.DepositIntrinsicGas(msg.Data)
	if err != nil {
		return nil, err
	}
	st.state.Prepare(rules, msg.From, st.evm.Context.Coinbase, msg.To, vm.ActivePrecompiles(rules), nil)

	var (
		ret   []byte
		vmerr error
	)
	if msg.GasLimit < intrinsicGas {
		st.initRuntimeGasBudget(rules, msg.GasLimit)
		vmerr = fmt.Errorf("%w: have %d, want %d", ErrIntrinsicGas, msg.GasLimit, intrinsicGas)
	} else {
		st.initRuntimeGasBudget(rules, intrinsicGas)
		ret, vmerr = st.executeCall(rules, msg.Value)
	}
	gasUsed, peakUsed, err := st.settleGas(rules, 0)
	if err != nil {
		return nil, err
	}
	fee := new(uint256.Int).Mul(uint256.NewInt(gasUsed), msg.GasPrice)
	st.state.AddBalance(st.evm.Context.Coinbase, fee, tracing.BalanceIncreaseRewardTransactionFee)

	return &ExecutionResult{
		UsedGas:    gasUsed,
		MaxUsedGas: peakUsed,
		Err:        vmerr,
		ReturnData: ret,
	}, nil
}
