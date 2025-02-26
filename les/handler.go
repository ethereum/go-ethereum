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

// Package les implements the Light Ethereum Subprotocol.
package les

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sync"
	"time"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/txpool"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/eth/downloader"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/light"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/p2p"
	"github.com/XinFinOrg/XDPoSChain/p2p/discover"
	"github.com/XinFinOrg/XDPoSChain/p2p/discv5"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/trie"
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

// errIncompatibleConfig is returned if the requested protocols and configs are
// not compatible (low protocol version restrictions and high requirements).
var errIncompatibleConfig = errors.New("incompatible configuration")

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
	State() (*state.StateDB, error)
	InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error)
	Rollback(chain []common.Hash)
	GetHeaderByNumber(number uint64) *types.Header
	GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash
	Genesis() *types.Block
	SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription
}

type txPool interface {
	AddRemotes(txs []*types.Transaction) []error
	AddRemotesSync(txs []*types.Transaction) []error
	Status(hashes []common.Hash) []txpool.TxStatus
}

type ProtocolManager struct {
	lightSync   bool
	txpool      txPool
	txrelay     *LesTxRelay
	networkId   uint64
	chainConfig *params.ChainConfig
	blockchain  BlockChain
	chainDb     ethdb.Database
	odr         *LesOdr
	server      *LesServer
	serverPool  *serverPool
	lesTopic    discv5.Topic
	reqDist     *requestDistributor
	retriever   *retrieveManager

	downloader *downloader.Downloader
	fetcher    *lightFetcher
	peers      *peerSet
	maxPeers   int

	SubProtocols []p2p.Protocol

	eventMux *event.TypeMux

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	quitSync    chan struct{}
	noMorePeers chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg *sync.WaitGroup
}

// NewProtocolManager returns a new ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the ethereum network.
func NewProtocolManager(chainConfig *params.ChainConfig, lightSync bool, protocolVersions []uint, networkId uint64, mux *event.TypeMux, engine consensus.Engine, peers *peerSet, blockchain BlockChain, txpool txPool, chainDb ethdb.Database, odr *LesOdr, txrelay *LesTxRelay, quitSync chan struct{}, wg *sync.WaitGroup) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		lightSync:   lightSync,
		eventMux:    mux,
		blockchain:  blockchain,
		chainConfig: chainConfig,
		chainDb:     chainDb,
		odr:         odr,
		networkId:   networkId,
		txpool:      txpool,
		txrelay:     txrelay,
		peers:       peers,
		newPeerCh:   make(chan *peer),
		quitSync:    quitSync,
		wg:          wg,
		noMorePeers: make(chan struct{}),
	}
	if odr != nil {
		manager.retriever = odr.retriever
		manager.reqDist = odr.retriever.dist
	}

	// Initiate a sub-protocol for every implemented version we can handle
	manager.SubProtocols = make([]p2p.Protocol, 0, len(protocolVersions))
	for _, version := range protocolVersions {
		// Compatible, initialize the sub-protocol
		version := version // Closure for the run
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    "les",
			Version: version,
			Length:  ProtocolLengths[version],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				var entry *poolEntry
				peer := manager.newPeer(int(version), networkId, p, rw)
				if manager.serverPool != nil {
					addr := p.RemoteAddr().(*net.TCPAddr)
					entry = manager.serverPool.connect(peer, addr.IP, uint16(addr.Port))
				}
				peer.poolEntry = entry
				select {
				case manager.newPeerCh <- peer:
					manager.wg.Add(1)
					defer manager.wg.Done()
					err := manager.handle(peer)
					if entry != nil {
						manager.serverPool.disconnect(entry)
					}
					return err
				case <-manager.quitSync:
					if entry != nil {
						manager.serverPool.disconnect(entry)
					}
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

	removePeer := manager.removePeer
	if disableClientRemovePeer {
		removePeer = func(id string) {}
	}

	if lightSync {
		manager.downloader = downloader.New(chainDb, manager.eventMux, nil, blockchain, removePeer, manager.handleProposedBlock)
		manager.peers.notify((*downloaderPeerNotify)(manager))
		manager.fetcher = newLightFetcher(manager)
	}

	return manager, nil
}

// removePeer initiates disconnection from a peer by removing it from the peer set
func (pm *ProtocolManager) removePeer(id string) {
	pm.peers.Unregister(id)
}

// an empty handleProposedBlock function
func (pm *ProtocolManager) handleProposedBlock(header *types.Header) error {
	return nil
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

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pm.peers yet
	// will exit when they try to register.
	pm.peers.Close()

	// Wait for any process action
	pm.wg.Wait()

	log.Info("Light Ethereum protocol stopped")
}

func (pm *ProtocolManager) newPeer(pv int, nv uint64, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, nv, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of a les peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
	// Ignore maxPeers if this is a trusted peer
	if pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
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
	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}
	// Register the peer locally
	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("Light Ethereum peer registration failed", "err", err)
		return err
	}
	defer func() {
		if pm.server != nil && pm.server.fcManager != nil && p.fcClient != nil {
			p.fcClient.Remove(pm.server.fcManager)
		}
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

	stop := make(chan struct{})
	defer close(stop)
	go func() {
		// new block announce loop
		for {
			select {
			case announce := <-p.announceChn:
				p.SendAnnounce(announce)
			case <-stop:
				return
			}
		}
	}()

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("Light Ethereum message handling failed", "err", err)
			return err
		}
	}
}

var reqList = []uint64{GetBlockHeadersMsg, GetBlockBodiesMsg, GetCodeMsg, GetReceiptsMsg, GetProofsV1Msg, SendTxMsg, SendTxV2Msg, GetTxStatusMsg, GetHeaderProofsMsg, GetProofsV2Msg, GetHelperTrieProofsMsg}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (pm *ProtocolManager) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	p.Log().Trace("Light Ethereum message arrived", "code", msg.Code, "bytes", msg.Size)

	costs := p.fcCosts[msg.Code]
	reject := func(reqCnt, maxCnt uint64) bool {
		if p.fcClient == nil || reqCnt > maxCnt {
			return true
		}
		bufValue, _ := p.fcClient.AcceptRequest()
		cost := costs.baseCost + reqCnt*costs.reqCost
		if cost > pm.server.defParams.BufLimit {
			cost = pm.server.defParams.BufLimit
		}
		if cost > bufValue {
			recharge := time.Duration((cost - bufValue) * 1000000 / pm.server.defParams.MinRecharge)
			p.Log().Error("Request came too early", "recharge", common.PrettyDuration(recharge))
			return true
		}
		return false
	}

	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()

	var deliverMsg *Msg

	// Handle the message depending on its contents
	switch msg.Code {
	case StatusMsg:
		p.Log().Trace("Received status message")
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	// Block header query, collect the requested headers and reply
	case AnnounceMsg:
		p.Log().Trace("Received announce message")
		if p.requestAnnounceType == announceTypeNone {
			return errResp(ErrUnexpectedResponse, "")
		}

		var req announceData
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}

		if p.requestAnnounceType == announceTypeSigned {
			if err := req.checkSignature(p.pubKey); err != nil {
				p.Log().Trace("Invalid announcement signature", "err", err)
				return err
			}
			p.Log().Trace("Valid announcement signature")
		}

		p.Log().Trace("Announce message content", "number", req.Number, "hash", req.Hash, "td", req.Td, "reorg", req.ReorgDepth)
		if pm.fetcher != nil {
			pm.fetcher.announce(p, &req)
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
		if reject(query.Amount, MaxHeaderFetch) {
			return errResp(ErrRequestRejected, "")
		}

		hashMode := query.Origin.Hash != (common.Hash{})

		// Gather headers until the fetch or network limits is reached
		var (
			bytes   common.StorageSize
			headers []*types.Header
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit {
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
			case hashMode && query.Reverse:
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

		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + query.Amount*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, query.Amount, rcost)
		return p.SendBlockHeaders(req.ReqID, bv, headers)

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
		p.fcServer.GotReply(resp.ReqID, resp.BV)
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
		if reject(uint64(reqCnt), MaxBodyFetch) {
			return errResp(ErrRequestRejected, "")
		}
		for _, hash := range req.Hashes {
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
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)
		return p.SendBlockBodiesRLP(req.ReqID, bv, bodies)

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
		p.fcServer.GotReply(resp.ReqID, resp.BV)
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
		if reject(uint64(reqCnt), MaxCodeFetch) {
			return errResp(ErrRequestRejected, "")
		}
		for _, req := range req.Reqs {
			// Retrieve the requested state entry, stopping if enough was found
			if number := rawdb.ReadHeaderNumber(pm.chainDb, req.BHash); number != nil {
				if header := rawdb.ReadHeader(pm.chainDb, req.BHash, *number); header != nil {
					statedb, err := pm.blockchain.State()
					if err != nil {
						continue
					}
					account, err := pm.getAccount(statedb, header.Root, common.BytesToHash(req.AccKey))
					if err != nil {
						continue
					}
					code, _ := statedb.Database().TrieDB().Node(common.BytesToHash(account.CodeHash))

					data = append(data, code)
					if bytes += len(code); bytes >= softResponseLimit {
						break
					}
				}
			}
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)
		return p.SendCode(req.ReqID, bv, data)

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
		p.fcServer.GotReply(resp.ReqID, resp.BV)
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
		if reject(uint64(reqCnt), MaxReceiptFetch) {
			return errResp(ErrRequestRejected, "")
		}
		for _, hash := range req.Hashes {
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
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)
		return p.SendReceiptsRLP(req.ReqID, bv, receipts)

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
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgReceipts,
			ReqID:   resp.ReqID,
			Obj:     resp.Receipts,
		}

	case GetProofsV1Msg:
		p.Log().Trace("Received proofs request")
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
			bytes  int
			proofs proofsData
		)
		reqCnt := len(req.Reqs)
		if reject(uint64(reqCnt), MaxProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}
		for _, req := range req.Reqs {
			// Retrieve the requested state entry, stopping if enough was found
			if number := rawdb.ReadHeaderNumber(pm.chainDb, req.BHash); number != nil {
				if header := rawdb.ReadHeader(pm.chainDb, req.BHash, *number); header != nil {
					statedb, err := pm.blockchain.State()
					if err != nil {
						continue
					}
					var trie state.Trie
					if len(req.AccKey) > 0 {
						account, err := pm.getAccount(statedb, header.Root, common.BytesToHash(req.AccKey))
						if err != nil {
							continue
						}
						trie, _ = statedb.Database().OpenStorageTrie(common.BytesToHash(req.AccKey), account.Root)
					} else {
						trie, _ = statedb.Database().OpenTrie(header.Root)
					}
					if trie != nil {
						var proof light.NodeList
						trie.Prove(req.Key, 0, &proof)

						proofs = append(proofs, proof)
						if bytes += proof.DataSize(); bytes >= softResponseLimit {
							break
						}
					}
				}
			}
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)
		return p.SendProofs(req.ReqID, bv, proofs)

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
			statedb   *state.StateDB
			root      common.Hash
		)
		reqCnt := len(req.Reqs)
		if reject(uint64(reqCnt), MaxProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}

		nodes := light.NewNodeSet()

		for _, req := range req.Reqs {
			// Look up the state belonging to the request
			if statedb == nil || req.BHash != lastBHash {
				statedb, root, lastBHash = nil, common.Hash{}, req.BHash

				if number := rawdb.ReadHeaderNumber(pm.chainDb, req.BHash); number != nil {
					if header := rawdb.ReadHeader(pm.chainDb, req.BHash, *number); header != nil {
						statedb, _ = pm.blockchain.State()
						root = header.Root
					}
				}
			}
			if statedb == nil {
				continue
			}
			// Pull the account or storage trie of the request
			var trie state.Trie
			if len(req.AccKey) > 0 {
				account, err := pm.getAccount(statedb, root, common.BytesToHash(req.AccKey))
				if err != nil {
					continue
				}
				trie, _ = statedb.Database().OpenStorageTrie(common.BytesToHash(req.AccKey), account.Root)
			} else {
				trie, _ = statedb.Database().OpenTrie(root)
			}
			if trie == nil {
				continue
			}
			// Prove the user's request from the account or stroage trie
			trie.Prove(req.Key, req.FromLevel, nodes)
			if nodes.DataSize() >= softResponseLimit {
				break
			}
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)
		return p.SendProofsV2(req.ReqID, bv, nodes.NodeList())

	case ProofsV1Msg:
		if pm.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.Log().Trace("Received proofs response")
		// A batch of merkle proofs arrived to one of our previous requests
		var resp struct {
			ReqID, BV uint64
			Data      []light.NodeList
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgProofsV1,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

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
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgProofsV2,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case GetHeaderProofsMsg:
		p.Log().Trace("Received headers proof request")
		// Decode the retrieval message
		var req struct {
			ReqID uint64
			Reqs  []ChtReq
		}
		if err := msg.Decode(&req); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Gather state data until the fetch or network limits is reached
		var (
			bytes  int
			proofs []ChtResp
		)
		reqCnt := len(req.Reqs)
		if reject(uint64(reqCnt), MaxHelperTrieProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}
		trieDb := trie.NewDatabase(rawdb.NewTable(pm.chainDb, light.ChtTablePrefix))
		for _, req := range req.Reqs {
			if header := pm.blockchain.GetHeaderByNumber(req.BlockNum); header != nil {
				sectionHead := rawdb.ReadCanonicalHash(pm.chainDb, req.ChtNum*light.CHTFrequencyServer-1)
				if root := light.GetChtRoot(pm.chainDb, req.ChtNum-1, sectionHead); root != (common.Hash{}) {
					trie, err := trie.New(root, trieDb)
					if err != nil {
						continue
					}
					var encNumber [8]byte
					binary.BigEndian.PutUint64(encNumber[:], req.BlockNum)

					var proof light.NodeList
					trie.Prove(encNumber[:], 0, &proof)

					proofs = append(proofs, ChtResp{Header: header, Proof: proof})
					if bytes += proof.DataSize() + estHeaderRlpSize; bytes >= softResponseLimit {
						break
					}
				}
			}
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)
		return p.SendHeaderProofs(req.ReqID, bv, proofs)

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
		if reject(uint64(reqCnt), MaxHelperTrieProofsFetch) {
			return errResp(ErrRequestRejected, "")
		}

		var (
			lastIdx  uint64
			lastType uint
			root     common.Hash
			auxTrie  *trie.Trie
		)
		nodes := light.NewNodeSet()
		for _, req := range req.Reqs {
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
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)
		return p.SendHelperTrieProofs(req.ReqID, bv, HelperTrieResps{Proofs: nodes.NodeList(), AuxData: auxData})

	case HeaderProofsMsg:
		if pm.odr == nil {
			return errResp(ErrUnexpectedResponse, "")
		}

		p.Log().Trace("Received headers proof response")
		var resp struct {
			ReqID, BV uint64
			Data      []ChtResp
		}
		if err := msg.Decode(&resp); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgHeaderProofs,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

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

		p.fcServer.GotReply(resp.ReqID, resp.BV)
		deliverMsg = &Msg{
			MsgType: MsgHelperTrieProofs,
			ReqID:   resp.ReqID,
			Obj:     resp.Data,
		}

	case SendTxMsg:
		if pm.txpool == nil {
			return errResp(ErrRequestRejected, "")
		}
		// Transactions arrived, parse all of them and deliver to the pool
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		reqCnt := len(txs)
		if reject(uint64(reqCnt), MaxTxSend) {
			return errResp(ErrRequestRejected, "")
		}
		pm.txpool.AddRemotes(txs)

		_, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)

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
		if reject(uint64(reqCnt), MaxTxSend) {
			return errResp(ErrRequestRejected, "")
		}

		hashes := make([]common.Hash, len(req.Txs))
		for i, tx := range req.Txs {
			hashes[i] = tx.Hash()
		}
		stats := pm.txStatus(hashes)
		for i, stat := range stats {
			if stat.Status == txpool.TxStatusUnknown {
				if errs := pm.txpool.AddRemotes([]*types.Transaction{req.Txs[i]}); errs[0] != nil {
					stats[i].Error = errs[0].Error()
					continue
				}
				stats[i] = pm.txStatus([]common.Hash{hashes[i]})[0]
			}
		}

		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)

		return p.SendTxStatus(req.ReqID, bv, stats)

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
		if reject(uint64(reqCnt), MaxTxStatus) {
			return errResp(ErrRequestRejected, "")
		}
		bv, rcost := p.fcClient.RequestProcessed(costs.baseCost + uint64(reqCnt)*costs.reqCost)
		pm.server.fcCostStats.update(msg.Code, uint64(reqCnt), rcost)

		return p.SendTxStatus(req.ReqID, bv, pm.txStatus(req.Hashes))

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

		p.fcServer.GotReply(resp.ReqID, resp.BV)

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
func (pm *ProtocolManager) getAccount(statedb *state.StateDB, root, hash common.Hash) (state.Account, error) {
	trie, err := trie.New(root, statedb.Database().TrieDB())
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
		sectionHead := rawdb.ReadCanonicalHash(pm.chainDb, (idx+1)*light.CHTFrequencyClient-1)
		return light.GetChtV2Root(pm.chainDb, idx, sectionHead), light.ChtTablePrefix
	case htBloomBits:
		sectionHead := rawdb.ReadCanonicalHash(pm.chainDb, (idx+1)*light.BloomTrieFrequency-1)
		return light.GetBloomTrieRoot(pm.chainDb, idx, sectionHead), light.BloomTrieTablePrefix
	}
	return common.Hash{}, ""
}

// getHelperTrieAuxData returns requested auxiliary data for the given HelperTrie request
func (pm *ProtocolManager) getHelperTrieAuxData(req HelperTrieReq) []byte {
	switch {
	case req.Type == htCanonical && req.AuxReq == auxHeader && len(req.Key) == 8:
		blockNum := binary.BigEndian.Uint64(req.Key)
		hash := rawdb.ReadCanonicalHash(pm.chainDb, blockNum)
		return rawdb.ReadHeaderRLP(pm.chainDb, hash, blockNum)
	}
	return nil
}

func (pm *ProtocolManager) txStatus(hashes []common.Hash) []txStatus {
	stats := make([]txStatus, len(hashes))
	for i, stat := range pm.txpool.Status(hashes) {
		// Save the status we've got from the transaction pool
		stats[i].Status = stat

		// If the transaction is unknown to the pool, try looking it up locally
		if stat == txpool.TxStatusUnknown {
			if tx, blockHash, blockNumber, txIndex := rawdb.ReadTransaction(pm.chainDb, hashes[i]); tx != nil {
				stats[i].Status = txpool.TxStatusIncluded
				stats[i].Lookup = &rawdb.LegacyTxLookupEntry{BlockHash: blockHash, BlockIndex: blockNumber, Index: txIndex}
			}
		}
	}
	return stats
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
	head := pm.blockchain.CurrentHeader()
	hash := head.Hash()

	return &NodeInfo{
		Network:    pm.networkId,
		Difficulty: pm.blockchain.GetTd(hash, head.Number.Uint64()),
		Genesis:    pm.blockchain.Genesis().Hash(),
		Config:     pm.blockchain.Config(),
		Head:       hash,
	}
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
			peer.fcServer.QueueRequest(reqID, cost)
			return func() { peer.RequestHeadersByHash(reqID, cost, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pc.manager.reqDist.queue(rq)
	if !ok {
		return ErrNoPeers
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
			peer.fcServer.QueueRequest(reqID, cost)
			return func() { peer.RequestHeadersByNumber(reqID, cost, origin, amount, skip, reverse) }
		},
	}
	_, ok := <-pc.manager.reqDist.queue(rq)
	if !ok {
		return ErrNoPeers
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
