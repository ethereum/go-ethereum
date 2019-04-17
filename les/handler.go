// Copyright 2016 The go-ethereum Authors
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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header

	ethVersion = 63 // equivalent eth version for the downloader

	MaxHeaderFetch           = 192 // Amount of block headers to be fetched per retrieval request
	MaxBodyFetch             = 32  // Amount of block bodies to be fetched per retrieval request
	MaxReceiptFetch          = 128 // Amount of transaction receipts to allow fetching per request
	MaxCodeFetch             = 64  // Amount of contract codes to allow fetching per request
	MaxProofsFetch           = 64  // Amount of merkle proofs to be fetched per retrieval request
	MaxHelperTrieProofsFetch = 64  // Amount of merkle proofs to be fetched per retrieval request
	MaxTxSend                = 64  // Amount of transactions to be send per request
	MaxTxStatus              = 256 // Amount of transactions to queried per request

	disableClientRemovePeer = false
)

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type BlockChain interface {
	Config() *params.ChainConfig
	HasHeader(hash common.Hash, number uint64) bool
	GetHeader(hash common.Hash, number uint64) *types.Header
	GetHeaderByHash(hash common.Hash) *types.Header
	CurrentHeader() *types.Header
	GetTd(hash common.Hash, number uint64) *big.Int
	StateCache() state.Database
	InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error)
	Rollback(chain []common.Hash)
	GetHeaderByNumber(number uint64) *types.Header
	GetAncestor(hash common.Hash, number, ancestor uint64, maxNonCanonical *uint64) (common.Hash, uint64)
	Genesis() *types.Block
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
}

type txPool interface {
	AddRemotes(txs []*types.Transaction) []error
	Status(hashes []common.Hash) []core.TxStatus
}

type ProtocolManager struct {
	lightSync    bool
	txpool       txPool
	txrelay      *LesTxRelay
	networkId    uint64
	chainConfig  *params.ChainConfig
	iConfig      *light.IndexerConfig
	blockchain   BlockChain
	chainDb      ethdb.Database
	odr          *LesOdr
	server       *LesServer
	serverPool   *serverPool
	lesTopic     discv5.Topic
	reqDist      *requestDistributor
	retriever    *retrieveManager
	servingQueue *servingQueue

	downloader *downloader.Downloader
	fetcher    *lightFetcher
	peers      *peerSet
	maxPeers   int

	eventMux *event.TypeMux

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	quitSync    chan struct{}
	noMorePeers chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg  *sync.WaitGroup
	ulc *ulc
}

// NewProtocolManager returns a new ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the ethereum network.
func NewProtocolManager(
	chainConfig *params.ChainConfig,
	indexerConfig *light.IndexerConfig,
	lightSync bool,
	networkId uint64,
	mux *event.TypeMux,
	engine consensus.Engine,
	peers *peerSet,
	blockchain BlockChain,
	txpool txPool,
	chainDb ethdb.Database,
	odr *LesOdr,
	txrelay *LesTxRelay,
	serverPool *serverPool,
	quitSync chan struct{},
	wg *sync.WaitGroup,
	ulcConfig *eth.ULCConfig) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		lightSync:   lightSync,
		eventMux:    mux,
		blockchain:  blockchain,
		chainConfig: chainConfig,
		iConfig:     indexerConfig,
		chainDb:     chainDb,
		odr:         odr,
		networkId:   networkId,
		txpool:      txpool,
		txrelay:     txrelay,
		serverPool:  serverPool,
		peers:       peers,
		newPeerCh:   make(chan *peer),
		quitSync:    quitSync,
		wg:          wg,
		noMorePeers: make(chan struct{}),
	}
	if odr != nil {
		manager.retriever = odr.retriever
		manager.reqDist = odr.retriever.dist
	} else {
		manager.servingQueue = newServingQueue(int64(time.Millisecond * 10))
	}

	if ulcConfig != nil {
		manager.ulc = newULC(ulcConfig)
	}

	removePeer := manager.removePeer
	if disableClientRemovePeer {
		removePeer = func(id string) {}
	}
	if lightSync {
		var checkpoint uint64
		if cht, ok := params.TrustedCheckpoints[blockchain.Genesis().Hash()]; ok {
			checkpoint = (cht.SectionIndex+1)*params.CHTFrequency - 1
		}
		manager.downloader = downloader.New(downloader.LightSync, checkpoint, chainDb, manager.eventMux, nil, blockchain, removePeer)
		manager.peers.notify((*downloaderPeerNotify)(manager))
		manager.fetcher = newLightFetcher(manager)
	}
	return manager, nil
}

// removePeer initiates disconnection from a peer by removing it from the peer set
func (pm *ProtocolManager) removePeer(id string) {
	pm.peers.Unregister(id)
}

func (pm *ProtocolManager) Start(maxPeers int) {
	pm.maxPeers = maxPeers
	if pm.lightSync {
		go pm.syncer()
	} else {
		go func() {
			for range pm.newPeerCh {
			}
		}()
	}
}

func (pm *ProtocolManager) Stop() {
	// Showing a log message. During download / process this could actually
	// take between 5 to 10 seconds and therefor feedback is required.
	log.Info("Stopping light Ethereum protocol")

	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	pm.noMorePeers <- struct{}{}

	close(pm.quitSync) // quits syncer, fetcher

	if pm.servingQueue != nil {
		pm.servingQueue.stop()
	}

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pm.peers yet
	// will exit when they try to register.
	pm.peers.Close()

	// Wait for any process action
	pm.wg.Wait()

	log.Info("Light Ethereum protocol stopped")
}

// runPeer is the p2p protocol run function for the given version.
func (pm *ProtocolManager) runPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter) error {
	var entry *poolEntry
	peer := pm.newPeer(int(version), pm.networkId, p, rw)
	if pm.serverPool != nil {
		entry = pm.serverPool.connect(peer, peer.Node())
	}
	peer.poolEntry = entry
	select {
	case pm.newPeerCh <- peer:
		pm.wg.Add(1)
		defer pm.wg.Done()
		err := pm.handle(peer)
		if entry != nil {
			pm.serverPool.disconnect(entry)
		}
		return err
	case <-pm.quitSync:
		if entry != nil {
			pm.serverPool.disconnect(entry)
		}
		return p2p.DiscQuitting
	}
}

func (pm *ProtocolManager) newPeer(pv int, nv uint64, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	var isTrusted bool
	if pm.isULCEnabled() {
		isTrusted = pm.ulc.isTrusted(p.ID())
	}
	return newPeer(pv, nv, isTrusted, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of a les peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
	// Ignore maxPeers if this is a trusted peer
	// In server mode we try to check into the client pool after handshake
	if pm.lightSync && pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}

	p.Log().Debug("Light Ethereum peer connected", "name", p.Name())

	// Execute the LES handshake
	var (
		genesis = pm.blockchain.Genesis()
		head    = pm.blockchain.CurrentHeader()
		hash    = head.Hash()
		number  = head.Number.Uint64()
		td      = pm.blockchain.GetTd(hash, number)
	)
	if err := p.Handshake(td, hash, number, genesis.Hash(), pm.server); err != nil {
		p.Log().Debug("Light Ethereum handshake failed", "err", err)
		return err
	}
	if p.fcClient != nil {
		defer p.fcClient.Disconnect()
	}

	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}

	// Register the peer locally
	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("Light Ethereum peer registration failed", "err", err)
		return err
	}
	defer func() {
		pm.removePeer(p.id)
	}()

	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if pm.lightSync {
		p.lock.Lock()
		head := p.headInfo
		p.lock.Unlock()
		if pm.fetcher != nil {
			pm.fetcher.announce(p, head)
		}

		if p.poolEntry != nil {
			pm.serverPool.registered(p.poolEntry)
		}
	}

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("Light Ethereum message handling failed", "err", err)
			if p.fcServer != nil {
				p.fcServer.DumpLogs()
			}
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pm *ProtocolManager) handleMsg(p *peer) error {
	select {
	case err := <-p.errCh:
		return err
	default:
	}
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	p.Log().Trace("Light Ethereum message arrived", "code", msg.Code, "bytes", msg.Size)

	p.responseCount++
	responseCount := p.responseCount
	var (
		maxCost uint64
		task    *servingTask
	)

	accept := func(reqID, reqCnt, maxCnt uint64) bool {
		if reqCnt == 0 {
			return false
		}
		if p.fcClient == nil || reqCnt > maxCnt {
			return false
		}
		maxCost = p.fcCosts.getCost(msg.Code, reqCnt)

		if accepted, bufShort, servingPriority := p.fcClient.AcceptRequest(reqID, responseCount, maxCost); !accepted {
			if bufShort > 0 {
				p.Log().Error("Request came too early", "remaining", common.PrettyDuration(time.Duration(bufShort*1000000/p.fcParams.MinRecharge)))
			}
			return false
		} else {
			task = pm.servingQueue.newTask(servingPriority)
		}
		return task.start()
	}

	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	var deliverMsg *Msg

	sendResponse := func(reqID, amount uint64, reply *reply, servingTime uint64) {
		p.responseLock.Lock()
		defer p.responseLock.Unlock()

		var replySize uint32
		if reply != nil {
			replySize = reply.size()
		}
		var realCost uint64
		if pm.server.costTracker != nil {
			realCost = pm.server.costTracker.realCost(servingTime, msg.Size, replySize)
			pm.server.costTracker.updateStats(msg.Code, amount, servingTime, realCost)
		} else {
			realCost = maxCost
		}
		bv := p.fcClient.RequestProcessed(reqID, responseCount, maxCost, realCost)
		if reply != nil {
			p.queueSend(func() {
				if err := reply.send(bv); err != nil {
					select {
					case p.errCh <- err:
					default:
					}
				}
			})
		}
	}

	// Handle the message depending on its contents
	switch msg.Code {
	case StatusMsg:
		p.Log().Trace("Received status message")
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	// Block header query, collect the requested headers and reply
	case AnnounceMsg:
		p.Log().Trace("Received announce message")
		var req announceData
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}

		update, size := req.Update.decode()
		if p.rejectUpdate(size) {
			return errResp(ErrRequestRejected, "")
		}
		p.updateFlowControl(update)

		if req.Hash != (common.Hash{}) {
			if p.announceType == announceTypeNone {
				return errResp(ErrUnexpectedResponse, "")
			}
			if p.announceType == announceTypeSigned {
				if err := req.checkSignature(p.ID(), update); err != nil {
					p.Log().Trace("Invalid announcement signature", "err", err)
					return err
				}
				p.Log().Trace("Valid announcement signature")
			}

			p.Log().Trace("Announce message content", "number", req.Number, "hash", req.Hash, "td", req.Td, "reorg", req.ReorgDepth)
			if pm.fetcher != nil {
				pm.fetcher.announce(p, &req)
			}
		}

	case GetBlockHeadersMsg:
		p.Log().Trace("Received block header request")
		// Decode the complex header query
		var req struct {
			ReqID uint64
			Query getBlockHeadersData
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}

		query := req.Query
		if !accept(req.ReqID, query.Amount, MaxHeaderFetch) {
			return errResp(ErrRequestRejected, "")
		}
		go func() {
			hashMode := query.Origin.Hash != (common.Hash{})
			first := true
			maxNonCanonical := uint64(100)

			// Gather headers until the fetch or network limits is reached
			var (
				bytes   common.StorageSize
				headers []*types.Header
				unknown bool
			)
			for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit {
				if !first && !task.waitOrStop() {
					return
				}
				// Retrieve the next header satisfying the query
				var origin *types.Header
				if hashMode {
					if first {
						origin = pm.blockchain.GetHeaderByHash(query.Origin.Hash)
						if origin != nil {
							query.Origin.Number = origin.Number.Uint64()
						}
					} else {
						origin = pm.blockchain.GetHeader(query.Origin.Hash, query.Origin.Number)
					}
				} else {
					origin = pm.blockchain.GetHeaderByNumber(query.Origin.Number)
				}
				if origin == nil {
					break
				}
				headers = append(headers, origin)
				bytes += estHeaderRlpSize

				// Advance to the next header of the query
				switch {
				case hashMode && query.Reverse:
					// Hash based traversal towards the genesis block
					ancestor := query.Skip + 1
					if ancestor == 0 {
						unknown = true
					} else {
						query.Origin.Hash, query.Origin.Number = pm.blockchain.GetAncestor(query.Origin.Hash, query.Origin.Number, ancestor, &maxNonCanonical)
						unknown = (query.Origin.Hash == common.Hash{})
					}
				case hashMode && !query.Reverse:
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
							nextHash := header.Hash()
							expOldHash, _ := pm.blockchain.GetAncestor(nextHash, next, query.Skip+1, &maxNonCanonical)
							if expOldHash == query.Origin.Hash {
								query.Origin.Hash, query.Origin.Number = nextHash, next
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
				first = false
			}
			sendResponse(req.ReqID, query.Amount, p.ReplyBlockHeaders(req.ReqID, headers), task.done())
		}()

	case BlockHeadersMsg:
		if pm.downloader == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.Log().Trace("Received block header response message")
		// A batch of headers arrived to one of our previous requests
		var resp struct {
			ReqID, BV uint64
			Headers   []*types.Header
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		if pm.fetcher != nil && pm.fetcher.requestedID(resp.ReqID) {
			pm.fetcher.deliverHeaders(p, resp.ReqID, resp.Headers)
		} else {
			err := pm.downloader.DeliverHeaders(p.id, resp.Headers)
			if err != nil {
				log.Debug(fmt.Sprint(err))
			}
		}

	case GetBlockBodiesMsg:
		p.Log().Trace("Received block bodies request")
		// Decode the retrieval message
		var req struct {
			ReqID  uint64
			Hashes []common.Hash
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather blocks until the fetch or network limits is reached
		var (
			bytes  int
			bodies []rlp.RawValue
		)
		reqCnt := len(req.Hashes)
		if !accept(req.ReqID, uint64(reqCnt), MaxBodyFetch) {
			return errResp(ErrRequestRejected, "")
		}
		go func() {
			for i, hash := range req.Hashes {
				if i != 0 && !task.waitOrStop() {
					return
				}
				if bytes >= softResponseLimit {
					break
				}
				// Retrieve the requested block body, stopping if enough was found
				if number := rawdb.ReadHeaderNumber(pm.chainDb, hash); number != nil {
					if data := rawdb.ReadBodyRLP(pm.chainDb, hash, *number); len(data) != 0 {
						bodies = append(bodies, data)
						bytes += len(data)
					}
				}
			}
			sendResponse(req.ReqID, uint64(reqCnt), p.ReplyBlockBodiesRLP(req.ReqID, bodies), task.done())
		}()

	case BlockBodiesMsg:
		if pm.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.Log().Trace("Received block bodies response")
		// A batch of block bodies arrived to one of our previous requests
		var resp struct {
			ReqID, BV uint64
			Data      []*types.Body
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgBlockBodies,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case GetCodeMsg:
		p.Log().Trace("Received code request")
		// Decode the retrieval message
		var req struct {
			ReqID uint64
			Reqs  []CodeReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			bytes int
			data  [][]byte
		)
		reqCnt := len(req.Reqs)
		if !accept(req.ReqID, uint64(reqCnt), MaxCodeFetch) {
			return errResp(ErrRequestRejected, "")
		}
		go func() {
			for i, req := range req.Reqs {
				if i != 0 && !task.waitOrStop() {
					return
				}
				// Look up the root hash belonging to the request
				number := rawdb.ReadHeaderNumber(pm.chainDb, req.BHash)
				if number == nil {
					p.Log().Warn("Failed to retrieve block num for code", "hash", req.BHash)
					continue
				}
				header := rawdb.ReadHeader(pm.chainDb, req.BHash, *number)
				if header == nil {
					p.Log().Warn("Failed to retrieve header for code", "block", *number, "hash", req.BHash)
					continue
				}
				triedb := pm.blockchain.StateCache().TrieDB()

				account, err := pm.getAccount(triedb, header.Root, common.BytesToHash(req.AccKey))
				if err != nil {
					p.Log().Warn("Failed to retrieve account for code", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(req.AccKey), "err", err)
					continue
				}
				code, err := triedb.Node(common.BytesToHash(account.CodeHash))
				if err != nil {
					p.Log().Warn("Failed to retrieve account code", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(req.AccKey), "codehash", common.BytesToHash(account.CodeHash), "err", err)
					continue
				}
				// Accumulate the code and abort if enough data was retrieved
				data = append(data, code)
				if bytes += len(code); bytes >= softResponseLimit {
					break
				}
			}
			sendResponse(req.ReqID, uint64(reqCnt), p.ReplyCode(req.ReqID, data), task.done())
		}()

	case CodeMsg:
		if pm.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.Log().Trace("Received code response")
		// A batch of node state data arrived to one of our previous requests
		var resp struct {
			ReqID, BV uint64
			Data      [][]byte
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgCode,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case GetReceiptsMsg:
		p.Log().Trace("Received receipts request")
		// Decode the retrieval message
		var req struct {
			ReqID  uint64
			Hashes []common.Hash
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			bytes    int
			receipts []rlp.RawValue
		)
		reqCnt := len(req.Hashes)
		if !accept(req.ReqID, uint64(reqCnt), MaxReceiptFetch) {
			return errResp(ErrRequestRejected, "")
		}
		go func() {
			for i, hash := range req.Hashes {
				if i != 0 && !task.waitOrStop() {
					return
				}
				if bytes >= softResponseLimit {
					break
				}
				// Retrieve the requested block's receipts, skipping if unknown to us
				var results types.Receipts
				if number := rawdb.ReadHeaderNumber(pm.chainDb, hash); number != nil {
					results = rawdb.ReadRawReceipts(pm.chainDb, hash, *number)
				}
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
			sendResponse(req.ReqID, uint64(reqCnt), p.ReplyReceiptsRLP(req.ReqID, receipts), task.done())
		}()

	case ReceiptsMsg:
		if pm.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.Log().Trace("Received receipts response")
		// A batch of receipts arrived to one of our previous requests
		var resp struct {
			ReqID, BV uint64
			Receipts  []types.Receipts
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgReceipts,
			ReqID:   resp.ReqID,
			Obj:     resp.Receipts,
		}

	case GetProofsV2Msg:
		p.Log().Trace("Received les/2 proofs request")
		// Decode the retrieval message
		var req struct {
			ReqID uint64
			Reqs  []ProofReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			lastBHash common.Hash
			root      common.Hash
		)
		reqCnt := len(req.Reqs)
		if !accept(req.ReqID, uint64(reqCnt), MaxProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}
		go func() {
			nodes := light.NewNodeSet()

			for i, req := range req.Reqs {
				if i != 0 && !task.waitOrStop() {
					return
				}
				// Look up the root hash belonging to the request
				var (
					number *uint64
					header *types.Header
					trie   state.Trie
				)
				if req.BHash != lastBHash {
					root, lastBHash = common.Hash{}, req.BHash

					if number = rawdb.ReadHeaderNumber(pm.chainDb, req.BHash); number == nil {
						p.Log().Warn("Failed to retrieve block num for proof", "hash", req.BHash)
						continue
					}
					if header = rawdb.ReadHeader(pm.chainDb, req.BHash, *number); header == nil {
						p.Log().Warn("Failed to retrieve header for proof", "block", *number, "hash", req.BHash)
						continue
					}
					root = header.Root
				}
				// Open the account or storage trie for the request
				statedb := pm.blockchain.StateCache()

				switch len(req.AccKey) {
				case 0:
					// No account key specified, open an account trie
					trie, err = statedb.OpenTrie(root)
					if trie == nil || err != nil {
						p.Log().Warn("Failed to open storage trie for proof", "block", header.Number, "hash", header.Hash(), "root", root, "err", err)
						continue
					}
				default:
					// Account key specified, open a storage trie
					account, err := pm.getAccount(statedb.TrieDB(), root, common.BytesToHash(req.AccKey))
					if err != nil {
						p.Log().Warn("Failed to retrieve account for proof", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(req.AccKey), "err", err)
						continue
					}
					trie, err = statedb.OpenStorageTrie(common.BytesToHash(req.AccKey), account.Root)
					if trie == nil || err != nil {
						p.Log().Warn("Failed to open storage trie for proof", "block", header.Number, "hash", header.Hash(), "account", common.BytesToHash(req.AccKey), "root", account.Root, "err", err)
						continue
					}
				}
				// Prove the user's request from the account or stroage trie
				if err := trie.Prove(req.Key, req.FromLevel, nodes); err != nil {
					p.Log().Warn("Failed to prove state request", "block", header.Number, "hash", header.Hash(), "err", err)
					continue
				}
				if nodes.DataSize() >= softResponseLimit {
					break
				}
			}
			sendResponse(req.ReqID, uint64(reqCnt), p.ReplyProofsV2(req.ReqID, nodes.NodeList()), task.done())
		}()

	case ProofsV2Msg:
		if pm.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.Log().Trace("Received les/2 proofs response")
		// A batch of merkle proofs arrived to one of our previous requests
		var resp struct {
			ReqID, BV uint64
			Data      light.NodeList
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgProofsV2,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case GetHelperTrieProofsMsg:
		p.Log().Trace("Received helper trie proof request")
		// Decode the retrieval message
		var req struct {
			ReqID uint64
			Reqs  []HelperTrieReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			auxBytes int
			auxData  [][]byte
		)
		reqCnt := len(req.Reqs)
		if !accept(req.ReqID, uint64(reqCnt), MaxHelperTrieProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}
		go func() {

			var (
				lastIdx  uint64
				lastType uint
				root     common.Hash
				auxTrie  *trie.Trie
			)
			nodes := light.NewNodeSet()
			for i, req := range req.Reqs {
				if i != 0 && !task.waitOrStop() {
					return
				}
				if auxTrie == nil || req.Type != lastType || req.TrieIdx != lastIdx {
					auxTrie, lastType, lastIdx = nil, req.Type, req.TrieIdx

					var prefix string
					if root, prefix = pm.getHelperTrie(req.Type, req.TrieIdx); root != (common.Hash{}) {
						auxTrie, _ = trie.New(root, trie.NewDatabase(rawdb.NewTable(pm.chainDb, prefix)))
					}
				}
				if req.AuxReq == auxRoot {
					var data []byte
					if root != (common.Hash{}) {
						data = root[:]
					}
					auxData = append(auxData, data)
					auxBytes += len(data)
				} else {
					if auxTrie != nil {
						auxTrie.Prove(req.Key, req.FromLevel, nodes)
					}
					if req.AuxReq != 0 {
						data := pm.getHelperTrieAuxData(req)
						auxData = append(auxData, data)
						auxBytes += len(data)
					}
				}
				if nodes.DataSize()+auxBytes >= softResponseLimit {
					break
				}
			}
			sendResponse(req.ReqID, uint64(reqCnt), p.ReplyHelperTrieProofs(req.ReqID, HelperTrieResps{Proofs: nodes.NodeList(), AuxData: auxData}), task.done())
		}()

	case HelperTrieProofsMsg:
		if pm.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.Log().Trace("Received helper trie proof response")
		var resp struct {
			ReqID, BV uint64
			Data      HelperTrieResps
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgHelperTrieProofs,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case SendTxV2Msg:
		if pm.txpool == nil {
			return errResp(ErrRequestRejected, "")
		}
		// Transactions arrived, parse all of them and deliver to the pool
		var req struct {
			ReqID uint64
			Txs   []*types.Transaction
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(req.Txs)
		if !accept(req.ReqID, uint64(reqCnt), MaxTxSend) {
			return errResp(ErrRequestRejected, "")
		}
		go func() {
			stats := make([]txStatus, len(req.Txs))
			for i, tx := range req.Txs {
				if i != 0 && !task.waitOrStop() {
					return
				}
				hash := tx.Hash()
				stats[i] = pm.txStatus(hash)
				if stats[i].Status == core.TxStatusUnknown {
					if errs := pm.txpool.AddRemotes([]*types.Transaction{tx}); errs[0] != nil {
						stats[i].Error = errs[0].Error()
						continue
					}
					stats[i] = pm.txStatus(hash)
				}
			}
			sendResponse(req.ReqID, uint64(reqCnt), p.ReplyTxStatus(req.ReqID, stats), task.done())
		}()

	case GetTxStatusMsg:
		if pm.txpool == nil {
			return errResp(ErrUnexpectedResponse, "")
		}
		// Transactions arrived, parse all of them and deliver to the pool
		var req struct {
			ReqID  uint64
			Hashes []common.Hash
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(req.Hashes)
		if !accept(req.ReqID, uint64(reqCnt), MaxTxStatus) {
			return errResp(ErrRequestRejected, "")
		}
		go func() {
			stats := make([]txStatus, len(req.Hashes))
			for i, hash := range req.Hashes {
				if i != 0 && !task.waitOrStop() {
					return
				}
				stats[i] = pm.txStatus(hash)
			}
			sendResponse(req.ReqID, uint64(reqCnt), p.ReplyTxStatus(req.ReqID, stats), task.done())
		}()

	case TxStatusMsg:
		if pm.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.Log().Trace("Received tx status response")
		var resp struct {
			ReqID, BV uint64
			Status    []txStatus
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		p.fcServer.ReceivedReply(resp.ReqID, resp.BV)

	default:
		p.Log().Trace("Received unknown message", "code", msg.Code)
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}

	if deliverMsg != nil {
		err := pm.retriever.deliver(p, deliverMsg)
		if err != nil {
			p.responseErrors++
			if p.responseErrors > maxResponseErrors {
				return err
			}
		}
	}
	return nil
}

// getAccount retrieves an account from the state based at root.
func (pm *ProtocolManager) getAccount(triedb *trie.Database, root, hash common.Hash) (state.Account, error) {
	trie, err := trie.New(root, triedb)
	if err != nil {
		return state.Account{}, err
	}
	blob, err := trie.TryGet(hash[:])
	if err != nil {
		return state.Account{}, err
	}
	var account state.Account
	if err = rlp.DecodeBytes(blob, &account); err != nil {
		return state.Account{}, err
	}
	return account, nil
}

// getHelperTrie returns the post-processed trie root for the given trie ID and section index
func (pm *ProtocolManager) getHelperTrie(id uint, idx uint64) (common.Hash, string) {
	switch id {
	case htCanonical:
		sectionHead := rawdb.ReadCanonicalHash(pm.chainDb, (idx+1)*pm.iConfig.ChtSize-1)
		return light.GetChtRoot(pm.chainDb, idx, sectionHead), light.ChtTablePrefix
	case htBloomBits:
		sectionHead := rawdb.ReadCanonicalHash(pm.chainDb, (idx+1)*pm.iConfig.BloomTrieSize-1)
		return light.GetBloomTrieRoot(pm.chainDb, idx, sectionHead), light.BloomTrieTablePrefix
	}
	return common.Hash{}, ""
}

// getHelperTrieAuxData returns requested auxiliary data for the given HelperTrie request
func (pm *ProtocolManager) getHelperTrieAuxData(req HelperTrieReq) []byte {
	if req.Type == htCanonical && req.AuxReq == auxHeader && len(req.Key) == 8 {
		blockNum := binary.BigEndian.Uint64(req.Key)
		hash := rawdb.ReadCanonicalHash(pm.chainDb, blockNum)
		return rawdb.ReadHeaderRLP(pm.chainDb, hash, blockNum)
	}
	return nil
}

func (pm *ProtocolManager) txStatus(hash common.Hash) txStatus {
	var stat txStatus
	stat.Status = pm.txpool.Status([]common.Hash{hash})[0]
	// If the transaction is unknown to the pool, try looking it up locally
	if stat.Status == core.TxStatusUnknown {
		if tx, blockHash, blockNumber, txIndex := rawdb.ReadTransaction(pm.chainDb, hash); tx != nil {
			stat.Status = core.TxStatusIncluded
			stat.Lookup = &rawdb.LegacyTxLookupEntry{BlockHash: blockHash, BlockIndex: blockNumber, Index: txIndex}
		}
	}
	return stat
}

// isULCEnabled returns true if we can use ULC
func (pm *ProtocolManager) isULCEnabled() bool {
	if pm.ulc == nil || len(pm.ulc.trustedKeys) == 0 {
		return false
	}
	return true
}

// downloaderPeerNotify implements peerSetNotify
type downloaderPeerNotify ProtocolManager

type peerConnection struct {
	manager *ProtocolManager
	peer    *peer
}

func (pc *peerConnection) Head() (common.Hash, *big.Int) {
	return pc.peer.HeadAndTd()
}

func (pc *peerConnection) RequestHeadersByHash(origin common.Hash, amount int, skip int, reverse bool) error {
	reqID := genReqID()
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			peer := dp.(*peer)
			return peer.GetRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*peer) == pc.peer
		},
		request: func(dp distPeer) func() {
			peer := dp.(*peer)
			cost := peer.GetRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.RequestHeadersByHash(reqID, cost, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pc.manager.reqDist.queue(rq)
	if !ok {
		return light.ErrNoPeers
	}
	return nil
}

func (pc *peerConnection) RequestHeadersByNumber(origin uint64, amount int, skip int, reverse bool) error {
	reqID := genReqID()
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			peer := dp.(*peer)
			return peer.GetRequestCost(GetBlockHeadersMsg, amount)
		},
		canSend: func(dp distPeer) bool {
			return dp.(*peer) == pc.peer
		},
		request: func(dp distPeer) func() {
			peer := dp.(*peer)
			cost := peer.GetRequestCost(GetBlockHeadersMsg, amount)
			peer.fcServer.QueuedRequest(reqID, cost)
			return func() { peer.RequestHeadersByNumber(reqID, cost, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pc.manager.reqDist.queue(rq)
	if !ok {
		return light.ErrNoPeers
	}
	return nil
}

func (d *downloaderPeerNotify) registerPeer(p *peer) {
	pm := (*ProtocolManager)(d)
	pc := &peerConnection{
		manager: pm,
		peer:    p,
	}
	pm.downloader.RegisterLightPeer(p.id, ethVersion, pc)
}

func (d *downloaderPeerNotify) unregisterPeer(p *peer) {
	pm := (*ProtocolManager)(d)
	pm.downloader.UnregisterPeer(p.id)
}
