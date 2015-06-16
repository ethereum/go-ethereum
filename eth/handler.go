package eth

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/eth/fetcher"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
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

	SubProtocol p2p.Protocol

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
func NewProtocolManager(protocolVersion, networkId int, mux *event.TypeMux, txpool txPool, chainman *core.ChainManager) *ProtocolManager {
	// Create the protocol manager and initialize peer handlers
	manager := &ProtocolManager{
		eventMux:  mux,
		txpool:    txpool,
		chainman:  chainman,
		peers:     newPeerSet(),
		newPeerCh: make(chan *peer, 1),
		txsyncCh:  make(chan *txsync),
		quitSync:  make(chan struct{}),
	}
	manager.SubProtocol = p2p.Protocol{
		Name:    "eth",
		Version: uint(protocolVersion),
		Length:  ProtocolLength,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			peer := manager.newPeer(protocolVersion, networkId, p, rw)
			manager.newPeerCh <- peer
			return manager.handle(peer)
		},
	}
	// Construct the different synchronisation mechanisms
	manager.downloader = downloader.New(manager.eventMux, manager.chainman.HasBlock, manager.chainman.GetBlock, manager.chainman.InsertChain, manager.removePeer)

	importer := func(peer string, block *types.Block) error {
		if p := manager.peers.Peer(peer); p != nil {
			return manager.importBlock(manager.peers.Peer(peer), block, nil)
		}
		return errors.New("unknown peer")
	}
	heighter := func() uint64 {
		return manager.chainman.CurrentBlock().NumberU64()
	}
	manager.fetcher = fetcher.New(manager.chainman.HasBlock, importer, heighter)

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
	td, current, genesis := pm.chainman.Status()

	return newPeer(pv, nv, genesis, current, td, p, rw)
}

func (pm *ProtocolManager) handle(p *peer) error {
	// Execute the Ethereum handshake.
	if err := p.handleStatus(); err != nil {
		return err
	}

	// Register the peer locally.
	glog.V(logger.Detail).Infoln("Adding peer", p.id)
	if err := pm.peers.Register(p); err != nil {
		glog.V(logger.Error).Infoln("Addition failed:", err)
		return err
	}
	defer pm.removePeer(p.id)

	// Register the peer in the downloader. If the downloader
	// considers it banned, we disconnect.
	if err := pm.downloader.RegisterPeer(p.id, p.Head(), p.requestHashes, p.requestBlocks); err != nil {
		return err
	}

	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	pm.syncTransactions(p)

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			return err
		}
	}
	return nil
}

func (self *ProtocolManager) handleMsg(p *peer) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// make sure that the payload has been fully consumed
	defer msg.Discard()

	switch msg.Code {
	case StatusMsg:
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	case TxMsg:
		// TODO: rework using lazy RLP stream
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for i, tx := range txs {
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			jsonlogger.LogJson(&logger.EthTxReceived{
				TxHash:   tx.Hash().Hex(),
				RemoteId: p.ID().String(),
			})
		}
		self.txpool.AddTransactions(txs)

	case GetBlockHashesMsg:
		var request getBlockHashesMsgData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "->msg %v: %v", msg, err)
		}

		if request.Amount > uint64(downloader.MaxHashFetch) {
			request.Amount = uint64(downloader.MaxHashFetch)
		}

		hashes := self.chainman.GetBlockHashesFromHash(request.Hash, request.Amount)

		if glog.V(logger.Debug) {
			if len(hashes) == 0 {
				glog.Infof("invalid block hash %x", request.Hash.Bytes()[:4])
			}
		}

		// returns either requested hashes or nothing (i.e. not found)
		return p.sendBlockHashes(hashes)

	case BlockHashesMsg:
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))

		var hashes []common.Hash
		if err := msgStream.Decode(&hashes); err != nil {
			break
		}
		err := self.downloader.DeliverHashes(p.id, hashes)
		if err != nil {
			glog.V(logger.Debug).Infoln(err)
		}

	case GetBlocksMsg:
		var blocks []*types.Block

		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		var (
			i         int
			totalsize common.StorageSize
		)
		for {
			i++
			var hash common.Hash
			err := msgStream.Decode(&hash)
			if err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}

			block := self.chainman.GetBlock(hash)
			if block != nil {
				blocks = append(blocks, block)
				totalsize += block.Size()
			}
			if i == downloader.MaxBlockFetch || totalsize > maxBlockRespSize {
				break
			}
		}
		return p.sendBlocks(blocks)

	case BlocksMsg:
		// Decode the arrived block message
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))

		var blocks []*types.Block
		if err := msgStream.Decode(&blocks); err != nil {
			glog.V(logger.Detail).Infoln("Decode error", err)
			blocks = nil
		}
		// Filter out any explicitly requested blocks, deliver the rest to the downloader
		if blocks := self.fetcher.Filter(blocks); len(blocks) > 0 {
			self.downloader.DeliverBlocks(p.id, blocks)
		}

	case NewBlockHashesMsg:
		// Retrieve and deseralize the remote new block hashes notification
		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))

		var hashes []common.Hash
		if err := msgStream.Decode(&hashes); err != nil {
			break
		}
		// Mark the hashes as present at the remote node
		for _, hash := range hashes {
			p.blockHashes.Add(hash)
			p.SetHead(hash)
		}
		// Schedule all the unknown hashes for retrieval
		unknown := make([]common.Hash, 0, len(hashes))
		for _, hash := range hashes {
			if !self.chainman.HasBlock(hash) {
				unknown = append(unknown, hash)
			}
		}
		for _, hash := range unknown {
			self.fetcher.Notify(p.id, hash, time.Now(), p.requestBlocks)
		}

	case NewBlockMsg:
		var request newBlockMsgData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if err := request.Block.ValidateFields(); err != nil {
			return errResp(ErrDecode, "block validation %v: %v", msg, err)
		}
		request.Block.ReceivedAt = msg.ReceivedAt

		if err := self.importBlock(p, request.Block, request.TD); err != nil {
			return err
		}

	default:
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

// importBlocks injects a new block retrieved from the given peer into the chain
// manager.
func (pm *ProtocolManager) importBlock(p *peer, block *types.Block, td *big.Int) error {
	hash := block.Hash()

	// Mark the block as present at the remote node (don't duplicate already held data)
	p.blockHashes.Add(hash)
	p.SetHead(hash)
	if td != nil {
		p.SetTd(td)
	}
	// Log the block's arrival
	_, chainHead, _ := pm.chainman.Status()
	jsonlogger.LogJson(&logger.EthChainReceivedNewBlock{
		BlockHash:     hash.Hex(),
		BlockNumber:   block.Number(),
		ChainHeadHash: chainHead.Hex(),
		BlockPrevHash: block.ParentHash().Hex(),
		RemoteId:      p.ID().String(),
	})
	// If the block's already known or its difficulty is lower than ours, drop
	if pm.chainman.HasBlock(hash) {
		p.SetTd(pm.chainman.GetBlock(hash).Td) // update the peer's TD to the real value
		return nil
	}
	if td != nil && pm.chainman.Td().Cmp(td) > 0 && new(big.Int).Add(block.Number(), big.NewInt(7)).Cmp(pm.chainman.CurrentBlock().Number()) < 0 {
		glog.V(logger.Debug).Infof("[%s] dropped block %v due to low TD %v\n", p.id, block.Number(), td)
		return nil
	}
	// Attempt to insert the newly received block and propagate to our peers
	if pm.chainman.HasBlock(block.ParentHash()) {
		if _, err := pm.chainman.InsertChain(types.Blocks{block}); err != nil {
			glog.V(logger.Error).Infoln("removed peer (", p.id, ") due to block error", err)
			return err
		}
		if td != nil && block.Td.Cmp(td) != 0 {
			err := fmt.Errorf("invalid TD on block(%v) from peer(%s): block.td=%v, request.td=%v", block.Number(), p.id, block.Td, td)
			glog.V(logger.Error).Infoln(err)
			return err
		}
		pm.BroadcastBlock(hash, block)
		return nil
	}
	// Parent of the block is unknown, try to sync with this peer if it seems to be good
	if td != nil {
		go pm.synchronise(p)
	}
	return nil
}

// BroadcastBlock will propagate the block to a subset of its connected peers,
// only notifying the rest of the block's appearance.
func (pm *ProtocolManager) BroadcastBlock(hash common.Hash, block *types.Block) {
	// Retrieve all the target peers and split between full broadcast or only notification
	peers := pm.peers.PeersWithoutBlock(hash)
	split := int(math.Sqrt(float64(len(peers))))

	transfer := peers[:split]
	notify := peers[split:]

	// Send out the data transfers and the notifications
	for _, peer := range notify {
		peer.sendNewBlockHashes([]common.Hash{hash})
	}
	glog.V(logger.Detail).Infoln("broadcast hash to", len(notify), "peers.")

	for _, peer := range transfer {
		peer.sendNewBlock(block)
	}
	glog.V(logger.Detail).Infoln("broadcast block to", len(transfer), "peers. Total processing time:", time.Since(block.ReceivedAt))
}

// BroadcastTx will propagate the block to its connected peers. It will sort
// out which peers do not contain the block in their block set and will do a
// sqrt(peers) to determine the amount of peers we broadcast to.
func (pm *ProtocolManager) BroadcastTx(hash common.Hash, tx *types.Transaction) {
	// Broadcast transaction to a batch of peers not knowing about it
	peers := pm.peers.PeersWithoutTx(hash)
	//FIXME include this again: peers = peers[:int(math.Sqrt(float64(len(peers))))]
	for _, peer := range peers {
		peer.sendTransaction(tx)
	}
	glog.V(logger.Detail).Infoln("broadcast tx to", len(peers), "peers")
}

// Mined broadcast loop
func (self *ProtocolManager) minedBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range self.minedBlockSub.Chan() {
		switch ev := obj.(type) {
		case core.NewMinedBlockEvent:
			self.BroadcastBlock(ev.Block.Hash(), ev.Block)
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
