// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCxlending

import (
	"context"
	"errors"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/XDCxlending/lendingstate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// ErrInsufficientLendingBalance is returned when lending balance is insufficient
	ErrInsufficientLendingBalance = errors.New("insufficient lending balance")
	// ErrOrderAlreadyExists is returned when order already exists
	ErrOrderAlreadyExists = errors.New("order already exists")
	// ErrOrderCannotBeCancelled is returned when order cannot be cancelled
	ErrOrderCannotBeCancelled = errors.New("order cannot be cancelled")
	// ErrInvalidSignature is returned when signature is invalid
	ErrInvalidSignature = errors.New("invalid signature")
)

// OrderProcessor handles lending order processing
type OrderProcessor struct {
	lending *XDCxLending
	lock    sync.Mutex
}

// NewOrderProcessor creates a new order processor
func NewOrderProcessor(lending *XDCxLending) *OrderProcessor {
	return &OrderProcessor{
		lending: lending,
	}
}

// Process processes a lending order and returns matched trades
func (op *OrderProcessor) Process(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, order *LendingOrder) ([]*LendingTrade, error) {
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

	// Check balance and collateral
	if err := op.checkBalanceAndCollateral(statedb, order); err != nil {
		return nil, err
	}

	// Match order
	trades, err := op.matchOrder(lendingState, order)
	if err != nil {
		return nil, err
	}

	// If order is not fully filled, add to order book
	if !order.IsFilled() && order.Type == LimitOrder {
		if err := op.addToOrderBook(lendingState, order); err != nil {
			return nil, err
		}
	}

	log.Debug("Lending order processed", "orderID", order.ID.Hex(), "trades", len(trades))
	return trades, nil
}

// Cancel cancels an existing lending order
func (op *OrderProcessor) Cancel(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, orderID common.Hash) error {
	op.lock.Lock()
	defer op.lock.Unlock()

	// Get order from state
	order := lendingState.GetLendingOrder(orderID)
	if order == nil {
		return ErrLoanNotFound
	}

	// Check if order can be cancelled
	if order.Status == OrderStatusFilled || order.Status == OrderStatusCancelled {
		return ErrOrderCannotBeCancelled
	}

	// Remove from order book
	if err := op.removeFromOrderBook(lendingState, order); err != nil {
		return err
	}

	// Update order status
	order.Status = OrderStatusCancelled
	lendingState.UpdateLendingOrder(order)

	log.Debug("Lending order cancelled", "orderID", orderID.Hex())
	return nil
}

// validateOrder validates a lending order
func (op *OrderProcessor) validateOrder(order *LendingOrder) error {
	if order.InterestRate == nil || order.InterestRate.Sign() < 0 {
		return ErrInvalidInterestRate
	}
	if order.Term == 0 {
		return ErrInvalidTerm
	}
	if order.Quantity == nil || order.Quantity.Sign() <= 0 {
		return errors.New("invalid quantity")
	}
	return nil
}

// checkBalanceAndCollateral checks if user has sufficient balance and collateral
func (op *OrderProcessor) checkBalanceAndCollateral(statedb *state.StateDB, order *LendingOrder) error {
	if order.Side == Borrow {
		// Borrowers need collateral
		collateralNeeded := op.calculateRequiredCollateral(order)
		balance := op.getTokenBalance(statedb, order.CollateralToken, order.UserAddress)
		if balance.Cmp(collateralNeeded) < 0 {
			return ErrInsufficientCollateral
		}
	} else {
		// Lenders need the lending token
		balance := op.getTokenBalance(statedb, order.LendingToken, order.UserAddress)
		if balance.Cmp(order.Quantity) < 0 {
			return ErrInsufficientLendingBalance
		}
	}
	return nil
}

// calculateRequiredCollateral calculates the required collateral for a borrow order
func (op *OrderProcessor) calculateRequiredCollateral(order *LendingOrder) *big.Int {
	// Required collateral = quantity * minCollateralRatio / 100
	minCollateral := op.lending.GetConfig().MinCollateral
	required := new(big.Int).Mul(order.Quantity, minCollateral)
	required = required.Div(required, big.NewInt(100))
	return required
}

// getTokenBalance gets the token balance for a user
func (op *OrderProcessor) getTokenBalance(statedb *state.StateDB, token, user common.Address) *big.Int {
	// This would call the ERC20 balanceOf function
	// Simplified implementation
	return statedb.GetBalance(user).ToBig()
}

// matchOrder matches a lending order against the order book
func (op *OrderProcessor) matchOrder(lendingState *lendingstate.LendingStateDB, order *LendingOrder) ([]*LendingTrade, error) {
	var trades []*LendingTrade

	pairKey := order.PairKey()
	orderBook := lendingState.GetLendingOrderBook(pairKey)
	if orderBook == nil {
		return trades, nil
	}

	// Match based on order side
	if order.Side == Borrow {
		trades = op.matchBorrowOrder(lendingState, orderBook, order)
	} else {
		trades = op.matchLendOrder(lendingState, orderBook, order)
	}

	return trades, nil
}

// matchBorrowOrder matches a borrow order against lend orders
func (op *OrderProcessor) matchBorrowOrder(lendingState *lendingstate.LendingStateDB, orderBook *lendingstate.LendingOrderBook, borrowOrder *LendingOrder) []*LendingTrade {
	var trades []*LendingTrade

	// Match against lend orders from lowest to highest interest rate
	for !borrowOrder.IsFilled() {
		bestLend := orderBook.GetBestLend()
		if bestLend == nil || bestLend.InterestRate.Cmp(borrowOrder.InterestRate) > 0 {
			break
		}

		trade := op.executeTrade(borrowOrder, bestLend)
		trades = append(trades, trade)

		if bestLend.IsFilled() {
			orderBook.RemoveLend(bestLend.ID)
		} else {
			lendingState.UpdateLendingOrder(bestLend)
		}
	}

	return trades
}

// matchLendOrder matches a lend order against borrow orders
func (op *OrderProcessor) matchLendOrder(lendingState *lendingstate.LendingStateDB, orderBook *lendingstate.LendingOrderBook, lendOrder *LendingOrder) []*LendingTrade {
	var trades []*LendingTrade

	// Match against borrow orders from highest to lowest interest rate
	for !lendOrder.IsFilled() {
		bestBorrow := orderBook.GetBestBorrow()
		if bestBorrow == nil || bestBorrow.InterestRate.Cmp(lendOrder.InterestRate) < 0 {
			break
		}

		trade := op.executeTrade(bestBorrow, lendOrder)
		trades = append(trades, trade)

		if bestBorrow.IsFilled() {
			orderBook.RemoveBorrow(bestBorrow.ID)
		} else {
			lendingState.UpdateLendingOrder(bestBorrow)
		}
	}

	return trades
}

// executeTrade executes a lending trade between two orders
func (op *OrderProcessor) executeTrade(borrowOrder, lendOrder *LendingOrder) *LendingTrade {
	// Determine trade quantity
	borrowRemaining := borrowOrder.RemainingQuantity()
	lendRemaining := lendOrder.RemainingQuantity()

	var tradeQuantity *big.Int
	if borrowRemaining.Cmp(lendRemaining) < 0 {
		tradeQuantity = borrowRemaining
	} else {
		tradeQuantity = lendRemaining
	}

	// Use market maker's interest rate
	tradeInterestRate := lendOrder.InterestRate

	// Update filled quantities
	borrowOrder.FilledQuantity = new(big.Int).Add(borrowOrder.FilledQuantity, tradeQuantity)
	lendOrder.FilledQuantity = new(big.Int).Add(lendOrder.FilledQuantity, tradeQuantity)

	// Update order statuses
	if borrowOrder.IsFilled() {
		borrowOrder.Status = OrderStatusFilled
	} else {
		borrowOrder.Status = OrderStatusPartialFilled
	}

	if lendOrder.IsFilled() {
		lendOrder.Status = OrderStatusFilled
	} else {
		lendOrder.Status = OrderStatusPartialFilled
	}

	// Calculate collateral
	collateral := op.calculateRequiredCollateral(borrowOrder)

	// Create trade
	return NewLendingTrade(
		borrowOrder.ID,
		lendOrder.ID,
		borrowOrder.UserAddress,
		lendOrder.UserAddress,
		borrowOrder.LendingToken,
		borrowOrder.CollateralToken,
		tradeQuantity,
		collateral,
		tradeInterestRate,
		borrowOrder.Term,
	)
}

// addToOrderBook adds an order to the order book
func (op *OrderProcessor) addToOrderBook(lendingState *lendingstate.LendingStateDB, order *LendingOrder) error {
	pairKey := order.PairKey()
	orderBook := lendingState.GetOrCreateLendingOrderBook(pairKey)

	if order.Side == Borrow {
		orderBook.AddBorrow(order)
	} else {
		orderBook.AddLend(order)
	}

	lendingState.SetLendingOrder(order)
	return nil
}

// removeFromOrderBook removes an order from the order book
func (op *OrderProcessor) removeFromOrderBook(lendingState *lendingstate.LendingStateDB, order *LendingOrder) error {
	pairKey := order.PairKey()
	orderBook := lendingState.GetLendingOrderBook(pairKey)
	if orderBook == nil {
		return nil
	}

	if order.Side == Borrow {
		orderBook.RemoveBorrow(order.ID)
	} else {
		orderBook.RemoveLend(order.ID)
	}

	return nil
}
