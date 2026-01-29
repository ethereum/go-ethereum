// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCx

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// TradeStatus represents the status of a trade
type TradeStatus uint8

const (
	// TradeStatusPending represents a pending trade
	TradeStatusPending TradeStatus = iota
	// TradeStatusSettled represents a settled trade
	TradeStatusSettled
	// TradeStatusFailed represents a failed trade
	TradeStatusFailed
)

// Trade represents a matched trade
type Trade struct {
	hash        common.Hash
	MakerOrderID common.Hash    `json:"makerOrderId"`
	TakerOrderID common.Hash    `json:"takerOrderId"`
	Maker       common.Address `json:"maker"`
	Taker       common.Address `json:"taker"`
	BaseToken   common.Address `json:"baseToken"`
	QuoteToken  common.Address `json:"quoteToken"`
	Price       *big.Int       `json:"price"`
	Quantity    *big.Int       `json:"quantity"`
	Amount      *big.Int       `json:"amount"`
	MakerFee    *big.Int       `json:"makerFee"`
	TakerFee    *big.Int       `json:"takerFee"`
	Timestamp   uint64         `json:"timestamp"`
	Status      TradeStatus    `json:"status"`
	BlockNumber uint64         `json:"blockNumber"`
	TxHash      common.Hash    `json:"txHash"`
}

// NewTrade creates a new trade
func NewTrade(
	makerOrderID, takerOrderID common.Hash,
	maker, taker common.Address,
	baseToken, quoteToken common.Address,
	price, quantity *big.Int,
) *Trade {
	amount := new(big.Int).Mul(price, quantity)
	amount = new(big.Int).Div(amount, big.NewInt(1e18))

	trade := &Trade{
		MakerOrderID: makerOrderID,
		TakerOrderID: takerOrderID,
		Maker:       maker,
		Taker:       taker,
		BaseToken:   baseToken,
		QuoteToken:  quoteToken,
		Price:       new(big.Int).Set(price),
		Quantity:    new(big.Int).Set(quantity),
		Amount:      amount,
		MakerFee:    big.NewInt(0),
		TakerFee:    big.NewInt(0),
		Status:      TradeStatusPending,
	}
	trade.hash = trade.ComputeHash()
	return trade
}

// ComputeHash computes the hash of the trade
func (t *Trade) ComputeHash() common.Hash {
	data := append(t.MakerOrderID.Bytes(), t.TakerOrderID.Bytes()...)
	data = append(data, t.Maker.Bytes()...)
	data = append(data, t.Taker.Bytes()...)
	data = append(data, t.BaseToken.Bytes()...)
	data = append(data, t.QuoteToken.Bytes()...)
	data = append(data, common.BigToHash(t.Price).Bytes()...)
	data = append(data, common.BigToHash(t.Quantity).Bytes()...)
	return crypto.Keccak256Hash(data)
}

// Hash returns the trade hash
func (t *Trade) Hash() common.Hash {
	if t.hash == (common.Hash{}) {
		t.hash = t.ComputeHash()
	}
	return t.hash
}

// SetFees sets the maker and taker fees
func (t *Trade) SetFees(makerFee, takerFee *big.Int) {
	if makerFee != nil {
		t.MakerFee = new(big.Int).Set(makerFee)
	}
	if takerFee != nil {
		t.TakerFee = new(big.Int).Set(takerFee)
	}
}

// SetSettled marks the trade as settled
func (t *Trade) SetSettled(blockNumber uint64, txHash common.Hash) {
	t.Status = TradeStatusSettled
	t.BlockNumber = blockNumber
	t.TxHash = txHash
}

// SetFailed marks the trade as failed
func (t *Trade) SetFailed() {
	t.Status = TradeStatusFailed
}

// PairKey returns the trading pair key
func (t *Trade) PairKey() common.Hash {
	return GetPairKey(t.BaseToken, t.QuoteToken)
}

// Clone creates a copy of the trade
func (t *Trade) Clone() *Trade {
	return &Trade{
		hash:        t.hash,
		MakerOrderID: t.MakerOrderID,
		TakerOrderID: t.TakerOrderID,
		Maker:       t.Maker,
		Taker:       t.Taker,
		BaseToken:   t.BaseToken,
		QuoteToken:  t.QuoteToken,
		Price:       new(big.Int).Set(t.Price),
		Quantity:    new(big.Int).Set(t.Quantity),
		Amount:      new(big.Int).Set(t.Amount),
		MakerFee:    new(big.Int).Set(t.MakerFee),
		TakerFee:    new(big.Int).Set(t.TakerFee),
		Timestamp:   t.Timestamp,
		Status:      t.Status,
		BlockNumber: t.BlockNumber,
		TxHash:      t.TxHash,
	}
}
