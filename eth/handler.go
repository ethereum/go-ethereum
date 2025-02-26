// Copyright 2015 The go-ethereum Authors
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

package eth

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/lru"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/misc"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/eth/bft"
	"github.com/XinFinOrg/XDPoSChain/eth/downloader"
	"github.com/XinFinOrg/XDPoSChain/eth/fetcher"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header

	// txChanSize is the size of channel listening to NewTxsEvent.
	// The number is referenced from the size of tx pool.
	txChanSize = 4096
)

var (
	daoChallengeTimeout = 15 * time.Second // Time allowance for a node to reply to the DAO handshake challenge
)

// errIncompatibleConfig is returned if the requested protocols and configs are
// not compatible (low protocol version restrictions and high requirements).
var errIncompatibleConfig = errors.New("incompatible configuration")

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type ProtocolManager struct {
	networkId uint64

	fastSync  uint32 // Flag whether fast sync is enabled (gets disabled if we already have blocks)
	acceptTxs uint32 // Flag whether we're considered synchronised (enables transaction processing)

	txpool      txPool
	orderpool   orderPool
	lendingpool lendingPool
	blockchain  *core.BlockChain
	chainconfig *params.ChainConfig
	maxPeers    int

	downloader *downloader.Downloader
	fetcher    *fetcher.Fetcher
	peers      *peerSet
	bft        *bft.Bfter

	SubProtocols []p2p.Protocol

	eventMux      *event.TypeMux
	txsCh         chan core.NewTxsEvent
	orderTxCh     chan core.OrderTxPreEvent
	lendingTxCh   chan core.LendingTxPreEvent
	txsSub        event.Subscription
	orderTxSub    event.Subscription
	lendingTxSub  event.Subscription
	minedBlockSub *event.TypeMuxSubscription

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	txsyncCh    chan *txsync
	quitSync    chan struct{}
	noMorePeers chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg             sync.WaitGroup
	knownTxs       *lru.Cache[common.Hash, struct{}]
	knowOrderTxs   *lru.Cache[common.Hash, struct{}]
	knowLendingTxs *lru.Cache[common.Hash, struct{}]

	// V2 messages
	knownVotes     *lru.Cache[common.Hash, struct{}]
	knownSyncInfos *lru.Cache[common.Hash, struct{}]
	knownTimeouts  *lru.Cache[common.Hash, struct{}]
}

// NewProtocolManagerEx add order pool to protocol
func NewProtocolManagerEx(config *params.ChainConfig, mode downloader.SyncMode, networkID uint64, mux *event.TypeMux, txpool txPool, orderpool orderPool, lendingpool lendingPool, engine consensus.Engine, blockchain *core.BlockChain, chaindb ethdb.Database) (*ProtocolManager, error) {
	protocol, err := NewProtocolManager(config, mode, networkID, mux, txpool, engine, blockchain, chaindb)
	if err != nil {
		return nil, err
	}
	protocol.addOrderPoolProtocol(orderpool)
	protocol.addLendingPoolProtocol(lendingpool)
	return protocol, nil
}

// NewProtocolManager returns a new ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the ethereum network.
func NewProtocolManager(config *params.ChainConfig, mode downloader.SyncMode, networkID uint64, mux *event.TypeMux, txpool txPool, engine consensus.Engine, blockchain *core.BlockChain, chaindb ethdb.Database) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		networkId:      networkID,
		eventMux:       mux,
		txpool:         txpool,
		blockchain:     blockchain,
		chainconfig:    config,
		peers:          newPeerSet(),
		newPeerCh:      make(chan *peer),
		noMorePeers:    make(chan struct{}),
		txsyncCh:       make(chan *txsync),
		quitSync:       make(chan struct{}),
		knownTxs:       lru.NewCache[common.Hash, struct{}](maxKnownTxs),
		knowOrderTxs:   lru.NewCache[common.Hash, struct{}](maxKnownOrderTxs),
		knowLendingTxs: lru.NewCache[common.Hash, struct{}](maxKnownLendingTxs),
		knownVotes:     lru.NewCache[common.Hash, struct{}](maxKnownVote),
		knownSyncInfos: lru.NewCache[common.Hash, struct{}](maxKnownSyncInfo),
		knownTimeouts:  lru.NewCache[common.Hash, struct{}](maxKnownTimeout),
		orderpool:      nil,
		lendingpool:    nil,
		orderTxSub:     nil,
		lendingTxSub:   nil,
	}
	// Figure out whether to allow fast sync or not
	if mode == downloader.FastSync && blockchain.CurrentBlock().NumberU64() > 0 {
		log.Warn("Blockchain not empty, fast sync disabled")
		mode = downloader.FullSync
	}
	if mode == downloader.FastSync {
		manager.fastSync = uint32(1)
	}
	// Initiate a sub-protocol for every implemented version we can handle
	manager.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {
		// Skip protocol version if incompatible with the mode of operation
		if mode == downloader.FastSync && version < eth63 {
			continue
		}
		// Compatible; initialise the sub-protocol
		version := version // Closure for the run
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := manager.newPeer(int(version), p, rw)
				select {
				case manager.newPeerCh <- peer:
					manager.wg.Add(1)
					defer manager.wg.Done()
					return manager.handle(peer)
				case <-manager.quitSync:
					return p2p.DiscQuitting
				}
			},
			NodeInfo: func() interface{} {
				return manager.NodeInfo()
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := manager.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}
	if len(manager.SubProtocols) == 0 {
		return nil, errIncompatibleConfig
	}

	var handleProposedBlock func(header *types.Header) error
	if config.XDPoS != nil {
		handleProposedBlock = func(header *types.Header) error {
			return engine.(*XDPoS.XDPoS).HandleProposedBlock(blockchain, header)
		}
	} else {
		handleProposedBlock = func(header *types.Header) error {
			return nil
		}
	}

	// Construct the different synchronisation mechanisms
	manager.downloader = downloader.New(chaindb, manager.eventMux, blockchain, nil, manager.removePeer, handleProposedBlock)

	validator := func(header *types.Header) error {
		return engine.VerifyHeader(blockchain, header, true)
	}

	heighter := func() uint64 {
		return blockchain.CurrentBlock().NumberU64()
	}

	inserter := func(block *types.Block) error {
		// If fast sync is running, deny importing weird blocks
		if atomic.LoadUint32(&manager.fastSync) == 1 {
			log.Warn("Discarded bad propagated block", "number", block.Number(), "hash", block.Hash())
			return nil
		}
		atomic.StoreUint32(&manager.acceptTxs, 1) // Mark initial sync done on any fetcher import
		return manager.blockchain.InsertBlock(block)
	}

	prepare := func(block *types.Block) error {
		// If fast sync is running, deny importing weird blocks
		if atomic.LoadUint32(&manager.fastSync) == 1 {
			log.Warn("Discarded bad propagated block", "number", block.Number(), "hash", block.Hash())
			return nil
		}
		atomic.StoreUint32(&manager.acceptTxs, 1) // Mark initial sync done on any fetcher import
		return manager.blockchain.PrepareBlock(block)
	}
	manager.fetcher = fetcher.New(blockchain.GetBlockByHash, validator, handleProposedBlock, manager.BroadcastBlock, heighter, inserter, prepare, manager.removePeer)
	//Define bft function
	broadcasts := bft.BroadcastFns{
		Vote:     manager.BroadcastVote,
		Timeout:  manager.BroadcastTimeout,
		SyncInfo: manager.BroadcastSyncInfo,
	}
	manager.bft = bft.New(broadcasts, blockchain, heighter)
	if blockchain.Config().XDPoS != nil {
		manager.bft.InitEpochNumber()
		manager.bft.SetConsensusFuns(engine)
	}

	return manager, nil
}

func (pm *ProtocolManager) addOrderPoolProtocol(orderpool orderPool) {
	pm.orderpool = orderpool
}

func (pm *ProtocolManager) addLendingPoolProtocol(lendingpool lendingPool) {
	pm.lendingpool = lendingpool
}

func (pm *ProtocolManager) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debug("Removing Ethereum peer", "peer", id)

	// Unregister the peer from the downloader and Ethereum peer set
	pm.downloader.UnregisterPeer(id)
	if err := pm.peers.Unregister(id); err != nil {
		log.Debug("Peer removal failed", "peer", id, "err", err)
	}
	// Hard disconnect at the networking layer
	peer.Peer.Disconnect(p2p.DiscUselessPeer)
}

func (pm *ProtocolManager) Start(maxPeers int) {
	pm.maxPeers = maxPeers

	// broadcast transactions
	pm.txsCh = make(chan core.NewTxsEvent, txChanSize)
	pm.txsSub = pm.txpool.SubscribeNewTxsEvent(pm.txsCh)
	pm.orderTxCh = make(chan core.OrderTxPreEvent, txChanSize)
	if pm.orderpool != nil {
		pm.orderTxSub = pm.orderpool.SubscribeTxPreEvent(pm.orderTxCh)
	}
	pm.lendingTxCh = make(chan core.LendingTxPreEvent, txChanSize)
	if pm.lendingpool != nil {
		pm.lendingTxSub = pm.lendingpool.SubscribeTxPreEvent(pm.lendingTxCh)
	}
	go pm.txBroadcastLoop()
	go pm.orderTxBroadcastLoop()
	go pm.lendingTxBroadcastLoop()
	// broadcast mined blocks
	pm.minedBlockSub = pm.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go pm.minedBroadcastLoop()

	// start sync handlers
	go pm.syncer()
	go pm.txsyncLoop()
}

func (pm *ProtocolManager) Stop() {
	log.Info("Stopping Ethereum protocol")

	pm.txsSub.Unsubscribe() // quits txBroadcastLoop
	if pm.orderTxSub != nil {
		pm.orderTxSub.Unsubscribe()
	}
	if pm.lendingTxSub != nil {
		pm.lendingTxSub.Unsubscribe()
	}
	pm.minedBlockSub.Unsubscribe() // quits blockBroadcastLoop

	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	pm.noMorePeers <- struct{}{}

	// Quit fetcher, txsyncLoop.
	close(pm.quitSync)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pm.peers yet
	// will exit when they try to register.
	pm.peers.Close()

	// Wait for all peer handler goroutines and the loops to come down.
	pm.wg.Wait()

	log.Info("Ethereum protocol stopped")
}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
	// Ignore maxPeers if this is a trusted peer
	if pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("Ethereum peer connected", "name", p.Name())

	// Execute the Ethereum handshake
	var (
		genesis = pm.blockchain.Genesis()
		head    = pm.blockchain.CurrentHeader()
		hash    = head.Hash()
		number  = head.Number.Uint64()
		td      = pm.blockchain.GetTd(hash, number)
	)
	if err := p.Handshake(pm.networkId, td, hash, genesis.Hash()); err != nil {
		p.Log().Debug("Ethereum handshake failed", "err", err)
		return err
	}
	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}
	// Register the peer locally
	err := pm.peers.Register(p)
	if err != nil && err != p2p.ErrAddPairPeer {
		p.Log().Error("Ethereum peer registration failed", "err", err)
		return err
	}
	defer pm.removePeer(p.id)
	if err != p2p.ErrAddPairPeer {
		// Register the peer in the downloader. If the downloader considers it banned, we disconnect
		if err := pm.downloader.RegisterPeer(p.id, p.version, p); err != nil {
			return err
		}
		// Propagate existing transactions. new transactions appearing
		// after this will be sent via broadcasts.
		pm.syncTransactions(p)

		// If we're DAO hard-fork aware, validate any remote peer with regard to the hard-fork
		if daoBlock := pm.chainconfig.DAOForkBlock; daoBlock != nil {
			// Request the peer's DAO fork header for extra-data validation
			if err := p.RequestHeadersByNumber(daoBlock.Uint64(), 1, 0, false); err != nil {
				return err
			}
			// Start a timer to disconnect if the peer doesn't reply in time
			p.forkDrop = time.AfterFunc(daoChallengeTimeout, func() {
				p.Log().Debug("Timed out DAO fork-check, dropping")
				pm.removePeer(p.id)
			})
			// Make sure it's cleaned up if the peer dies off
			defer func() {
				if p.forkDrop != nil {
					p.forkDrop.Stop()
					p.forkDrop = nil
				}
			}()
		}
	}
	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("Ethereum message handling failed", "err", err)
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pm *ProtocolManager) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	// Handle the message depending on its contents
	switch {
	case msg.Code == StatusMsg:
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

		// Block header query, collect the requested headers and reply
	case msg.Code == GetBlockHeadersMsg:
		// Decode the complex header query
		var query getBlockHeadersData
		if err := msg.Decode(&query); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		hashMode := query.Origin.Hash != (common.Hash{})

		// Gather headers until the fetch or network limits is reached
		var (
			bytes   common.StorageSize
			headers []*types.Header
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {
			// Retrieve the next header satisfying the query
			var origin *types.Header
			if hashMode {
				origin = pm.blockchain.GetHeaderByHash(query.Origin.Hash)
			} else {
				origin = pm.blockchain.GetHeaderByNumber(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			number := origin.Number.Uint64()
			headers = append(headers, origin)
			bytes += estHeaderRlpSize

			// Advance to the next header of the query
			switch {
			case query.Origin.Hash != (common.Hash{}) && query.Reverse:
				// Hash based traversal towards the genesis block
				for i := 0; i < int(query.Skip)+1; i++ {
					if header := pm.blockchain.GetHeader(query.Origin.Hash, number); header != nil {
						query.Origin.Hash = header.ParentHash
						number--
					} else {
						unknown = true
						break
					}
				}
			case query.Origin.Hash != (common.Hash{}) && !query.Reverse:
				// Hash based traversal towards the leaf block
				var (
					current = origin.Number.Uint64()
					next    = current + query.Skip + 1
				)
				if next <= current {
					infos, _ := json.MarshalIndent(p.Peer.Info(), "", "  ")
					p.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
					unknown = true
				} else {
					if header := pm.blockchain.GetHeaderByNumber(next); header != nil {
						if pm.blockchain.GetBlockHashesFromHash(header.Hash(), query.Skip+1)[query.Skip] == query.Origin.Hash {
							query.Origin.Hash = header.Hash()
						} else {
							unknown = true
						}
					} else {
						unknown = true
					}
				}
			case query.Reverse:
				// Number based traversal towards the genesis block
				if query.Origin.Number >= query.Skip+1 {
					query.Origin.Number -= query.Skip + 1
				} else {
					unknown = true
				}

			case !query.Reverse:
				// Number based traversal towards the leaf block
				query.Origin.Number += query.Skip + 1
			}
		}
		return p.SendBlockHeaders(headers)

	case msg.Code == BlockHeadersMsg:
		// A batch of headers arrived to one of our previous requests
		var headers []*types.Header
		if err := msg.Decode(&headers); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// If no headers were received, but we're expending a DAO fork check, maybe it's that
		if len(headers) == 0 && p.forkDrop != nil {
			// Possibly an empty reply to the fork header checks, sanity check TDs
			verifyDAO := true

			// If we already have a DAO header, we can check the peer's TD against it. If
			// the peer's ahead of this, it too must have a reply to the DAO check
			if daoHeader := pm.blockchain.GetHeaderByNumber(pm.chainconfig.DAOForkBlock.Uint64()); daoHeader != nil {
				if _, td := p.Head(); td.Cmp(pm.blockchain.GetTd(daoHeader.Hash(), daoHeader.Number.Uint64())) >= 0 {
					verifyDAO = false
				}
			}
			// If we're seemingly on the same chain, disable the drop timer
			if verifyDAO {
				p.Log().Debug("Seems to be on the same side of the DAO fork")
				p.forkDrop.Stop()
				p.forkDrop = nil
				return nil
			}
		}
		// Filter out any explicitly requested headers, deliver the rest to the downloader
		filter := len(headers) == 1
		if filter {
			// If it's a potential DAO fork check, validate against the rules
			if p.forkDrop != nil && pm.chainconfig.DAOForkBlock.Cmp(headers[0].Number) == 0 {
				// Disable the fork drop timer
				p.forkDrop.Stop()
				p.forkDrop = nil

				// Validate the header and either drop the peer or continue
				if err := misc.VerifyDAOHeaderExtraData(pm.chainconfig, headers[0]); err != nil {
					p.Log().Debug("Verified to be on the other side of the DAO fork, dropping")
					return err
				}
				p.Log().Debug("Verified to be on the same side of the DAO fork")
				return nil
			}
			// Irrelevant of the fork checks, send the header to the fetcher just in case
			headers = pm.fetcher.FilterHeaders(p.id, headers, time.Now())
		}
		if len(headers) > 0 || !filter {
			err := pm.downloader.DeliverHeaders(p.id, headers)
			if err != nil {
				log.Debug("Failed to deliver headers", "err", err)
			}
		}

	case msg.Code == GetBlockBodiesMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather blocks until the fetch or network limits is reached
		var (
			hash   common.Hash
			bytes  int
			bodies []rlp.RawValue
		)
		for bytes < softResponseLimit && len(bodies) < downloader.MaxBlockFetch {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block body, stopping if enough was found
			if data := pm.blockchain.GetBodyRLP(hash); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)
			}
		}
		return p.SendBlockBodiesRLP(bodies)

	case msg.Code == BlockBodiesMsg:
		// A batch of block bodies arrived to one of our previous requests
		var request blockBodiesData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver them all to the downloader for queuing
		trasactions := make([][]*types.Transaction, len(request))
		uncles := make([][]*types.Header, len(request))

		for i, body := range request {
			trasactions[i] = body.Transactions
			uncles[i] = body.Uncles
		}
		// Filter out any explicitly requested bodies, deliver the rest to the downloader
		filter := len(trasactions) > 0 || len(uncles) > 0
		if filter {
			trasactions, uncles = pm.fetcher.FilterBodies(p.id, trasactions, uncles, time.Now())
		}
		if len(trasactions) > 0 || len(uncles) > 0 || !filter {
			err := pm.downloader.DeliverBodies(p.id, trasactions, uncles)
			if err != nil {
				log.Debug("Failed to deliver bodies", "err", err)
			}
		}

	case p.version >= eth63 && msg.Code == GetNodeDataMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash  common.Hash
			bytes int
			data  [][]byte
		)
		for bytes < softResponseLimit && len(data) < downloader.MaxStateFetch {
			// Retrieve the hash of the next state entry
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested state entry, stopping if enough was found
			if entry, err := pm.blockchain.TrieNode(hash); err == nil {
				data = append(data, entry)
				bytes += len(entry)
			}
		}
		return p.SendNodeData(data)

	case p.version >= eth63 && msg.Code == NodeDataMsg:
		// A batch of node state data arrived to one of our previous requests
		var data [][]byte
		if err := msg.Decode(&data); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver all to the downloader
		if err := pm.downloader.DeliverNodeData(p.id, data); err != nil {
			log.Debug("Failed to deliver node state data", "err", err)
		}

	case p.version >= eth63 && msg.Code == GetReceiptsMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather state data until the fetch or network limits is reached
		var (
			hash     common.Hash
			bytes    int
			receipts []rlp.RawValue
		)
		for bytes < softResponseLimit && len(receipts) < downloader.MaxReceiptFetch {
			// Retrieve the hash of the next block
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block's receipts, skipping if unknown to us
			results := pm.blockchain.GetReceiptsByHash(hash)
			if results == nil {
				if header := pm.blockchain.GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
					continue
				}
			}
			// If known, encode and queue for response packet
			if encoded, err := rlp.EncodeToBytes(results); err != nil {
				log.Error("Failed to encode receipt", "err", err)
			} else {
				receipts = append(receipts, encoded)
				bytes += len(encoded)
			}
		}
		return p.SendReceiptsRLP(receipts)

	case p.version >= eth63 && msg.Code == ReceiptsMsg:
		// A batch of receipts arrived to one of our previous requests
		var receipts [][]*types.Receipt
		if err := msg.Decode(&receipts); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Deliver all to the downloader
		if err := pm.downloader.DeliverReceipts(p.id, receipts); err != nil {
			log.Debug("Failed to deliver receipts", "err", err)
		}

	case msg.Code == NewBlockHashesMsg:
		var announces newBlockHashesData
		if err := msg.Decode(&announces); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		// Mark the hashes as present at the remote node
		for _, block := range announces {
			p.MarkBlock(block.Hash)
		}
		// Schedule all the unknown hashes for retrieval
		unknown := make(newBlockHashesData, 0, len(announces))
		for _, block := range announces {
			if !pm.blockchain.HasBlock(block.Hash, block.Number) {
				unknown = append(unknown, block)
			}
		}
		for _, block := range unknown {
			pm.fetcher.Notify(p.id, block.Hash, block.Number, time.Now(), p.RequestOneHeader, p.RequestBodies)
		}

	case msg.Code == NewBlockMsg:
		// Retrieve and decode the propagated block
		var request newBlockData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		request.Block.ReceivedAt = msg.ReceivedAt
		request.Block.ReceivedFrom = p

		// Mark the peer as owning the block and schedule it for import
		p.MarkBlock(request.Block.Hash())
		pm.fetcher.Enqueue(p.id, request.Block)

		// Assuming the block is importable by the peer, but possibly not yet done so,
		// calculate the head hash and TD that the peer truly must have.
		var (
			trueHead = request.Block.ParentHash()
			trueTD   = new(big.Int).Sub(request.TD, request.Block.Difficulty())
		)
		// Update the peers total difficulty if better than the previous
		if _, td := p.Head(); trueTD.Cmp(td) > 0 {
			p.SetHead(trueHead, trueTD)

			// Schedule a sync if above ours. Note, this will not fire a sync for a gap of
			// a singe block (as the true TD is below the propagated block), however this
			// scenario should easily be covered by the fetcher.
			currentBlock := pm.blockchain.CurrentBlock()
			if trueTD.Cmp(pm.blockchain.GetTd(currentBlock.Hash(), currentBlock.NumberU64())) > 0 {
				go pm.synchronise(p)
			}
		}

	case msg.Code == TxMsg:
		// Transactions arrived, make sure we have a valid and fresh chain to handle them
		if atomic.LoadUint32(&pm.acceptTxs) == 0 {
			break
		}
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			p.MarkTransaction(tx.Hash())
			if pm.knownTxs.Contains(tx.Hash()) {
				log.Trace("Discard known tx", "hash", tx.Hash(), "nonce", tx.Nonce(), "to", tx.To())
			} else {
				pm.knownTxs.Add(tx.Hash(), struct{}{})
			}

		}
		pm.txpool.AddRemotes(txs)

	case msg.Code == OrderTxMsg:
		// Transactions arrived, make sure we have a valid and fresh chain to handle them
		if atomic.LoadUint32(&pm.acceptTxs) == 0 {
			break
		}
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*types.OrderTransaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			p.MarkOrderTransaction(tx.Hash())
			if pm.knowOrderTxs.Contains(tx.Hash()) {
				log.Trace("Discard known tx", "hash", tx.Hash(), "nonce", tx.Nonce())
			} else {
				pm.knowOrderTxs.Add(tx.Hash(), struct{}{})
			}
		}

		if pm.orderpool != nil {
			pm.orderpool.AddRemotes(txs)
		}

	case msg.Code == LendingTxMsg:
		// Transactions arrived, make sure we have a valid and fresh chain to handle them
		if atomic.LoadUint32(&pm.acceptTxs) == 0 {
			break
		}
		// Transactions can be processed, parse all of them and deliver to the pool
		var txs []*types.LendingTransaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			p.MarkLendingTransaction(tx.Hash())
			if pm.knowLendingTxs.Contains(tx.Hash()) {
				log.Trace("Discard known tx", "hash", tx.Hash(), "nonce", tx.Nonce())
			} else {
				pm.knowLendingTxs.Add(tx.Hash(), struct{}{})
			}
		}

		if pm.lendingpool != nil {
			pm.lendingpool.AddRemotes(txs)
		}
	case msg.Code == VoteMsg:
		if pm.downloader.Synchronising() {
			break
		}

		var vote types.Vote
		if err := msg.Decode(&vote); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.MarkVote(vote.Hash())

		if pm.knownVotes.Contains(vote.Hash()) {
			log.Trace("Discarded vote, known vote", "vote hash", vote.Hash(), "voted block hash", vote.ProposedBlockInfo.Hash.Hex(), "number", vote.ProposedBlockInfo.Number, "round", vote.ProposedBlockInfo.Round)
		} else {
			pm.knownVotes.Add(vote.Hash(), struct{}{})
			go pm.bft.Vote(p.id, &vote)
		}

	case msg.Code == TimeoutMsg:
		if pm.downloader.Synchronising() {
			break
		}

		var timeout types.Timeout
		if err := msg.Decode(&timeout); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.MarkTimeout(timeout.Hash())

		if pm.knownTimeouts.Contains(timeout.Hash()) {
			log.Trace("Discarded Timeout, known Timeout", "Signature", timeout.Signature, "hash", timeout.Hash(), "round", timeout.Round)
		} else {
			pm.knownTimeouts.Add(timeout.Hash(), struct{}{})
			go pm.bft.Timeout(p.id, &timeout)
		}

	case msg.Code == SyncInfoMsg:
		if pm.downloader.Synchronising() {
			break
		}

		var syncInfo types.SyncInfo
		if err := msg.Decode(&syncInfo); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.MarkSyncInfo(syncInfo.Hash())

		if pm.knownSyncInfos.Contains(syncInfo.Hash()) {
			log.Trace("Discarded SyncInfo, known SyncInfo", "hash", syncInfo.Hash())
		} else {
			pm.knownSyncInfos.Add(syncInfo.Hash(), struct{}{})
			go pm.bft.SyncInfo(p.id, &syncInfo)
		}

	default:
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

// BroadcastBlock will either propagate a block to a subset of it's peers, or
// will only announce it's availability (depending what's requested).
func (pm *ProtocolManager) BroadcastBlock(block *types.Block, propagate bool) {
	hash := block.Hash()
	peers := pm.peers.PeersWithoutBlock(hash)

	// If propagation is requested, send to a subset of the peer
	if propagate {
		// Calculate the TD of the block (it's not imported yet, so block.Td is not valid)
		var td *big.Int
		if parent := pm.blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1); parent != nil {
			td = new(big.Int).Add(block.Difficulty(), pm.blockchain.GetTd(block.ParentHash(), block.NumberU64()-1))
		} else {
			log.Error("Propagating dangling block", "number", block.Number(), "hash", hash)
			return
		}
		// Send the block to a subset of our peers
		for _, peer := range peers {
			peer.SendNewBlock(block, td)
		}
		log.Trace("Propagated block", "hash", hash, "recipients", len(peers), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
		return
	}
	// Otherwise if the block is indeed in out own chain, announce it
	if pm.blockchain.HasBlock(hash, block.NumberU64()) {
		for _, peer := range peers {
			peer.SendNewBlockHashes([]common.Hash{hash}, []uint64{block.NumberU64()})
		}
		log.Trace("Announced block", "hash", hash, "recipients", len(peers), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
	}
}

// BroadcastTxs will propagate a batch of transactions to all peers which are not known to
// already have the given transaction.
func (pm *ProtocolManager) BroadcastTxs(txs types.Transactions) {
	var txset = make(map[*peer]types.Transactions)

	// Broadcast transactions to a batch of peers not knowing about it
	for _, tx := range txs {
		peers := pm.peers.PeersWithoutTx(tx.Hash())
		for _, peer := range peers {
			txset[peer] = append(txset[peer], tx)
		}
		log.Trace("Broadcast transaction", "hash", tx.Hash(), "recipients", len(peers))
	}
	// FIXME include this again: peers = peers[:int(math.Sqrt(float64(len(peers))))]
	for peer, txs := range txset {
		peer.SendTransactions(txs)
	}
}

// BroadcastVote will propagate a Vote to all peers which are not known to
// already have the given vote.
func (pm *ProtocolManager) BroadcastVote(vote *types.Vote) {
	hash := vote.Hash()
	peers := pm.peers.PeersWithoutVote(hash)
	if len(peers) > 0 {
		for _, peer := range peers {
			err := peer.SendVote(vote)
			if err != nil {
				log.Debug("[BroadcastVote] Fail to broadcast vote message", "peerId", peer.id, "version", peer.version, "blockNum", vote.ProposedBlockInfo.Number, "err", err)
				pm.removePeer(peer.id)
			}
		}
		log.Trace("Propagated Vote", "vote hash", vote.Hash(), "voted block hash", vote.ProposedBlockInfo.Hash.Hex(), "number", vote.ProposedBlockInfo.Number, "round", vote.ProposedBlockInfo.Round, "recipients", len(peers))
	}
}

// BroadcastTimeout will propagate a Timeout to all peers which are not known to
// already have the given timeout.
func (pm *ProtocolManager) BroadcastTimeout(timeout *types.Timeout) {
	hash := timeout.Hash()
	peers := pm.peers.PeersWithoutTimeout(hash)
	if len(peers) > 0 {
		for _, peer := range peers {
			err := peer.SendTimeout(timeout)
			if err != nil {
				log.Debug("[BroadcastTimeout] Fail to broadcast timeout message, remove peer", "peerId", peer.id, "version", peer.version, "timeout", timeout, "err", err)
				pm.removePeer(peer.id)
			}
		}
		log.Trace("Propagated Timeout", "hash", hash, "recipients", len(peers))
	}
}

// BroadcastSyncInfo will propagate a SyncInfo to all peers which are not known to
// already have the given SyncInfo.
func (pm *ProtocolManager) BroadcastSyncInfo(syncInfo *types.SyncInfo) {
	hash := syncInfo.Hash()
	peers := pm.peers.PeersWithoutSyncInfo(hash)
	if len(peers) > 0 {
		for _, peer := range peers {
			err := peer.SendSyncInfo(syncInfo)
			if err != nil {
				log.Debug("[BroadcastSyncInfo] Fail to broadcast syncInfo message, remove peer", "peerId", peer.id, "version", peer.version, "syncInfo", syncInfo, "err", err)
				pm.removePeer(peer.id)
			}
		}
		log.Trace("Propagated SyncInfo", "hash", hash, "recipients", len(peers))
	}

}

// OrderBroadcastTx will propagate a transaction to all peers which are not known to
// already have the given transaction.
func (pm *ProtocolManager) OrderBroadcastTx(hash common.Hash, tx *types.OrderTransaction) {
	// Broadcast transaction to a batch of peers not knowing about it
	peers := pm.peers.OrderPeersWithoutTx(hash)
	//FIXME include this again: peers = peers[:int(math.Sqrt(float64(len(peers))))]
	for _, peer := range peers {
		peer.SendOrderTransactions(types.OrderTransactions{tx})
	}
	log.Trace("Broadcast order transaction", "hash", hash, "recipients", len(peers))
}

// LendingBroadcastTx will propagate a transaction to all peers which are not known to
// already have the given transaction.
func (pm *ProtocolManager) LendingBroadcastTx(hash common.Hash, tx *types.LendingTransaction) {
	// Broadcast transaction to a batch of peers not knowing about it
	peers := pm.peers.LendingPeersWithoutTx(hash)
	//FIXME include this again: peers = peers[:int(math.Sqrt(float64(len(peers))))]
	for _, peer := range peers {
		peer.SendLendingTransactions(types.LendingTransactions{tx})
	}
	log.Trace("Broadcast lending transaction", "hash", hash, "recipients", len(peers))
}

// minedBroadcastLoop broadcast loop
func (pm *ProtocolManager) minedBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range pm.minedBlockSub.Chan() {
		switch ev := obj.Data.(type) {
		case core.NewMinedBlockEvent:
			pm.BroadcastBlock(ev.Block, true) // First propagate block to peers
			//self.BroadcastBlock(ev.Block, false) // Only then announce to the rest
		}
	}
}

func (pm *ProtocolManager) txBroadcastLoop() {
	for {
		select {
		case event := <-pm.txsCh:
			pm.BroadcastTxs(event.Txs)

		// Err() channel will be closed when unsubscribing.
		case <-pm.txsSub.Err():
			return
		}
	}
}

// orderTxBroadcastLoop broadcast order
func (pm *ProtocolManager) orderTxBroadcastLoop() {
	if pm.orderTxSub == nil {
		return
	}
	for {
		select {
		case event := <-pm.orderTxCh:
			pm.OrderBroadcastTx(event.Tx.Hash(), event.Tx)

			// Err() channel will be closed when unsubscribing.
		case <-pm.orderTxSub.Err():
			return
		}
	}
}

// lendingTxBroadcastLoop broadcast order
func (pm *ProtocolManager) lendingTxBroadcastLoop() {
	if pm.lendingTxSub == nil {
		return
	}
	for {
		select {
		case event := <-pm.lendingTxCh:
			pm.LendingBroadcastTx(event.Tx.Hash(), event.Tx)

			// Err() channel will be closed when unsubscribing.
		case <-pm.lendingTxSub.Err():
			return
		}
	}
}

// NodeInfo represents a short summary of the Ethereum sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network    uint64              `json:"network"`    // XDC network ID (50=xinfin, 51=apothem, 551=devnet)
	Difficulty *big.Int            `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    common.Hash         `json:"genesis"`    // SHA3 hash of the host's genesis block
	Config     *params.ChainConfig `json:"config"`     // Chain configuration for the fork rules
	Head       common.Hash         `json:"head"`       // SHA3 hash of the host's best owned block
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (pm *ProtocolManager) NodeInfo() *NodeInfo {
	currentBlock := pm.blockchain.CurrentBlock()
	return &NodeInfo{
		Network:    pm.networkId,
		Difficulty: pm.blockchain.GetTd(currentBlock.Hash(), currentBlock.NumberU64()),
		Genesis:    pm.blockchain.Genesis().Hash(),
		Config:     pm.blockchain.Config(),
		Head:       currentBlock.Hash(),
	}
}
