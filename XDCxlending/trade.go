// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCxlending

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// LendingTradeStatus represents the status of a lending trade
type LendingTradeStatus uint8

const (
	// TradeStatusPending represents a pending trade
	TradeStatusPending LendingTradeStatus = iota
	// TradeStatusSettled represents a settled trade
	TradeStatusSettled
	// TradeStatusFailed represents a failed trade
	TradeStatusFailed
)

// LendingTrade represents a matched lending trade
type LendingTrade struct {
	hash             common.Hash
	BorrowOrderID    common.Hash    `json:"borrowOrderId"`
	LendOrderID      common.Hash    `json:"lendOrderId"`
	Borrower         common.Address `json:"borrower"`
	Lender           common.Address `json:"lender"`
	LendingToken     common.Address `json:"lendingToken"`
	CollateralToken  common.Address `json:"collateralToken"`
	Principal        *big.Int       `json:"principal"`
	CollateralAmount *big.Int       `json:"collateralAmount"`
	InterestRate     *big.Int       `json:"interestRate"`
	Term             uint64         `json:"term"`
	BorrowFee        *big.Int       `json:"borrowFee"`
	LendFee          *big.Int       `json:"lendFee"`
	Timestamp        uint64         `json:"timestamp"`
	Status           LendingTradeStatus `json:"status"`
	BlockNumber      uint64         `json:"blockNumber"`
	TxHash           common.Hash    `json:"txHash"`
	LoanID           common.Hash    `json:"loanId"`
}

// NewLendingTrade creates a new lending trade
func NewLendingTrade(
	borrowOrderID, lendOrderID common.Hash,
	borrower, lender common.Address,
	lendingToken, collateralToken common.Address,
	principal, collateral *big.Int,
	interestRate *big.Int,
	term uint64,
) *LendingTrade {
	trade := &LendingTrade{
		BorrowOrderID:    borrowOrderID,
		LendOrderID:      lendOrderID,
		Borrower:         borrower,
		Lender:           lender,
		LendingToken:     lendingToken,
		CollateralToken:  collateralToken,
		Principal:        new(big.Int).Set(principal),
		CollateralAmount: new(big.Int).Set(collateral),
		InterestRate:     new(big.Int).Set(interestRate),
		Term:             term,
		BorrowFee:        big.NewInt(0),
		LendFee:          big.NewInt(0),
		Status:           TradeStatusPending,
	}
	trade.hash = trade.ComputeHash()
	return trade
}

// ComputeHash computes the hash of the lending trade
func (t *LendingTrade) ComputeHash() common.Hash {
	data := append(t.BorrowOrderID.Bytes(), t.LendOrderID.Bytes()...)
	data = append(data, t.Borrower.Bytes()...)
	data = append(data, t.Lender.Bytes()...)
	data = append(data, t.LendingToken.Bytes()...)
	data = append(data, t.CollateralToken.Bytes()...)
	data = append(data, common.BigToHash(t.Principal).Bytes()...)
	data = append(data, common.BigToHash(t.CollateralAmount).Bytes()...)
	data = append(data, common.BigToHash(t.InterestRate).Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(t.Term))).Bytes()...)
	return crypto.Keccak256Hash(data)
}

// Hash returns the trade hash
func (t *LendingTrade) Hash() common.Hash {
	if t.hash == (common.Hash{}) {
		t.hash = t.ComputeHash()
	}
	return t.hash
}

// SetFees sets the borrower and lender fees
func (t *LendingTrade) SetFees(borrowFee, lendFee *big.Int) {
	if borrowFee != nil {
		t.BorrowFee = new(big.Int).Set(borrowFee)
	}
	if lendFee != nil {
		t.LendFee = new(big.Int).Set(lendFee)
	}
}

// SetSettled marks the trade as settled
func (t *LendingTrade) SetSettled(blockNumber uint64, txHash common.Hash, loanID common.Hash) {
	t.Status = TradeStatusSettled
	t.BlockNumber = blockNumber
	t.TxHash = txHash
	t.LoanID = loanID
}

// SetFailed marks the trade as failed
func (t *LendingTrade) SetFailed() {
	t.Status = TradeStatusFailed
}

// PairKey returns the lending pair key
func (t *LendingTrade) PairKey() common.Hash {
	return GetLendingPairKey(t.LendingToken, t.CollateralToken, t.Term)
}

// CalculateInterest calculates the interest for the loan
func (t *LendingTrade) CalculateInterest() *big.Int {
	// Simple interest: Principal * Rate * Time / (365 * 100)
	interest := new(big.Int).Mul(t.Principal, t.InterestRate)
	interest = interest.Mul(interest, big.NewInt(int64(t.Term)))
	interest = interest.Div(interest, big.NewInt(365*24*3600*100))
	return interest
}

// TotalRepayment calculates the total repayment amount
func (t *LendingTrade) TotalRepayment() *big.Int {
	interest := t.CalculateInterest()
	return new(big.Int).Add(t.Principal, interest)
}

// Clone creates a copy of the lending trade
func (t *LendingTrade) Clone() *LendingTrade {
	return &LendingTrade{
		hash:             t.hash,
		BorrowOrderID:    t.BorrowOrderID,
		LendOrderID:      t.LendOrderID,
		Borrower:         t.Borrower,
		Lender:           t.Lender,
		LendingToken:     t.LendingToken,
		CollateralToken:  t.CollateralToken,
		Principal:        new(big.Int).Set(t.Principal),
		CollateralAmount: new(big.Int).Set(t.CollateralAmount),
		InterestRate:     new(big.Int).Set(t.InterestRate),
		Term:             t.Term,
		BorrowFee:        new(big.Int).Set(t.BorrowFee),
		LendFee:          new(big.Int).Set(t.LendFee),
		Timestamp:        t.Timestamp,
		Status:           t.Status,
		BlockNumber:      t.BlockNumber,
		TxHash:           t.TxHash,
		LoanID:           t.LoanID,
	}
}
