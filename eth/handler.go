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
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/fetcher"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header
)

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type hashFetcherFn func(common.Hash) error
type blockFetcherFn func([]common.Hash) error

// extProt is an interface which is passed around so we can expose GetHashes and GetBlock without exposing it to the rest of the protocol
// extProt is passed around to peers which require to GetHashes and GetBlocks
type extProt struct {
	getHashes hashFetcherFn
	getBlocks blockFetcherFn
}

func (ep extProt) GetHashes(hash common.Hash) error    { return ep.getHashes(hash) }
func (ep extProt) GetBlock(hashes []common.Hash) error { return ep.getBlocks(hashes) }

type ProtocolManager struct {
	txpool   txPool
	chainman *core.ChainManager
	chaindb  ethdb.Database

	downloader *downloader.Downloader
	fetcher    *fetcher.Fetcher
	peers      *peerSet

	SubProtocols []p2p.Protocol

	eventMux      *event.TypeMux
	txSub         event.Subscription
	minedBlockSub event.Subscription

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh chan *peer
	txsyncCh  chan *txsync
	quitSync  chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg   sync.WaitGroup
	quit bool
}

// NewProtocolManager returns a new ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the ethereum network.
func NewProtocolManager(networkId int, mux *event.TypeMux, txpool txPool, pow pow.PoW, chainman *core.ChainManager, chaindb ethdb.Database) *ProtocolManager {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		eventMux:  mux,
		txpool:    txpool,
		chainman:  chainman,
		chaindb:   chaindb,
		peers:     newPeerSet(),
		newPeerCh: make(chan *peer, 1),
		txsyncCh:  make(chan *txsync),
		quitSync:  make(chan struct{}),
	}
	// Initiate a sub-protocol for every implemented version we can handle
	manager.SubProtocols = make([]p2p.Protocol, len(ProtocolVersions))
	for i := 0; i < len(manager.SubProtocols); i++ {
		version := ProtocolVersions[i]

		manager.SubProtocols[i] = p2p.Protocol{
			Name:    "eth",
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := manager.newPeer(int(version), networkId, p, rw)
				manager.newPeerCh <- peer
				return manager.handle(peer)
			},
		}
	}
	// Construct the different synchronisation mechanisms
	manager.downloader = downloader.New(manager.eventMux, manager.chainman.HasBlock, manager.chainman.GetBlock, manager.chainman.CurrentBlock, manager.chainman.GetTd, manager.chainman.InsertChain, manager.removePeer)

	validator := func(block *types.Block, parent *types.Block) error {
		return core.ValidateHeader(pow, block.Header(), parent, true, false)
	}
	heighter := func() uint64 {
		return manager.chainman.CurrentBlock().NumberU64()
	}
	manager.fetcher = fetcher.New(manager.chainman.GetBlock, validator, manager.BroadcastBlock, heighter, manager.chainman.InsertChain, manager.removePeer)

	return manager
}

func (pm *ProtocolManager) removePeer(id string) {
	// Short circuit if the peer was already removed
	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	glog.V(logger.Debug).Infoln("Removing peer", id)

	// Unregister the peer from the downloader and Ethereum peer set
	pm.downloader.UnregisterPeer(id)
	if err := pm.peers.Unregister(id); err != nil {
		glog.V(logger.Error).Infoln("Removal failed:", err)
	}
	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (pm *ProtocolManager) Start() {
	// broadcast transactions
	pm.txSub = pm.eventMux.Subscribe(core.TxPreEvent{})
	go pm.txBroadcastLoop()
	// broadcast mined blocks
	pm.minedBlockSub = pm.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go pm.minedBroadcastLoop()

	// start sync handlers
	go pm.syncer()
	go pm.txsyncLoop()
}

func (pm *ProtocolManager) Stop() {
	// Showing a log message. During download / process this could actually
	// take between 5 to 10 seconds and therefor feedback is required.
	glog.V(logger.Info).Infoln("Stopping ethereum protocol handler...")

	pm.quit = true
	pm.txSub.Unsubscribe()         // quits txBroadcastLoop
	pm.minedBlockSub.Unsubscribe() // quits blockBroadcastLoop
	close(pm.quitSync)             // quits syncer, fetcher, txsyncLoop

	// Wait for any process action
	pm.wg.Wait()

	glog.V(logger.Info).Infoln("Ethereum protocol handler stopped")
}

func (pm *ProtocolManager) newPeer(pv, nv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, nv, p, newMeteredMsgWriter(rw))
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
	glog.V(logger.Debug).Infof("%v: peer connected [%s]", p, p.Name())

	// Execute the Ethereum handshake
	td, head, genesis := pm.chainman.Status()
	if err := p.Handshake(td, head, genesis); err != nil {
		glog.V(logger.Debug).Infof("%v: handshake failed: %v", p, err)
		return err
	}
	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}
	// Register the peer locally
	glog.V(logger.Detail).Infof("%v: adding peer", p)
	if err := pm.peers.Register(p); err != nil {
		glog.V(logger.Error).Infof("%v: addition failed: %v", p, err)
		return err
	}
	defer pm.removePeer(p.id)

	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := pm.downloader.RegisterPeer(p.id, p.version, p.Head(),
		p.RequestHashes, p.RequestHashesFromNumber, p.RequestBlocks,
		p.RequestHeadersByHash, p.RequestHeadersByNumber, p.RequestBodies); err != nil {
		return err
	}
	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	pm.syncTransactions(p)

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			glog.V(logger.Debug).Infof("%v: message handling failed: %v", p, err)
			return err
		}
	}
	return nil
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

	case p.version < eth62 && msg.Code == GetBlockHashesMsg:
		// Retrieve the number of hashes to return and from which origin hash
		var request getBlockHashesData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if request.Amount > uint64(downloader.MaxHashFetch) {
			request.Amount = uint64(downloader.MaxHashFetch)
		}
		// Retrieve the hashes from the block chain and return them
		hashes := pm.chainman.GetBlockHashesFromHash(request.Hash, request.Amount)
		if len(hashes) == 0 {
			glog.V(logger.Debug).Infof("invalid block hash %x", request.Hash.Bytes()[:4])
		}
		return p.SendBlockHashes(hashes)

	case p.version < eth62 && msg.Code == GetBlockHashesFromNumberMsg:
		// Retrieve and decode the number of hashes to return and from which origin number
		var request getBlockHashesFromNumberData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if request.Amount > uint64(downloader.MaxHashFetch) {
			request.Amount = uint64(downloader.MaxHashFetch)
		}
		// Calculate the last block that should be retrieved, and short circuit if unavailable
		last := pm.chainman.GetBlockByNumber(request.Number + request.Amount - 1)
		if last == nil {
			last = pm.chainman.CurrentBlock()
			request.Amount = last.NumberU64() - request.Number + 1
		}
		if last.NumberU64() < request.Number {
			return p.SendBlockHashes(nil)
		}
		// Retrieve the hashes from the last block backwards, reverse and return
		hashes := []common.Hash{last.Hash()}
		hashes = append(hashes, pm.chainman.GetBlockHashesFromHash(last.Hash(), request.Amount-1)...)

		for i := 0; i < len(hashes)/2; i++ {
			hashes[i], hashes[len(hashes)-1-i] = hashes[len(hashes)-1-i], hashes[i]
		}
		return p.SendBlockHashes(hashes)

	case p.version < eth62 && msg.Code == BlockHashesMsg:
		// A batch of hashes arrived to one of our previous requests
		var hashes []common.Hash
		if err := msg.Decode(&hashes); err != nil {
			break
		}
		// Deliver them all to the downloader for queuing
		err := pm.downloader.DeliverHashes61(p.id, hashes)
		if err != nil {
			glog.V(logger.Debug).Infoln(err)
		}

	case p.version < eth62 && msg.Code == GetBlocksMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather blocks until the fetch or network limits is reached
		var (
			hash   common.Hash
			bytes  common.StorageSize
			blocks []*types.Block
		)
		for len(blocks) < downloader.MaxBlockFetch && bytes < softResponseLimit {
			//Retrieve the hash of the next block
			err := msgStream.Decode(&hash)
			if err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested block, stopping if enough was found
			if block := pm.chainman.GetBlock(hash); block != nil {
				blocks = append(blocks, block)
				bytes += block.Size()
			}
		}
		return p.SendBlocks(blocks)

	case p.version < eth62 && msg.Code == BlocksMsg:
		// Decode the arrived block message
		var blocks []*types.Block
		if err := msg.Decode(&blocks); err != nil {
			glog.V(logger.Detail).Infoln("Decode error", err)
			blocks = nil
		}
		// Update the receive timestamp of each block
		for _, block := range blocks {
			block.ReceivedAt = msg.ReceivedAt
		}
		// Filter out any explicitly requested blocks, deliver the rest to the downloader
		if blocks := pm.fetcher.FilterBlocks(blocks); len(blocks) > 0 {
			pm.downloader.DeliverBlocks61(p.id, blocks)
		}

	// Block header query, collect the requested headers and reply
	case p.version >= eth62 && msg.Code == GetBlockHeadersMsg:
		// Decode the complex header query
		var query getBlockHeadersData
		if err := msg.Decode(&query); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		// Gather headers until the fetch or network limits is reached
		var (
			bytes   common.StorageSize
			headers []*types.Header
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {
			// Retrieve the next header satisfying the query
			var origin *types.Header
			if query.Origin.Hash != (common.Hash{}) {
				origin = pm.chainman.GetHeader(query.Origin.Hash)
			} else {
				origin = pm.chainman.GetHeaderByNumber(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			headers = append(headers, origin)
			bytes += estHeaderRlpSize

			// Advance to the next header of the query
			switch {
			case query.Origin.Hash != (common.Hash{}) && query.Reverse:
				// Hash based traversal towards the genesis block
				for i := 0; i < int(query.Skip)+1; i++ {
					if header := pm.chainman.GetHeader(query.Origin.Hash); header != nil {
						query.Origin.Hash = header.ParentHash
					} else {
						unknown = true
						break
					}
				}
			case query.Origin.Hash != (common.Hash{}) && !query.Reverse:
				// Hash based traversal towards the leaf block
				if header := pm.chainman.GetHeaderByNumber(origin.Number.Uint64() + query.Skip + 1); header != nil {
					if pm.chainman.GetBlockHashesFromHash(header.Hash(), query.Skip+1)[query.Skip] == query.Origin.Hash {
						query.Origin.Hash = header.Hash()
					} else {
						unknown = true
					}
				} else {
					unknown = true
				}
			case query.Reverse:
				// Number based traversal towards the genesis block
				if query.Origin.Number >= query.Skip+1 {
					query.Origin.Number -= (query.Skip + 1)
				} else {
					unknown = true
				}

			case !query.Reverse:
				// Number based traversal towards the leaf block
				query.Origin.Number += (query.Skip + 1)
			}
		}
		return p.SendBlockHeaders(headers)

	case p.version >= eth62 && msg.Code == BlockHeadersMsg:
		// A batch of headers arrived to one of our previous requests
		var headers []*types.Header
		if err := msg.Decode(&headers); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		// Filter out any explicitly requested headers, deliver the rest to the downloader
		filter := len(headers) == 1
		if filter {
			headers = pm.fetcher.FilterHeaders(headers, time.Now())
		}
		if len(headers) > 0 || !filter {
			err := pm.downloader.DeliverHeaders(p.id, headers)
			if err != nil {
				glog.V(logger.Debug).Infoln(err)
			}
		}

	case p.version >= eth62 && msg.Code == BlockBodiesMsg:
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
		if trasactions, uncles := pm.fetcher.FilterBodies(trasactions, uncles, time.Now()); len(trasactions) > 0 || len(uncles) > 0 {
			err := pm.downloader.DeliverBodies(p.id, trasactions, uncles)
			if err != nil {
				glog.V(logger.Debug).Infoln(err)
			}
		}

	case p.version >= eth62 && msg.Code == GetBlockBodiesMsg:
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
			if data := pm.chainman.GetBodyRLP(hash); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)
			}
		}
		return p.SendBlockBodiesRLP(bodies)

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
			if entry, err := pm.chaindb.Get(hash.Bytes()); err == nil {
				data = append(data, entry)
				bytes += len(entry)
			}
		}
		return p.SendNodeData(data)

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
			receipts []*types.Receipt
		)
		for bytes < softResponseLimit && len(receipts) < downloader.MaxReceiptsFetch {
			// Retrieve the hash of the next transaction receipt
			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			// Retrieve the requested receipt, stopping if enough was found
			if receipt := core.GetReceipt(pm.chaindb, hash); receipt != nil {
				receipts = append(receipts, receipt)
				bytes += len(receipt.RlpEncode())
			}
		}
		return p.SendReceipts(receipts)

	case msg.Code == NewBlockHashesMsg:
		// Retrieve and deseralize the remote new block hashes notification
		type announce struct {
			Hash   common.Hash
			Number uint64
		}
		var announces = []announce{}

		if p.version < eth62 {
			// We're running the old protocol, make block number unknown (0)
			var hashes []common.Hash
			if err := msg.Decode(&hashes); err != nil {
				return errResp(ErrDecode, "%v: %v", msg, err)
			}
			for _, hash := range hashes {
				announces = append(announces, announce{hash, 0})
			}
		} else {
			// Otherwise extract both block hash and number
			var request newBlockHashesData
			if err := msg.Decode(&request); err != nil {
				return errResp(ErrDecode, "%v: %v", msg, err)
			}
			for _, block := range request {
				announces = append(announces, announce{block.Hash, block.Number})
			}
		}
		// Mark the hashes as present at the remote node
		for _, block := range announces {
			p.MarkBlock(block.Hash)
			p.SetHead(block.Hash)
		}
		// Schedule all the unknown hashes for retrieval
		unknown := make([]announce, 0, len(announces))
		for _, block := range announces {
			if !pm.chainman.HasBlock(block.Hash) {
				unknown = append(unknown, block)
			}
		}
		for _, block := range unknown {
			if p.version < eth62 {
				pm.fetcher.Notify(p.id, block.Hash, block.Number, time.Now(), p.RequestBlocks, nil, nil)
			} else {
				pm.fetcher.Notify(p.id, block.Hash, block.Number, time.Now(), nil, p.RequestOneHeader, p.RequestBodies)
			}
		}

	case msg.Code == NewBlockMsg:
		// Retrieve and decode the propagated block
		var request newBlockData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if err := request.Block.ValidateFields(); err != nil {
			return errResp(ErrDecode, "block validation %v: %v", msg, err)
		}
		request.Block.ReceivedAt = msg.ReceivedAt

		// Mark the block's arrival for whatever reason
		_, chainHead, _ := pm.chainman.Status()
		jsonlogger.LogJson(&logger.EthChainReceivedNewBlock{
			BlockHash:     request.Block.Hash().Hex(),
			BlockNumber:   request.Block.Number(),
			ChainHeadHash: chainHead.Hex(),
			BlockPrevHash: request.Block.ParentHash().Hex(),
			RemoteId:      p.ID().String(),
		})
		// Mark the peer as owning the block and schedule it for import
		p.MarkBlock(request.Block.Hash())
		p.SetHead(request.Block.Hash())

		pm.fetcher.Enqueue(p.id, request.Block)

		// Update the peers total difficulty if needed, schedule a download if gapped
		if request.TD.Cmp(p.Td()) > 0 {
			p.SetTd(request.TD)
			if request.TD.Cmp(new(big.Int).Add(pm.chainman.Td(), request.Block.Difficulty())) > 0 {
				go pm.synchronise(p)
			}
		}

	case msg.Code == TxMsg:
		// Transactions arrived, parse all of them and deliver to the pool
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

			// Log it's arrival for later analysis
			jsonlogger.LogJson(&logger.EthTxReceived{
				TxHash:   tx.Hash().Hex(),
				RemoteId: p.ID().String(),
			})
		}
		pm.txpool.AddTransactions(txs)

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
		if parent := pm.chainman.GetBlock(block.ParentHash()); parent != nil {
			td = new(big.Int).Add(block.Difficulty(), pm.chainman.GetTd(block.ParentHash()))
		} else {
			glog.V(logger.Error).Infof("propagating dangling block #%d [%x]", block.NumberU64(), hash[:4])
			return
		}
		// Send the block to a subset of our peers
		transfer := peers[:int(math.Sqrt(float64(len(peers))))]
		for _, peer := range transfer {
			peer.SendNewBlock(block, td)
		}
		glog.V(logger.Detail).Infof("propagated block %x to %d peers in %v", hash[:4], len(transfer), time.Since(block.ReceivedAt))
	}
	// Otherwise if the block is indeed in out own chain, announce it
	if pm.chainman.HasBlock(hash) {
		for _, peer := range peers {
			if peer.version < eth62 {
				peer.SendNewBlockHashes61([]common.Hash{hash})
			} else {
				peer.SendNewBlockHashes([]common.Hash{hash}, []uint64{block.NumberU64()})
			}
		}
		glog.V(logger.Detail).Infof("announced block %x to %d peers in %v", hash[:4], len(peers), time.Since(block.ReceivedAt))
	}
}

// BroadcastTx will propagate a transaction to all peers which are not known to
// already have the given transaction.
func (pm *ProtocolManager) BroadcastTx(hash common.Hash, tx *types.Transaction) {
	// Broadcast transaction to a batch of peers not knowing about it
	peers := pm.peers.PeersWithoutTx(hash)
	//FIXME include this again: peers = peers[:int(math.Sqrt(float64(len(peers))))]
	for _, peer := range peers {
		peer.SendTransactions(types.Transactions{tx})
	}
	glog.V(logger.Detail).Infoln("broadcast tx to", len(peers), "peers")
}

// Mined broadcast loop
func (self *ProtocolManager) minedBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.minedBlockSub.Chan() {
		switch ev := obj.(type) {
		case core.NewMinedBlockEvent:
			self.BroadcastBlock(ev.Block, true)  // First propagate block to peers
			self.BroadcastBlock(ev.Block, false) // Only then announce to the rest
		}
	}
}

func (self *ProtocolManager) txBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.txSub.Chan() {
		event := obj.(core.TxPreEvent)
		self.BroadcastTx(event.Tx.Hash(), event.Tx)
	}
}
