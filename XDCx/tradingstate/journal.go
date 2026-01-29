// Copyright 2019 XDC Network
// This file is part of the XDC library.

package tradingstate

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// journalEntry is a modification entry in the trading state change journal
type journalEntry interface {
	// revert undoes the changes introduced by this journal entry
	revert(*TradingStateDB)

	// dirtied returns the hash of the object modified by this journal entry
	dirtied() *common.Hash
}

// journal contains the list of state modifications applied since the last state commit
type journal struct {
	entries []journalEntry
	dirties map[common.Hash]int
}

// newJournal creates a new initialized journal
func newJournal() *journal {
	return &journal{
		dirties: make(map[common.Hash]int),
	}
}

// append adds a new journal entry
func (j *journal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
	if hash := entry.dirtied(); hash != nil {
		j.dirties[*hash]++
	}
}

// revert undoes a batch of journal entries
func (j *journal) revert(statedb *TradingStateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		j.entries[i].revert(statedb)

		if hash := j.entries[i].dirtied(); hash != nil {
			if j.dirties[*hash]--; j.dirties[*hash] == 0 {
				delete(j.dirties, *hash)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

// dirty returns the dirty count for a specific object
func (j *journal) dirty(hash common.Hash) int {
	return j.dirties[hash]
}

// length returns the current number of entries in the journal
func (j *journal) length() int {
	return len(j.entries)
}

// orderChange represents a change to an order
type orderChange struct {
	orderID common.Hash
	prev    *OrderState
}

func (ch orderChange) revert(s *TradingStateDB) {
	if ch.prev == nil {
		delete(s.orders, ch.orderID)
	} else {
		s.orders[ch.orderID] = ch.prev
	}
}

func (ch orderChange) dirtied() *common.Hash {
	return &ch.orderID
}

// orderBookChange represents a change to an order book
type orderBookChange struct {
	pairKey common.Hash
	prev    *OrderBook
}

func (ch orderBookChange) revert(s *TradingStateDB) {
	if ch.prev == nil {
		delete(s.orderBooks, ch.pairKey)
	} else {
		s.orderBooks[ch.pairKey] = ch.prev
	}
}

func (ch orderBookChange) dirtied() *common.Hash {
	return &ch.pairKey
}

// bidAddChange represents adding a bid
type bidAddChange struct {
	pairKey common.Hash
	orderID common.Hash
}

func (ch bidAddChange) revert(s *TradingStateDB) {
	if ob, exists := s.orderBooks[ch.pairKey]; exists {
		ob.RemoveBid(ch.orderID)
	}
}

func (ch bidAddChange) dirtied() *common.Hash {
	return &ch.pairKey
}

// askAddChange represents adding an ask
type askAddChange struct {
	pairKey common.Hash
	orderID common.Hash
}

func (ch askAddChange) revert(s *TradingStateDB) {
	if ob, exists := s.orderBooks[ch.pairKey]; exists {
		ob.RemoveAsk(ch.orderID)
	}
}

func (ch askAddChange) dirtied() *common.Hash {
	return &ch.pairKey
}

// bidRemoveChange represents removing a bid
type bidRemoveChange struct {
	pairKey common.Hash
	order   *OrderState
}

func (ch bidRemoveChange) revert(s *TradingStateDB) {
	if ob, exists := s.orderBooks[ch.pairKey]; exists {
		ob.AddBid(ch.order)
	}
}

func (ch bidRemoveChange) dirtied() *common.Hash {
	return &ch.pairKey
}

// askRemoveChange represents removing an ask
type askRemoveChange struct {
	pairKey common.Hash
	order   *OrderState
}

func (ch askRemoveChange) revert(s *TradingStateDB) {
	if ob, exists := s.orderBooks[ch.pairKey]; exists {
		ob.AddAsk(ch.order)
	}
}

func (ch askRemoveChange) dirtied() *common.Hash {
	return &ch.pairKey
}

// balanceChange represents a balance change
type balanceChange struct {
	key    common.Hash
	amount *big.Int
}

func (ch balanceChange) revert(s *TradingStateDB) {
	if obj, exists := s.stateObjects[ch.key]; exists {
		obj.SubBalance(ch.amount)
	}
}

func (ch balanceChange) dirtied() *common.Hash {
	return &ch.key
}

// nonceChange represents a nonce change
type nonceChange struct {
	key  common.Hash
	prev uint64
}

func (ch nonceChange) revert(s *TradingStateDB) {
	if obj, exists := s.stateObjects[ch.key]; exists {
		obj.SetNonce(ch.prev)
	}
}

func (ch nonceChange) dirtied() *common.Hash {
	return &ch.key
}
