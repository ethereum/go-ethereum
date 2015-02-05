package eth

import (
	"bytes"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	ethlogger "github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow"
)

var poolLogger = ethlogger.NewLogger("Blockpool")

const (
	blockHashesBatchSize       = 256
	blockBatchSize             = 64
	blocksRequestInterval      = 500 // ms
	blocksRequestRepetition    = 1
	blockHashesRequestInterval = 500 // ms
	blocksRequestMaxIdleRounds = 100
	blockHashesTimeout         = 60  // seconds
	blocksTimeout              = 120 // seconds
)

type poolNode struct {
	lock    sync.RWMutex
	hash    []byte
	td      *big.Int
	block   *types.Block
	parent  *poolNode
	peer    string
	blockBy string
}

type poolEntry struct {
	node    *poolNode
	section *section
	index   int
}

type BlockPool struct {
	lock      sync.RWMutex
	chainLock sync.RWMutex

	pool map[string]*poolEntry

	peersLock sync.RWMutex
	peers     map[string]*peerInfo
	peer      *peerInfo

	quit    chan bool
	purgeC  chan bool
	flushC  chan bool
	wg      sync.WaitGroup
	procWg  sync.WaitGroup
	running bool

	// the minimal interface with blockchain
	hasBlock    func(hash []byte) bool
	insertChain func(types.Blocks) error
	verifyPoW   func(pow.Block) bool
}

type peerInfo struct {
	lock sync.RWMutex

	td               *big.Int
	currentBlockHash []byte
	currentBlock     *types.Block
	currentBlockC    chan *types.Block
	parentHash       []byte
	headSection      *section
	headSectionC     chan *section
	id               string

	requestBlockHashes func([]byte) error
	requestBlocks      func([][]byte) error
	peerError          func(int, string, ...interface{})

	sections map[string]*section

	quitC chan bool
}

// structure to store long range links on chain to skip along
type section struct {
	lock        sync.RWMutex
	parent      *section
	child       *section
	top         *poolNode
	bottom      *poolNode
	nodes       []*poolNode
	controlC    chan *peerInfo
	suicideC    chan bool
	blockChainC chan bool
	forkC       chan chan bool
	offC        chan bool
}

func NewBlockPool(hasBlock func(hash []byte) bool, insertChain func(types.Blocks) error, verifyPoW func(pow.Block) bool,
) *BlockPool {
	return &BlockPool{
		hasBlock:    hasBlock,
		insertChain: insertChain,
		verifyPoW:   verifyPoW,
	}
}

// allows restart
func (self *BlockPool) Start() {
	self.lock.Lock()
	if self.running {
		self.lock.Unlock()
		return
	}
	self.running = true
	self.quit = make(chan bool)
	self.flushC = make(chan bool)
	self.pool = make(map[string]*poolEntry)

	self.lock.Unlock()

	self.peersLock.Lock()
	self.peers = make(map[string]*peerInfo)
	self.peersLock.Unlock()

	poolLogger.Infoln("Started")

}

func (self *BlockPool) Stop() {
	self.lock.Lock()
	if !self.running {
		self.lock.Unlock()
		return
	}
	self.running = false

	self.lock.Unlock()

	poolLogger.Infoln("Stopping...")

	close(self.quit)
	//self.wg.Wait()

	self.peersLock.Lock()
	self.peers = nil
	self.peer = nil
	self.peersLock.Unlock()

	self.lock.Lock()
	self.pool = nil
	self.lock.Unlock()

	poolLogger.Infoln("Stopped")
}

func (self *BlockPool) Purge() {
	self.lock.Lock()
	if !self.running {
		self.lock.Unlock()
		return
	}
	self.lock.Unlock()

	poolLogger.Infoln("Purging...")

	close(self.purgeC)
	self.wg.Wait()

	self.purgeC = make(chan bool)

	poolLogger.Infoln("Stopped")

}

func (self *BlockPool) Wait(t time.Duration) {
	self.lock.Lock()
	if !self.running {
		self.lock.Unlock()
		return
	}
	self.lock.Unlock()

	poolLogger.Infoln("Waiting for processes to complete...")
	close(self.flushC)
	w := make(chan bool)
	go func() {
		self.procWg.Wait()
		close(w)
	}()

	select {
	case <-w:
		poolLogger.Infoln("Processes complete")
	case <-time.After(t):
		poolLogger.Warnf("Timeout")
	}
	self.flushC = make(chan bool)
}

// AddPeer is called by the eth protocol instance running on the peer after
// the status message has been received with total difficulty and current block hash
// AddPeer can only be used once, RemovePeer needs to be called when the peer disconnects
func (self *BlockPool) AddPeer(td *big.Int, currentBlockHash []byte, peerId string, requestBlockHashes func([]byte) error, requestBlocks func([][]byte) error, peerError func(int, string, ...interface{})) (best bool) {

	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	peer, ok := self.peers[peerId]
	if ok {
		if bytes.Compare(peer.currentBlockHash, currentBlockHash) != 0 {
			poolLogger.Debugf("Update peer %v with td %v and current block %s", peerId, td, name(currentBlockHash))
			peer.lock.Lock()
			peer.td = td
			peer.currentBlockHash = currentBlockHash
			peer.currentBlock = nil
			peer.parentHash = nil
			peer.headSection = nil
			peer.lock.Unlock()
		}
	} else {
		peer = &peerInfo{
			td:                 td,
			currentBlockHash:   currentBlockHash,
			id:                 peerId, //peer.Identity().Pubkey()
			requestBlockHashes: requestBlockHashes,
			requestBlocks:      requestBlocks,
			peerError:          peerError,
			sections:           make(map[string]*section),
			currentBlockC:      make(chan *types.Block),
			headSectionC:       make(chan *section),
		}
		self.peers[peerId] = peer
		poolLogger.Debugf("add new peer %v with td %v and current block %x", peerId, td, currentBlockHash[:4])
	}
	// check peer current head
	if self.hasBlock(currentBlockHash) {
		// peer not ahead
		return false
	}

	if self.peer == peer {
		// new block update
		// peer is already active best peer, request hashes
		poolLogger.Debugf("[%s] already the best peer. Request new head section info from %s", peerId, name(currentBlockHash))
		peer.headSectionC <- nil
		best = true
	} else {
		currentTD := ethutil.Big0
		if self.peer != nil {
			currentTD = self.peer.td
		}
		if td.Cmp(currentTD) > 0 {
			poolLogger.Debugf("peer %v promoted best peer", peerId)
			self.switchPeer(self.peer, peer)
			self.peer = peer
			best = true
		}
	}
	return
}

func (self *BlockPool) requestHeadSection(peer *peerInfo) {
	self.wg.Add(1)
	self.procWg.Add(1)
	poolLogger.Debugf("[%s] head section at [%s] requesting info", peer.id, name(peer.currentBlockHash))

	go func() {
		var idle bool
		peer.lock.RLock()
		quitC := peer.quitC
		currentBlockHash := peer.currentBlockHash
		peer.lock.RUnlock()
		blockHashesRequestTimer := time.NewTimer(0)
		blocksRequestTimer := time.NewTimer(0)
		suicide := time.NewTimer(blockHashesTimeout * time.Second)
		blockHashesRequestTimer.Stop()
		defer blockHashesRequestTimer.Stop()
		defer blocksRequestTimer.Stop()

		entry := self.get(currentBlockHash)
		if entry != nil {
			entry.node.lock.RLock()
			currentBlock := entry.node.block
			entry.node.lock.RUnlock()
			if currentBlock != nil {
				peer.lock.Lock()
				peer.currentBlock = currentBlock
				peer.parentHash = currentBlock.ParentHash()
				poolLogger.Debugf("[%s] head block [%s] found", peer.id, name(currentBlockHash))
				peer.lock.Unlock()
				blockHashesRequestTimer.Reset(0)
				blocksRequestTimer.Stop()
			}
		}

	LOOP:
		for {

			select {
			case <-self.quit:
				break LOOP

			case <-quitC:
				poolLogger.Debugf("[%s] head section at [%s] incomplete - quit request loop", peer.id, name(currentBlockHash))
				break LOOP

			case headSection := <-peer.headSectionC:
				peer.lock.Lock()
				peer.headSection = headSection
				if headSection == nil {
					oldBlockHash := currentBlockHash
					currentBlockHash = peer.currentBlockHash
					poolLogger.Debugf("[%s] head section changed [%s] -> [%s]", peer.id, name(oldBlockHash), name(currentBlockHash))
					if idle {
						idle = false
						suicide.Reset(blockHashesTimeout * time.Second)
						self.procWg.Add(1)
					}
					blocksRequestTimer.Reset(blocksRequestInterval * time.Millisecond)
				} else {
					poolLogger.DebugDetailf("[%s] head section at [%s] created", peer.id, name(currentBlockHash))
					if !idle {
						idle = true
						suicide.Stop()
						self.procWg.Done()
					}
				}
				peer.lock.Unlock()
				blockHashesRequestTimer.Stop()

			case <-blockHashesRequestTimer.C:
				poolLogger.DebugDetailf("[%s] head section at [%s] not found, requesting block hashes", peer.id, name(currentBlockHash))
				peer.requestBlockHashes(currentBlockHash)
				blockHashesRequestTimer.Reset(blockHashesRequestInterval * time.Millisecond)

			case currentBlock := <-peer.currentBlockC:
				peer.lock.Lock()
				peer.currentBlock = currentBlock
				peer.parentHash = currentBlock.ParentHash()
				poolLogger.DebugDetailf("[%s] head block [%s] found", peer.id, name(currentBlockHash))
				peer.lock.Unlock()
				if self.hasBlock(currentBlock.ParentHash()) {
					if err := self.insertChain(types.Blocks([]*types.Block{currentBlock})); err != nil {
						peer.peerError(ErrInvalidBlock, "%v", err)
					}
					if !idle {
						idle = true
						suicide.Stop()
						self.procWg.Done()
					}
				} else {
					blockHashesRequestTimer.Reset(0)
				}
				blocksRequestTimer.Stop()

			case <-blocksRequestTimer.C:
				peer.lock.RLock()
				poolLogger.DebugDetailf("[%s] head block [%s] not found, requesting", peer.id, name(currentBlockHash))
				peer.requestBlocks([][]byte{peer.currentBlockHash})
				peer.lock.RUnlock()
				blocksRequestTimer.Reset(blocksRequestInterval * time.Millisecond)

			case <-suicide.C:
				peer.peerError(ErrInsufficientChainInfo, "peer failed to provide block hashes or head block for block hash %x", currentBlockHash)
				break LOOP
			}
		}
		self.wg.Done()
		if !idle {
			self.procWg.Done()
		}
	}()
}

// RemovePeer is called by the eth protocol when the peer disconnects
func (self *BlockPool) RemovePeer(peerId string) {
	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	peer, ok := self.peers[peerId]
	if !ok {
		return
	}
	delete(self.peers, peerId)
	poolLogger.Debugf("remove peer %v", peerId)

	// if current best peer is removed, need find a better one
	if self.peer == peer {
		var newPeer *peerInfo
		max := ethutil.Big0
		// peer with the highest self-acclaimed TD is chosen
		for _, info := range self.peers {
			if info.td.Cmp(max) > 0 {
				max = info.td
				newPeer = info
			}
		}
		if newPeer != nil {
			poolLogger.Debugf("peer %v with td %v promoted to best peer", newPeer.id, newPeer.td)
		} else {
			poolLogger.Warnln("no peers")
		}
		self.peer = newPeer
		self.switchPeer(peer, newPeer)
	}
}

// Entry point for eth protocol to add block hashes received via BlockHashesMsg
// only hashes from the best peer is handled
// this method is always responsible to initiate further hash requests until
// a known parent is reached unless cancelled by a peerChange event
// this process also launches all request processes on each chain section
// this function needs to run asynchronously for one peer since the message is discarded???
func (self *BlockPool) AddBlockHashes(next func() ([]byte, bool), peerId string) {

	// register with peer manager loop

	peer, best := self.getPeer(peerId)
	if !best {
		return
	}
	// peer is still the best

	var size, n int
	var hash []byte
	var ok, headSection bool
	var sec, child, parent *section
	var entry *poolEntry
	var nodes []*poolNode
	bestPeer := peer

	hash, ok = next()
	peer.lock.Lock()
	if bytes.Compare(peer.parentHash, hash) == 0 {
		if self.hasBlock(peer.currentBlockHash) {
			return
		}
		poolLogger.Debugf("adding hashes at chain head for best peer %s starting from [%s]", peerId, name(peer.currentBlockHash))
		headSection = true

		if entry := self.get(peer.currentBlockHash); entry == nil {
			node := &poolNode{
				hash:    peer.currentBlockHash,
				block:   peer.currentBlock,
				peer:    peerId,
				blockBy: peerId,
			}
			if size == 0 {
				sec = newSection()
			}
			nodes = append(nodes, node)
			size++
			n++
		} else {
			child = entry.section
		}
	} else {
		poolLogger.Debugf("adding hashes for best peer %s starting from [%s]", peerId, name(hash))
	}
	quitC := peer.quitC
	peer.lock.Unlock()

LOOP:
	// iterate using next (rlp stream lazy decoder) feeding hashesC
	for ; ok; hash, ok = next() {
		n++
		select {
		case <-self.quit:
			return
		case <-quitC:
			// if the peer is demoted, no more hashes taken
			bestPeer = nil
			break LOOP
		default:
		}
		if self.hasBlock(hash) {
			// check if known block connecting the downloaded chain to our blockchain
			poolLogger.DebugDetailf("[%s] known block", name(hash))
			// mark child as absolute pool root with parent known to blockchain
			if sec != nil {
				self.connectToBlockChain(sec)
			} else {
				if child != nil {
					self.connectToBlockChain(child)
				}
			}
			break LOOP
		}
		// look up node in pool
		entry = self.get(hash)
		if entry != nil {
			// reached a known chain in the pool
			if entry.node == entry.section.bottom && n == 1 {
				// the first block hash received is an orphan in the pool, so rejoice and continue
				poolLogger.DebugDetailf("[%s] connecting child section", sectionName(entry.section))
				child = entry.section
				continue LOOP
			}
			poolLogger.DebugDetailf("[%s] reached blockpool chain", name(hash))
			parent = entry.section
			break LOOP
		}
		// if node for block hash does not exist, create it and index in the pool
		node := &poolNode{
			hash: hash,
			peer: peerId,
		}
		if size == 0 {
			sec = newSection()
		}
		nodes = append(nodes, node)
		size++
	} //for

	self.chainLock.Lock()

	poolLogger.DebugDetailf("added %v hashes sent by %s", n, peerId)

	if parent != nil && entry != nil && entry.node != parent.top {
		poolLogger.DebugDetailf("[%s] split section at fork", sectionName(parent))
		parent.controlC <- nil
		waiter := make(chan bool)
		parent.forkC <- waiter
		chain := parent.nodes
		parent.nodes = chain[entry.index:]
		parent.top = parent.nodes[0]
		orphan := newSection()
		self.link(orphan, parent.child)
		self.processSection(orphan, chain[0:entry.index])
		orphan.controlC <- nil
		close(waiter)
	}

	if size > 0 {
		self.processSection(sec, nodes)
		poolLogger.DebugDetailf("[%s]->[%s](%v)->[%s] new chain section", sectionName(parent), sectionName(sec), size, sectionName(child))
		self.link(parent, sec)
		self.link(sec, child)
	} else {
		poolLogger.DebugDetailf("[%s]->[%s] connecting known sections", sectionName(parent), sectionName(child))
		self.link(parent, child)
	}

	self.chainLock.Unlock()

	if parent != nil && bestPeer != nil {
		self.activateChain(parent, peer)
		poolLogger.Debugf("[%s] activate parent section [%s]", name(parent.top.hash), sectionName(parent))
	}

	if sec != nil {
		peer.addSection(sec.top.hash, sec)
		// request next section here once, only repeat if bottom block arrives,
		// otherwise no way to check if it arrived
		peer.requestBlockHashes(sec.bottom.hash)
		sec.controlC <- bestPeer
		poolLogger.Debugf("[%s] activate new section", sectionName(sec))
	}

	if headSection {
		var headSec *section
		switch {
		case sec != nil:
			headSec = sec
		case child != nil:
			headSec = child
		default:
			headSec = parent
		}
		peer.headSectionC <- headSec
	}
}

func name(hash []byte) (name string) {
	if hash == nil {
		name = ""
	} else {
		name = fmt.Sprintf("%x", hash[:4])
	}
	return
}

func sectionName(section *section) (name string) {
	if section == nil {
		name = ""
	} else {
		name = fmt.Sprintf("%x-%x", section.bottom.hash[:4], section.top.hash[:4])
	}
	return
}

// AddBlock is the entry point for the eth protocol when blockmsg is received upon requests
// It has a strict interpretation of the protocol in that if the block received has not been requested, it results in an error (which can be ignored)
// block is checked for PoW
// only the first PoW-valid block for a hash is considered legit
func (self *BlockPool) AddBlock(block *types.Block, peerId string) {
	hash := block.Hash()
	self.peersLock.Lock()
	peer := self.peer
	self.peersLock.Unlock()

	entry := self.get(hash)
	if bytes.Compare(hash, peer.currentBlockHash) == 0 {
		poolLogger.Debugf("add head block [%s] for peer %s", name(hash), peerId)
		peer.currentBlockC <- block
	} else {
		if entry == nil {
			poolLogger.Warnf("unrequested block [%s] by peer %s", name(hash), peerId)
			self.peerError(peerId, ErrUnrequestedBlock, "%x", hash)
		}
	}
	if entry == nil {
		return
	}

	node := entry.node
	node.lock.Lock()
	defer node.lock.Unlock()

	// check if block already present
	if node.block != nil {
		poolLogger.DebugDetailf("block [%s] already sent by %s", name(hash), node.blockBy)
		return
	}

	if self.hasBlock(hash) {
		poolLogger.DebugDetailf("block [%s] already known", name(hash))
	} else {

		// validate block for PoW
		if !self.verifyPoW(block) {
			poolLogger.Warnf("invalid pow on block [%s] by peer %s", name(hash), peerId)
			self.peerError(peerId, ErrInvalidPoW, "%x", hash)
			return
		}
	}
	poolLogger.Debugf("added block [%s] sent by peer %s", name(hash), peerId)
	node.block = block
	node.blockBy = peerId

}

func (self *BlockPool) connectToBlockChain(section *section) {
	select {
	case <-section.offC:
		self.addSectionToBlockChain(section)
	case <-section.blockChainC:
	default:
		close(section.blockChainC)
	}
}

func (self *BlockPool) addSectionToBlockChain(section *section) (rest int, err error) {

	var blocks types.Blocks
	var node *poolNode
	var keys []string
	rest = len(section.nodes)
	for rest > 0 {
		rest--
		node = section.nodes[rest]
		node.lock.RLock()
		block := node.block
		node.lock.RUnlock()
		if block == nil {
			break
		}
		keys = append(keys, string(node.hash))
		blocks = append(blocks, block)
	}

	self.lock.Lock()
	for _, key := range keys {
		delete(self.pool, key)
	}
	self.lock.Unlock()

	poolLogger.Infof("insert %v blocks into blockchain", len(blocks))
	err = self.insertChain(blocks)
	if err != nil {
		// TODO: not clear which peer we need to address
		// peerError should dispatch to peer if still connected and disconnect
		self.peerError(node.blockBy, ErrInvalidBlock, "%v", err)
		poolLogger.Warnf("invalid block %x", node.hash)
		poolLogger.Warnf("penalise peers %v (hash), %v (block)", node.peer, node.blockBy)
		// penalise peer in node.blockBy
		// self.disconnect()
	}
	return
}

func (self *BlockPool) activateChain(section *section, peer *peerInfo) {
	poolLogger.DebugDetailf("[%s] activate known chain for peer %s", sectionName(section), peer.id)
	i := 0
LOOP:
	for section != nil {
		// register this section with the peer and quit if registered
		poolLogger.DebugDetailf("[%s] register section with peer %s", sectionName(section), peer.id)
		if peer.addSection(section.top.hash, section) == section {
			return
		}
		poolLogger.DebugDetailf("[%s] activate section process", sectionName(section))
		select {
		case section.controlC <- peer:
		case <-section.offC:
		}
		i++
		section = self.getParent(section)
		select {
		case <-peer.quitC:
			break LOOP
		case <-self.quit:
			break LOOP
		default:
		}
	}
}

// main worker thread on each section in the poolchain
// - kills the section if there are blocks missing after an absolute time
// - kills the section if there are maxIdleRounds of idle rounds of block requests with no response
// - periodically polls the chain section for missing blocks which are then requested from peers
// - registers the process controller on the peer so that if the peer is promoted as best peer the second time (after a disconnect of a better one), all active processes are switched back on unless they expire and killed ()
// - when turned off (if peer disconnects and new peer connects with alternative chain), no blockrequests are made but absolute expiry timer is ticking
// - when turned back on it recursively calls itself on the root of the next chain section
// - when exits, signals to
func (self *BlockPool) processSection(sec *section, nodes []*poolNode) {

	for i, node := range nodes {
		entry := &poolEntry{node: node, section: sec, index: i}
		self.set(node.hash, entry)
	}

	sec.bottom = nodes[len(nodes)-1]
	sec.top = nodes[0]
	sec.nodes = nodes
	poolLogger.DebugDetailf("[%s] setup section process", sectionName(sec))

	self.wg.Add(1)
	go func() {

		// absolute time after which sub-chain is killed if not complete (some blocks are missing)
		suicideTimer := time.After(blocksTimeout * time.Second)

		var peer, newPeer *peerInfo

		var blocksRequestTimer, blockHashesRequestTimer <-chan time.Time
		var blocksRequestTime, blockHashesRequestTime bool
		var blocksRequests, blockHashesRequests int
		var blocksRequestsComplete, blockHashesRequestsComplete bool

		// node channels for the section
		var missingC, processC, offC chan *poolNode
		// container for missing block hashes
		var hashes [][]byte

		var i, missing, lastMissing, depth int
		var idle int
		var init, done, same, ready bool
		var insertChain bool
		var quitC chan bool

		var blockChainC = sec.blockChainC

		var parentHash []byte

	LOOP:
		for {

			if insertChain {
				insertChain = false
				rest, err := self.addSectionToBlockChain(sec)
				if err != nil {
					close(sec.suicideC)
					continue LOOP
				}
				if rest == 0 {
					blocksRequestsComplete = true
					child := self.getChild(sec)
					if child != nil {
						self.connectToBlockChain(child)
					}
				}
			}

			if blockHashesRequestsComplete && blocksRequestsComplete {
				// not waiting for hashes any more
				poolLogger.Debugf("[%s] section complete %v blocks retrieved (%v attempts), hash requests complete on root (%v attempts)", sectionName(sec), depth, blocksRequests, blockHashesRequests)
				break LOOP
			} // otherwise suicide if no hashes coming

			if done {
				// went through all blocks in section
				if missing == 0 {
					// no missing blocks
					poolLogger.DebugDetailf("[%s] got all blocks. process complete (%v total blocksRequests): missing %v/%v/%v", sectionName(sec), blocksRequests, missing, lastMissing, depth)
					blocksRequestsComplete = true
					blocksRequestTimer = nil
					blocksRequestTime = false
				} else {
					poolLogger.DebugDetailf("[%s] section checked: missing %v/%v/%v", sectionName(sec), missing, lastMissing, depth)
					// some missing blocks
					blocksRequests++
					if len(hashes) > 0 {
						// send block requests to peers
						self.requestBlocks(blocksRequests, hashes)
						hashes = nil
					}
					if missing == lastMissing {
						// idle round
						if same {
							// more than once
							idle++
							// too many idle rounds
							if idle >= blocksRequestMaxIdleRounds {
								poolLogger.DebugDetailf("[%s] block requests had %v idle rounds (%v total attempts): missing %v/%v/%v\ngiving up...", sectionName(sec), idle, blocksRequests, missing, lastMissing, depth)
								close(sec.suicideC)
							}
						} else {
							idle = 0
						}
						same = true
					} else {
						same = false
					}
				}
				lastMissing = missing
				ready = true
				done = false
				// save a new processC (blocks still missing)
				offC = missingC
				missingC = processC
				// put processC offline
				processC = nil
			}
			//

			if ready && blocksRequestTime && !blocksRequestsComplete {
				poolLogger.DebugDetailf("[%s] check if new blocks arrived (attempt %v): missing %v/%v/%v", sectionName(sec), blocksRequests, missing, lastMissing, depth)
				blocksRequestTimer = time.After(blocksRequestInterval * time.Millisecond)
				blocksRequestTime = false
				processC = offC
			}

			if blockHashesRequestTime {
				var parentSection = self.getParent(sec)
				if parentSection == nil {
					if parent := self.get(parentHash); parent != nil {
						parentSection = parent.section
						self.chainLock.Lock()
						self.link(parentSection, sec)
						self.chainLock.Unlock()
					} else {
						if self.hasBlock(parentHash) {
							insertChain = true
							blockHashesRequestTime = false
							blockHashesRequestTimer = nil
							blockHashesRequestsComplete = true
							continue LOOP
						}
					}
				}
				if parentSection != nil {
					// if not root of chain, switch off
					poolLogger.DebugDetailf("[%s] parent found, hash requests deactivated (after %v total attempts)\n", sectionName(sec), blockHashesRequests)
					blockHashesRequestTimer = nil
					blockHashesRequestsComplete = true
				} else {
					blockHashesRequests++
					poolLogger.Debugf("[%s] hash request on root (%v total attempts)\n", sectionName(sec), blockHashesRequests)
					peer.requestBlockHashes(sec.bottom.hash)
					blockHashesRequestTimer = time.After(blockHashesRequestInterval * time.Millisecond)
				}
				blockHashesRequestTime = false
			}

			select {
			case <-self.quit:
				break LOOP

			case <-quitC:
				// peer quit or demoted, put section in idle mode
				quitC = nil
				go func() {
					sec.controlC <- nil
				}()

			case <-self.purgeC:
				suicideTimer = time.After(0)

			case <-suicideTimer:
				close(sec.suicideC)
				poolLogger.Debugf("[%s] timeout. (%v total attempts): missing %v/%v/%v", sectionName(sec), blocksRequests, missing, lastMissing, depth)

			case <-sec.suicideC:
				poolLogger.Debugf("[%s] suicide", sectionName(sec))

				// first delink from child and parent under chainlock
				self.chainLock.Lock()
				self.link(nil, sec)
				self.link(sec, nil)
				self.chainLock.Unlock()
				// delete node entries from pool index under pool lock
				self.lock.Lock()
				for _, node := range sec.nodes {
					delete(self.pool, string(node.hash))
				}
				self.lock.Unlock()

				break LOOP

			case <-blocksRequestTimer:
				poolLogger.DebugDetailf("[%s] block request time", sectionName(sec))
				blocksRequestTime = true

			case <-blockHashesRequestTimer:
				poolLogger.DebugDetailf("[%s] hash request time", sectionName(sec))
				blockHashesRequestTime = true

			case newPeer = <-sec.controlC:

				// active -> idle
				if peer != nil && newPeer == nil {
					self.procWg.Done()
					if init {
						poolLogger.Debugf("[%s] idle mode (%v total attempts): missing %v/%v/%v", sectionName(sec), blocksRequests, missing, lastMissing, depth)
					}
					blocksRequestTime = false
					blocksRequestTimer = nil
					blockHashesRequestTime = false
					blockHashesRequestTimer = nil
					if processC != nil {
						offC = processC
						processC = nil
					}
				}

				// idle -> active
				if peer == nil && newPeer != nil {
					self.procWg.Add(1)

					poolLogger.Debugf("[%s] active mode", sectionName(sec))
					if !blocksRequestsComplete {
						blocksRequestTime = true
					}
					if !blockHashesRequestsComplete && parentHash != nil {
						blockHashesRequestTime = true
					}
					if !init {
						processC = make(chan *poolNode, blockHashesBatchSize)
						missingC = make(chan *poolNode, blockHashesBatchSize)
						i = 0
						missing = 0
						self.wg.Add(1)
						self.procWg.Add(1)
						depth = len(sec.nodes)
						lastMissing = depth
						// if not run at least once fully, launch iterator
						go func() {
							var node *poolNode
						IT:
							for _, node = range sec.nodes {
								select {
								case processC <- node:
								case <-self.quit:
									break IT
								}
							}
							close(processC)
							self.wg.Done()
							self.procWg.Done()
						}()
					} else {
						poolLogger.Debugf("[%s] restore earlier state", sectionName(sec))
						processC = offC
					}
				}
				// reset quitC to current best peer
				if newPeer != nil {
					quitC = newPeer.quitC
				}
				peer = newPeer

			case waiter := <-sec.forkC:
				// this case just blocks the process until section is split at the fork
				<-waiter
				init = false
				done = false
				ready = false

			case node, ok := <-processC:
				if !ok && !init {
					// channel closed, first iteration finished
					init = true
					done = true
					processC = make(chan *poolNode, missing)
					poolLogger.DebugDetailf("[%s] section initalised: missing %v/%v/%v", sectionName(sec), missing, lastMissing, depth)
					continue LOOP
				}
				if ready {
					i = 0
					missing = 0
					ready = false
				}
				i++
				// if node has no block
				node.lock.RLock()
				block := node.block
				node.lock.RUnlock()
				if block == nil {
					missing++
					hashes = append(hashes, node.hash)
					if len(hashes) == blockBatchSize {
						poolLogger.Debugf("[%s] request %v missing blocks", sectionName(sec), len(hashes))
						self.requestBlocks(blocksRequests, hashes)
						hashes = nil
					}
					missingC <- node
				} else {
					if i == lastMissing {
						if blockChainC == nil {
							insertChain = true
						} else {
							if parentHash == nil {
								parentHash = block.ParentHash()
								poolLogger.Debugf("[%s] found root block [%s]", sectionName(sec), name(parentHash))
								blockHashesRequestTime = true
							}
						}
					}
				}
				if i == lastMissing && init {
					done = true
				}

			case <-blockChainC:
				// closed blockChain channel indicates that the blockpool is reached
				// connected to the blockchain, insert the longest chain of blocks
				poolLogger.Debugf("[%s] reached blockchain", sectionName(sec))
				blockChainC = nil
				// switch off hash requests in case they were on
				blockHashesRequestTime = false
				blockHashesRequestTimer = nil
				blockHashesRequestsComplete = true
				// section root has block
				if len(sec.nodes) > 0 && sec.nodes[len(sec.nodes)-1].block != nil {
					insertChain = true
				}
				continue LOOP

			} // select
		} // for

		close(sec.offC)

		self.wg.Done()
		if peer != nil {
			self.procWg.Done()
		}
	}()
	return
}

func (self *BlockPool) peerError(peerId string, code int, format string, params ...interface{}) {
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	peer, ok := self.peers[peerId]
	if ok {
		peer.peerError(code, format, params...)
	}
}

func (self *BlockPool) requestBlocks(attempts int, hashes [][]byte) {
	self.wg.Add(1)
	self.procWg.Add(1)
	go func() {
		// distribute block request among known peers
		self.peersLock.Lock()
		defer self.peersLock.Unlock()
		peerCount := len(self.peers)
		// on first attempt use the best peer
		if attempts == 0 {
			poolLogger.Debugf("request %v missing blocks from best peer %s", len(hashes), self.peer.id)
			self.peer.requestBlocks(hashes)
			return
		}
		repetitions := int(math.Min(float64(peerCount), float64(blocksRequestRepetition)))
		i := 0
		indexes := rand.Perm(peerCount)[0:repetitions]
		sort.Ints(indexes)
		poolLogger.Debugf("request %v missing blocks from %v/%v peers: chosen %v", len(hashes), repetitions, peerCount, indexes)
		for _, peer := range self.peers {
			if i == indexes[0] {
				poolLogger.Debugf("request %v missing blocks [%x/%x] from peer %s", len(hashes), hashes[0][:4], hashes[len(hashes)-1][:4], peer.id)
				peer.requestBlocks(hashes)
				indexes = indexes[1:]
				if len(indexes) == 0 {
					break
				}
			}
			i++
		}
		self.wg.Done()
		self.procWg.Done()
	}()
}

func (self *BlockPool) getPeer(peerId string) (*peerInfo, bool) {
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	if self.peer != nil && self.peer.id == peerId {
		return self.peer, true
	}
	info, ok := self.peers[peerId]
	if !ok {
		return nil, false
	}
	return info, false
}

func (self *peerInfo) addSection(hash []byte, section *section) (found *section) {
	self.lock.Lock()
	defer self.lock.Unlock()
	key := string(hash)
	found = self.sections[key]
	poolLogger.DebugDetailf("[%s] section process stored for %s", sectionName(section), self.id)
	self.sections[key] = section
	return
}

func (self *BlockPool) switchPeer(oldPeer, newPeer *peerInfo) {
	if newPeer != nil {
		newPeer.quitC = make(chan bool)
		poolLogger.DebugDetailf("[%s] activate section processes", newPeer.id)
		var addSections []*section
		for hash, section := range newPeer.sections {
			// split sections get reorganised here
			if string(section.top.hash) != hash {
				addSections = append(addSections, section)
				if entry := self.get([]byte(hash)); entry != nil {
					addSections = append(addSections, entry.section)
				}
			}
		}
		for _, section := range addSections {
			newPeer.sections[string(section.top.hash)] = section
		}
		for hash, section := range newPeer.sections {
			// this will block if section process is waiting for peer lock
			select {
			case <-section.offC:
				poolLogger.DebugDetailf("[%s][%x] section process complete - remove", newPeer.id, hash[:4])
				delete(newPeer.sections, hash)
			case section.controlC <- newPeer:
				poolLogger.DebugDetailf("[%s][%x] activates section [%s]", newPeer.id, hash[:4], sectionName(section))
			}
		}
		newPeer.lock.Lock()
		headSection := newPeer.headSection
		currentBlockHash := newPeer.currentBlockHash
		newPeer.lock.Unlock()
		if headSection == nil {
			poolLogger.DebugDetailf("[%s] head section for [%s] not created, requesting info", newPeer.id, name(currentBlockHash))
			self.requestHeadSection(newPeer)
		} else {
			if entry := self.get(currentBlockHash); entry != nil {
				headSection = entry.section
			}
			poolLogger.DebugDetailf("[%s] activate chain at head section [%s] for current head [%s]", newPeer.id, sectionName(headSection), name(currentBlockHash))
			self.activateChain(headSection, newPeer)
		}
	}
	if oldPeer != nil {
		poolLogger.DebugDetailf("[%s] quit section processes", oldPeer.id)
		close(oldPeer.quitC)
	}
}

func (self *BlockPool) getParent(sec *section) *section {
	self.chainLock.RLock()
	defer self.chainLock.RUnlock()
	return sec.parent
}

func (self *BlockPool) getChild(sec *section) *section {
	self.chainLock.RLock()
	defer self.chainLock.RUnlock()
	return sec.child
}

func newSection() (sec *section) {
	sec = &section{
		controlC:    make(chan *peerInfo),
		suicideC:    make(chan bool),
		blockChainC: make(chan bool),
		offC:        make(chan bool),
		forkC:       make(chan chan bool),
	}
	return
}

// link should only be called under chainLock
func (self *BlockPool) link(parent *section, child *section) {
	if parent != nil {
		exChild := parent.child
		parent.child = child
		if exChild != nil && exChild != child {
			poolLogger.Debugf("[%s] chain fork [%s] -> [%s]", sectionName(parent), sectionName(exChild), sectionName(child))
			exChild.parent = nil
		}
	}
	if child != nil {
		exParent := child.parent
		if exParent != nil && exParent != parent {
			poolLogger.Debugf("[%s] chain reverse fork [%s] -> [%s]", sectionName(child), sectionName(exParent), sectionName(parent))
			exParent.child = nil
		}
		child.parent = parent
	}
}

func (self *BlockPool) get(hash []byte) (node *poolEntry) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	return self.pool[string(hash)]
}

func (self *BlockPool) set(hash []byte, node *poolEntry) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.pool[string(hash)] = node
}
