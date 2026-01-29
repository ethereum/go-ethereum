// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCxlending

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/XDCxlending/lendingstate"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/log"
)

// Liquidator handles loan liquidation
type Liquidator struct {
	lending *XDCxLending
	lock    sync.Mutex
}

// NewLiquidator creates a new liquidator
func NewLiquidator(lending *XDCxLending) *Liquidator {
	return &Liquidator{
		lending: lending,
	}
}

// Liquidate liquidates an undercollateralized loan
func (l *Liquidator) Liquidate(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, loanID common.Hash) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	loan := lendingState.GetLoan(loanID)
	if loan == nil {
		return ErrLoanNotFound
	}

	// Check if loan is eligible for liquidation
	collateralPrice := l.getCollateralPrice(statedb, loan.CollateralToken)
	if !l.isLiquidatable(loan, collateralPrice) {
		return ErrInsufficientCollateral
	}

	// Perform liquidation
	if err := l.performLiquidation(statedb, lendingState, loan, collateralPrice); err != nil {
		return err
	}

	// Update loan status
	loan.Status = LoanStatusLiquidated
	lendingState.UpdateLoan(loan)

	log.Info("Loan liquidated", "loanID", loanID.Hex())
	return nil
}

// CheckLoansForLiquidation checks all active loans for liquidation
func (l *Liquidator) CheckLoansForLiquidation(lendingState *lendingstate.LendingStateDB) ([]common.Hash, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	liquidatableLoans := make([]common.Hash, 0)
	allLoans := lendingState.GetAllLoans()

	for _, loan := range allLoans {
		if loan.Status != LoanStatusActive {
			continue
		}

		// Check collateral ratio
		collateralPrice := big.NewInt(1e18) // Placeholder - should get from oracle
		if l.isLiquidatable(loan, collateralPrice) {
			liquidatableLoans = append(liquidatableLoans, loan.ID)
		}
	}

	return liquidatableLoans, nil
}

// isLiquidatable checks if a loan is eligible for liquidation
func (l *Liquidator) isLiquidatable(loan *Loan, collateralPrice *big.Int) bool {
	// Calculate current collateral ratio
	ratio := loan.GetCollateralRatio(collateralPrice)

	// Compare with liquidation threshold
	liquidationRate := l.lending.GetConfig().LiquidationRate
	return ratio.Cmp(liquidationRate) < 0
}

// performLiquidation performs the actual liquidation
func (l *Liquidator) performLiquidation(statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, loan *Loan, collateralPrice *big.Int) error {
	// Calculate liquidation amounts
	principal := loan.Principal
	interest := l.lending.calculateInterest(loan)
	totalDebt := new(big.Int).Add(principal, interest)

	// Calculate collateral to sell
	collateralToSell := new(big.Int).Mul(totalDebt, big.NewInt(1e18))
	collateralToSell = collateralToSell.Div(collateralToSell, collateralPrice)

	// Add liquidation penalty (5%)
	penalty := new(big.Int).Mul(collateralToSell, big.NewInt(5))
	penalty = penalty.Div(penalty, big.NewInt(100))
	totalCollateralSold := new(big.Int).Add(collateralToSell, penalty)

	// Ensure we don't sell more than available
	if totalCollateralSold.Cmp(loan.CollateralAmount) > 0 {
		totalCollateralSold = loan.CollateralAmount
	}

	// Calculate remaining collateral for borrower
	remainingCollateral := new(big.Int).Sub(loan.CollateralAmount, totalCollateralSold)

	log.Debug("Liquidation performed",
		"loanID", loan.ID.Hex(),
		"totalDebt", totalDebt,
		"collateralSold", totalCollateralSold,
		"remainingCollateral", remainingCollateral,
	)

	return nil
}

// getCollateralPrice gets the current price of the collateral token
func (l *Liquidator) getCollateralPrice(statedb *state.StateDB, collateralToken common.Address) *big.Int {
	// This should query the price oracle
	// Placeholder implementation
	return big.NewInt(1e18)
}

// LiquidationEvent represents a liquidation event
type LiquidationEvent struct {
	LoanID           common.Hash    `json:"loanId"`
	Borrower         common.Address `json:"borrower"`
	Lender           common.Address `json:"lender"`
	Liquidator       common.Address `json:"liquidator"`
	CollateralSold   *big.Int       `json:"collateralSold"`
	DebtRepaid       *big.Int       `json:"debtRepaid"`
	LiquidationBonus *big.Int       `json:"liquidationBonus"`
	BlockNumber      uint64         `json:"blockNumber"`
	TxHash           common.Hash    `json:"txHash"`
}

// RecallLoans recalls loans that have exceeded their term
func (l *Liquidator) RecallLoans(ctx context.Context, statedb *state.StateDB, lendingState *lendingstate.LendingStateDB, currentTime uint64) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	allLoans := lendingState.GetAllLoans()
	for _, loan := range allLoans {
		if loan.Status != LoanStatusActive {
			continue
		}

		if loan.IsExpired(currentTime) {
			log.Info("Recalling expired loan", "loanID", loan.ID.Hex())
			// Mark for liquidation or automatic repayment
			loan.Status = LoanStatusDefaulted
			lendingState.UpdateLoan(loan)
		}
	}

	return nil
}

// GetLiquidationPrice calculates the price at which a loan becomes liquidatable
func (l *Liquidator) GetLiquidationPrice(loan *Loan) *big.Int {
	// Liquidation price = (Principal * LiquidationRate) / (CollateralAmount * 100)
	liquidationRate := l.lending.GetConfig().LiquidationRate
	numerator := new(big.Int).Mul(loan.Principal, liquidationRate)
	denominator := new(big.Int).Mul(loan.CollateralAmount, big.NewInt(100))
	price := new(big.Int).Div(numerator, denominator)
	return price
}
