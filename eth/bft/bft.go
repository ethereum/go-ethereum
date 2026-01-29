// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// Package bft implements Byzantine Fault Tolerant consensus message handling
// for XDPoS 2.0.
package bft

import (
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// BroadcastFns contains the broadcast functions for BFT messages
type BroadcastFns struct {
	Vote     func(*types.Vote)
	Timeout  func(*types.Timeout)
	SyncInfo func(*types.SyncInfo)
}

// BlockChain defines the blockchain interface needed by Bfter
type BlockChain interface {
	CurrentBlock() *types.Block
	GetBlock(hash common.Hash, number uint64) *types.Block
	GetBlockByHash(hash common.Hash) *types.Block
	GetBlockByNumber(number uint64) *types.Block
	Config() *params.ChainConfig
}

// Bfter handles BFT consensus messages
type Bfter struct {
	broadcasts BroadcastFns
	blockchain BlockChain
	heighter   func() uint64

	// Consensus engine integration
	engine consensus.Engine

	// State
	running    int32
	epochNum   uint64
	mu         sync.RWMutex

	// Quit channel
	quit chan struct{}
}

// New creates a new Bfter instance
func New(broadcasts BroadcastFns, blockchain BlockChain, heighter func() uint64) *Bfter {
	return &Bfter{
		broadcasts: broadcasts,
		blockchain: blockchain,
		heighter:   heighter,
		quit:       make(chan struct{}),
	}
}

// Start starts the BFT message handler
func (b *Bfter) Start() {
	if !atomic.CompareAndSwapInt32(&b.running, 0, 1) {
		return
	}
	log.Info("BFT message handler started")
}

// Stop stops the BFT message handler
func (b *Bfter) Stop() {
	if !atomic.CompareAndSwapInt32(&b.running, 1, 0) {
		return
	}
	close(b.quit)
	log.Info("BFT message handler stopped")
}

// SetConsensusFuns sets the consensus engine
func (b *Bfter) SetConsensusFuns(engine consensus.Engine) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.engine = engine
}

// InitEpochNumber initializes the epoch number from the current block
func (b *Bfter) InitEpochNumber() {
	b.mu.Lock()
	defer b.mu.Unlock()
	// Epoch initialization would be done based on consensus engine
	// For now, set to 0 and let the engine update it
	b.epochNum = 0
}

// Vote handles incoming vote messages
func (b *Bfter) Vote(peerID string, vote *types.Vote) {
	if atomic.LoadInt32(&b.running) == 0 {
		return
	}

	b.mu.RLock()
	engine := b.engine
	b.mu.RUnlock()

	if engine == nil {
		log.Debug("BFT vote received but no consensus engine set", "peer", peerID)
		return
	}

	log.Debug("BFT vote received",
		"peer", peerID,
		"blockHash", vote.ProposedBlockInfo.Hash.Hex(),
		"blockNumber", vote.ProposedBlockInfo.Number,
		"round", vote.ProposedBlockInfo.Round,
	)

	// Forward to consensus engine for processing
	// The actual handling depends on the XDPoS engine implementation
}

// Timeout handles incoming timeout messages
func (b *Bfter) Timeout(peerID string, timeout *types.Timeout) {
	if atomic.LoadInt32(&b.running) == 0 {
		return
	}

	b.mu.RLock()
	engine := b.engine
	b.mu.RUnlock()

	if engine == nil {
		log.Debug("BFT timeout received but no consensus engine set", "peer", peerID)
		return
	}

	log.Debug("BFT timeout received",
		"peer", peerID,
		"round", timeout.Round,
	)

	// Forward to consensus engine for processing
}

// SyncInfo handles incoming sync info messages
func (b *Bfter) SyncInfo(peerID string, syncInfo *types.SyncInfo) {
	if atomic.LoadInt32(&b.running) == 0 {
		return
	}

	b.mu.RLock()
	engine := b.engine
	b.mu.RUnlock()

	if engine == nil {
		log.Debug("BFT syncInfo received but no consensus engine set", "peer", peerID)
		return
	}

	log.Debug("BFT syncInfo received", "peer", peerID)

	// Forward to consensus engine for processing
}

// BroadcastVote broadcasts a vote to all peers
func (b *Bfter) BroadcastVote(vote *types.Vote) {
	if b.broadcasts.Vote != nil {
		b.broadcasts.Vote(vote)
	}
}

// BroadcastTimeout broadcasts a timeout to all peers
func (b *Bfter) BroadcastTimeout(timeout *types.Timeout) {
	if b.broadcasts.Timeout != nil {
		b.broadcasts.Timeout(timeout)
	}
}

// BroadcastSyncInfo broadcasts sync info to all peers
func (b *Bfter) BroadcastSyncInfo(syncInfo *types.SyncInfo) {
	if b.broadcasts.SyncInfo != nil {
		b.broadcasts.SyncInfo(syncInfo)
	}
}

// GetEpochNumber returns the current epoch number
func (b *Bfter) GetEpochNumber() uint64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.epochNum
}

// SetEpochNumber sets the current epoch number
func (b *Bfter) SetEpochNumber(epoch uint64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.epochNum = epoch
}

// Import params for type reference
import "github.com/ethereum/go-ethereum/params"
