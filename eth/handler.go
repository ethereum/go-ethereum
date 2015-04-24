package eth

// XXX Fair warning, most of the code is re-used from the old protocol. Please be aware that most of this will actually change
// The idea is that most of the calls within the protocol will become synchronous.
// Block downloading and block processing will be complete seperate processes
/*
# Possible scenarios

// Synching scenario
// Use the best peer to synchronise
blocks, err := pm.downloader.Synchronise()
if err != nil {
	// handle
	break
}
pm.chainman.InsertChain(blocks)

// Receiving block with known parent
if parent_exist {
	if err := pm.chainman.InsertChain(block); err != nil {
		// handle
		break
	}
	pm.BroadcastBlock(block)
}

// Receiving block with unknown parent
blocks, err := pm.downloader.SynchroniseWithPeer(peer)
if err != nil {
	// handle
	break
}
pm.chainman.InsertChain(blocks)

*/

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
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	peerCountTimeout    = 12 * time.Second // Amount of time it takes for the peer handler to ignore minDesiredPeerCount
	minDesiredPeerCount = 5                // Amount of peers desired to start syncing
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
	protVer, netId int
	txpool         txPool
	chainman       *core.ChainManager
	downloader     *downloader.Downloader

	pmu   sync.Mutex
	peers map[string]*peer

	SubProtocol p2p.Protocol

	eventMux      *event.TypeMux
	txSub         event.Subscription
	minedBlockSub event.Subscription

	newPeerCh chan *peer
	quit      chan struct{}
}

// NewProtocolManager returns a new ethereum sub protocol manager. The Ethereum sub protocol manages peers capable
// with the ethereum network.
func NewProtocolManager(protocolVersion, networkId int, mux *event.TypeMux, txpool txPool, chainman *core.ChainManager, downloader *downloader.Downloader) *ProtocolManager {
	manager := &ProtocolManager{
		eventMux:   mux,
		txpool:     txpool,
		chainman:   chainman,
		downloader: downloader,
		peers:      make(map[string]*peer),
		newPeerCh:  make(chan *peer, 1),
		quit:       make(chan struct{}),
	}
	go manager.peerHandler()

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

	return manager
}

func (pm *ProtocolManager) peerHandler() {
	// itimer is used to determine when to start ignoring `minDesiredPeerCount`
	itimer := time.NewTimer(peerCountTimeout)
out:
	for {
		select {
		case <-pm.newPeerCh:
			// Meet the `minDesiredPeerCount` before we select our best peer
			if len(pm.peers) < minDesiredPeerCount {
				break
			}
			itimer.Stop()

			// Find the best peer
			peer := getBestPeer(pm.peers)
			if peer == nil {
				glog.V(logger.Debug).Infoln("Sync attempt cancelled. No peers available")
				return
			}
			go pm.synchronise(peer)
		case <-itimer.C:
			// The timer will make sure that the downloader keeps an active state
			// in which it attempts to always check the network for highest td peers
			// Either select the peer or restart the timer if no peers could
			// be selected.
			if peer := getBestPeer(pm.peers); peer != nil {
				go pm.synchronise(peer)
			} else {
				itimer.Reset(5 * time.Second)
			}
		case <-pm.quit:
			break out
		}
	}
}

func (pm *ProtocolManager) synchronise(peer *peer) {
	// Get the hashes from the peer (synchronously)
	_, err := pm.downloader.Synchronise(peer.id, peer.recentHash)
	if err != nil {
		// handle error
		glog.V(logger.Debug).Infoln("error downloading:", err)
	}
}

func (pm *ProtocolManager) Start() {
	// broadcast transactions
	pm.txSub = pm.eventMux.Subscribe(core.TxPreEvent{})
	go pm.txBroadcastLoop()

	// broadcast mined blocks
	pm.minedBlockSub = pm.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go pm.minedBroadcastLoop()
}

func (pm *ProtocolManager) Stop() {
	pm.txSub.Unsubscribe()         // quits txBroadcastLoop
	pm.minedBlockSub.Unsubscribe() // quits blockBroadcastLoop
}

func (pm *ProtocolManager) newPeer(pv, nv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {

	td, current, genesis := pm.chainman.Status()

	return newPeer(pv, nv, genesis, current, td, p, rw)
}

func (pm *ProtocolManager) handle(p *peer) error {
	if err := p.handleStatus(); err != nil {
		return err
	}
	pm.pmu.Lock()
	pm.peers[p.id] = p
	pm.pmu.Unlock()

	pm.downloader.RegisterPeer(p.id, p.td, p.recentHash, p.requestHashes, p.requestBlocks)
	defer func() {
		pm.pmu.Lock()
		defer pm.pmu.Unlock()
		delete(pm.peers, p.id)
		pm.downloader.UnregisterPeer(p.id)
	}()

	// propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	if err := p.sendTransactions(pm.txpool.GetTransactions()); err != nil {
		return err
	}

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
	case GetTxMsg: // ignore
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

		if request.Amount > maxHashes {
			request.Amount = maxHashes
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
		err := self.downloader.AddHashes(p.id, hashes)
		if err != nil {
			glog.V(logger.Debug).Infoln(err)
		}

	case GetBlocksMsg:
		var blocks []*types.Block

		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}
		var i int
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
			}
			if i == maxBlocks {
				break
			}
		}
		return p.sendBlocks(blocks)
	case BlocksMsg:
		var blocks []*types.Block

		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if err := msgStream.Decode(&blocks); err != nil {
			glog.V(logger.Detail).Infoln("Decode error", err)
			blocks = nil
		}
		self.downloader.DeliverChunk(p.id, blocks)

	case NewBlockMsg:
		var request newBlockMsgData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if err := request.Block.ValidateFields(); err != nil {
			return errResp(ErrDecode, "block validation %v: %v", msg, err)
		}
		hash := request.Block.Hash()
		// Add the block hash as a known hash to the peer. This will later be used to determine
		// who should receive this.
		p.blockHashes.Add(hash)

		_, chainHead, _ := self.chainman.Status()

		jsonlogger.LogJson(&logger.EthChainReceivedNewBlock{
			BlockHash:     hash.Hex(),
			BlockNumber:   request.Block.Number(), // this surely must be zero
			ChainHeadHash: chainHead.Hex(),
			BlockPrevHash: request.Block.ParentHash().Hex(),
			RemoteId:      p.ID().String(),
		})

		// Make sure the block isn't already known. If this is the case simply drop
		// the message and move on. If the TD is < currentTd; drop it as well. If this
		// chain at some point becomes canonical, the downloader will fetch it.
		if self.chainman.HasBlock(hash) {
			break
		}
		if self.chainman.Td().Cmp(request.TD) > 0 && new(big.Int).Add(request.Block.Number(), big.NewInt(7)).Cmp(self.chainman.CurrentBlock().Number()) < 0 {
			glog.V(logger.Debug).Infof("[%s] dropped block %v due to low TD %v\n", p.id, request.Block.Number(), request.TD)
			break
		}

		// Attempt to insert the newly received by checking if the parent exists.
		// if the parent exists we process the block and propagate to our peers
		// if the parent does not exists we delegate to the downloader.
		if self.chainman.HasBlock(request.Block.ParentHash()) {
			if err := self.chainman.InsertChain(types.Blocks{request.Block}); err != nil {
				// handle error
				return nil
			}
			self.BroadcastBlock(hash, request.Block)
		} else {
			// adding blocks is synchronous
			go func() {
				err := self.downloader.AddBlock(p.id, request.Block, request.TD)
				if err != nil {
					glog.V(logger.Detail).Infoln("downloader err:", err)
					return
				}
				self.BroadcastBlock(hash, request.Block)
			}()
		}
	default:
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

// BroadcastBlock will propagate the block to its connected peers. It will sort
// out which peers do not contain the block in their block set and will do a
// sqrt(peers) to determine the amount of peers we broadcast to.
func (pm *ProtocolManager) BroadcastBlock(hash common.Hash, block *types.Block) {
	pm.pmu.Lock()
	defer pm.pmu.Unlock()

	// Find peers who don't know anything about the given hash. Peers that
	// don't know about the hash will be a candidate for the broadcast loop
	var peers []*peer
	for _, peer := range pm.peers {
		if !peer.blockHashes.Has(hash) {
			peers = append(peers, peer)
		}
	}
	// Broadcast block to peer set
	peers = peers[:int(math.Sqrt(float64(len(peers))))]
	for _, peer := range peers {
		peer.sendNewBlock(block)
	}
	glog.V(logger.Detail).Infoln("broadcast block to", len(peers), "peers")
}

// BroadcastTx will propagate the block to its connected peers. It will sort
// out which peers do not contain the block in their block set and will do a
// sqrt(peers) to determine the amount of peers we broadcast to.
func (pm *ProtocolManager) BroadcastTx(hash common.Hash, tx *types.Transaction) {
	pm.pmu.Lock()
	defer pm.pmu.Unlock()

	// Find peers who don't know anything about the given hash. Peers that
	// don't know about the hash will be a candidate for the broadcast loop
	var peers []*peer
	for _, peer := range pm.peers {
		if !peer.txHashes.Has(hash) {
			peers = append(peers, peer)
		}
	}
	// Broadcast block to peer set
	peers = peers[:int(math.Sqrt(float64(len(peers))))]
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
