// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package types

import (
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// OrderTransaction represents an XDCx order transaction
type OrderTransaction struct {
	Nonce           uint64         `json:"nonce"`
	Quantity        *big.Int       `json:"quantity"`
	Price           *big.Int       `json:"price"`
	ExchangeAddress common.Address `json:"exchangeAddress"`
	UserAddress     common.Address `json:"userAddress"`
	BaseToken       common.Address `json:"baseToken"`
	QuoteToken      common.Address `json:"quoteToken"`
	Status          string         `json:"status"`
	Side            string         `json:"side"`   // "BUY" or "SELL"
	Type            string         `json:"type"`   // "LO" (limit) or "MO" (market)
	Hash            common.Hash    `json:"hash"`
	OrderID         uint64         `json:"orderId"`
	PairName        string         `json:"pairName"`
	
	// Signature
	V *big.Int `json:"v"`
	R *big.Int `json:"r"`
	S *big.Int `json:"s"`
	
	// Cache
	hash atomic.Value
	size atomic.Value
}

// OrderTransactions is a slice of order transactions
type OrderTransactions []*OrderTransaction

// Len returns the length of the slice
func (s OrderTransactions) Len() int { return len(s) }

// Swap swaps two elements
func (s OrderTransactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp returns the RLP encoding of an order transaction
func (s OrderTransactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// ComputeHash computes the hash of the order transaction
func (tx *OrderTransaction) ComputeHash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	
	// Hash the key fields
	data, _ := rlp.EncodeToBytes([]interface{}{
		tx.Nonce,
		tx.Quantity,
		tx.Price,
		tx.ExchangeAddress,
		tx.UserAddress,
		tx.BaseToken,
		tx.QuoteToken,
		tx.Side,
		tx.Type,
	})
	hash := crypto.Keccak256Hash(data)
	tx.hash.Store(hash)
	return hash
}

// GetHash returns the cached hash or computes it
func (tx *OrderTransaction) GetHash() common.Hash {
	return tx.ComputeHash()
}

// Size returns the approximate size of the transaction
func (tx *OrderTransaction) Size() common.StorageSize {
	if size := tx.size.Load(); size != nil {
		return size.(common.StorageSize)
	}
	
	c := common.StorageSize(0)
	c += 8 // Nonce
	if tx.Quantity != nil {
		c += common.StorageSize(len(tx.Quantity.Bytes()))
	}
	if tx.Price != nil {
		c += common.StorageSize(len(tx.Price.Bytes()))
	}
	c += 20 * 4 // addresses
	c += 32     // hash
	c += common.StorageSize(len(tx.Status) + len(tx.Side) + len(tx.Type) + len(tx.PairName))
	
	tx.size.Store(c)
	return c
}

// EncodingSize returns the RLP encoding size
func (tx *OrderTransaction) EncodingSize() int {
	enc, _ := rlp.EncodeToBytes(tx)
	return len(enc)
}

// IsBuy returns true if this is a buy order
func (tx *OrderTransaction) IsBuy() bool {
	return tx.Side == "BUY"
}

// IsSell returns true if this is a sell order  
func (tx *OrderTransaction) IsSell() bool {
	return tx.Side == "SELL"
}

// IsLimit returns true if this is a limit order
func (tx *OrderTransaction) IsLimit() bool {
	return tx.Type == "LO"
}

// IsMarket returns true if this is a market order
func (tx *OrderTransaction) IsMarket() bool {
	return tx.Type == "MO"
}

// OrderTxByNonce sorts order transactions by nonce
type OrderTxByNonce OrderTransactions

func (s OrderTxByNonce) Len() int           { return len(s) }
func (s OrderTxByNonce) Less(i, j int) bool { return s[i].Nonce < s[j].Nonce }
func (s OrderTxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// OrderTxByPrice sorts order transactions by price
type OrderTxByPrice struct {
	txs  OrderTransactions
	side string
}

func (s OrderTxByPrice) Len() int { return len(s.txs) }
func (s OrderTxByPrice) Less(i, j int) bool {
	if s.side == "BUY" {
		// Higher price first for buy orders
		return s.txs[i].Price.Cmp(s.txs[j].Price) > 0
	}
	// Lower price first for sell orders
	return s.txs[i].Price.Cmp(s.txs[j].Price) < 0
}
func (s OrderTxByPrice) Swap(i, j int) { s.txs[i], s.txs[j] = s.txs[j], s.txs[i] }

// LendingTransaction represents an XDCx lending transaction
type LendingTransaction struct {
	Nonce             uint64         `json:"nonce"`
	Quantity          *big.Int       `json:"quantity"`
	Interest          uint64         `json:"interest"`      // Interest rate in basis points
	Term              uint64         `json:"term"`          // Term in seconds
	RelayerAddress    common.Address `json:"relayerAddress"`
	UserAddress       common.Address `json:"userAddress"`
	LendingToken      common.Address `json:"lendingToken"`
	CollateralToken   common.Address `json:"collateralToken"`
	AutoTopUp         bool           `json:"autoTopUp"`
	Status            string         `json:"status"`
	Side              string         `json:"side"`          // "INVEST" or "BORROW"
	Type              string         `json:"type"`          // "LO" or "MO"
	LendingId         uint64         `json:"lendingId"`
	LendingTradeId    uint64         `json:"lendingTradeId"`
	ExtraData         string         `json:"extraData"`
	
	// Signature
	V *big.Int `json:"v"`
	R *big.Int `json:"r"`
	S *big.Int `json:"s"`
	
	// Cache
	hash atomic.Value
}

// LendingTransactions is a slice of lending transactions
type LendingTransactions []*LendingTransaction

// Len returns the length of the slice
func (s LendingTransactions) Len() int { return len(s) }

// Swap swaps two elements
func (s LendingTransactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp returns the RLP encoding of a lending transaction
func (s LendingTransactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}

// Hash returns the hash of the lending transaction
func (tx *LendingTransaction) Hash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	
	data, _ := rlp.EncodeToBytes([]interface{}{
		tx.Nonce,
		tx.Quantity,
		tx.Interest,
		tx.Term,
		tx.RelayerAddress,
		tx.UserAddress,
		tx.LendingToken,
		tx.CollateralToken,
		tx.Side,
		tx.Type,
	})
	hash := crypto.Keccak256Hash(data)
	tx.hash.Store(hash)
	return hash
}

// Nonce returns the transaction nonce
func (tx *LendingTransaction) GetNonce() uint64 {
	return tx.Nonce
}

// IsInvest returns true if this is an invest order
func (tx *LendingTransaction) IsInvest() bool {
	return tx.Side == "INVEST"
}

// IsBorrow returns true if this is a borrow order
func (tx *LendingTransaction) IsBorrow() bool {
	return tx.Side == "BORROW"
}

// LendingTxByNonce sorts lending transactions by nonce
type LendingTxByNonce LendingTransactions

func (s LendingTxByNonce) Len() int           { return len(s) }
func (s LendingTxByNonce) Less(i, j int) bool { return s[i].Nonce < s[j].Nonce }
func (s LendingTxByNonce) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
