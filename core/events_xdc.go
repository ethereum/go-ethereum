// Copyright 2023 The XDC Network Authors
// XDC-specific event types for the core package

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// OrderTxPreEvent is posted when an order transaction enters the transaction pool.
type OrderTxPreEvent struct {
	Tx *types.OrderTransaction
}

// LendingTxPreEvent is posted when a lending transaction enters the transaction pool.
type LendingTxPreEvent struct {
	Tx interface{} // *types.LendingTransaction when implemented
}

// MasternodeEvent is posted when masternode state changes.
type MasternodeEvent struct {
	Masternode common.Address
	Action     string // "join", "leave", "reward", "penalty"
}

// EpochSwitchEvent is posted when epoch switches.
type EpochSwitchEvent struct {
	EpochNumber   uint64
	OldMasternodes []common.Address
	NewMasternodes []common.Address
}

// TradeEvent is posted when a trade is executed on XDCx.
type TradeEvent struct {
	TakerOrderHash common.Hash
	MakerOrderHash common.Hash
	Pair           common.Hash
	Price          *big.Int
	Quantity       *big.Int
	TakerAddress   common.Address
	MakerAddress   common.Address
	Side           string
	BlockNumber    uint64
}

// LendingTradeEvent is posted when a lending trade is executed.
type LendingTradeEvent struct {
	LendingId        uint64
	BorrowAmount     *big.Int
	CollateralAmount *big.Int
	Term             uint64
	Interest         uint64
	BlockNumber      uint64
}

// OrderCancelledEvent is posted when an order is cancelled.
type OrderCancelledEvent struct {
	OrderHash common.Hash
	Pair      common.Hash
}

// LendingCancelledEvent is posted when a lending order is cancelled.
type LendingCancelledEvent struct {
	OrderHash common.Hash
	Token     common.Hash
}
