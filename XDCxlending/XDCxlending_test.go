// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCxlending

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

func TestNewXDCxLending(t *testing.T) {
	db := memorydb.New()
	config := DefaultConfig()

	lending, err := New(config, db)
	if err != nil {
		t.Fatalf("Failed to create XDCxLending: %v", err)
	}

	if lending == nil {
		t.Fatal("XDCxLending instance is nil")
	}

	if lending.config != config {
		t.Error("Config not set correctly")
	}
}

func TestXDCxLendingStartStop(t *testing.T) {
	db := memorydb.New()
	lending, _ := New(DefaultConfig(), db)

	// Test start
	if err := lending.Start(); err != nil {
		t.Fatalf("Failed to start XDCxLending: %v", err)
	}

	if !lending.IsRunning() {
		t.Error("XDCxLending should be running after Start")
	}

	// Test stop
	if err := lending.Stop(); err != nil {
		t.Fatalf("Failed to stop XDCxLending: %v", err)
	}

	if lending.IsRunning() {
		t.Error("XDCxLending should not be running after Stop")
	}
}

func TestDefaultLendingConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DBEngine != "leveldb" {
		t.Errorf("Expected DBEngine 'leveldb', got '%s'", config.DBEngine)
	}

	if config.DefaultTerm != 86400*30 {
		t.Errorf("Expected DefaultTerm 30 days, got %d", config.DefaultTerm)
	}

	if config.MinCollateral.Cmp(big.NewInt(150)) != 0 {
		t.Errorf("Expected MinCollateral 150, got %s", config.MinCollateral.String())
	}

	if config.LiquidationRate.Cmp(big.NewInt(110)) != 0 {
		t.Errorf("Expected LiquidationRate 110, got %s", config.LiquidationRate.String())
	}
}

func TestNewLendingOrder(t *testing.T) {
	userAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	lendingToken := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	collateralToken := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	relayerAddr := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")

	interestRate := big.NewInt(500) // 5%
	term := uint64(86400 * 30)      // 30 days
	quantity := big.NewInt(1000)
	nonce := uint64(1)

	order := NewLendingOrder(
		userAddr, lendingToken, collateralToken,
		Borrow, LimitOrder,
		interestRate, term, quantity, nonce, relayerAddr,
	)

	if order.UserAddress != userAddr {
		t.Errorf("Expected user address %s, got %s", userAddr.Hex(), order.UserAddress.Hex())
	}

	if order.LendingToken != lendingToken {
		t.Errorf("Expected lending token %s, got %s", lendingToken.Hex(), order.LendingToken.Hex())
	}

	if order.Side != Borrow {
		t.Errorf("Expected side Borrow, got %d", order.Side)
	}

	if order.Status != OrderStatusNew {
		t.Errorf("Expected status New, got %d", order.Status)
	}

	if order.Term != term {
		t.Errorf("Expected term %d, got %d", term, order.Term)
	}
}

func TestLendingOrderRemainingQuantity(t *testing.T) {
	order := &LendingOrder{
		Quantity:       big.NewInt(100),
		FilledQuantity: big.NewInt(30),
	}

	remaining := order.RemainingQuantity()
	expected := big.NewInt(70)

	if remaining.Cmp(expected) != 0 {
		t.Errorf("Expected remaining %s, got %s", expected.String(), remaining.String())
	}
}

func TestLendingOrderIsFilled(t *testing.T) {
	order := &LendingOrder{
		Quantity:       big.NewInt(100),
		FilledQuantity: big.NewInt(30),
	}

	if order.IsFilled() {
		t.Error("Order should not be filled")
	}

	order.FilledQuantity = big.NewInt(100)
	if !order.IsFilled() {
		t.Error("Order should be filled")
	}
}

func TestNewLoan(t *testing.T) {
	borrower := common.HexToAddress("0x1111")
	lender := common.HexToAddress("0x2222")
	lendingToken := common.HexToAddress("0x3333")
	collateralToken := common.HexToAddress("0x4444")
	principal := big.NewInt(1000)
	collateral := big.NewInt(1500)
	interestRate := big.NewInt(500)
	term := uint64(86400 * 30)
	startTime := uint64(1000000)

	loan := NewLoan(
		borrower, lender,
		lendingToken, collateralToken,
		principal, collateral,
		interestRate, term, startTime,
	)

	if loan.BorrowerAddress != borrower {
		t.Error("Borrower address mismatch")
	}

	if loan.LenderAddress != lender {
		t.Error("Lender address mismatch")
	}

	if loan.ExpiryTime != startTime+term {
		t.Errorf("Expected expiry %d, got %d", startTime+term, loan.ExpiryTime)
	}

	if loan.Status != LoanStatusActive {
		t.Errorf("Expected status Active, got %d", loan.Status)
	}
}

func TestLoanIsExpired(t *testing.T) {
	loan := &Loan{
		ExpiryTime: 1000000,
	}

	// Not expired
	if loan.IsExpired(500000) {
		t.Error("Loan should not be expired")
	}

	// Exactly expired
	if !loan.IsExpired(1000000) {
		t.Error("Loan should be expired at expiry time")
	}

	// Past expiry
	if !loan.IsExpired(1500000) {
		t.Error("Loan should be expired past expiry time")
	}
}

func TestLoanGetCollateralRatio(t *testing.T) {
	loan := &Loan{
		Principal:        big.NewInt(1000),
		CollateralAmount: big.NewInt(1500),
	}

	// Collateral price = 1
	collateralPrice := big.NewInt(1)
	ratio := loan.GetCollateralRatio(collateralPrice)

	// Expected: (1500 * 1) / 1000 * 100 = 150
	expected := big.NewInt(150)
	if ratio.Cmp(expected) != 0 {
		t.Errorf("Expected ratio %s, got %s", expected.String(), ratio.String())
	}
}

func TestGetLendingPairKey(t *testing.T) {
	lendingToken := common.HexToAddress("0xaaaa")
	collateralToken := common.HexToAddress("0xbbbb")
	term := uint64(86400 * 30)

	key1 := GetLendingPairKey(lendingToken, collateralToken, term)
	key2 := GetLendingPairKey(lendingToken, collateralToken, term)

	if key1 != key2 {
		t.Error("Same parameters should produce same key")
	}

	// Different term should produce different key
	key3 := GetLendingPairKey(lendingToken, collateralToken, term*2)
	if key1 == key3 {
		t.Error("Different term should produce different key")
	}
}

func TestNewLendingTrade(t *testing.T) {
	borrowOrderID := common.HexToHash("0x1111")
	lendOrderID := common.HexToHash("0x2222")
	borrower := common.HexToAddress("0x3333")
	lender := common.HexToAddress("0x4444")
	lendingToken := common.HexToAddress("0x5555")
	collateralToken := common.HexToAddress("0x6666")
	principal := big.NewInt(1000)
	collateral := big.NewInt(1500)
	interestRate := big.NewInt(500)
	term := uint64(86400 * 30)

	trade := NewLendingTrade(
		borrowOrderID, lendOrderID,
		borrower, lender,
		lendingToken, collateralToken,
		principal, collateral,
		interestRate, term,
	)

	if trade.BorrowOrderID != borrowOrderID {
		t.Error("Borrow order ID mismatch")
	}

	if trade.Borrower != borrower {
		t.Error("Borrower address mismatch")
	}

	if trade.Status != TradeStatusPending {
		t.Errorf("Expected status Pending, got %d", trade.Status)
	}
}

func TestLendingTradeCalculateInterest(t *testing.T) {
	trade := &LendingTrade{
		Principal:    big.NewInt(10000),
		InterestRate: big.NewInt(500), // 5%
		Term:         86400 * 30,      // 30 days
	}

	interest := trade.CalculateInterest()

	// Interest = 10000 * 5 * 30 / (365 * 100) ≈ 41
	// This is approximate due to integer division
	if interest.Sign() <= 0 {
		t.Error("Interest should be positive")
	}
}

func TestLendingTradeTotalRepayment(t *testing.T) {
	trade := &LendingTrade{
		Principal:    big.NewInt(10000),
		InterestRate: big.NewInt(500),
		Term:         86400 * 30,
	}

	total := trade.TotalRepayment()
	interest := trade.CalculateInterest()
	expected := new(big.Int).Add(trade.Principal, interest)

	if total.Cmp(expected) != 0 {
		t.Errorf("Expected total %s, got %s", expected.String(), total.String())
	}
}

func TestXDCxLendingAPIs(t *testing.T) {
	db := memorydb.New()
	lending, _ := New(DefaultConfig(), db)

	apis := lending.APIs()

	if len(apis) != 2 {
		t.Errorf("Expected 2 APIs, got %d", len(apis))
	}

	// Check namespaces
	namespaces := make(map[string]bool)
	for _, api := range apis {
		namespaces[api.Namespace] = true
	}

	if !namespaces["xdcxlending"] {
		t.Error("Expected 'xdcxlending' namespace")
	}
}

func TestCalculateInterest(t *testing.T) {
	db := memorydb.New()
	lending, _ := New(DefaultConfig(), db)

	loan := &Loan{
		Principal:    big.NewInt(10000),
		InterestRate: big.NewInt(1000), // 10%
		Term:         86400 * 365,      // 1 year
	}

	interest := lending.calculateInterest(loan)

	// With 10% annual rate for 1 year, interest should be around 1000
	// Allow some tolerance due to integer division
	if interest.Cmp(big.NewInt(900)) < 0 || interest.Cmp(big.NewInt(1100)) > 0 {
		t.Errorf("Expected interest around 1000, got %s", interest.String())
	}
}
