// Copyright 2019 XDC Network
// This file is part of the XDC library.

package XDCx

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// OrderSide represents the side of an order (buy/sell)
type OrderSide uint8

const (
	// Buy represents a buy order
	Buy OrderSide = iota
	// Sell represents a sell order
	Sell
)

// OrderType represents the type of an order
type OrderType uint8

const (
	// Limit represents a limit order
	Limit OrderType = iota
	// Market represents a market order
	Market
)

// OrderStatus represents the status of an order
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

// Order represents a trading order
type Order struct {
	ID              common.Hash    `json:"id"`
	UserAddress     common.Address `json:"userAddress"`
	BaseToken       common.Address `json:"baseToken"`
	QuoteToken      common.Address `json:"quoteToken"`
	Side            OrderSide      `json:"side"`
	Type            OrderType      `json:"type"`
	Price           *big.Int       `json:"price"`
	Quantity        *big.Int       `json:"quantity"`
	FilledQuantity  *big.Int       `json:"filledQuantity"`
	Status          OrderStatus    `json:"status"`
	Nonce           uint64         `json:"nonce"`
	Timestamp       uint64         `json:"timestamp"`
	ExchangeAddress common.Address `json:"exchangeAddress"`
	Signature       []byte         `json:"signature"`
}

// NewOrder creates a new order
func NewOrder(
	userAddress common.Address,
	baseToken, quoteToken common.Address,
	side OrderSide,
	orderType OrderType,
	price, quantity *big.Int,
	nonce uint64,
	exchangeAddress common.Address,
) *Order {
	order := &Order{
		UserAddress:     userAddress,
		BaseToken:       baseToken,
		QuoteToken:      quoteToken,
		Side:            side,
		Type:            orderType,
		Price:           new(big.Int).Set(price),
		Quantity:        new(big.Int).Set(quantity),
		FilledQuantity:  big.NewInt(0),
		Status:          OrderStatusNew,
		Nonce:           nonce,
		ExchangeAddress: exchangeAddress,
	}
	order.ID = order.ComputeHash()
	return order
}

// ComputeHash computes the hash of the order
func (o *Order) ComputeHash() common.Hash {
	data := append(o.UserAddress.Bytes(), o.BaseToken.Bytes()...)
	data = append(data, o.QuoteToken.Bytes()...)
	data = append(data, byte(o.Side))
	data = append(data, byte(o.Type))
	data = append(data, common.BigToHash(o.Price).Bytes()...)
	data = append(data, common.BigToHash(o.Quantity).Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(o.Nonce))).Bytes()...)
	data = append(data, o.ExchangeAddress.Bytes()...)
	return crypto.Keccak256Hash(data)
}

// Sign signs the order with the given private key
func (o *Order) Sign(privateKey *ecdsa.PrivateKey) error {
	hash := o.ComputeHash()
	sig, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		return err
	}
	o.Signature = sig
	return nil
}

// VerifySignature verifies the order signature
func (o *Order) VerifySignature() bool {
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
func (o *Order) RemainingQuantity() *big.Int {
	return new(big.Int).Sub(o.Quantity, o.FilledQuantity)
}

// IsFilled returns whether the order is fully filled
func (o *Order) IsFilled() bool {
	return o.FilledQuantity.Cmp(o.Quantity) >= 0
}

// Clone creates a copy of the order
func (o *Order) Clone() *Order {
	return &Order{
		ID:              o.ID,
		UserAddress:     o.UserAddress,
		BaseToken:       o.BaseToken,
		QuoteToken:      o.QuoteToken,
		Side:            o.Side,
		Type:            o.Type,
		Price:           new(big.Int).Set(o.Price),
		Quantity:        new(big.Int).Set(o.Quantity),
		FilledQuantity:  new(big.Int).Set(o.FilledQuantity),
		Status:          o.Status,
		Nonce:           o.Nonce,
		Timestamp:       o.Timestamp,
		ExchangeAddress: o.ExchangeAddress,
		Signature:       append([]byte{}, o.Signature...),
	}
}

// PairKey returns the trading pair key
func (o *Order) PairKey() common.Hash {
	return GetPairKey(o.BaseToken, o.QuoteToken)
}

// GetPairKey returns the trading pair key for base and quote tokens
func GetPairKey(baseToken, quoteToken common.Address) common.Hash {
	return crypto.Keccak256Hash(baseToken.Bytes(), quoteToken.Bytes())
}
