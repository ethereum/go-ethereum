// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCx

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

// mockStateDB implements a minimal state.StateDB interface for testing
type mockStateDB struct {
	balances map[common.Address]*big.Int
}

func newMockStateDB() *mockStateDB {
	return &mockStateDB{
		balances: make(map[common.Address]*big.Int),
	}
}

func (m *mockStateDB) GetBalance(addr common.Address) *big.Int {
	if balance, ok := m.balances[addr]; ok {
		return balance
	}
	return big.NewInt(0)
}

func (m *mockStateDB) SetBalance(addr common.Address, amount *big.Int) {
	m.balances[addr] = amount
}

func TestNewXDCx(t *testing.T) {
	db := memorydb.New()
	config := DefaultConfig()

	xdcx, err := New(config, db)
	if err != nil {
		t.Fatalf("Failed to create XDCx: %v", err)
	}

	if xdcx == nil {
		t.Fatal("XDCx instance is nil")
	}

	if xdcx.config != config {
		t.Error("Config not set correctly")
	}
}

func TestXDCxStartStop(t *testing.T) {
	db := memorydb.New()
	xdcx, _ := New(DefaultConfig(), db)

	// Test start
	if err := xdcx.Start(); err != nil {
		t.Fatalf("Failed to start XDCx: %v", err)
	}

	if !xdcx.IsRunning() {
		t.Error("XDCx should be running after Start")
	}

	// Test double start
	if err := xdcx.Start(); err != nil {
		t.Error("Double start should not error")
	}

	// Test stop
	if err := xdcx.Stop(); err != nil {
		t.Fatalf("Failed to stop XDCx: %v", err)
	}

	if xdcx.IsRunning() {
		t.Error("XDCx should not be running after Stop")
	}

	// Test double stop
	if err := xdcx.Stop(); err != nil {
		t.Error("Double stop should not error")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.DBEngine != "leveldb" {
		t.Errorf("Expected DBEngine 'leveldb', got '%s'", config.DBEngine)
	}

	if config.TradingStateDB != "XDCx" {
		t.Errorf("Expected TradingStateDB 'XDCx', got '%s'", config.TradingStateDB)
	}
}

func TestNewOrder(t *testing.T) {
	userAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
	baseToken := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	quoteToken := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	exchangeAddr := common.HexToAddress("0xcccccccccccccccccccccccccccccccccccccccc")

	price := big.NewInt(1000)
	quantity := big.NewInt(5)
	nonce := uint64(1)

	order := NewOrder(userAddr, baseToken, quoteToken, Buy, Limit, price, quantity, nonce, exchangeAddr)

	if order.UserAddress != userAddr {
		t.Errorf("Expected user address %s, got %s", userAddr.Hex(), order.UserAddress.Hex())
	}

	if order.BaseToken != baseToken {
		t.Errorf("Expected base token %s, got %s", baseToken.Hex(), order.BaseToken.Hex())
	}

	if order.Side != Buy {
		t.Errorf("Expected side Buy, got %d", order.Side)
	}

	if order.Status != OrderStatusNew {
		t.Errorf("Expected status New, got %d", order.Status)
	}

	if order.FilledQuantity.Cmp(big.NewInt(0)) != 0 {
		t.Errorf("Expected filled quantity 0, got %s", order.FilledQuantity.String())
	}
}

func TestOrderRemainingQuantity(t *testing.T) {
	order := &Order{
		Quantity:       big.NewInt(100),
		FilledQuantity: big.NewInt(30),
	}

	remaining := order.RemainingQuantity()
	expected := big.NewInt(70)

	if remaining.Cmp(expected) != 0 {
		t.Errorf("Expected remaining %s, got %s", expected.String(), remaining.String())
	}
}

func TestOrderIsFilled(t *testing.T) {
	// Not filled
	order := &Order{
		Quantity:       big.NewInt(100),
		FilledQuantity: big.NewInt(30),
	}

	if order.IsFilled() {
		t.Error("Order should not be filled")
	}

	// Filled
	order.FilledQuantity = big.NewInt(100)
	if !order.IsFilled() {
		t.Error("Order should be filled")
	}

	// Over filled
	order.FilledQuantity = big.NewInt(110)
	if !order.IsFilled() {
		t.Error("Order should be filled when over-filled")
	}
}

func TestOrderClone(t *testing.T) {
	order := &Order{
		ID:             common.HexToHash("0x1234"),
		UserAddress:    common.HexToAddress("0x5678"),
		Price:          big.NewInt(1000),
		Quantity:       big.NewInt(100),
		FilledQuantity: big.NewInt(50),
		Status:         OrderStatusPartialFilled,
	}

	clone := order.Clone()

	// Check values are equal
	if clone.ID != order.ID {
		t.Error("Clone ID should match")
	}

	if clone.Price.Cmp(order.Price) != 0 {
		t.Error("Clone price should match")
	}

	// Check it's a deep copy
	clone.Price = big.NewInt(2000)
	if order.Price.Cmp(big.NewInt(1000)) != 0 {
		t.Error("Modifying clone should not affect original")
	}
}

func TestPairKey(t *testing.T) {
	baseToken := common.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	quoteToken := common.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	key1 := GetPairKey(baseToken, quoteToken)
	key2 := GetPairKey(baseToken, quoteToken)

	if key1 != key2 {
		t.Error("Same tokens should produce same key")
	}

	key3 := GetPairKey(quoteToken, baseToken)
	if key1 == key3 {
		t.Error("Reversed tokens should produce different key")
	}
}

func TestNewTrade(t *testing.T) {
	makerOrderID := common.HexToHash("0x1111")
	takerOrderID := common.HexToHash("0x2222")
	maker := common.HexToAddress("0x3333")
	taker := common.HexToAddress("0x4444")
	baseToken := common.HexToAddress("0x5555")
	quoteToken := common.HexToAddress("0x6666")
	price := big.NewInt(1000)
	quantity := big.NewInt(10)

	trade := NewTrade(makerOrderID, takerOrderID, maker, taker, baseToken, quoteToken, price, quantity)

	if trade.MakerOrderID != makerOrderID {
		t.Error("Maker order ID mismatch")
	}

	if trade.Maker != maker {
		t.Error("Maker address mismatch")
	}

	// Check amount calculation
	expectedAmount := new(big.Int).Mul(price, quantity)
	expectedAmount = expectedAmount.Div(expectedAmount, big.NewInt(1e18))
	if trade.Amount.Cmp(expectedAmount) != 0 {
		t.Errorf("Expected amount %s, got %s", expectedAmount.String(), trade.Amount.String())
	}

	if trade.Status != TradeStatusPending {
		t.Errorf("Expected status Pending, got %d", trade.Status)
	}
}

func TestTradeSettled(t *testing.T) {
	trade := NewTrade(
		common.Hash{}, common.Hash{},
		common.Address{}, common.Address{},
		common.Address{}, common.Address{},
		big.NewInt(1000), big.NewInt(10),
	)

	blockNumber := uint64(12345)
	txHash := common.HexToHash("0xabcd")

	trade.SetSettled(blockNumber, txHash)

	if trade.Status != TradeStatusSettled {
		t.Errorf("Expected status Settled, got %d", trade.Status)
	}

	if trade.BlockNumber != blockNumber {
		t.Errorf("Expected block %d, got %d", blockNumber, trade.BlockNumber)
	}

	if trade.TxHash != txHash {
		t.Error("TxHash mismatch")
	}
}

func TestMatcherGetBestBidError(t *testing.T) {
	db := memorydb.New()
	xdcx, _ := New(DefaultConfig(), db)

	baseToken := common.HexToAddress("0xaaaa")
	quoteToken := common.HexToAddress("0xbbbb")

	// Should return error for non-existent order book
	_, err := xdcx.matcher.GetBestBid(baseToken, quoteToken, nil)
	if err == nil {
		t.Error("Expected error for nil trading state")
	}
}

func TestXDCxAPIs(t *testing.T) {
	db := memorydb.New()
	xdcx, _ := New(DefaultConfig(), db)

	apis := xdcx.APIs()

	if len(apis) != 2 {
		t.Errorf("Expected 2 APIs, got %d", len(apis))
	}

	// Check namespaces
	namespaces := make(map[string]bool)
	for _, api := range apis {
		namespaces[api.Namespace] = true
	}

	if !namespaces["xdcx"] {
		t.Error("Expected 'xdcx' namespace")
	}
}

func TestOrderProcessorValidation(t *testing.T) {
	db := memorydb.New()
	xdcx, _ := New(DefaultConfig(), db)

	// Invalid price
	order := &Order{
		Price:    nil,
		Quantity: big.NewInt(100),
	}

	_, err := xdcx.ProcessOrder(context.Background(), nil, nil, order)
	if err == nil || err != ErrXDCxServiceNotRunning {
		// Service not running is expected since we didn't start it
	}

	// Start service
	xdcx.Start()

	// Invalid price
	order.Price = big.NewInt(0)
	_, err = xdcx.ProcessOrder(context.Background(), nil, nil, order)
	if err == nil {
		t.Error("Expected error for zero price")
	}

	// Invalid quantity
	order.Price = big.NewInt(1000)
	order.Quantity = big.NewInt(0)
	_, err = xdcx.ProcessOrder(context.Background(), nil, nil, order)
	if err == nil {
		t.Error("Expected error for zero quantity")
	}
}
