// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package eth

import (
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

// XDCHandler extends the base handler with XDPoS-specific functionality
type XDCHandler struct {
	networkID   uint64
	txpool      TxPool
	orderpool   OrderPool
	lendingpool LendingPool
	chain       *core.BlockChain
	maxPeers    int

	// Accept transactions flag
	acceptTxs uint32

	// XDPoS peer management
	xdcPeers *xdcPeerSet

	// Event subscriptions
	orderTxCh    chan core.OrderTxPreEvent
	lendingTxCh  chan core.LendingTxPreEvent
	orderTxSub   event.Subscription
	lendingTxSub event.Subscription

	// Vote and consensus message channels
	voteCh     chan *types.Vote
	timeoutCh  chan *types.Timeout
	syncInfoCh chan *types.SyncInfo

	// Quit channel
	quitSync chan struct{}
}

// TxPool interface for transaction pool
type TxPool interface {
	Pending(enforceTips bool) map[common.Address][]*types.Transaction
	SubscribeNewTxsEvent(chan<- core.NewTxsEvent) event.Subscription
	AddRemotes([]*types.Transaction) []error
}

// NewXDCHandler creates a new XDC protocol handler
func NewXDCHandler(config *HandlerConfig) (*XDCHandler, error) {
	h := &XDCHandler{
		networkID:  config.Network,
		chain:      config.Chain,
		txpool:     config.TxPool,
		maxPeers:   config.MaxPeers,
		xdcPeers:   newXDCPeerSet(),
		orderTxCh:  make(chan core.OrderTxPreEvent, 4096),
		lendingTxCh: make(chan core.LendingTxPreEvent, 4096),
		voteCh:     make(chan *types.Vote, 4096),
		timeoutCh:  make(chan *types.Timeout, 4096),
		syncInfoCh: make(chan *types.SyncInfo, 4096),
		quitSync:   make(chan struct{}),
	}
	return h, nil
}

// HandlerConfig contains configuration for the handler
type HandlerConfig struct {
	Network  uint64
	Chain    *core.BlockChain
	TxPool   TxPool
	MaxPeers int
}

// Start starts the XDC handler
func (h *XDCHandler) Start(maxPeers int) {
	h.maxPeers = maxPeers
	atomic.StoreUint32(&h.acceptTxs, 1)

	// Start broadcast loops
	go h.orderTxBroadcastLoop()
	go h.lendingTxBroadcastLoop()
	go h.consensusMsgLoop()
}

// Stop stops the XDC handler
func (h *XDCHandler) Stop() {
	close(h.quitSync)
	h.xdcPeers.Close()
}

// SetOrderPool sets the order pool
func (h *XDCHandler) SetOrderPool(orderpool OrderPool) {
	h.orderpool = orderpool
	if orderpool != nil {
		h.orderTxSub = orderpool.SubscribeTxPreEvent(h.orderTxCh)
	}
}

// SetLendingPool sets the lending pool
func (h *XDCHandler) SetLendingPool(lendingpool LendingPool) {
	h.lendingpool = lendingpool
	if lendingpool != nil {
		h.lendingTxSub = lendingpool.SubscribeTxPreEvent(h.lendingTxCh)
	}
}

// HandleMsg handles an incoming message from a peer
func (h *XDCHandler) HandleMsg(peer *p2p.Peer, rw p2p.MsgReadWriter, msg p2p.Msg) error {
	switch msg.Code {
	case OrderTxMsgCode:
		return h.handleOrderTxMsg(peer, msg)
	case LendingTxMsgCode:
		return h.handleLendingTxMsg(peer, msg)
	case VoteMsgCode:
		return h.handleVoteMsg(peer, msg)
	case TimeoutMsgCode:
		return h.handleTimeoutMsg(peer, msg)
	case SyncInfoMsgCode:
		return h.handleSyncInfoMsg(peer, msg)
	}
	return nil
}

// handleOrderTxMsg handles order transaction messages
func (h *XDCHandler) handleOrderTxMsg(peer *p2p.Peer, msg p2p.Msg) error {
	if atomic.LoadUint32(&h.acceptTxs) == 0 {
		return nil
	}

	var txs []*types.OrderTransaction
	if err := msg.Decode(&txs); err != nil {
		return err
	}

	if h.orderpool != nil {
		h.orderpool.AddRemotes(txs)
	}

	return nil
}

// handleLendingTxMsg handles lending transaction messages
func (h *XDCHandler) handleLendingTxMsg(peer *p2p.Peer, msg p2p.Msg) error {
	if atomic.LoadUint32(&h.acceptTxs) == 0 {
		return nil
	}

	var txs []*types.LendingTransaction
	if err := msg.Decode(&txs); err != nil {
		return err
	}

	if h.lendingpool != nil {
		h.lendingpool.AddRemotes(txs)
	}

	return nil
}

// handleVoteMsg handles vote messages
func (h *XDCHandler) handleVoteMsg(peer *p2p.Peer, msg p2p.Msg) error {
	var vote types.Vote
	if err := msg.Decode(&vote); err != nil {
		return err
	}

	select {
	case h.voteCh <- &vote:
	default:
		log.Warn("Vote channel full, dropping vote")
	}

	return nil
}

// handleTimeoutMsg handles timeout messages
func (h *XDCHandler) handleTimeoutMsg(peer *p2p.Peer, msg p2p.Msg) error {
	var timeout types.Timeout
	if err := msg.Decode(&timeout); err != nil {
		return err
	}

	select {
	case h.timeoutCh <- &timeout:
	default:
		log.Warn("Timeout channel full, dropping timeout")
	}

	return nil
}

// handleSyncInfoMsg handles sync info messages
func (h *XDCHandler) handleSyncInfoMsg(peer *p2p.Peer, msg p2p.Msg) error {
	var syncInfo types.SyncInfo
	if err := msg.Decode(&syncInfo); err != nil {
		return err
	}

	select {
	case h.syncInfoCh <- &syncInfo:
	default:
		log.Warn("SyncInfo channel full, dropping syncInfo")
	}

	return nil
}

// orderTxBroadcastLoop broadcasts order transactions
func (h *XDCHandler) orderTxBroadcastLoop() {
	if h.orderTxSub == nil {
		return
	}
	for {
		select {
		case event := <-h.orderTxCh:
			h.BroadcastOrderTx(event.Tx)
		case <-h.orderTxSub.Err():
			return
		case <-h.quitSync:
			return
		}
	}
}

// lendingTxBroadcastLoop broadcasts lending transactions
func (h *XDCHandler) lendingTxBroadcastLoop() {
	if h.lendingTxSub == nil {
		return
	}
	for {
		select {
		case event := <-h.lendingTxCh:
			h.BroadcastLendingTx(event.Tx)
		case <-h.lendingTxSub.Err():
			return
		case <-h.quitSync:
			return
		}
	}
}

// consensusMsgLoop handles consensus messages
func (h *XDCHandler) consensusMsgLoop() {
	for {
		select {
		case vote := <-h.voteCh:
			// Forward to consensus engine
			log.Debug("Processing vote", "hash", vote.Hash())
		case timeout := <-h.timeoutCh:
			// Forward to consensus engine
			log.Debug("Processing timeout", "round", timeout.Round)
		case syncInfo := <-h.syncInfoCh:
			// Forward to consensus engine
			log.Debug("Processing syncInfo")
			_ = syncInfo
		case <-h.quitSync:
			return
		}
	}
}

// BroadcastOrderTx broadcasts an order transaction to peers
func (h *XDCHandler) BroadcastOrderTx(tx *types.OrderTransaction) {
	hash := tx.GetHash()
	peers := h.xdcPeers.PeersWithoutTx(hash)
	for _, peer := range peers {
		peer.MarkOrderTransaction(hash)
	}
	log.Trace("Broadcast order transaction", "hash", hash, "recipients", len(peers))
}

// BroadcastLendingTx broadcasts a lending transaction to peers
func (h *XDCHandler) BroadcastLendingTx(tx *types.LendingTransaction) {
	hash := tx.Hash()
	peers := h.xdcPeers.PeersWithoutTx(hash)
	for _, peer := range peers {
		peer.MarkLendingTransaction(hash)
	}
	log.Trace("Broadcast lending transaction", "hash", hash, "recipients", len(peers))
}

// BroadcastVote broadcasts a vote to peers
func (h *XDCHandler) BroadcastVote(vote *types.Vote) {
	hash := vote.Hash()
	peers := h.xdcPeers.PeersWithoutVote(hash)
	for _, peer := range peers {
		peer.MarkVote(hash)
	}
	log.Debug("Broadcast vote",
		"hash", hash,
		"blockHash", vote.ProposedBlockInfo.Hash,
		"recipients", len(peers),
	)
}

// BroadcastTimeout broadcasts a timeout to peers
func (h *XDCHandler) BroadcastTimeout(timeout *types.Timeout) {
	hash := timeout.Hash()
	peers := h.xdcPeers.PeersWithoutTimeout(hash)
	for _, peer := range peers {
		peer.MarkTimeout(hash)
	}
	log.Debug("Broadcast timeout", "round", timeout.Round, "recipients", len(peers))
}

// BroadcastSyncInfo broadcasts sync info to peers
func (h *XDCHandler) BroadcastSyncInfo(syncInfo *types.SyncInfo) {
	hash := syncInfo.Hash()
	peers := h.xdcPeers.PeersWithoutSyncInfo(hash)
	for _, peer := range peers {
		peer.MarkSyncInfo(hash)
	}
	log.Debug("Broadcast syncInfo", "recipients", len(peers))
}

// BroadcastBlock broadcasts a new block to peers
func (h *XDCHandler) BroadcastBlock(block *types.Block, propagate bool) {
	hash := block.Hash()
	peers := h.xdcPeers.PeersWithoutBlock(hash)

	if propagate {
		td := h.chain.GetTd(block.ParentHash(), block.NumberU64()-1)
		if td == nil {
			log.Error("Propagating block with unknown parent", "number", block.Number(), "hash", hash)
			return
		}
		td = new(big.Int).Add(td, block.Difficulty())

		for _, peer := range peers {
			peer.MarkBlock(hash)
		}
		log.Debug("Propagated block", "hash", hash, "recipients", len(peers),
			"duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
	}
}

// NodeInfo represents XDC node information
type XDCNodeInfo struct {
	Network    uint64      `json:"network"`
	Difficulty *big.Int    `json:"difficulty"`
	Genesis    common.Hash `json:"genesis"`
	Head       common.Hash `json:"head"`
	Epoch      uint64      `json:"epoch"`
}

// NodeInfo returns XDC-specific node information
func (h *XDCHandler) NodeInfo() *XDCNodeInfo {
	currentBlock := h.chain.CurrentBlock()
	return &XDCNodeInfo{
		Network:    h.networkID,
		Difficulty: h.chain.GetTd(currentBlock.Hash(), currentBlock.NumberU64()),
		Genesis:    h.chain.Genesis().Hash(),
		Head:       currentBlock.Hash(),
	}
}
