// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package p2p

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// XDCDialScheduler extends the dial scheduler with masternode prioritization
type XDCDialScheduler struct {
	mu              sync.Mutex
	masternodeNodes map[enode.ID]*enode.Node
	priorityQueue   []*enode.Node
	dialer          NodeDialer
	maxDialing      int
	dialingCount    int
}

// NewXDCDialScheduler creates a new XDC dial scheduler
func NewXDCDialScheduler(dialer NodeDialer, maxDialing int) *XDCDialScheduler {
	return &XDCDialScheduler{
		masternodeNodes: make(map[enode.ID]*enode.Node),
		priorityQueue:   make([]*enode.Node, 0),
		dialer:          dialer,
		maxDialing:      maxDialing,
	}
}

// SetMasternodes sets the current masternode list
func (s *XDCDialScheduler) SetMasternodes(nodes []*enode.Node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.masternodeNodes = make(map[enode.ID]*enode.Node)
	s.priorityQueue = make([]*enode.Node, 0, len(nodes))
	
	for _, node := range nodes {
		s.masternodeNodes[node.ID()] = node
		s.priorityQueue = append(s.priorityQueue, node)
	}
	
	log.Debug("Updated dial scheduler masternode list", "count", len(nodes))
}

// IsMasternode checks if a node ID is a masternode
func (s *XDCDialScheduler) IsMasternode(id enode.ID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	_, exists := s.masternodeNodes[id]
	return exists
}

// GetPriorityNode returns the next priority node to dial
func (s *XDCDialScheduler) GetPriorityNode() *enode.Node {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if len(s.priorityQueue) == 0 {
		return nil
	}
	
	node := s.priorityQueue[0]
	s.priorityQueue = s.priorityQueue[1:]
	return node
}

// AddPriorityNode adds a node to the priority queue
func (s *XDCDialScheduler) AddPriorityNode(node *enode.Node) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Don't add duplicates
	for _, n := range s.priorityQueue {
		if n.ID() == node.ID() {
			return
		}
	}
	
	s.priorityQueue = append(s.priorityQueue, node)
}

// DialContext dials a node with context
func (s *XDCDialScheduler) DialContext(ctx context.Context, node *enode.Node) (Conn, error) {
	s.mu.Lock()
	if s.dialingCount >= s.maxDialing {
		s.mu.Unlock()
		return nil, errTooManyPeers
	}
	s.dialingCount++
	s.mu.Unlock()
	
	defer func() {
		s.mu.Lock()
		s.dialingCount--
		s.mu.Unlock()
	}()
	
	return s.dialer.Dial(node)
}

var errTooManyPeers = &DiscReason{DiscTooManyPeers}

// MasternodeDialer dials masternodes with priority
type MasternodeDialer struct {
	scheduler *XDCDialScheduler
	interval  time.Duration
	quit      chan struct{}
}

// NewMasternodeDialer creates a new masternode dialer
func NewMasternodeDialer(scheduler *XDCDialScheduler, interval time.Duration) *MasternodeDialer {
	return &MasternodeDialer{
		scheduler: scheduler,
		interval:  interval,
		quit:      make(chan struct{}),
	}
}

// Start starts the masternode dialing loop
func (d *MasternodeDialer) Start() {
	go d.loop()
}

// Stop stops the masternode dialer
func (d *MasternodeDialer) Stop() {
	close(d.quit)
}

func (d *MasternodeDialer) loop() {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			d.dialNext()
		case <-d.quit:
			return
		}
	}
}

func (d *MasternodeDialer) dialNext() {
	node := d.scheduler.GetPriorityNode()
	if node == nil {
		return
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	_, err := d.scheduler.DialContext(ctx, node)
	if err != nil {
		log.Debug("Failed to dial masternode", "id", node.ID(), "err", err)
		// Re-add to queue for retry
		d.scheduler.AddPriorityNode(node)
	}
}

// PeerPriority defines peer priority levels
type PeerPriority int

const (
	PriorityNormal PeerPriority = iota
	PriorityMasternode
	PriorityValidator
)

// XDCPeerSelector selects peers based on priority
type XDCPeerSelector struct {
	mu       sync.RWMutex
	peers    map[enode.ID]PeerPriority
	maxPeers int
}

// NewXDCPeerSelector creates a new peer selector
func NewXDCPeerSelector(maxPeers int) *XDCPeerSelector {
	return &XDCPeerSelector{
		peers:    make(map[enode.ID]PeerPriority),
		maxPeers: maxPeers,
	}
}

// SetPriority sets the priority for a peer
func (s *XDCPeerSelector) SetPriority(id enode.ID, priority PeerPriority) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.peers[id] = priority
}

// GetPriority gets the priority for a peer
func (s *XDCPeerSelector) GetPriority(id enode.ID) PeerPriority {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.peers[id]
}

// ShouldConnect determines if we should connect to a peer
func (s *XDCPeerSelector) ShouldConnect(id enode.ID, priority PeerPriority) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Always accept masternodes and validators
	if priority >= PriorityMasternode {
		return true
	}
	
	// Count by priority
	normalCount := 0
	for _, p := range s.peers {
		if p == PriorityNormal {
			normalCount++
		}
	}
	
	// Reserve slots for high priority peers
	reservedSlots := s.maxPeers / 3
	maxNormal := s.maxPeers - reservedSlots
	
	return normalCount < maxNormal
}

// RemovePeer removes a peer from the selector
func (s *XDCPeerSelector) RemovePeer(id enode.ID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.peers, id)
}

// GetMasternodePeers returns all masternode peers
func (s *XDCPeerSelector) GetMasternodePeers() []enode.ID {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []enode.ID
	for id, priority := range s.peers {
		if priority >= PriorityMasternode {
			result = append(result, id)
		}
	}
	return result
}

// MasternodeAddressToNodeID converts a masternode address to enode ID
// This is a placeholder - actual implementation would use the masternode registry
func MasternodeAddressToNodeID(addr common.Address) enode.ID {
	// In practice, this would look up the enode from the masternode registry
	return enode.ID{}
}
