// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCxlending

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// OrderSide represents the side of a lending order (borrow/lend)
type OrderSide uint8

const (
	// Borrow represents a borrow order
	Borrow OrderSide = iota
	// Lend represents a lend order
	Lend
)

// OrderType represents the type of a lending order
type OrderType uint8

const (
	// LimitOrder represents a limit order
	LimitOrder OrderType = iota
	// MarketOrder represents a market order
	MarketOrder
)

// OrderStatus represents the status of a lending order
type OrderStatus uint8

const (
	// OrderStatusNew represents a new order
	OrderStatusNew OrderStatus = iota
	// OrderStatusPartialFilled represents a partially filled order
	OrderStatusPartialFilled
	// OrderStatusFilled represents a filled order
	OrderStatusFilled
	// OrderStatusCancelled represents a cancelled order
	OrderStatusCancelled
	// OrderStatusRejected represents a rejected order
	OrderStatusRejected
)

// LoanStatus represents the status of a loan
type LoanStatus uint8

const (
	// LoanStatusActive represents an active loan
	LoanStatusActive LoanStatus = iota
	// LoanStatusRepaid represents a repaid loan
	LoanStatusRepaid
	// LoanStatusLiquidated represents a liquidated loan
	LoanStatusLiquidated
	// LoanStatusDefaulted represents a defaulted loan
	LoanStatusDefaulted
)

// LendingOrder represents a lending order
type LendingOrder struct {
	ID               common.Hash    `json:"id"`
	UserAddress      common.Address `json:"userAddress"`
	RelayerAddress   common.Address `json:"relayerAddress"`
	LendingToken     common.Address `json:"lendingToken"`
	CollateralToken  common.Address `json:"collateralToken"`
	Side             OrderSide      `json:"side"`
	Type             OrderType      `json:"type"`
	InterestRate     *big.Int       `json:"interestRate"` // Annual rate in basis points
	Term             uint64         `json:"term"`         // Duration in seconds
	Quantity         *big.Int       `json:"quantity"`
	FilledQuantity   *big.Int       `json:"filledQuantity"`
	Status           OrderStatus    `json:"status"`
	Nonce            uint64         `json:"nonce"`
	Timestamp        uint64         `json:"timestamp"`
	Signature        []byte         `json:"signature"`
}

// NewLendingOrder creates a new lending order
func NewLendingOrder(
	userAddress common.Address,
	lendingToken, collateralToken common.Address,
	side OrderSide,
	orderType OrderType,
	interestRate *big.Int,
	term uint64,
	quantity *big.Int,
	nonce uint64,
	relayerAddress common.Address,
) *LendingOrder {
	order := &LendingOrder{
		UserAddress:     userAddress,
		RelayerAddress:  relayerAddress,
		LendingToken:    lendingToken,
		CollateralToken: collateralToken,
		Side:            side,
		Type:            orderType,
		InterestRate:    new(big.Int).Set(interestRate),
		Term:            term,
		Quantity:        new(big.Int).Set(quantity),
		FilledQuantity:  big.NewInt(0),
		Status:          OrderStatusNew,
		Nonce:           nonce,
	}
	order.ID = order.ComputeHash()
	return order
}

// ComputeHash computes the hash of the lending order
func (o *LendingOrder) ComputeHash() common.Hash {
	data := append(o.UserAddress.Bytes(), o.LendingToken.Bytes()...)
	data = append(data, o.CollateralToken.Bytes()...)
	data = append(data, byte(o.Side))
	data = append(data, byte(o.Type))
	data = append(data, common.BigToHash(o.InterestRate).Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(o.Term))).Bytes()...)
	data = append(data, common.BigToHash(o.Quantity).Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(o.Nonce))).Bytes()...)
	data = append(data, o.RelayerAddress.Bytes()...)
	return crypto.Keccak256Hash(data)
}

// Sign signs the lending order with the given private key
func (o *LendingOrder) Sign(privateKey *ecdsa.PrivateKey) error {
	hash := o.ComputeHash()
	sig, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return err
	}
	o.Signature = sig
	return nil
}

// VerifySignature verifies the lending order signature
func (o *LendingOrder) VerifySignature() bool {
	if len(o.Signature) != 65 {
		return false
	}
	hash := o.ComputeHash()
	pubKey, err := crypto.SigToPub(hash.Bytes(), o.Signature)
	if err != nil {
		return false
	}
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	return recoveredAddr == o.UserAddress
}

// RemainingQuantity returns the remaining quantity to be filled
func (o *LendingOrder) RemainingQuantity() *big.Int {
	return new(big.Int).Sub(o.Quantity, o.FilledQuantity)
}

// IsFilled returns whether the order is fully filled
func (o *LendingOrder) IsFilled() bool {
	return o.FilledQuantity.Cmp(o.Quantity) >= 0
}

// Clone creates a copy of the lending order
func (o *LendingOrder) Clone() *LendingOrder {
	return &LendingOrder{
		ID:              o.ID,
		UserAddress:     o.UserAddress,
		RelayerAddress:  o.RelayerAddress,
		LendingToken:    o.LendingToken,
		CollateralToken: o.CollateralToken,
		Side:            o.Side,
		Type:            o.Type,
		InterestRate:    new(big.Int).Set(o.InterestRate),
		Term:            o.Term,
		Quantity:        new(big.Int).Set(o.Quantity),
		FilledQuantity:  new(big.Int).Set(o.FilledQuantity),
		Status:          o.Status,
		Nonce:           o.Nonce,
		Timestamp:       o.Timestamp,
		Signature:       append([]byte{}, o.Signature...),
	}
}

// PairKey returns the lending pair key
func (o *LendingOrder) PairKey() common.Hash {
	return GetLendingPairKey(o.LendingToken, o.CollateralToken, o.Term)
}

// GetLendingPairKey returns the lending pair key
func GetLendingPairKey(lendingToken, collateralToken common.Address, term uint64) common.Hash {
	data := append(lendingToken.Bytes(), collateralToken.Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(term))).Bytes()...)
	return crypto.Keccak256Hash(data)
}

// Loan represents an active loan
type Loan struct {
	ID               common.Hash    `json:"id"`
	BorrowerAddress  common.Address `json:"borrowerAddress"`
	LenderAddress    common.Address `json:"lenderAddress"`
	LendingToken     common.Address `json:"lendingToken"`
	CollateralToken  common.Address `json:"collateralToken"`
	Principal        *big.Int       `json:"principal"`
	CollateralAmount *big.Int       `json:"collateralAmount"`
	InterestRate     *big.Int       `json:"interestRate"`
	Term             uint64         `json:"term"`
	StartTime        uint64         `json:"startTime"`
	ExpiryTime       uint64         `json:"expiryTime"`
	Status           LoanStatus     `json:"status"`
	LiquidationPrice *big.Int       `json:"liquidationPrice"`
}

// NewLoan creates a new loan
func NewLoan(
	borrower, lender common.Address,
	lendingToken, collateralToken common.Address,
	principal, collateral *big.Int,
	interestRate *big.Int,
	term, startTime uint64,
) *Loan {
	loan := &Loan{
		BorrowerAddress:  borrower,
		LenderAddress:    lender,
		LendingToken:     lendingToken,
		CollateralToken:  collateralToken,
		Principal:        new(big.Int).Set(principal),
		CollateralAmount: new(big.Int).Set(collateral),
		InterestRate:     new(big.Int).Set(interestRate),
		Term:             term,
		StartTime:        startTime,
		ExpiryTime:       startTime + term,
		Status:           LoanStatusActive,
	}
	loan.ID = loan.ComputeHash()
	return loan
}

// ComputeHash computes the loan hash
func (l *Loan) ComputeHash() common.Hash {
	data := append(l.BorrowerAddress.Bytes(), l.LenderAddress.Bytes()...)
	data = append(data, l.LendingToken.Bytes()...)
	data = append(data, l.CollateralToken.Bytes()...)
	data = append(data, common.BigToHash(l.Principal).Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(l.StartTime))).Bytes()...)
	return crypto.Keccak256Hash(data)
}

// IsExpired returns whether the loan has expired
func (l *Loan) IsExpired(currentTime uint64) bool {
	return currentTime >= l.ExpiryTime
}

// GetCollateralRatio returns the current collateral ratio
func (l *Loan) GetCollateralRatio(collateralPrice *big.Int) *big.Int {
	// Ratio = (CollateralAmount * CollateralPrice) / Principal * 100
	collateralValue := new(big.Int).Mul(l.CollateralAmount, collateralPrice)
	ratio := new(big.Int).Mul(collateralValue, big.NewInt(100))
	ratio = ratio.Div(ratio, l.Principal)
	return ratio
}
