// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package p2p

import (
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	// ErrAddPairPeer is returned when adding a paired peer (not an error, signals pairing)
	ErrAddPairPeer = errors.New("add pair peer")
	
	// ErrMasternodeNotFound is returned when masternode is not in the list
	ErrMasternodeNotFound = errors.New("masternode not found")
)

// MasternodeConfig contains XDPoS masternode configuration
type MasternodeConfig struct {
	// Enable masternode mode
	Enabled bool
	
	// Masternode account address
	Address common.Address
	
	// Priority dial for masternodes
	PriorityDial bool
	
	// Max masternode peers
	MaxMasternodePeers int
}

// MasternodeManager manages masternode peer connections
type MasternodeManager struct {
	mu          sync.RWMutex
	masternodes map[common.Address]*enode.Node
	active      map[common.Address]bool
	config      *MasternodeConfig
}

// NewMasternodeManager creates a new masternode manager
func NewMasternodeManager(config *MasternodeConfig) *MasternodeManager {
	if config == nil {
		config = &MasternodeConfig{
			MaxMasternodePeers: 50,
		}
	}
	return &MasternodeManager{
		masternodes: make(map[common.Address]*enode.Node),
		active:      make(map[common.Address]bool),
		config:      config,
	}
}

// UpdateMasternodes updates the masternode list
func (m *MasternodeManager) UpdateMasternodes(nodes map[common.Address]*enode.Node) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Clear old list
	m.masternodes = make(map[common.Address]*enode.Node)
	
	// Add new masternodes
	for addr, node := range nodes {
		m.masternodes[addr] = node
	}
	
	log.Info("Updated masternode list", "count", len(m.masternodes))
}

// GetMasternodes returns the current masternode list
func (m *MasternodeManager) GetMasternodes() map[common.Address]*enode.Node {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	result := make(map[common.Address]*enode.Node)
	for addr, node := range m.masternodes {
		result[addr] = node
	}
	return result
}

// IsMasternode checks if an address is a masternode
func (m *MasternodeManager) IsMasternode(addr common.Address) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	_, exists := m.masternodes[addr]
	return exists
}

// SetActive marks a masternode as active/inactive
func (m *MasternodeManager) SetActive(addr common.Address, active bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.active[addr] = active
}

// IsActive checks if a masternode is active
func (m *MasternodeManager) IsActive(addr common.Address) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.active[addr]
}

// ActiveCount returns the count of active masternodes
func (m *MasternodeManager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	count := 0
	for _, active := range m.active {
		if active {
			count++
		}
	}
	return count
}

// GetMasternodeNode gets the enode for a masternode address
func (m *MasternodeManager) GetMasternodeNode(addr common.Address) *enode.Node {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.masternodes[addr]
}

// GetMasternodeAddresses returns all masternode addresses
func (m *MasternodeManager) GetMasternodeAddresses() []common.Address {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	addrs := make([]common.Address, 0, len(m.masternodes))
	for addr := range m.masternodes {
		addrs = append(addrs, addr)
	}
	return addrs
}

// PrioritizeMasternodes returns nodes that should be prioritized for connection
func (m *MasternodeManager) PrioritizeMasternodes() []*enode.Node {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	nodes := make([]*enode.Node, 0)
	for addr, node := range m.masternodes {
		if !m.active[addr] && node != nil {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// XDCServerConfig extends server config with XDPoS options
type XDCServerConfig struct {
	// Base server config
	Config
	
	// Masternode configuration
	Masternode *MasternodeConfig
	
	// Whether to accept non-masternode peers
	AcceptNonMasternode bool
}

// XDCDialer implements a dialer that prioritizes masternodes
type XDCDialer struct {
	manager *MasternodeManager
	dialer  NodeDialer
}

// NewXDCDialer creates a new XDC dialer
func NewXDCDialer(manager *MasternodeManager, dialer NodeDialer) *XDCDialer {
	return &XDCDialer{
		manager: manager,
		dialer:  dialer,
	}
}

// Dial attempts to dial a node, prioritizing masternodes
func (d *XDCDialer) Dial(dest *enode.Node) (Conn, error) {
	// TODO: Implement priority dialing for masternodes
	return d.dialer.Dial(dest)
}

// PeerHook is called when a peer connects or disconnects
type PeerHook func(peer *Peer, added bool)

// XDCPeerHooks contains hooks for XDPoS peer events
type XDCPeerHooks struct {
	OnConnect    PeerHook
	OnDisconnect PeerHook
}

// MasternodePeerInfo contains masternode-specific peer info
type MasternodePeerInfo struct {
	Address     common.Address `json:"address"`
	IsMaster    bool           `json:"isMaster"`
	Epoch       uint64         `json:"epoch"`
	IsValidator bool           `json:"isValidator"`
}

// GetMasternodePeerInfo extracts masternode info from a peer
func GetMasternodePeerInfo(peer *Peer) *MasternodePeerInfo {
	// This would be extracted from peer handshake data
	return &MasternodePeerInfo{
		IsMaster: false,
	}
}
