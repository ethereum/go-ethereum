// Copyright 2019 XDC Network
// This file is part of the XDC library.

// Package lendingstate provides lending state management for XDCxLending
package lendingstate

import (
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
)

// LendingStateDB represents the lending state database
type LendingStateDB struct {
	db   Database
	trie Trie

	// Order books cache
	orderBooks      map[common.Hash]*LendingOrderBook
	orderBooksDirty map[common.Hash]struct{}

	// Orders cache
	orders      map[common.Hash]*LendingOrderState
	ordersDirty map[common.Hash]struct{}

	// Loans cache
	loans      map[common.Hash]*LoanState
	loansDirty map[common.Hash]struct{}

	lock sync.Mutex
}

// New creates a new lending state database
func New(root common.Hash, db Database) (*LendingStateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}

	return &LendingStateDB{
		db:              db,
		trie:            tr,
		orderBooks:      make(map[common.Hash]*LendingOrderBook),
		orderBooksDirty: make(map[common.Hash]struct{}),
		orders:          make(map[common.Hash]*LendingOrderState),
		ordersDirty:     make(map[common.Hash]struct{}),
		loans:           make(map[common.Hash]*LoanState),
		loansDirty:      make(map[common.Hash]struct{}),
	}, nil
}

// GetLendingOrder returns a lending order by ID
func (s *LendingStateDB) GetLendingOrder(orderID common.Hash) *LendingOrderState {
	s.lock.Lock()
	defer s.lock.Unlock()

	if order, exists := s.orders[orderID]; exists {
		return order
	}

	// Load from trie
	order := s.loadLendingOrder(orderID)
	if order != nil {
		s.orders[orderID] = order
	}
	return order
}

// SetLendingOrder sets a lending order in the state
func (s *LendingStateDB) SetLendingOrder(order interface{}) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Convert order interface to LendingOrderState
	orderState := &LendingOrderState{}
	// Implementation depends on order type
	s.orders[orderState.ID] = orderState
	s.ordersDirty[orderState.ID] = struct{}{}
}

// UpdateLendingOrder updates a lending order in the state
func (s *LendingStateDB) UpdateLendingOrder(order interface{}) {
	s.SetLendingOrder(order)
}

// DeleteLendingOrder deletes a lending order from the state
func (s *LendingStateDB) DeleteLendingOrder(orderID common.Hash) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.orders, orderID)
	s.ordersDirty[orderID] = struct{}{}
}

// GetLendingOrderBook returns a lending order book by pair key
func (s *LendingStateDB) GetLendingOrderBook(pairKey common.Hash) *LendingOrderBook {
	s.lock.Lock()
	defer s.lock.Unlock()

	if ob, exists := s.orderBooks[pairKey]; exists {
		return ob
	}

	// Load from trie
	ob := s.loadLendingOrderBook(pairKey)
	if ob != nil {
		s.orderBooks[pairKey] = ob
	}
	return ob
}

// GetOrCreateLendingOrderBook returns or creates a lending order book
func (s *LendingStateDB) GetOrCreateLendingOrderBook(pairKey common.Hash) *LendingOrderBook {
	ob := s.GetLendingOrderBook(pairKey)
	if ob == nil {
		s.lock.Lock()
		ob = NewLendingOrderBook(pairKey)
		s.orderBooks[pairKey] = ob
		s.orderBooksDirty[pairKey] = struct{}{}
		s.lock.Unlock()
	}
	return ob
}

// GetLoan returns a loan by ID
func (s *LendingStateDB) GetLoan(loanID common.Hash) *LoanState {
	s.lock.Lock()
	defer s.lock.Unlock()

	if loan, exists := s.loans[loanID]; exists {
		return loan
	}

	// Load from trie
	loan := s.loadLoan(loanID)
	if loan != nil {
		s.loans[loanID] = loan
	}
	return loan
}

// SetLoan sets a loan in the state
func (s *LendingStateDB) SetLoan(loan *LoanState) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.loans[loan.ID] = loan
	s.loansDirty[loan.ID] = struct{}{}
}

// UpdateLoan updates a loan in the state
func (s *LendingStateDB) UpdateLoan(loan interface{}) {
	// Convert interface to LoanState
	if l, ok := loan.(*LoanState); ok {
		s.SetLoan(l)
	}
}

// GetAllLoans returns all loans
func (s *LendingStateDB) GetAllLoans() []*LoanState {
	s.lock.Lock()
	defer s.lock.Unlock()

	loans := make([]*LoanState, 0, len(s.loans))
	for _, loan := range s.loans {
		loans = append(loans, loan.Copy())
	}
	return loans
}

// Commit commits all changes to the underlying database
func (s *LendingStateDB) Commit() (common.Hash, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	// Commit dirty order books
	for pairKey := range s.orderBooksDirty {
		if ob, exists := s.orderBooks[pairKey]; exists {
			if err := s.commitOrderBook(ob); err != nil {
				return common.Hash{}, err
			}
		}
		delete(s.orderBooksDirty, pairKey)
	}

	// Commit dirty orders
	for orderID := range s.ordersDirty {
		if order, exists := s.orders[orderID]; exists {
			if err := s.commitOrder(order); err != nil {
				return common.Hash{}, err
			}
		}
		delete(s.ordersDirty, orderID)
	}

	// Commit dirty loans
	for loanID := range s.loansDirty {
		if loan, exists := s.loans[loanID]; exists {
			if err := s.commitLoan(loan); err != nil {
				return common.Hash{}, err
			}
		}
		delete(s.loansDirty, loanID)
	}

	// Commit trie
	root, err := s.trie.Commit(nil)
	if err != nil {
		return common.Hash{}, err
	}

	log.Debug("Lending state committed", "root", root.Hex())
	return root, nil
}

// Copy creates a copy of the lending state
func (s *LendingStateDB) Copy() *LendingStateDB {
	s.lock.Lock()
	defer s.lock.Unlock()

	state := &LendingStateDB{
		db:              s.db,
		trie:            s.db.CopyTrie(s.trie),
		orderBooks:      make(map[common.Hash]*LendingOrderBook),
		orderBooksDirty: make(map[common.Hash]struct{}),
		orders:          make(map[common.Hash]*LendingOrderState),
		ordersDirty:     make(map[common.Hash]struct{}),
		loans:           make(map[common.Hash]*LoanState),
		loansDirty:      make(map[common.Hash]struct{}),
	}

	// Copy order books
	for key, ob := range s.orderBooks {
		state.orderBooks[key] = ob.Copy()
	}
	for key := range s.orderBooksDirty {
		state.orderBooksDirty[key] = struct{}{}
	}

	// Copy orders
	for key, order := range s.orders {
		state.orders[key] = order.Copy()
	}
	for key := range s.ordersDirty {
		state.ordersDirty[key] = struct{}{}
	}

	// Copy loans
	for key, loan := range s.loans {
		state.loans[key] = loan.Copy()
	}
	for key := range s.loansDirty {
		state.loansDirty[key] = struct{}{}
	}

	return state
}

// loadLendingOrder loads a lending order from the trie
func (s *LendingStateDB) loadLendingOrder(orderID common.Hash) *LendingOrderState {
	return nil
}

// loadLendingOrderBook loads a lending order book from the trie
func (s *LendingStateDB) loadLendingOrderBook(pairKey common.Hash) *LendingOrderBook {
	return nil
}

// loadLoan loads a loan from the trie
func (s *LendingStateDB) loadLoan(loanID common.Hash) *LoanState {
	return nil
}

// commitOrderBook commits a lending order book
func (s *LendingStateDB) commitOrderBook(ob *LendingOrderBook) error {
	return nil
}

// commitOrder commits a lending order
func (s *LendingStateDB) commitOrder(order *LendingOrderState) error {
	return nil
}

// commitLoan commits a loan
func (s *LendingStateDB) commitLoan(loan *LoanState) error {
	return nil
}

// Database wraps access to tries and contract code
type Database interface {
	// OpenTrie opens the main lending state trie
	OpenTrie(root common.Hash) (Trie, error)

	// CopyTrie returns an independent copy of the given trie
	CopyTrie(Trie) Trie
}

// Trie is a XDCx Merkle Patricia trie
type Trie interface {
	GetKey([]byte) []byte
	TryGet(key []byte) ([]byte, error)
	TryUpdate(key, value []byte) error
	TryDelete(key []byte) error
	Hash() common.Hash
	Commit(onleaf trie.LeafCallback) (common.Hash, error)
	NodeIterator(startKey []byte) trie.NodeIterator
}

// NewDatabase creates a new lending state database
func NewDatabase(db ethdb.Database) Database {
	return &cachingDB{
		db: db,
	}
}

type cachingDB struct {
	db ethdb.Database
}

func (db *cachingDB) OpenTrie(root common.Hash) (Trie, error) {
	return nil, nil
}

func (db *cachingDB) CopyTrie(t Trie) Trie {
	return nil
}

// LendingOrderState represents a lending order in the state
type LendingOrderState struct {
	ID              common.Hash
	UserAddress     common.Address
	RelayerAddress  common.Address
	LendingToken    common.Address
	CollateralToken common.Address
	Side            uint8
	Type            uint8
	InterestRate    *big.Int
	Term            uint64
	Quantity        *big.Int
	FilledQuantity  *big.Int
	Status          uint8
	Nonce           uint64
	Timestamp       uint64
	Signature       []byte
}

// Copy creates a copy of the lending order state
func (o *LendingOrderState) Copy() *LendingOrderState {
	return &LendingOrderState{
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

// RemainingQuantity returns the remaining quantity
func (o *LendingOrderState) RemainingQuantity() *big.Int {
	return new(big.Int).Sub(o.Quantity, o.FilledQuantity)
}

// IsFilled returns whether the order is filled
func (o *LendingOrderState) IsFilled() bool {
	return o.FilledQuantity.Cmp(o.Quantity) >= 0
}

// LoanState represents a loan in the state
type LoanState struct {
	ID               common.Hash
	BorrowerAddress  common.Address
	LenderAddress    common.Address
	LendingToken     common.Address
	CollateralToken  common.Address
	Principal        *big.Int
	CollateralAmount *big.Int
	InterestRate     *big.Int
	Term             uint64
	StartTime        uint64
	ExpiryTime       uint64
	Status           uint8
	LiquidationPrice *big.Int
}

// Copy creates a copy of the loan state
func (l *LoanState) Copy() *LoanState {
	return &LoanState{
		ID:               l.ID,
		BorrowerAddress:  l.BorrowerAddress,
		LenderAddress:    l.LenderAddress,
		LendingToken:     l.LendingToken,
		CollateralToken:  l.CollateralToken,
		Principal:        new(big.Int).Set(l.Principal),
		CollateralAmount: new(big.Int).Set(l.CollateralAmount),
		InterestRate:     new(big.Int).Set(l.InterestRate),
		Term:             l.Term,
		StartTime:        l.StartTime,
		ExpiryTime:       l.ExpiryTime,
		Status:           l.Status,
		LiquidationPrice: new(big.Int).Set(l.LiquidationPrice),
	}
}
