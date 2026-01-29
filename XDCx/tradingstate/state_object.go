// Copyright 2019 XDC Network
// This file is part of the XDC library.

package tradingstate

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// stateObject represents a trading state object
type stateObject struct {
	key      common.Hash
	data     StateData
	db       *TradingStateDB
	trie     Trie
	dirty    bool
	deleted  bool
}

// StateData holds the trading state data
type StateData struct {
	Nonce       uint64
	Balance     *big.Int
	Root        common.Hash // merkle root of the state trie
	CodeHash    []byte
}

// newObject creates a new state object
func newObject(db *TradingStateDB, key common.Hash, data StateData) *stateObject {
	return &stateObject{
		key:   key,
		data:  data,
		db:    db,
		dirty: false,
	}
}

// GetNonce returns the nonce
func (s *stateObject) GetNonce() uint64 {
	return s.data.Nonce
}

// SetNonce sets the nonce
func (s *stateObject) SetNonce(nonce uint64) {
	s.data.Nonce = nonce
	s.dirty = true
}

// GetBalance returns the balance
func (s *stateObject) GetBalance() *big.Int {
	return s.data.Balance
}

// SetBalance sets the balance
func (s *stateObject) SetBalance(balance *big.Int) {
	s.data.Balance = new(big.Int).Set(balance)
	s.dirty = true
}

// AddBalance adds amount to balance
func (s *stateObject) AddBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Add(s.GetBalance(), amount))
}

// SubBalance subtracts amount from balance
func (s *stateObject) SubBalance(amount *big.Int) {
	if amount.Sign() == 0 {
		return
	}
	s.SetBalance(new(big.Int).Sub(s.GetBalance(), amount))
}

// deepCopy creates a deep copy of the state object
func (s *stateObject) deepCopy() *stateObject {
	return &stateObject{
		key:     s.key,
		data:    s.data,
		db:      s.db,
		trie:    s.trie,
		dirty:   s.dirty,
		deleted: s.deleted,
	}
}

// EncodeRLP implements rlp.Encoder
func (s *stateObject) EncodeRLP() ([]byte, error) {
	return rlp.EncodeToBytes(&s.data)
}

// OrderState represents an order in the state
type OrderState struct {
	ID              common.Hash
	UserAddress     common.Address
	ExchangeAddress common.Address
	BaseToken       common.Address
	QuoteToken      common.Address
	Side            uint8
	Type            uint8
	Price           *big.Int
	Quantity        *big.Int
	FilledQuantity  *big.Int
	Status          uint8
	Nonce           uint64
	Timestamp       uint64
	Signature       []byte
}

// NewOrderState creates a new order state
func NewOrderState(
	id common.Hash,
	userAddress common.Address,
	baseToken, quoteToken common.Address,
	side, orderType uint8,
	price, quantity *big.Int,
) *OrderState {
	return &OrderState{
		ID:             id,
		UserAddress:    userAddress,
		BaseToken:      baseToken,
		QuoteToken:     quoteToken,
		Side:           side,
		Type:           orderType,
		Price:          new(big.Int).Set(price),
		Quantity:       new(big.Int).Set(quantity),
		FilledQuantity: big.NewInt(0),
		Status:         0, // New
	}
}

// Hash computes the order hash
func (o *OrderState) Hash() common.Hash {
	data := append(o.UserAddress.Bytes(), o.BaseToken.Bytes()...)
	data = append(data, o.QuoteToken.Bytes()...)
	data = append(data, byte(o.Side))
	data = append(data, byte(o.Type))
	data = append(data, common.BigToHash(o.Price).Bytes()...)
	data = append(data, common.BigToHash(o.Quantity).Bytes()...)
	data = append(data, common.BigToHash(big.NewInt(int64(o.Nonce))).Bytes()...)
	return crypto.Keccak256Hash(data)
}

// Copy creates a copy of the order state
func (o *OrderState) Copy() *OrderState {
	return &OrderState{
		ID:              o.ID,
		UserAddress:     o.UserAddress,
		ExchangeAddress: o.ExchangeAddress,
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
		Signature:       append([]byte{}, o.Signature...),
	}
}

// RemainingQuantity returns the remaining quantity
func (o *OrderState) RemainingQuantity() *big.Int {
	return new(big.Int).Sub(o.Quantity, o.FilledQuantity)
}

// IsFilled returns whether the order is filled
func (o *OrderState) IsFilled() bool {
	return o.FilledQuantity.Cmp(o.Quantity) >= 0
}
