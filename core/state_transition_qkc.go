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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// ErrDepositNotMove is returned when a block has a cross-shard deposit other
// than a move, the only deposit the source shards send for now.
var ErrDepositNotMove = errors.New("deposit is not a move")

// executeDeposit applies a QuarkChain cross-shard deposit. Only moves are sent
// for now, so the deposit runs no code, even if its recipient is a contract,
// and cannot fail: the source shard burnt its value and the cost of its gas,
// at the gas price of the sending transaction, and the deposit
//
//  1. uses its gas, XShardSendGas, out of the block gas,
//  2. mints the value to the recipient,
//  3. mints the gas cost to the coinbase in full, as there is no base fee to burn.
//
// The consensus layer forces deposits into blocks, and only a deposit no valid
// source shard could have sent is a consensus error.
func (st *stateTransition) executeDeposit(rules params.Rules) (*ExecutionResult, error) {
	msg := st.msg
	if !rules.IsQuarkChain {
		return nil, fmt.Errorf("%w: deposit tx (sender %v)", ErrTxTypeNotSupported, msg.From)
	}
	if !msg.isMove || len(msg.Data) != 0 || msg.GasLimit != params.XShardSendGas {
		return nil, fmt.Errorf("%w: sender %v, gas %d, data length %d", ErrDepositNotMove, msg.From, msg.GasLimit, len(msg.Data))
	}
	fee, overflow := new(uint256.Int).MulOverflow(uint256.NewInt(msg.GasLimit), msg.GasPrice)
	if overflow {
		return nil, fmt.Errorf("%w: deposit gas cost exceeds 256 bits (sender %v)", ErrInsufficientFunds, msg.From)
	}
	// The consensus layer only forces in the deposits that fit in the block.
	var err error
	if rules.IsAmsterdam {
		err = st.gp.ChargeGasAmsterdam(msg.GasLimit, 0, msg.GasLimit)
	} else if err = st.gp.CheckGasLegacy(msg.GasLimit); err == nil {
		err = st.gp.ChargeGasLegacy(0, msg.GasLimit)
	}
	if err != nil {
		return nil, err
	}
	// Start the transaction in the state, e.g. its block access list entry (EIP-7928).
	st.state.Prepare(rules, msg.From, st.evm.Context.Coinbase, msg.To, nil, nil)
	st.state.AddBalance(*msg.To, msg.Value, tracing.BalanceIncreaseXShardDeposit)
	st.state.AddBalance(st.evm.Context.Coinbase, fee, tracing.BalanceIncreaseRewardTransactionFee)

	return &ExecutionResult{
		UsedGas:    msg.GasLimit,
		MaxUsedGas: msg.GasLimit,
	}, nil
}
