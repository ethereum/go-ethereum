package eth

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/fetcher"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/rlp"
)

// This is the target maximum size of returned blocks for the
// getBlocks message. The reply message may exceed it
// if a single block is larger than the limit.
const maxBlockRespSize = 2 * 1024 * 1024

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
	protVer, netId int
	txpool         txPool
	chainman       *core.ChainManager
	downloader     *downloader.Downloader
	fetcher        *fetcher.Fetcher
	peers          *peerSet

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
func NewProtocolManager(networkId int, mux *event.TypeMux, txpool txPool, pow pow.PoW, chainman *core.ChainManager) *ProtocolManager {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		eventMux:  mux,
		txpool:    txpool,
		chainman:  chainman,
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
	manager.downloader = downloader.New(manager.eventMux, manager.chainman.HasBlock, manager.chainman.GetBlock, manager.chainman.InsertChain, manager.removePeer)

	validator := func(block *types.Block, parent *types.Block) error {
		return core.ValidateHeader(pow, block.Header(), parent, true)
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
	return newPeer(pv, nv, p, rw)
}

// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
	glog.V(logger.Debug).Infof("%v: peer connected", p)

	// Execute the Ethereum handshake
	td, head, genesis := pm.chainman.Status()
	if err := p.Handshake(td, head, genesis); err != nil {
		glog.V(logger.Debug).Infof("%v: handshake failed: %v", p, err)
		return err
	}
	// Register the peer locally
	glog.V(logger.Detail).Infof("%v: adding peer", p)
	if err := pm.peers.Register(p); err != nil {
		glog.V(logger.Error).Infof("%v: addition failed: %v", p, err)
		return err
	}
	defer pm.removePeer(p.id)

	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := pm.downloader.RegisterPeer(p.id, p.Head(), p.RequestHashes, p.RequestBlocks); err != nil {
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
	switch msg.Code {
	case StatusMsg:
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	case TxMsg:
		// Transactions arrived, parse all of them and deliver to the pool
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		propTxnInPacketsMeter.Mark(1)
		for i, tx := range txs {
			// Validate and mark the remote transaction
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			p.MarkTransaction(tx.Hash())

			// Log it's arrival for later analysis
			propTxnInTrafficMeter.Mark(tx.Size().Int64())
			jsonlogger.LogJson(&logger.EthTxReceived{
				TxHash:   tx.Hash().Hex(),
				RemoteId: p.ID().String(),
			})
		}
		pm.txpool.AddTransactions(txs)

	case GetBlockHashesMsg:
		var request getBlockHashesMsgData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "->msg %v: %v", msg, err)
		}

		if request.Amount > uint64(downloader.MaxHashFetch) {
			request.Amount = uint64(downloader.MaxHashFetch)
		}

		hashes := pm.chainman.GetBlockHashesFromHash(request.Hash, request.Amount)

		if glog.V(logger.Debug) {
			if len(hashes) == 0 {
				glog.Infof("invalid block hash %x", request.Hash.Bytes()[:4])
			}
		}

		// returns either requested hashes or nothing (i.e. not found)
		return p.SendBlockHashes(hashes)

	case BlockHashesMsg:
		// A batch of hashes arrived to one of our previous requests
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		reqHashInPacketsMeter.Mark(1)

		var hashes []common.Hash
		if err := msgStream.Decode(&hashes); err != nil {
			break
		}
		reqHashInTrafficMeter.Mark(int64(32 * len(hashes)))

		// Deliver them all to the downloader for queuing
		err := pm.downloader.DeliverHashes(p.id, hashes)
		if err != nil {
			glog.V(logger.Debug).Infoln(err)
		}

	case GetBlocksMsg:
		// Decode the retrieval message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		// Gather blocks until the fetch or network limits is reached
		var (
			hash   common.Hash
			bytes  common.StorageSize
			hashes []common.Hash
			blocks []*types.Block
		)
		for {
			err := msgStream.Decode(&hash)
			if err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}
			hashes = append(hashes, hash)

			// Retrieve the requested block, stopping if enough was found
			if block := pm.chainman.GetBlock(hash); block != nil {
				blocks = append(blocks, block)
				bytes += block.Size()
				if len(blocks) >= downloader.MaxBlockFetch || bytes > maxBlockRespSize {
					break
				}
			}
		}
		if glog.V(logger.Detail) && len(blocks) == 0 && len(hashes) > 0 {
			list := "["
			for _, hash := range hashes {
				list += fmt.Sprintf("%x, ", hash[:4])
			}
			list = list[:len(list)-2] + "]"

			glog.Infof("%v: no blocks found for requested hashes %s", p, list)
		}
		return p.SendBlocks(blocks)

	case BlocksMsg:
		// Decode the arrived block message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		reqBlockInPacketsMeter.Mark(1)

		var blocks []*types.Block
		if err := msgStream.Decode(&blocks); err != nil {
			glog.V(logger.Detail).Infoln("Decode error", err)
			blocks = nil
		}
		// Update the receive timestamp of each block
		for _, block := range blocks {
			reqBlockInTrafficMeter.Mark(block.Size().Int64())
			block.ReceivedAt = msg.ReceivedAt
		}
		// Filter out any explicitly requested blocks, deliver the rest to the downloader
		if blocks := pm.fetcher.Filter(blocks); len(blocks) > 0 {
			pm.downloader.DeliverBlocks(p.id, blocks)
		}

	case NewBlockHashesMsg:
		// Retrieve and deseralize the remote new block hashes notification
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))

		var hashes []common.Hash
		if err := msgStream.Decode(&hashes); err != nil {
			break
		}
		propHashInPacketsMeter.Mark(1)
		propHashInTrafficMeter.Mark(int64(32 * len(hashes)))

		// Mark the hashes as present at the remote node
		for _, hash := range hashes {
			p.MarkBlock(hash)
			p.SetHead(hash)
		}
		// Schedule all the unknown hashes for retrieval
		unknown := make([]common.Hash, 0, len(hashes))
		for _, hash := range hashes {
			if !pm.chainman.HasBlock(hash) {
				unknown = append(unknown, hash)
			}
		}
		for _, hash := range unknown {
			pm.fetcher.Notify(p.id, hash, time.Now(), p.RequestBlocks)
		}

	case NewBlockMsg:
		// Retrieve and decode the propagated block
		var request newBlockMsgData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		propBlockInPacketsMeter.Mark(1)
		propBlockInTrafficMeter.Mark(request.Block.Size().Int64())

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

		// TODO: Schedule a sync to cover potential gaps (this needs proto update)
		p.SetTd(request.TD)
		go pm.synchronise(p)

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
		transfer := peers[:int(math.Sqrt(float64(len(peers))))]
		for _, peer := range transfer {
			peer.SendNewBlock(block)
		}
		glog.V(logger.Detail).Infof("propagated block %x to %d peers in %v", hash[:4], len(transfer), time.Since(block.ReceivedAt))
	}
	// Otherwise if the block is indeed in out own chain, announce it
	if pm.chainman.HasBlock(hash) {
		for _, peer := range peers {
			peer.SendNewBlockHashes([]common.Hash{hash})
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
