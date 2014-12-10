package eth

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	ethlogger "github.com/ethereum/go-ethereum/logger"
)

var poolLogger = ethlogger.NewLogger("Blockpool")

const (
	blockHashesBatchSize   = 256
	blockBatchSize         = 64
	blockRequestInterval   = 10 // seconds
	blockRequestRepetition = 1
	cacheTimeout           = 3 // minutes
	blockTimeout           = 5 // minutes
)

type poolNode struct {
	hash                []byte
	block               *types.Block
	child               *poolNode
	parent              *poolNode
	root                *nodePointer
	knownParent         bool
	suicide             chan bool
	peer                string
	source              string
	blockRequestRoot    bool
	blockRequestControl *bool
	blockRequestQuit    *(chan bool)
}

// the minimal interface for chain manager
type chainManager interface {
	KnownBlock(hash []byte) bool
	AddBlock(*types.Block) error
	CheckPoW(*types.Block) bool
}

type BlockPool struct {
	chainManager chainManager
	eventer      event.TypeMux

	// pool     Pool
	lock sync.Mutex
	pool map[string]*poolNode

	peersLock sync.Mutex
	peers     map[string]*peerInfo
	peer      *peerInfo

	quit    chan bool
	wg      sync.WaitGroup
	running bool
}

type peerInfo struct {
	td                 *big.Int
	currentBlock       []byte
	id                 string
	requestBlockHashes func([]byte) error
	requestBlocks      func([][]byte) error
	invalidBlock       func(error)
}

type nodePointer struct {
	hash []byte
}

type peerChangeEvent struct {
	*peerInfo
}

func NewBlockPool(chMgr chainManager) *BlockPool {
	return &BlockPool{
		chainManager: chMgr,
		pool:         make(map[string]*poolNode),
		peers:        make(map[string]*peerInfo),
		quit:         make(chan bool),
		running:      true,
	}
}

func (self *BlockPool) Stop() {
	self.lock.Lock()
	if !self.running {
		self.lock.Unlock()
		return
	}
	self.running = false
	self.lock.Unlock()

	poolLogger.Infoln("Stopping")

	close(self.quit)
	self.wg.Wait()
	poolLogger.Infoln("Stopped")

}

// Entry point for eth protocol to add block hashes received via BlockHashesMsg
// only hashes from the best peer is handled
// this method is always responsible to initiate further hash requests until
// a known parent is reached unless cancelled by a peerChange event
func (self *BlockPool) AddBlockHashes(next func() ([]byte, bool), peerId string) {
	// subscribe to peerChangeEvent before we check for best peer
	peerChange := self.eventer.Subscribe(peerChangeEvent{})
	defer peerChange.Unsubscribe()
	// check if this peer is the best
	peer, best := self.getPeer(peerId)
	if !best {
		return
	}
	root := &nodePointer{}
	// peer is still the best
	hashes := make(chan []byte)
	var lastPoolNode *poolNode

	// using a for select loop so that peer change (new best peer) can abort the parallel thread that processes hashes of the earlier best peer
	for {
		hash, ok := next()
		if ok {
			hashes <- hash
		} else {
			break
		}
		select {
		case <-self.quit:
			return
		case <-peerChange.Chan():
			// remember where we left off with this peer
			if lastPoolNode != nil {
				root.hash = lastPoolNode.hash
				go self.killChain(lastPoolNode)
			}
		case hash := <-hashes:
			self.lock.Lock()
			defer self.lock.Unlock()
			// check if known block connecting the downloaded chain to our blockchain
			if self.chainManager.KnownBlock(hash) {
				poolLogger.Infof("known block (%x...)\n", hash[0:4])
				if lastPoolNode != nil {
					lastPoolNode.knownParent = true
					go self.requestBlocksLoop(lastPoolNode)
				} else {
					// all hashes known if topmost one is in blockchain
				}
				return
			}
			//
			var currentPoolNode *poolNode
			// check if lastPoolNode has the correct parent node (hash matching),
			// then just assign to currentPoolNode
			if lastPoolNode != nil && lastPoolNode.parent != nil && bytes.Compare(lastPoolNode.parent.hash, hash) == 0 {
				currentPoolNode = lastPoolNode.parent
			} else {
				// otherwise look up in pool
				currentPoolNode = self.pool[string(hash)]
				// if node does not exist, create it and index in the pool
				if currentPoolNode == nil {
					currentPoolNode = &poolNode{
						hash: hash,
					}
					self.pool[string(hash)] = currentPoolNode
				}
			}
			// set up parent-child nodes (doubly linked list)
			self.link(currentPoolNode, lastPoolNode)
			// ! we trust the node iff
			// (1) node marked as by the same peer or
			// (2) it has a PoW valid block retrieved
			if currentPoolNode.peer == peer.id || currentPoolNode.block != nil {
				// the trusted checkpoint from which we request hashes down to known head
				lastPoolNode = self.pool[string(currentPoolNode.root.hash)]
				break
			}
			currentPoolNode.peer = peer.id
			currentPoolNode.root = root
			lastPoolNode = currentPoolNode
		}
	}
	// lastPoolNode is nil if and only if the node with stored root hash is already cleaned up
	// after valid block insertion, therefore in this case the blockpool active chain is connected to the blockchain, so no need to request further hashes or request blocks
	if lastPoolNode != nil {
		root.hash = lastPoolNode.hash
		peer.requestBlockHashes(lastPoolNode.hash)
		go self.requestBlocksLoop(lastPoolNode)
	}
	return
}

func (self *BlockPool) requestBlocksLoop(node *poolNode) {
	suicide := time.After(blockTimeout * time.Minute)
	requestTimer := time.After(0)
	var controlChan chan bool
	closedChan := make(chan bool)
	quit := make(chan bool)
	close(closedChan)
	requestBlocks := true
	origNode := node
	self.lock.Lock()
	node.blockRequestRoot = true
	b := false
	control := &b
	node.blockRequestControl = control
	node.blockRequestQuit = &quit
	self.lock.Unlock()
	blocks := 0
	self.wg.Add(1)
loop:
	for {
		if requestBlocks {
			controlChan = closedChan
		} else {
			self.lock.Lock()
			if *node.blockRequestControl {
				controlChan = closedChan
				*node.blockRequestControl = false
			}
			self.lock.Unlock()
		}
		select {
		case <-quit:
			break loop
		case <-suicide:
			go self.killChain(origNode)
			break loop

		case <-requestTimer:
			requestBlocks = true

		case <-controlChan:
			controlChan = nil
			// this iteration takes care of requesting blocks only starting from the first node with a missing block (moving target),
			// max up to the next checkpoint (n.blockRequestRoot true)
			nodes := []*poolNode{}
			n := node
			next := node
			self.lock.Lock()
			for n != nil && (n == node || !n.blockRequestRoot) && (requestBlocks || n.block != nil) {
				if n.block != nil {
					if len(nodes) == 0 {
						// nil control indicates that node is not needed anymore
						// block can be inserted to blockchain and deleted if knownParent
						n.blockRequestControl = nil
						blocks++
						next = next.child
					} else {
						// this is needed to indicate that when a new chain forks from an existing one
						// triggering a reorg will ? renew the blockTimeout period ???
						// if there is a block but control == nil should start fetching blocks, see link function
						n.blockRequestControl = control
					}
				} else {
					nodes = append(nodes, n)
					n.blockRequestControl = control
				}
				n = n.child
			}
			// if node is connected to the blockchain, we can immediately start inserting
			// blocks to the blockchain and delete nodes
			if node.knownParent {
				go self.insertChainFrom(node)
			}
			if next.blockRequestRoot && next != node {
				// no more missing blocks till the checkpoint, quitting
				poolLogger.Debugf("fetched %v blocks on active chain, batch %v-%v", blocks, origNode, n)
				break loop
			}
			self.lock.Unlock()

			// reset starting node to the first descendant node with missing block
			node = next
			if !requestBlocks {
				continue
			}
			go self.requestBlocks(nodes)
			requestTimer = time.After(blockRequestInterval * time.Second)
		}
	}
	self.wg.Done()
	return
}

func (self *BlockPool) requestBlocks(nodes []*poolNode) {
	// distribute block request among known peers
	self.peersLock.Lock()
	peerCount := len(self.peers)
	poolLogger.Debugf("requesting %v missing blocks from %v peers", len(nodes), peerCount)
	blockHashes := make([][][]byte, peerCount)
	repetitions := int(math.Max(float64(peerCount)/2.0, float64(blockRequestRepetition)))
	for n, node := range nodes {
		for i := 0; i < repetitions; i++ {
			blockHashes[n%peerCount] = append(blockHashes[n%peerCount], node.hash)
			n++
		}
	}
	i := 0
	for _, peer := range self.peers {
		peer.requestBlocks(blockHashes[i])
		i++
	}
	self.peersLock.Unlock()
}

func (self *BlockPool) insertChainFrom(node *poolNode) {
	self.lock.Lock()
	defer self.lock.Unlock()
	for node != nil && node.blockRequestControl == nil {
		err := self.chainManager.AddBlock(node.block)
		if err != nil {
			poolLogger.Debugf("invalid block %v", node.hash)
			poolLogger.Debugf("penalise peers %v (hash), %v (block)", node.peer, node.source)
			// penalise peer in node.source
			go self.killChain(node)
			return
		}
		poolLogger.Debugf("insert block %v into blockchain", node.hash)
		node = node.child
	}
	// if block insertion succeeds, mark the child as knownParent
	// trigger request blocks reorg
	if node != nil {
		node.knownParent = true
		*(node.blockRequestControl) = true
	}
}

// AddPeer is called by the eth protocol instance running on the peer after
// the status message has been received with total difficulty and current block hash
// AddPeer can only be used once, RemovePeer needs to be called when the peer disconnects
func (self *BlockPool) AddPeer(td *big.Int, currentBlock []byte, peerId string, requestBlockHashes func([]byte) error, requestBlocks func([][]byte) error, invalidBlock func(error)) bool {
	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	if self.peers[peerId] != nil {
		panic("peer already added")
	}
	info := &peerInfo{
		td:                 td,
		currentBlock:       currentBlock,
		id:                 peerId, //peer.Identity().Pubkey()
		requestBlockHashes: requestBlockHashes,
		requestBlocks:      requestBlocks,
		invalidBlock:       invalidBlock,
	}
	self.peers[peerId] = info
	poolLogger.Debugf("add new peer %v with td %v", peerId, td)
	currentTD := ethutil.Big0
	if self.peer != nil {
		currentTD = self.peer.td
	}
	if td.Cmp(currentTD) > 0 {
		self.peer = info
		self.eventer.Post(peerChangeEvent{info})
		poolLogger.Debugf("peer %v promoted to best peer", peerId)
		requestBlockHashes(currentBlock)
		return true
	}
	return false
}

// RemovePeer is called by the eth protocol when the peer disconnects
func (self *BlockPool) RemovePeer(peerId string) {
	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	if self.peers[peerId] != nil {
		panic("peer already removed")
	}
	self.peers[peerId] = nil
	poolLogger.Debugf("remove peer %v", peerId[0:4])

	// if current best peer is removed, need find a better one
	if peerId == self.peer.id {
		var newPeer *peerInfo
		max := ethutil.Big0
		// peer with the highest self-acclaimed TD is chosen
		for _, info := range self.peers {
			if info.td.Cmp(max) > 0 {
				max = info.td
				newPeer = info
			}
		}
		self.peer = newPeer
		self.eventer.Post(peerChangeEvent{newPeer})
		if newPeer != nil {
			poolLogger.Debugf("peer %v with td %v spromoted to best peer", newPeer.id[0:4], newPeer.td)
			newPeer.requestBlockHashes(newPeer.currentBlock)
		} else {
			poolLogger.Warnln("no peers left")
		}
	}
}

func (self *BlockPool) getPeer(peerId string) (*peerInfo, bool) {
	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	if self.peer.id == peerId {
		return self.peer, true
	}
	info, ok := self.peers[peerId]
	if !ok {
		panic("unknown peer")
	}
	return info, false
}

// if same peer gave different chain before, this will overwrite it
// if currentPoolNode existed as a non-leaf node the earlier fork is delinked
// if same parent hash is found, we can abort, we do not allow the same peer to change minds about parent of same hash, if errored first time round, will get penalized.
// if lastPoolNode had a different parent the earlier parent (with entire subtree) is delinked, this situation cannot normally arise though
// just in case reset lastPoolNode as non-root (unlikely)

func (self *BlockPool) link(parent, child *poolNode) {
	// reactivate node scheduled for suicide
	if parent.suicide != nil {
		close(parent.suicide)
		parent.suicide = nil
	}
	if parent.child != child {
		orphan := parent.child
		orphan.parent = nil
		go self.killChain(orphan)
		parent.child = child
	}
	if child != nil {
		if child.parent != parent {
			orphan := child.parent
			orphan.child = nil
			go func() {
				// if it is a aberrant reverse fork, zip down to bottom
				for orphan.parent != nil {
					orphan = orphan.parent
				}
				self.killChain(orphan)
			}()
			child.parent = parent
		}
		child.knownParent = false
	}
}

func (self *BlockPool) killChain(node *poolNode) {
	if node == nil {
		return
	}
	poolLogger.Debugf("suicide scheduled on node %v", node)
	suicide := make(chan bool)
	self.lock.Lock()
	node.suicide = suicide
	self.lock.Unlock()
	timer := time.After(cacheTimeout * time.Minute)
	self.wg.Add(1)
	select {
	case <-self.quit:
	case <-suicide:
		// cancel suicide = close node.suicide to reactivate node
	case <-timer:
		poolLogger.Debugf("suicide on node %v", node)
		self.lock.Lock()
		defer self.lock.Unlock()
		// proceed up via child links until another suicide root found or chain ends
		// abort request blocks loops that start above
		// and delete nodes from pool then quit the suicide process
		okToAbort := node.blockRequestRoot
		for node != nil && (node.suicide == suicide || node.suicide == nil) {
			self.pool[string(node.hash)] = nil
			if okToAbort && node.blockRequestQuit != nil {
				quit := *(node.blockRequestQuit)
				if quit != nil { // not yet closed
					*(node.blockRequestQuit) = nil
					close(quit)
				}
			} else {
				okToAbort = true
			}
			node = node.child
		}
	}
	self.wg.Done()
}

// AddBlock is the entry point for the eth protocol when blockmsg is received upon requests
// It has a strict interpretation of the protocol in that if the block received has not been requested, it results in an error (which can be ignored)
// block is checked for PoW
// only the first PoW-valid block for a hash is considered legit
func (self *BlockPool) AddBlock(block *types.Block, peerId string) (err error) {
	hash := block.Hash()
	self.lock.Lock()
	defer self.lock.Unlock()
	node, ok := self.pool[string(hash)]
	if !ok && !self.chainManager.KnownBlock(hash) {
		return fmt.Errorf("unrequested block %x", hash)
	}
	if node.block != nil {
		return
	}
	// validate block for PoW
	if !self.chainManager.CheckPoW(block) {
		return fmt.Errorf("invalid pow on block %x", hash)
	}
	node.block = block
	node.source = peerId
	return nil
}
