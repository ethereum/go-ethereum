// Copyright 2019 The go-ethereum Authors
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

package les

import (
	"crypto/ecdsa"
	"errors"

	//"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/les/flowcontrol"
	vfs "github.com/ethereum/go-ethereum/les/vflux/server"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header

	MaxHeaderFetch           = 192 // Amount of block headers to be fetched per retrieval request
	MaxBodyFetch             = 32  // Amount of block bodies to be fetched per retrieval request
	MaxReceiptFetch          = 128 // Amount of transaction receipts to allow fetching per request
	MaxCodeFetch             = 64  // Amount of contract codes to allow fetching per request
	MaxProofsFetch           = 64  // Amount of merkle proofs to be fetched per retrieval request
	MaxHelperTrieProofsFetch = 64  // Amount of helper tries to be fetched per retrieval request
	MaxTxSend                = 64  // Amount of transactions to be send per request
	MaxTxStatus              = 256 // Amount of transactions to queried per request
)

var (
	errTooManyInvalidRequest = errors.New("too many invalid requests made")
)

// serverHandler serves general LES requests (not including beacon chain-related ones)
type serverHandler struct {
	forkFilter forkid.Filter
	blockchain *core.BlockChain
	chainDb    ethdb.Database
	txpool     *core.TxPool
	server     *LesServer
	fcWrapper  *fcRequestWrapper

	privateKey                   *ecdsa.PrivateKey
	lastAnnounce, signedAnnounce announceData

	// Testing fields
	synced     func() bool // Callback function used to determine whether local node is synced.
	addTxsSync bool
}

// newServerHandler returns a new serverHandler
func newServerHandler(server *LesServer, blockchain *core.BlockChain, chainDb ethdb.Database, txpool *core.TxPool, fcWrapper *fcRequestWrapper, synced func() bool) *serverHandler {
	handler := &serverHandler{
		forkFilter: forkid.NewFilter(blockchain),
		server:     server,
		blockchain: blockchain,
		chainDb:    chainDb,
		txpool:     txpool,
		synced:     synced,
		fcWrapper:  fcWrapper,
	}
	return handler
}

// start implements auxModule
func (h *serverHandler) start(wg *sync.WaitGroup, closeCh chan struct{}) {
	if h.server.p2pSrv != nil {
		h.privateKey = h.server.p2pSrv.PrivateKey
	}
	wg.Add(1)
	go func() {
		defer wg.Done()

		headCh := make(chan core.ChainHeadEvent, 10)
		headSub := h.blockchain.SubscribeChainHeadEvent(headCh)
		defer headSub.Unsubscribe()

		var (
			lastHead = h.blockchain.CurrentHeader()
			lastTd   = common.Big0
		)
		for {
			select {
			case ev := <-headCh:
				header := ev.Block.Header()
				hash, number := header.Hash(), header.Number.Uint64()
				td := h.blockchain.GetTd(hash, number)
				if td == nil || td.Cmp(lastTd) <= 0 {
					continue
				}
				var reorg uint64
				if lastHead != nil {
					// If a setHead has been performed, the common ancestor can be nil.
					if ancestor := rawdb.FindCommonAncestor(h.chainDb, header, lastHead); ancestor != nil {
						reorg = lastHead.Number.Uint64() - ancestor.Number.Uint64()
					}
				}
				lastHead, lastTd = header, td
				log.Debug("Announcing block to peers", "number", number, "hash", hash, "td", td, "reorg", reorg)
				h.broadcast(announceData{Hash: hash, Number: number, Td: td, ReorgDepth: reorg})
			case <-closeCh:
				return
			}
		}
	}()
}

// broadcast broadcasts legacy PoW head announcements
func (h *serverHandler) broadcast(announce announceData) {
	h.lastAnnounce = announce
	for _, peer := range h.server.peers.allPeers() {
		h.announceOrStore(peer)
	}
}

// announceOrStore sends the requested type of announcement to the given peer or stores
// it for later if the peer is inactive (capacity == 0).
func (h *serverHandler) announceOrStore(p *peer) {
	if h.lastAnnounce.Td == nil {
		return
	}
	switch p.announceType {
	case announceTypeSimple:
		p.announceOrStore(h.lastAnnounce)
	case announceTypeSigned:
		if h.signedAnnounce.Hash != h.lastAnnounce.Hash {
			h.signedAnnounce = h.lastAnnounce
			h.signedAnnounce.sign(h.privateKey)
		}
		p.announceOrStore(h.signedAnnounce)
	}
}

// sendHandshake implements handshakeModule
func (h *serverHandler) sendHandshake(p *peer, send *keyValueList) {
	// Note: peer.headInfo should contain the last head announced to the client by us.
	// The values announced by the client in the handshake are dummy values for compatibility reasons and should be ignored.
	head := h.blockchain.CurrentHeader()
	hash := head.Hash()
	number := head.Number.Uint64()
	p.headInfo = blockInfo{Hash: hash, Number: number, Td: h.blockchain.GetTd(hash, number)}
	sendHeadInfo(send, p.headInfo)
	sendGeneralInfo(p, send, h.blockchain.Genesis().Hash(), forkid.NewID(h.blockchain.Config(), h.blockchain.Genesis().Hash(), number))

	recentTx := h.blockchain.TxLookupLimit()
	if recentTx != txIndexUnlimited {
		if recentTx < blockSafetyMargin {
			recentTx = txIndexDisabled
		} else {
			recentTx -= blockSafetyMargin - txIndexRecentOffset
		}
	}

	// Add some information which services server can offer.
	send.add("serveHeaders", nil)
	send.add("serveChainSince", uint64(0))
	send.add("serveStateSince", uint64(0))

	// If local ethereum node is running in archive mode, advertise ourselves we have
	// all version state data. Otherwise only recent state is available.
	stateRecent := uint64(core.TriesInMemory - blockSafetyMargin)
	if h.server.archiveMode {
		stateRecent = 0
	}
	send.add("serveRecentState", stateRecent)
	send.add("txRelay", nil)
	if p.version >= lpv4 {
		send.add("recentTxLookup", recentTx)
	}

	// Add advertised checkpoint and register block height which
	// client can verify the checkpoint validity.
	if h.server.oracle != nil && h.server.oracle.IsRunning() {
		cp, height := h.server.oracle.StableCheckpoint()
		if cp != nil {
			send.add("checkpoint/value", cp)
			send.add("checkpoint/registerHeight", height)
		}
	}
}

// receiveHandshake implements handshakeModule
func (h *serverHandler) receiveHandshake(p *peer, recv keyValueMap) error {
	if err := receiveGeneralInfo(p, recv, h.blockchain.Genesis().Hash(), h.forkFilter); err != nil {
		return err
	}

	p.server = recv.get("serveHeaders", nil) == nil
	if p.server {
		p.announceType = announceTypeNone // connected to another server, send no messages
	} else {
		if recv.get("announceType", &p.announceType) != nil {
			// set default announceType on server side
			p.announceType = announceTypeSimple
		}
	}
	return nil
}

// peerConnected implements connectionModule
func (h *serverHandler) peerConnected(p *peer) (func(), error) {
	if h.blockchain.TxLookupLimit() != txIndexUnlimited && p.version < lpv4 {
		return nil, errors.New("Cannot serve old clients without a complete tx index")
	}

	// Reject light clients if server is not synced. Put this checking here, so
	// that "non-synced" les-server peers are still allowed to keep the connection.
	if !h.synced() { //TODO synced status after merge
		p.Log().Debug("Light server not synced, rejecting peer")
		return nil, p2p.DiscRequested
	}

	if p.version <= lpv4 {
		h.announceOrStore(p)
	}

	// Mark the peer as being served.
	atomic.StoreUint32(&p.serving, 1) //TODO ???

	return func() {
		atomic.StoreUint32(&p.serving, 0)
	}, nil
}

// messageHandlers implements messageHandlerModule
func (h *serverHandler) messageHandlers() messageHandlers {
	return h.fcWrapper.wrapMessageHandlers((&RequestServer{
		ArchiveMode:   h.server.archiveMode,
		AddTxsSync:    h.addTxsSync,
		BlockChain:    h.blockchain,
		TxPool:        h.txpool,
		GetHelperTrie: h.GetHelperTrie,
	}).MessageHandlers())
}

// getAccount retrieves an account from the state based on root.
func getAccount(triedb *trie.Database, root, hash common.Hash) (types.StateAccount, error) {
	trie, err := trie.New(common.Hash{}, root, triedb)
	if err != nil {
		return types.StateAccount{}, err
	}
	blob, err := trie.TryGet(hash[:])
	if err != nil {
		return types.StateAccount{}, err
	}
	var acc types.StateAccount
	if err = rlp.DecodeBytes(blob, &acc); err != nil {
		return types.StateAccount{}, err
	}
	return acc, nil
}

// GetHelperTrie returns the post-processed trie root for the given trie ID and section index
func (h *serverHandler) GetHelperTrie(typ uint, index uint64) *trie.Trie {
	var (
		root   common.Hash
		prefix string
	)
	switch typ {
	case htCanonical:
		sectionHead := rawdb.ReadCanonicalHash(h.chainDb, (index+1)*h.server.iConfig.ChtSize-1)
		root, prefix = light.GetChtRoot(h.chainDb, index, sectionHead), light.ChtTablePrefix
	case htBloomBits:
		sectionHead := rawdb.ReadCanonicalHash(h.chainDb, (index+1)*h.server.iConfig.BloomTrieSize-1)
		root, prefix = light.GetBloomTrieRoot(h.chainDb, index, sectionHead), light.BloomTrieTablePrefix
	}
	if root == (common.Hash{}) {
		return nil
	}
	trie, _ := trie.New(common.Hash{}, root, trie.NewDatabase(rawdb.NewTable(h.chainDb, prefix)))
	return trie
}

// fcServerHandler performs flow control-related protocol handler tasks
type fcServerHandler struct {
	fcManager    *flowcontrol.ClientManager
	costTracker  *costTracker
	defParams    flowcontrol.ServerParams
	servingQueue *servingQueue
	blockchain   *core.BlockChain // only for SubscribeBlockProcessingEvent, could also use an interface
}

// start implements auxModule
func (h *fcServerHandler) start(wg *sync.WaitGroup, closeCh chan struct{}) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		processCh := make(chan bool, 100)
		sub := h.blockchain.SubscribeBlockProcessingEvent(processCh)
		defer sub.Unsubscribe()

		totalRechargeCh := make(chan uint64, 100)
		totalRecharge := h.costTracker.subscribeTotalRecharge(totalRechargeCh)

		threadsIdle := int(h.costTracker.utilTarget * 4 / flowcontrol.FixedPointMultiplier)
		if threadsIdle < 4 {
			threadsIdle = 4
		}
		threadsBusy := int(h.costTracker.utilTarget/flowcontrol.FixedPointMultiplier + 1)
		//fmt.Println("*** threadsIdle/Busy", threadsIdle, threadsBusy)

		var (
			busy         bool
			blockProcess mclock.AbsTime
		)
		updateRecharge := func() {
			if busy {
				h.servingQueue.setThreads(threadsBusy)
				h.fcManager.SetRechargeCurve(flowcontrol.PieceWiseLinear{{0, 0}, {totalRecharge, totalRecharge}})
			} else {
				h.servingQueue.setThreads(threadsIdle)
				h.fcManager.SetRechargeCurve(flowcontrol.PieceWiseLinear{{0, 0}, {totalRecharge / 10, totalRecharge}, {totalRecharge, totalRecharge}})
			}
		}
		updateRecharge()

		for {
			select {
			case busy = <-processCh:
				if busy {
					blockProcess = mclock.Now()
				} else {
					blockProcessingTimer.Update(time.Duration(mclock.Now() - blockProcess))
				}
				updateRecharge()
			case totalRecharge = <-totalRechargeCh:
				totalRechargeGauge.Update(int64(totalRecharge))
				updateRecharge()
			case <-closeCh:
				return
			}
		}
	}()
}

// sendHandshake implements handshakeModule
func (h *fcServerHandler) sendHandshake(p *peer, send *keyValueList) {
	send.add("flowControl/BL", h.defParams.BufLimit)
	send.add("flowControl/MRR", h.defParams.MinRecharge)

	var costList RequestCostList
	if h.costTracker.testCostList != nil {
		costList = h.costTracker.testCostList
	} else {
		costList = h.costTracker.makeCostList(h.costTracker.globalFactor())
	}
	send.add("flowControl/MRC", costList)
	p.fcCosts = costList.decode(ProtocolLengths[uint(p.version)])
	p.fcParams = h.defParams
}

// receiveHandshake implements handshakeModule
func (h *fcServerHandler) receiveHandshake(p *peer, recv keyValueMap) error {
	return nil
}

// peerConnected implements connectionModule
func (h *fcServerHandler) peerConnected(p *peer) (func(), error) {
	// Setup flow control mechanism for the peer
	p.fcClient = flowcontrol.NewClientNode(h.fcManager, p.fcParams)

	return func() {
		p.fcClient.Disconnect()
		p.fcClient = nil
	}, nil
}

// vfxServerHandler performs vflux-related protocol handler tasks (admits clients into the client pool)
type vfxServerHandler struct {
	fcManager   *flowcontrol.ClientManager
	clientPool  *vfs.ClientPool
	minCapacity uint64
	maxPeers    int
}

// peerConnected implements connectionModule
func (h *vfxServerHandler) peerConnected(p *peer) (func(), error) {
	if p.balance = h.clientPool.Register(p); p.balance == nil {
		p.Log().Debug("Client pool already closed")
		return nil, p2p.DiscRequested
	}

	return func() {
		h.clientPool.Unregister(p)
		p.balance = nil
	}, nil
}

// start implements auxModule
func (h *vfxServerHandler) start(wg *sync.WaitGroup, closeCh chan struct{}) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		totalCapacityCh := make(chan uint64, 100)
		totalCapacity := h.fcManager.SubscribeTotalCapacity(totalCapacityCh)
		h.clientPool.SetLimits(uint64(h.maxPeers), totalCapacity)

		var freePeers uint64

		for {
			select {
			case totalCapacity = <-totalCapacityCh:
				totalCapacityGauge.Update(int64(totalCapacity))
				newFreePeers := totalCapacity / h.minCapacity
				if newFreePeers < freePeers && newFreePeers < uint64(h.maxPeers) {
					log.Warn("Reduced free peer connections", "from", freePeers, "to", newFreePeers)
				}
				freePeers = newFreePeers
				h.clientPool.SetLimits(uint64(h.maxPeers), totalCapacity)
			case <-closeCh:
				return
			}
		}
	}()
}
