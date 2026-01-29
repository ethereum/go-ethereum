// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package core

import (
	"github.com/ethereum/go-ethereum/core/types"
)

// OrderTxPreEvent is posted when an order transaction enters the transaction pool.
type OrderTxPreEvent struct {
	Tx *types.OrderTransaction
}

// LendingTxPreEvent is posted when a lending transaction enters the transaction pool.
type LendingTxPreEvent struct {
	Tx *types.LendingTransaction
}

// NewOrderTxsEvent is posted when new order transactions are processed.
type NewOrderTxsEvent struct {
	Txs types.OrderTransactions
}

// NewLendingTxsEvent is posted when new lending transactions are processed.
type NewLendingTxsEvent struct {
	Txs types.LendingTransactions
}

// EpochSwitchEvent is posted when an epoch switch occurs.
type EpochSwitchEvent struct {
	Number      uint64
	Epoch       uint64
	Masternodes []types.Address
}

// MasternodeUpdateEvent is posted when the masternode list is updated.
type MasternodeUpdateEvent struct {
	Epoch       uint64
	Masternodes []types.Address
}

// PenaltyEvent is posted when a validator is penalized.
type PenaltyEvent struct {
	Address types.Address
	Epoch   uint64
	Reason  string
}

// RewardEvent is posted when rewards are distributed.
type RewardEvent struct {
	Block        uint64
	Signer       types.Address
	SignerReward uint64 // in wei
	VoterRewards map[types.Address]uint64
}

// VoteEvent is posted when a vote is received.
type VoteEvent struct {
	Vote *types.Vote
}

// TimeoutEvent is posted when a timeout is received.
type TimeoutEvent struct {
	Timeout *types.Timeout
}

// SyncInfoEvent is posted when sync info is received.
type SyncInfoEvent struct {
	SyncInfo *types.SyncInfo
}

// QuorumCertEvent is posted when a quorum certificate is formed.
type QuorumCertEvent struct {
	QC *types.QuorumCert
}

// TimeoutCertEvent is posted when a timeout certificate is formed.
type TimeoutCertEvent struct {
	TC *types.TimeoutCert
}

// XDCxTradeEvent is posted when a trade is executed on XDCx.
type XDCxTradeEvent struct {
	TakerOrderHash common.Hash
	MakerOrderHash common.Hash
	Amount         *big.Int
	Price          *big.Int
	BlockNumber    uint64
}

// XDCxOrderEvent is posted when an order status changes.
type XDCxOrderEvent struct {
	OrderHash   common.Hash
	Status      string
	BlockNumber uint64
}

// LendingTradeEvent is posted when a lending trade is executed.
type LendingTradeEvent struct {
	LendingId      uint64
	BorrowAmount   *big.Int
	CollateralAmount *big.Int
	Term           uint64
	Interest       uint64
	BlockNumber    uint64
}

// Import missing types for compilation
import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Address alias for common.Address in events
type Address = common.Address
