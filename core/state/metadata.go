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
	// BalanceChangeSuicideRefund is added to the recipient as indicated by a selfdestructing account.
	BalanceChangeSuicideRefund = 11
	// BalanceChangeSuicideWithdraw is deducted from a contract due to self-destruct.
	// This can happen either at the point of self-destruction, or at the end of the tx
	// if ether was sent to contract post-selfdestruct.
	BalanceChangeSuicideWithdraw = 12
	// BalanceChangeBurn accounts for:
	// - EIP-1559 burnt fees
	// - ether that is sent to a self-destructed contract within the same tx (captured at end of tx)
	// Note it doesn't account for a self-destruct which appoints same contract as recipient.
	BalanceChangeBurn = 13
	// BalanceChangeBurnRefund is refunded to an account at the end of transaction based on
	// gas usage from the estimated burn amount.
	BalanceChangeBurnRefund = 14
	BalanceChangeWithdrawal = 15
)
