// Copyright 2023 The go-ethereum Authors
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

package state

// BalanceChangeReason is used to indicate the reason for a balance change, useful
// for tracing and reporting.
type BalanceChangeReason byte

const (
	BalanceChangeUnspecified          BalanceChangeReason = 0
	BalanceChangeRewardMineUncle                          = 1
	BalanceChangeRewardMineBlock                          = 2
	BalanceChangeDaoRefundContract                        = 3
	BalanceChangeDaoAdjustBalance                         = 4
	BalanceChangeTransfer                                 = 5
	BalanceChangeGenesisBalance                           = 6
	BalanceChangeGasBuy                                   = 7
	BalanceChangeRewardTransactionFee                     = 8
	BalanceChangeGasRefund                                = 9
	BalanceChangeTouchAccount                             = 10
	// TODO: rename (debit, credit)
	BalanceChangeSuicideRefund   = 11
	BalanceChangeSuicideWithdraw = 12
	// TODO: add method on statedb to track burn without Add/Sub balance
	// Or: Track via OnBurn
	// 1559 burn, blob burn, withdraw to self via selfdestruct burn, receive ether after selfdestruct (at end of tx)
	BalanceChangeBurn       = 13
	BalanceChangeWithdrawal = 14
)
