// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCx

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/XDCx/tradingstate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// ErrInsufficientBalance is returned when balance is insufficient
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrOrderAlreadyExists is returned when order already exists
	ErrOrderAlreadyExists = errors.New("order already exists")
	// ErrOrderCannotBeCancelled is returned when order cannot be cancelled
	ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled")
)

// OrderProcessor handles order processing
type OrderProcessor struct {
	xdcx *XDCx
	lock sync.Mutex
}

// NewOrderProcessor creates a new order processor
func NewOrderProcessor(xdcx *XDCx) *OrderProcessor {
	return &OrderProcessor{
		xdcx: xdcx,
	}
}

// Process processes an order and returns matched trades
func (op *OrderProcessor) Process(ctx context.Context, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, order *Order) ([]*Trade, error) {
	op.lock.Lock()
	defer op.lock.Unlock()

	// Validate order
	if err := op.validateOrder(order); err != nil {
		return nil, err
	}

	// Verify signature
	if !order.VerifySignature() {
		return nil, ErrInvalidSignature
	}

	// Check balance
	if err := op.checkBalance(statedb, order); err != nil {
		return nil, err
	}

	// Match order
	trades, err := op.matchOrder(tradingState, order)
	if err != nil {
		return nil, err
	}

	// If order is not fully filled, add to order book
	if !order.IsFilled() && order.Type == Limit {
		if err := op.addToOrderBook(tradingState, order); err != nil {
			return nil, err
		}
	}

	log.Debug("Order processed", "orderID", order.ID.Hex(), "trades", len(trades))
	return trades, nil
}

// Cancel cancels an existing order
func (op *OrderProcessor) Cancel(ctx context.Context, statedb *state.StateDB, tradingState *tradingstate.TradingStateDB, orderID common.Hash) error {
	op.lock.Lock()
	defer op.lock.Unlock()

	// Get order from state
	order := tradingState.GetOrder(orderID)
	if order == nil {
		return ErrOrderNotFound
	}

	// Check if order can be cancelled
	if order.Status == OrderStatusFilled || order.Status == OrderStatusCancelled {
		return ErrOrderCannotBeCancelled
	}

	// Remove from order book
	if err := op.removeFromOrderBook(tradingState, order); err != nil {
		return err
	}

	// Update order status
	order.Status = OrderStatusCancelled
	tradingState.UpdateOrder(order)

	log.Debug("Order cancelled", "orderID", orderID.Hex())
	return nil
}

// validateOrder validates an order
func (op *OrderProcessor) validateOrder(order *Order) error {
	if order.Price == nil || order.Price.Sign() <= 0 {
		return ErrInvalidPrice
	}
	if order.Quantity == nil || order.Quantity.Sign() <= 0 {
		return ErrInvalidQuantity
	}
	return nil
}

// checkBalance checks if user has sufficient balance
func (op *OrderProcessor) checkBalance(statedb *state.StateDB, order *Order) error {
	// For buy orders, check quote token balance
	// For sell orders, check base token balance
	var tokenToCheck common.Address
	var amountNeeded *big.Int

	if order.Side == Buy {
		tokenToCheck = order.QuoteToken
		// Amount needed = price * quantity / 10^18
		amountNeeded = new(big.Int).Mul(order.Price, order.Quantity)
		amountNeeded = new(big.Int).Div(amountNeeded, big.NewInt(1e18))
	} else {
		tokenToCheck = order.BaseToken
		amountNeeded = order.Quantity
	}

	// Check ERC20 balance (simplified)
	balance := op.getTokenBalance(statedb, tokenToCheck, order.UserAddress)
	if balance.Cmp(amountNeeded) < 0 {
		return ErrInsufficientBalance
	}

	return nil
}

// getTokenBalance gets the token balance for a user
func (op *OrderProcessor) getTokenBalance(statedb *state.StateDB, token, user common.Address) *big.Int {
	// This would call the ERC20 balanceOf function
	// Simplified implementation
	return statedb.GetBalance(user).ToBig()
}

// matchOrder matches an order against the order book
func (op *OrderProcessor) matchOrder(tradingState *tradingstate.TradingStateDB, order *Order) ([]*Trade, error) {
	var trades []*Trade

	pairKey := order.PairKey()
	orderBook := tradingState.GetOrderBook(pairKey)
	if orderBook == nil {
		return trades, nil
	}

	// Match based on order side
	if order.Side == Buy {
		trades = op.matchBuyOrder(tradingState, orderBook, order)
	} else {
		trades = op.matchSellOrder(tradingState, orderBook, order)
	}

	return trades, nil
}

// matchBuyOrder matches a buy order against sell orders
func (op *OrderProcessor) matchBuyOrder(tradingState *tradingstate.TradingStateDB, orderBook *tradingstate.OrderBook, buyOrder *Order) []*Trade {
	var trades []*Trade

	// Match against asks (sell orders) from lowest to highest price
	for !buyOrder.IsFilled() {
		bestAsk := orderBook.GetBestAsk()
		if bestAsk == nil || bestAsk.Price.Cmp(buyOrder.Price) > 0 {
			break
		}

		trade := op.executeTrade(buyOrder, bestAsk)
		trades = append(trades, trade)

		if bestAsk.IsFilled() {
			orderBook.RemoveAsk(bestAsk.ID)
		} else {
			tradingState.UpdateOrder(bestAsk)
		}
	}

	return trades
}

// matchSellOrder matches a sell order against buy orders
func (op *OrderProcessor) matchSellOrder(tradingState *tradingstate.TradingStateDB, orderBook *tradingstate.OrderBook, sellOrder *Order) []*Trade {
	var trades []*Trade

	// Match against bids (buy orders) from highest to lowest price
	for !sellOrder.IsFilled() {
		bestBid := orderBook.GetBestBid()
		if bestBid == nil || bestBid.Price.Cmp(sellOrder.Price) < 0 {
			break
		}

		trade := op.executeTrade(bestBid, sellOrder)
		trades = append(trades, trade)

		if bestBid.IsFilled() {
			orderBook.RemoveBid(bestBid.ID)
		} else {
			tradingState.UpdateOrder(bestBid)
		}
	}

	return trades
}

// executeTrade executes a trade between two orders
func (op *OrderProcessor) executeTrade(buyOrder, sellOrder *Order) *Trade {
	// Determine trade quantity
	buyRemaining := buyOrder.RemainingQuantity()
	sellRemaining := sellOrder.RemainingQuantity()

	var tradeQuantity *big.Int
	if buyRemaining.Cmp(sellRemaining) < 0 {
		tradeQuantity = buyRemaining
	} else {
		tradeQuantity = sellRemaining
	}

	// Use maker's price (the order that was in the book)
	tradePrice := sellOrder.Price

	// Update filled quantities
	buyOrder.FilledQuantity = new(big.Int).Add(buyOrder.FilledQuantity, tradeQuantity)
	sellOrder.FilledQuantity = new(big.Int).Add(sellOrder.FilledQuantity, tradeQuantity)

	// Update order statuses
	if buyOrder.IsFilled() {
		buyOrder.Status = OrderStatusFilled
	} else {
		buyOrder.Status = OrderStatusPartialFilled
	}

	if sellOrder.IsFilled() {
		sellOrder.Status = OrderStatusFilled
	} else {
		sellOrder.Status = OrderStatusPartialFilled
	}

	// Create trade
	return NewTrade(
		buyOrder.ID,
		sellOrder.ID,
		buyOrder.UserAddress,
		sellOrder.UserAddress,
		buyOrder.BaseToken,
		buyOrder.QuoteToken,
		tradePrice,
		tradeQuantity,
	)
}

// addToOrderBook adds an order to the order book
func (op *OrderProcessor) addToOrderBook(tradingState *tradingstate.TradingStateDB, order *Order) error {
	pairKey := order.PairKey()
	orderBook := tradingState.GetOrCreateOrderBook(pairKey)

	if order.Side == Buy {
		orderBook.AddBid(order)
	} else {
		orderBook.AddAsk(order)
	}

	tradingState.SetOrder(order)
	return nil
}

// removeFromOrderBook removes an order from the order book
func (op *OrderProcessor) removeFromOrderBook(tradingState *tradingstate.TradingStateDB, order *Order) error {
	pairKey := order.PairKey()
	orderBook := tradingState.GetOrderBook(pairKey)
	if orderBook == nil {
		return nil
	}

	if order.Side == Buy {
		orderBook.RemoveBid(order.ID)
	} else {
		orderBook.RemoveAsk(order.ID)
	}

	return nil
}
