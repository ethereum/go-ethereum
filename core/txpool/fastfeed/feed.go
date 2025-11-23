// Copyright 2024 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package fastfeed

import (
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"errors"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const (
	// DefaultBufferSize is the default ring buffer capacity (must be power of 2)
	DefaultBufferSize = 16384
	
	// MaxReaders is the maximum number of concurrent consumers
	MaxReaders = 64
	
	// TxEventSize is the size of a transaction event in bytes
	TxEventSize = 184 // 32 (hash) + 20 (from) + 20 (to) + 32 (value) + 32 (gasPrice) + 8 (nonce) + 8 (gas) + 4 (type) + 8 (timestamp) + 20 (padding)
)

// TxEventType represents the type of transaction event
type TxEventType uint8

const (
	TxEventAdded TxEventType = iota
	TxEventRemoved
	TxEventReplaced
)

// TxEvent is a fixed-size transaction event optimized for zero-copy access.
// Layout is designed for CPU cache efficiency and minimal memory access.
type TxEvent struct {
	Hash      [32]byte    // Transaction hash
	From      [20]byte    // Sender address
	To        [20]byte    // Recipient address (0x0 for contract creation)
	Value     [32]byte    // Transfer value
	GasPrice  [32]byte    // Gas price or maxFeePerGas for EIP-1559
	Nonce     uint64      // Sender nonce
	Gas       uint64      // Gas limit
	Type      uint8       // Transaction type
	EventType TxEventType // Event type (added/removed/replaced)
	Timestamp uint64      // Event timestamp (nanoseconds)
	_         [6]byte     // Padding for alignment
}

// TxFilter defines filtering criteria for transaction events.
type TxFilter struct {
	// Addresses to watch (empty = all addresses)
	Addresses map[common.Address]struct{}
	
	// Contract methods to watch (first 4 bytes of calldata)
	Methods map[[4]byte]struct{}
	
	// Minimum gas price filter
	MinGasPrice uint64
	
	// Transaction types to include
	Types map[uint8]struct{}
}

// Matches returns true if the transaction matches the filter.
func (f *TxFilter) Matches(event *TxEvent) bool {
	// Check addresses
	if len(f.Addresses) > 0 {
		fromAddr := common.BytesToAddress(event.From[:])
		toAddr := common.BytesToAddress(event.To[:])
		_, fromMatch := f.Addresses[fromAddr]
		_, toMatch := f.Addresses[toAddr]
		if !fromMatch && !toMatch {
			return false
		}
	}
	
	// Check transaction type
	if len(f.Types) > 0 {
		if _, ok := f.Types[event.Type]; !ok {
			return false
		}
	}
	
	// Check gas price
	if f.MinGasPrice > 0 {
		// Simple comparison of first 8 bytes as uint64
		gasPrice := uint64(event.GasPrice[24])<<56 | 
		           uint64(event.GasPrice[25])<<48 | 
		           uint64(event.GasPrice[26])<<40 | 
		           uint64(event.GasPrice[27])<<32 |
		           uint64(event.GasPrice[28])<<24 | 
		           uint64(event.GasPrice[29])<<16 | 
		           uint64(event.GasPrice[30])<<8 | 
		           uint64(event.GasPrice[31])
		if gasPrice < f.MinGasPrice {
			return false
		}
	}
	
	return true
}

// TxFastFeed is a high-performance transaction event feed using lock-free ring buffers.
type TxFastFeed struct {
	ring     *RingBuffer
	mu       sync.RWMutex
	filters  map[int]*TxFilter
	nextID   int
	enabled  atomic.Bool
	
	// Metrics
	eventsPublished atomic.Uint64
	eventsDropped   atomic.Uint64
	lastPublish     atomic.Int64
}

// NewTxFastFeed creates a new fast transaction feed.
func NewTxFastFeed() *TxFastFeed {
	feed := &TxFastFeed{
		ring:    NewRingBuffer(DefaultBufferSize, MaxReaders),
		filters: make(map[int]*TxFilter),
	}
	feed.enabled.Store(true)
	return feed
}

// Publish publishes a transaction event to all subscribers.
func (f *TxFastFeed) Publish(tx *types.Transaction, eventType TxEventType) {
	if !f.enabled.Load() {
		return
	}
	
	// Convert transaction to fixed-size event
	event := f.txToEvent(tx, eventType)
	
	// Write to ring buffer
	eventPtr := unsafe.Pointer(&event)
	if !f.ring.Write(eventPtr) {
		f.eventsDropped.Add(1)
		log.Warn("Fast feed buffer full, event dropped", "hash", tx.Hash())
		return
	}
	
	f.eventsPublished.Add(1)
	f.lastPublish.Store(time.Now().UnixNano())
}

// Subscribe creates a new subscription with optional filtering.
func (f *TxFastFeed) Subscribe(filter *TxFilter) (*Subscription, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	if f.nextID >= MaxReaders {
		return nil, ErrTooManySubscribers
	}
	
	id := f.nextID
	f.nextID++
	
	if filter != nil {
		f.filters[id] = filter
	}
	
	sub := &Subscription{
		id:     id,
		feed:   f,
		events: make(chan *TxEvent, 256),
		quit:   make(chan struct{}),
	}
	
	// Reset reader position to current
	f.ring.Reset(id)
	
	// Start event delivery goroutine
	go sub.deliver()
	
	return sub, nil
}

// txToEvent converts a transaction to a fixed-size event.
func (f *TxFastFeed) txToEvent(tx *types.Transaction, eventType TxEventType) TxEvent {
	var event TxEvent
	
	// Hash
	copy(event.Hash[:], tx.Hash().Bytes())
	
	// From (will be filled by caller if available)
	// We don't compute sender here to avoid expensive ECDSA recovery
	
	// To
	if to := tx.To(); to != nil {
		copy(event.To[:], to.Bytes())
	}
	
	// Value
	if value := tx.Value(); value != nil {
		copy(event.Value[:], value.Bytes())
	}
	
	// Gas price
	if gasPrice := tx.GasPrice(); gasPrice != nil {
		copy(event.GasPrice[:], gasPrice.Bytes())
	}
	
	// Other fields
	event.Nonce = tx.Nonce()
	event.Gas = tx.Gas()
	event.Type = tx.Type()
	event.EventType = eventType
	event.Timestamp = uint64(time.Now().UnixNano())
	
	return event
}

// PublishWithSender publishes a transaction event with a known sender.
func (f *TxFastFeed) PublishWithSender(tx *types.Transaction, from common.Address, eventType TxEventType) {
	if !f.enabled.Load() {
		return
	}
	
	event := f.txToEvent(tx, eventType)
	copy(event.From[:], from.Bytes())
	
	eventPtr := unsafe.Pointer(&event)
	if !f.ring.Write(eventPtr) {
		f.eventsDropped.Add(1)
		log.Warn("Fast feed buffer full, event dropped", "hash", tx.Hash())
		return
	}
	
	f.eventsPublished.Add(1)
	f.lastPublish.Store(time.Now().UnixNano())
}

// Enable enables the fast feed.
func (f *TxFastFeed) Enable() {
	f.enabled.Store(true)
}

// Disable disables the fast feed.
func (f *TxFastFeed) Disable() {
	f.enabled.Store(false)
}

// Stats returns feed statistics.
type FeedStats struct {
	BufferStats     BufferStats
	EventsPublished uint64
	EventsDropped   uint64
	LastPublishNs   int64
	Subscribers     int
}

// Stats returns current feed statistics.
func (f *TxFastFeed) Stats() FeedStats {
	f.mu.RLock()
	subscribers := len(f.filters)
	f.mu.RUnlock()
	
	return FeedStats{
		BufferStats:     f.ring.Stats(),
		EventsPublished: f.eventsPublished.Load(),
		EventsDropped:   f.eventsDropped.Load(),
		LastPublishNs:   f.lastPublish.Load(),
		Subscribers:     subscribers,
	}
}

var ErrTooManySubscribers = errors.New("too many subscribers")

