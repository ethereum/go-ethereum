package eth

import (
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
	cacheTimeout               = 3 // minutes
	blockTimeout               = 5 // minutes
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

	td           *big.Int
	currentBlock []byte
	id           string

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
	self.wg.Wait()

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
func (self *BlockPool) AddPeer(td *big.Int, currentBlock []byte, peerId string, requestBlockHashes func([]byte) error, requestBlocks func([][]byte) error, peerError func(int, string, ...interface{})) bool {

	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	peer, ok := self.peers[peerId]
	if ok {
		poolLogger.Debugf("Update peer %v with td %v and current block %x", peerId, td, currentBlock[:4])
		peer.td = td
		peer.currentBlock = currentBlock
	} else {
		peer = &peerInfo{
			td:                 td,
			currentBlock:       currentBlock,
			id:                 peerId, //peer.Identity().Pubkey()
			requestBlockHashes: requestBlockHashes,
			requestBlocks:      requestBlocks,
			peerError:          peerError,
			sections:           make(map[string]*section),
		}
		self.peers[peerId] = peer
		poolLogger.Debugf("add new peer %v with td %v and current block %x", peerId, td, currentBlock[:4])
	}
	// check peer current head
	if self.hasBlock(currentBlock) {
		// peer not ahead
		return false
	}

	if self.peer == peer {
		// new block update
		// peer is already active best peer, request hashes
		poolLogger.Debugf("[%s] already the best peer. request hashes from %s", peerId, name(currentBlock))
		peer.requestBlockHashes(currentBlock)
		return true
	}

	currentTD := ethutil.Big0
	if self.peer != nil {
		currentTD = self.peer.td
	}
	if td.Cmp(currentTD) > 0 {
		poolLogger.Debugf("peer %v promoted best peer", peerId)
		self.switchPeer(self.peer, peer)
		self.peer = peer
		return true
	}
	return false
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
		self.peer = newPeer
		self.switchPeer(peer, newPeer)
		if newPeer != nil {
			poolLogger.Debugf("peer %v with td %v promoted to best peer", newPeer.id, newPeer.td)
		} else {
			poolLogger.Warnln("no peers")
		}
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
	poolLogger.Debugf("adding hashes for best peer %s", peerId)

	var size, n int
	var hash []byte
	var ok bool
	var section, child, parent *section
	var entry *poolEntry
	var nodes []*poolNode

LOOP:
	// iterate using next (rlp stream lazy decoder) feeding hashesC
	for hash, ok = next(); ok; hash, ok = next() {
		n++
		select {
		case <-self.quit:
			return
		case <-peer.quitC:
			// if the peer is demoted, no more hashes taken
			peer = nil
			break LOOP
		default:
		}
		if self.hasBlock(hash) {
			// check if known block connecting the downloaded chain to our blockchain
			poolLogger.DebugDetailf("[%s] known block", name(hash))
			// mark child as absolute pool root with parent known to blockchain
			if section != nil {
				self.connectToBlockChain(section)
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
			section = newSection()
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
		self.processSection(section, nodes)
		poolLogger.DebugDetailf("[%s]->[%s](%v)->[%s] new chain section", sectionName(parent), sectionName(section), size, sectionName(child))
		self.link(parent, section)
		self.link(section, child)
	} else {
		poolLogger.DebugDetailf("[%s]->[%s] connecting known sections", sectionName(parent), sectionName(child))
		self.link(parent, child)
	}

	self.chainLock.Unlock()

	if parent != nil && peer != nil {
		self.activateChain(parent, peer)
		poolLogger.Debugf("[%s] activate parent section [%s]", name(parent.top.hash), sectionName(parent))
	}

	if section != nil {
		peer.addSection(section.top.hash, section)
		section.controlC <- peer
		poolLogger.Debugf("[%s] activate new section", sectionName(section))
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
	if self.hasBlock(hash) {
		poolLogger.DebugDetailf("block [%s] already known", name(hash))
		return
	}
	entry := self.get(hash)
	if entry == nil {
		poolLogger.Warnf("unrequested block [%x] by peer %s", hash, peerId)
		self.peerError(peerId, ErrUnrequestedBlock, "%x", hash)
		return
	}

	node := entry.node
	node.lock.Lock()
	defer node.lock.Unlock()

	// check if block already present
	if node.block != nil {
		poolLogger.DebugDetailf("block [%x] already sent by %s", name(hash), node.blockBy)
		return
	}

	// validate block for PoW
	if !self.verifyPoW(block) {
		poolLogger.Warnf("invalid pow on block [%x] by peer %s", hash, peerId)
		self.peerError(peerId, ErrInvalidPoW, "%x", hash)
		return
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
func (self *BlockPool) processSection(section *section, nodes []*poolNode) {

	for i, node := range nodes {
		entry := &poolEntry{node: node, section: section, index: i}
		self.set(node.hash, entry)
	}

	section.bottom = nodes[len(nodes)-1]
	section.top = nodes[0]
	section.nodes = nodes
	poolLogger.DebugDetailf("[%s] setup section process", sectionName(section))

	self.wg.Add(1)
	go func() {

		// absolute time after which sub-chain is killed if not complete (some blocks are missing)
		suicideTimer := time.After(blockTimeout * time.Minute)

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

		var blockChainC = section.blockChainC

	LOOP:
		for {

			if insertChain {
				insertChain = false
				rest, err := self.addSectionToBlockChain(section)
				if err != nil {
					close(section.suicideC)
					continue LOOP
				}
				if rest == 0 {
					blocksRequestsComplete = true
					child := self.getChild(section)
					if child != nil {
						self.connectToBlockChain(child)
					}
				}
			}

			if blockHashesRequestsComplete && blocksRequestsComplete {
				// not waiting for hashes any more
				poolLogger.Debugf("[%s] section complete %v blocks retrieved (%v attempts), hash requests complete on root (%v attempts)", sectionName(section), depth, blocksRequests, blockHashesRequests)
				break LOOP
			} // otherwise suicide if no hashes coming

			if done {
				// went through all blocks in section
				if missing == 0 {
					// no missing blocks
					poolLogger.DebugDetailf("[%s] got all blocks. process complete (%v total blocksRequests): missing %v/%v/%v", sectionName(section), blocksRequests, missing, lastMissing, depth)
					blocksRequestsComplete = true
					blocksRequestTimer = nil
					blocksRequestTime = false
				} else {
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
								poolLogger.DebugDetailf("[%s] block requests had %v idle rounds (%v total attempts): missing %v/%v/%v\ngiving up...", sectionName(section), idle, blocksRequests, missing, lastMissing, depth)
								close(section.suicideC)
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
				poolLogger.DebugDetailf("[%s] check if new blocks arrived (attempt %v): missing %v/%v/%v", sectionName(section), blocksRequests, missing, lastMissing, depth)
				blocksRequestTimer = time.After(blocksRequestInterval * time.Millisecond)
				blocksRequestTime = false
				processC = offC
			}

			if blockHashesRequestTime {
				if self.getParent(section) != nil {
					// if not root of chain, switch off
					poolLogger.DebugDetailf("[%s] parent found, hash requests deactivated (after %v total attempts)\n", sectionName(section), blockHashesRequests)
					blockHashesRequestTimer = nil
					blockHashesRequestsComplete = true
				} else {
					blockHashesRequests++
					poolLogger.Debugf("[%s] hash request on root (%v total attempts)\n", sectionName(section), blockHashesRequests)
					peer.requestBlockHashes(section.bottom.hash)
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
					section.controlC <- nil
				}()

			case <-self.purgeC:
				suicideTimer = time.After(0)

			case <-suicideTimer:
				close(section.suicideC)
				poolLogger.Debugf("[%s] timeout. (%v total attempts): missing %v/%v/%v", sectionName(section), blocksRequests, missing, lastMissing, depth)

			case <-section.suicideC:
				poolLogger.Debugf("[%s] suicide", sectionName(section))

				// first delink from child and parent under chainlock
				self.chainLock.Lock()
				self.link(nil, section)
				self.link(section, nil)
				self.chainLock.Unlock()
				// delete node entries from pool index under pool lock
				self.lock.Lock()
				for _, node := range section.nodes {
					delete(self.pool, string(node.hash))
				}
				self.lock.Unlock()

				break LOOP

			case <-blocksRequestTimer:
				poolLogger.DebugDetailf("[%s] block request time", sectionName(section))
				blocksRequestTime = true

			case <-blockHashesRequestTimer:
				poolLogger.DebugDetailf("[%s] hash request time", sectionName(section))
				blockHashesRequestTime = true

			case newPeer = <-section.controlC:

				// active -> idle
				if peer != nil && newPeer == nil {
					self.procWg.Done()
					if init {
						poolLogger.Debugf("[%s] idle mode (%v total attempts): missing %v/%v/%v", sectionName(section), blocksRequests, missing, lastMissing, depth)
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

					poolLogger.Debugf("[%s] active mode", sectionName(section))
					if !blocksRequestsComplete {
						blocksRequestTime = true
					}
					if !blockHashesRequestsComplete {
						blockHashesRequestTime = true
					}
					if !init {
						processC = make(chan *poolNode, blockHashesBatchSize)
						missingC = make(chan *poolNode, blockHashesBatchSize)
						i = 0
						missing = 0
						self.wg.Add(1)
						self.procWg.Add(1)
						depth = len(section.nodes)
						lastMissing = depth
						// if not run at least once fully, launch iterator
						go func() {
							var node *poolNode
						IT:
							for _, node = range section.nodes {
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
						poolLogger.Debugf("[%s] restore earlier state", sectionName(section))
						processC = offC
					}
				}
				// reset quitC to current best peer
				if newPeer != nil {
					quitC = newPeer.quitC
				}
				peer = newPeer

			case waiter := <-section.forkC:
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
					poolLogger.DebugDetailf("[%s] section initalised: missing %v/%v/%v", sectionName(section), missing, lastMissing, depth)
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
						poolLogger.Debugf("[%s] request %v missing blocks", sectionName(section), len(hashes))
						self.requestBlocks(blocksRequests, hashes)
						hashes = nil
					}
					missingC <- node
				} else {
					if blockChainC == nil && i == lastMissing {
						insertChain = true
					}
				}
				poolLogger.Debugf("[%s] %v/%v/%v/%v", sectionName(section), i, missing, lastMissing, depth)
				if i == lastMissing && init {
					done = true
				}

			case <-blockChainC:
				// closed blockChain channel indicates that the blockpool is reached
				// connected to the blockchain, insert the longest chain of blocks
				poolLogger.Debugf("[%s] reached blockchain", sectionName(section))
				blockChainC = nil
				// switch off hash requests in case they were on
				blockHashesRequestTime = false
				blockHashesRequestTimer = nil
				blockHashesRequestsComplete = true
				// section root has block
				if len(section.nodes) > 0 && section.nodes[len(section.nodes)-1].block != nil {
					insertChain = true
				}
				continue LOOP

			} // select
		} // for
		poolLogger.Debugf("[%s] section complete: %v block hashes requests - %v block requests - missing %v/%v/%v", sectionName(section), blockHashesRequests, blocksRequests, missing, lastMissing, depth)

		close(section.offC)

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
				poolLogger.Debugf("request %v missing blocks from peer %s", len(hashes), peer.id)
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
	poolLogger.DebugDetailf("[%s] section process %s registered", sectionName(section), self.id)
	self.sections[key] = section
	return
}

func (self *BlockPool) switchPeer(oldPeer, newPeer *peerInfo) {
	if newPeer != nil {
		entry := self.get(newPeer.currentBlock)
		if entry == nil {
			poolLogger.Debugf("[%s] head block [%s] not found, requesting hashes", newPeer.id, name(newPeer.currentBlock))
			newPeer.requestBlockHashes(newPeer.currentBlock)
		} else {
			poolLogger.Debugf("[%s] head block [%s] found, activate chain at section [%s]", newPeer.id, name(newPeer.currentBlock), sectionName(entry.section))
			self.activateChain(entry.section, newPeer)
		}
		poolLogger.DebugDetailf("[%s] activate section processes", newPeer.id)
		for hash, section := range newPeer.sections {
			// this will block if section process is waiting for peer lock
			select {
			case <-section.offC:
				poolLogger.DebugDetailf("[%s][%x] section process complete - remove", newPeer.id, hash[:4])
				delete(newPeer.sections, hash)
			case section.controlC <- newPeer:
				poolLogger.DebugDetailf("[%s][%x] registered peer with section", newPeer.id, hash[:4])
			}
		}
		newPeer.quitC = make(chan bool)
	}
	if oldPeer != nil {
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
