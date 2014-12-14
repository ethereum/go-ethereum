package eth

import (
	"math"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	ethlogger "github.com/ethereum/go-ethereum/logger"
)

var poolLogger = ethlogger.NewLogger("Blockpool")

const (
	blockHashesBatchSize       = 256
	blockBatchSize             = 64
	blocksRequestInterval      = 10 // seconds
	blocksRequestRepetition    = 1
	blockHashesRequestInterval = 10 // seconds
	blocksRequestMaxIdleRounds = 10
	cacheTimeout               = 3 // minutes
	blockTimeout               = 5 // minutes
)

type poolNode struct {
	lock        sync.RWMutex
	hash        []byte
	block       *types.Block
	child       *poolNode
	parent      *poolNode
	section     *section
	knownParent bool
	peer        string
	source      string
	complete    bool
}

type BlockPool struct {
	lock sync.RWMutex
	pool map[string]*poolNode

	peersLock sync.RWMutex
	peers     map[string]*peerInfo
	peer      *peerInfo

	quit    chan bool
	wg      sync.WaitGroup
	running bool

	// the minimal interface with blockchain
	hasBlock    func(hash []byte) bool
	insertChain func(types.Blocks) error
	verifyPoW   func(*types.Block) bool
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
	roots    []*poolNode
	quitC    chan bool
}

func NewBlockPool(hasBlock func(hash []byte) bool, insertChain func(types.Blocks) error, verifyPoW func(*types.Block) bool,
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
	self.pool = make(map[string]*poolNode)
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

	poolLogger.Infoln("Stopping")

	close(self.quit)
	self.lock.Lock()
	self.peersLock.Lock()
	self.peers = nil
	self.pool = nil
	self.peer = nil
	self.wg.Wait()
	self.lock.Unlock()
	self.peersLock.Unlock()
	poolLogger.Infoln("Stopped")

}

// AddPeer is called by the eth protocol instance running on the peer after
// the status message has been received with total difficulty and current block hash
// AddPeer can only be used once, RemovePeer needs to be called when the peer disconnects
func (self *BlockPool) AddPeer(td *big.Int, currentBlock []byte, peerId string, requestBlockHashes func([]byte) error, requestBlocks func([][]byte) error, peerError func(int, string, ...interface{})) bool {
	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	if self.peers[peerId] != nil {
		panic("peer already added")
	}
	peer := &peerInfo{
		td:                 td,
		currentBlock:       currentBlock,
		id:                 peerId, //peer.Identity().Pubkey()
		requestBlockHashes: requestBlockHashes,
		requestBlocks:      requestBlocks,
		peerError:          peerError,
	}
	self.peers[peerId] = peer
	poolLogger.Debugf("add new peer %v with td %v", peerId, td)
	currentTD := ethutil.Big0
	if self.peer != nil {
		currentTD = self.peer.td
	}
	if td.Cmp(currentTD) > 0 {
		self.peer.stop(peer)
		peer.start(self.peer)
		poolLogger.Debugf("peer %v promoted to best peer", peerId)
		self.peer = peer
		return true
	}
	return false
}

// RemovePeer is called by the eth protocol when the peer disconnects
func (self *BlockPool) RemovePeer(peerId string) {
	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	peer := self.peers[peerId]
	if peer == nil {
		return
	}
	self.peers[peerId] = nil
	poolLogger.Debugf("remove peer %v", peerId[0:4])

	// if current best peer is removed, need find a better one
	if self.peer != nil && peerId == self.peer.id {
		var newPeer *peerInfo
		max := ethutil.Big0
		// peer with the highest self-acclaimed TD is chosen
		for _, info := range self.peers {
			if info.td.Cmp(max) > 0 {
				max = info.td
				newPeer = info
			}
		}
		self.peer.stop(peer)
		peer.start(self.peer)
		if newPeer != nil {
			poolLogger.Debugf("peer %v with td %v promoted to best peer", newPeer.id[0:4], newPeer.td)
		} else {
			poolLogger.Warnln("no peers left")
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

	// check if this peer is the best
	peer, best := self.getPeer(peerId)
	if !best {
		return
	}
	// peer is still the best

	var child *poolNode
	var depth int

	// iterate using next (rlp stream lazy decoder) feeding hashesC
	self.wg.Add(1)
	go func() {
		for {
			select {
			case <-self.quit:
				return
			case <-peer.quitC:
				// if the peer is demoted, no more hashes taken
				break
			default:
				hash, ok := next()
				if !ok {
					// message consumed chain skeleton built
					break
				}
				// check if known block connecting the downloaded chain to our blockchain
				if self.hasBlock(hash) {
					poolLogger.Infof("known block (%x...)\n", hash[0:4])
					if child != nil {
						child.Lock()
						// mark child as absolute pool root with parent known to blockchain
						child.knownParent = true
						child.Unlock()
					}
					break
				}
				//
				var parent *poolNode
				// look up node in pool
				parent = self.get(hash)
				if parent != nil {
					// reached a known chain in the pool
					// request blocks on the newly added part of the chain
					if child != nil {
						self.link(parent, child)

						// activate the current chain
						self.activateChain(parent, peer, true)
						poolLogger.Debugf("potential chain of %v blocks added, reached blockpool, activate chain", depth)
						break
					}
					// if this is the first hash, we expect to find it
					parent.RLock()
					grandParent := parent.parent
					parent.RUnlock()
					if grandParent != nil {
						// activate the current chain
						self.activateChain(parent, peer, true)
						poolLogger.Debugf("block hash found, activate chain")
						break
					}
					// the first node is the root of a chain in the pool, rejoice and continue
				}
				// if node does not exist, create it and index in the pool
				section := &section{}
				if child == nil {
					section.top = parent
				}
				parent = &poolNode{
					hash:    hash,
					child:   child,
					section: section,
					peer:    peerId,
				}
				self.set(hash, parent)
				poolLogger.Debugf("create potential block for %x...", hash[0:4])

				depth++
				child = parent
			}
		}
		if child != nil {
			poolLogger.Debugf("chain of %v hashes added", depth)
			// start a processSection on the last node, but switch off asking
			// hashes and blocks until next peer confirms this chain
			section := self.processSection(child)
			peer.addSection(child.hash, section)
			section.start()
		}
	}()
}

// AddBlock is the entry point for the eth protocol when blockmsg is received upon requests
// It has a strict interpretation of the protocol in that if the block received has not been requested, it results in an error (which can be ignored)
// block is checked for PoW
// only the first PoW-valid block for a hash is considered legit
func (self *BlockPool) AddBlock(block *types.Block, peerId string) {
	hash := block.Hash()
	node := self.get(hash)
	node.RLock()
	b := node.block
	node.RUnlock()
	if b != nil {
		return
	}
	if node == nil && !self.hasBlock(hash) {
		self.peerError(peerId, ErrUnrequestedBlock, "%x", hash)
		return
	}
	// validate block for PoW
	if !self.verifyPoW(block) {
		self.peerError(peerId, ErrInvalidPoW, "%x", hash)
	}
	node.Lock()
	node.block = block
	node.source = peerId
	node.Unlock()
}

// iterates down a known poolchain and activates fetching processes
// on each chain section for the peer
// stops if the peer is demoted
// registers last section root as root for the peer (in case peer is promoted a second time, to remember)
func (self *BlockPool) activateChain(node *poolNode, peer *peerInfo, on bool) {
	self.wg.Add(1)
	go func() {
		for {
			node.sectionRLock()
			bottom := node.section.bottom
			if bottom == nil { // the chain section is being created or killed
				break
			}
			// register this section with the peer
			if peer != nil {
				peer.addSection(bottom.hash, bottom.section)
				if on {
					bottom.section.start()
				} else {
					bottom.section.start()
				}
			}
			if bottom.parent == nil {
				node = bottom
				break
			}
			// if peer demoted stop activation
			select {
			case <-peer.quitC:
				break
			default:
			}

			node = bottom.parent
			bottom.sectionRUnlock()
		}
		// remember root for this peer
		peer.addRoot(node)
		self.wg.Done()
	}()
}

// main worker thread on each section in the poolchain
// - kills the section if there are blocks missing after an absolute time
// - kills the section if there are maxIdleRounds of idle rounds of block requests with no response
// - periodically polls the chain section for missing blocks which are then requested from peers
// - registers the process controller on the peer so that if the peer is promoted as best peer the second time (after a disconnect of a better one), all active processes are switched back on unless they expire and killed ()
// - when turned off (if peer disconnects and new peer connects with alternative chain), no blockrequests are made but absolute expiry timer is ticking
// - when turned back on it recursively calls itself on the root of the next chain section
// - when exits, signals to
func (self *BlockPool) processSection(node *poolNode) *section {
	// absolute time after which sub-chain is killed if not complete (some blocks are missing)
	suicideTimer := time.After(blockTimeout * time.Minute)
	var blocksRequestTimer, blockHashesRequestTimer <-chan time.Time
	var nodeC, missingC, processC chan *poolNode
	controlC := make(chan bool)
	resetC := make(chan bool)
	var hashes [][]byte
	var i, total, missing, lastMissing, depth int
	var blockHashesRequests, blocksRequests int
	var idle int
	var init, alarm, done, same, running, once bool
	orignode := node
	hash := node.hash

	node.sectionLock()
	defer node.sectionUnlock()
	section := &section{controlC: controlC, resetC: resetC}
	node.section = section

	go func() {
		self.wg.Add(1)
		for {
			node.sectionRLock()
			controlC = node.section.controlC
			node.sectionRUnlock()

			if init {
				// missing blocks read from nodeC
				// initialized section
				if depth == 0 {
					break
				}
				// enable select case to read missing block when ready
				processC = missingC
				missingC = make(chan *poolNode, lastMissing)
				nodeC = nil
				// only do once
				init = false
			} else {
				if !once {
					missingC = nil
					processC = nil
					i = 0
					total = 0
					lastMissing = 0
				}
			}

			// went through all blocks in section
			if i != 0 && i == lastMissing {
				if len(hashes) > 0 {
					// send block requests to peers
					self.requestBlocks(blocksRequests, hashes)
				}
				blocksRequests++
				poolLogger.Debugf("[%x] block request attempt %v: missing %v/%v/%v", hash[0:4], blocksRequests, missing, total, depth)
				if missing == lastMissing {
					// idle round
					if same {
						// more than once
						idle++
						// too many idle rounds
						if idle > blocksRequestMaxIdleRounds {
							poolLogger.Debugf("[%x] block requests had %v idle rounds (%v total attempts): missing %v/%v/%v\ngiving up...", hash[0:4], idle, blocksRequests, missing, total, depth)
							self.killChain(node, nil)
							break
						}
					} else {
						idle = 0
					}
					same = true
				} else {
					if missing == 0 {
						// no missing nodes
						poolLogger.Debugf("block request process complete on section %x... (%v total blocksRequests): missing %v/%v/%v", hash[0:4], blockHashesRequests, blocksRequests, missing, total, depth)
						node.Lock()
						orignode.complete = true
						node.Unlock()
						blocksRequestTimer = nil
						if blockHashesRequestTimer == nil {
							// not waiting for hashes any more
							poolLogger.Debugf("hash request on root %x... successful (%v total attempts)\nquitting...", hash[0:4], blockHashesRequests)
							break
						} // otherwise suicide if no hashes coming
					}
					same = false
				}
				lastMissing = missing
				i = 0
				missing = 0
				// ready for next round
				done = true
			}
			if done && alarm {
				poolLogger.Debugf("start checking if new blocks arrived (attempt %v): missing %v/%v/%v", blocksRequests, missing, total, depth)
				blocksRequestTimer = time.After(blocksRequestInterval * time.Second)
				alarm = false
				done = false
				// processC supposed to be empty and never closed so just swap,  no need to allocate
				tempC := processC
				processC = missingC
				missingC = tempC
			}
			select {
			case <-self.quit:
				break
			case <-suicideTimer:
				self.killChain(node, nil)
				poolLogger.Warnf("[%x] timeout. (%v total attempts): missing %v/%v/%v", hash[0:4], blocksRequests, missing, total, depth)
				break
			case <-blocksRequestTimer:
				alarm = true
			case <-blockHashesRequestTimer:
				orignode.RLock()
				parent := orignode.parent
				orignode.RUnlock()
				if parent != nil {
					// if not root of chain, switch off
					poolLogger.Debugf("[%x] parent found, hash requests deactivated (after %v total attempts)\n", hash[0:4], blockHashesRequests)
					blockHashesRequestTimer = nil
				} else {
					blockHashesRequests++
					poolLogger.Debugf("[%x] hash request on root (%v total attempts)\n", hash[0:4], blockHashesRequests)
					self.requestBlockHashes(parent.hash)
					blockHashesRequestTimer = time.After(blockHashesRequestInterval * time.Second)
				}
			case r, ok := <-controlC:
				if !ok {
					break
				}
				if running && !r {
					poolLogger.Debugf("process on section %x... (%v total attempts): missing %v/%v/%v", hash[0:4], blocksRequests, missing, total, depth)

					alarm = false
					blocksRequestTimer = nil
					blockHashesRequestTimer = nil
					processC = nil
				}
				if !running && r {
					poolLogger.Debugf("[%x] on", hash[0:4])

					orignode.RLock()
					parent := orignode.parent
					complete := orignode.complete
					knownParent := orignode.knownParent
					orignode.RUnlock()
					if !complete {
						poolLogger.Debugf("[%x] activate block requests", hash[0:4])
						blocksRequestTimer = time.After(0)
					}
					if parent == nil && !knownParent {
						// if no parent but not connected to blockchain
						poolLogger.Debugf("[%x] activate block hashes requests", hash[0:4])
						blockHashesRequestTimer = time.After(0)
					} else {
						blockHashesRequestTimer = nil
					}
					alarm = true
					processC = missingC
					if !once {
						// if not run at least once fully, launch iterator
						processC = make(chan *poolNode)
						missingC = make(chan *poolNode)
						self.foldUp(orignode, processC)
						once = true
					}
				}
				total = lastMissing
			case <-resetC:
				once = false
				init = false
				done = false
			case node, ok := <-processC:
				if !ok {
					// channel closed, first iteration finished
					init = true
					once = true
					continue
				}
				i++
				// if node has no block
				node.RLock()
				block := node.block
				nhash := node.hash
				knownParent := node.knownParent
				node.RUnlock()
				if !init {
					depth++
				}
				if block == nil {
					missing++
					if !init {
						total++
					}
					hashes = append(hashes, nhash)
					if len(hashes) == blockBatchSize {
						self.requestBlocks(blocksRequests, hashes)
						hashes = nil
					}
					missingC <- node
				} else {
					// block is found
					if knownParent {
						// connected to the blockchain, insert the longest chain of blocks
						var blocks types.Blocks
						child := node
						parent := node
						node.sectionRLock()
						for child != nil && child.block != nil {
							parent = child
							blocks = append(blocks, parent.block)
							child = parent.child
						}
						node.sectionRUnlock()
						poolLogger.Debugf("[%x] insert %v blocks into blockchain", hash[0:4], len(blocks))
						if err := self.insertChain(blocks); err != nil {
							// TODO: not clear which peer we need to address
							// peerError should dispatch to peer if still connected and disconnect
							self.peerError(node.source, ErrInvalidBlock, "%v", err)
							poolLogger.Debugf("invalid block %v", node.hash)
							poolLogger.Debugf("penalise peers %v (hash), %v (block)", node.peer, node.source)
							// penalise peer in node.source
							self.killChain(node, nil)
							// self.disconnect()
							break
						}
						// if suceeded mark the next one (no block yet) as connected to blockchain
						if child != nil {
							child.Lock()
							child.knownParent = true
							child.Unlock()
						}
						// reset starting node to first node with missing block
						orignode = child
						// pop the inserted ancestors off the channel
						for i := 1; i < len(blocks); i++ {
							<-processC
						}
						// delink inserted chain section
						self.killChain(node, parent)
					}
				}
			}
		}
		poolLogger.Debugf("[%x] quit after\n%v block hashes requests\n%v block requests: missing %v/%v/%v", hash[0:4], blockHashesRequests, blocksRequests, missing, total, depth)

		self.wg.Done()
		node.sectionLock()
		node.section.controlC = nil
		node.sectionUnlock()
		// this signals that controller not available
	}()
	return section

}

func (self *BlockPool) peerError(peerId string, code int, format string, params ...interface{}) {
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	peer, ok := self.peers[peerId]
	if ok {
		peer.peerError(code, format, params...)
	}
}

func (self *BlockPool) requestBlockHashes(hash []byte) {
	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	if self.peer != nil {
		self.peer.requestBlockHashes(hash)
	}
}

func (self *BlockPool) requestBlocks(attempts int, hashes [][]byte) {
	// distribute block request among known peers
	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	peerCount := len(self.peers)
	// on first attempt use the best peer
	if attempts == 0 {
		self.peer.requestBlocks(hashes)
		return
	}
	repetitions := int(math.Min(float64(peerCount), float64(blocksRequestRepetition)))
	poolLogger.Debugf("request %v missing blocks from %v/%v peers", len(hashes), repetitions, peerCount)
	i := 0
	indexes := rand.Perm(peerCount)[0:(repetitions - 1)]
	sort.Ints(indexes)
	for _, peer := range self.peers {
		if i == indexes[0] {
			peer.requestBlocks(hashes)
			indexes = indexes[1:]
			if len(indexes) == 0 {
				break
			}
		}
		i++
	}
}

func (self *BlockPool) getPeer(peerId string) (*peerInfo, bool) {
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	if self.peer != nil && self.peer.id == peerId {
		return self.peer, true
	}
	info, ok := self.peers[peerId]
	if !ok {
		panic("unknown peer")
	}
	return info, false
}

func (self *peerInfo) addSection(hash []byte, section *section) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.sections[string(hash)] = section
}

func (self *peerInfo) addRoot(node *poolNode) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.roots = append(self.roots, node)
}

// (re)starts processes registered for this peer (self)
func (self *peerInfo) start(peer *peerInfo) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.quitC = make(chan bool)
	for _, root := range self.roots {
		root.sectionRLock()
		if root.section.bottom != nil {
			if root.parent == nil {
				self.requestBlockHashes(root.hash)
			}
		}
		root.sectionRUnlock()
	}
	self.roots = nil
	self.controlSections(peer, true)
}

//  (re)starts process without requests, only suicide timer
func (self *peerInfo) stop(peer *peerInfo) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	close(self.quitC)
	self.controlSections(peer, false)
}

func (self *peerInfo) controlSections(peer *peerInfo, on bool) {
	if peer != nil {
		peer.lock.RLock()
		defer peer.lock.RUnlock()
	}
	for hash, section := range peer.sections {
		if section.done() {
			delete(self.sections, hash)
		}
		_, exists := peer.sections[hash]
		if on || peer == nil || exists {
			if on {
				// self is best peer
				section.start()
			} else {
				//  (re)starts process without requests, only suicide timer
				section.stop()
			}
		}
	}
}

// called when parent is found in pool
// parent and child are guaranteed to be on different sections
func (self *BlockPool) link(parent, child *poolNode) {
	var top bool
	parent.sectionLock()
	if child != nil {
		child.sectionLock()
	}
	if parent == parent.section.top && parent.section.top != nil {
		top = true
	}
	var bottom bool

	if child == child.section.bottom {
		bottom = true
	}
	if parent.child != child {
		orphan := parent.child
		if orphan != nil {
			// got a fork in the chain
			if top {
				orphan.lock.Lock()
				// make old child orphan
				orphan.parent = nil
				orphan.lock.Unlock()
			} else { // we are under section lock
				// make old child orphan
				orphan.parent = nil
				// reset section objects above the fork
				nchild := orphan.child
				node := orphan
				section := &section{bottom: orphan}
				for node.section == nchild.section {
					node = nchild
					node.section = section
					nchild = node.child
				}
				section.top = node
				// set up a suicide
				self.processSection(orphan).stop()
			}
		} else {
			// child is on top of a chain need to close section
			child.section.bottom = child
		}
		// adopt new child
		parent.child = child
		if !top {
			parent.section.top = parent
			// restart section process so that shorter section is scanned for blocks
			parent.section.reset()
		}
	}

	if child != nil {
		if child.parent != parent {
			stepParent := child.parent
			if stepParent != nil {
				if bottom {
					stepParent.Lock()
					stepParent.child = nil
					stepParent.Unlock()
				} else {
					// we are on the same section
					// if it is a aberrant reverse fork,
					stepParent.child = nil
					node := stepParent
					nparent := stepParent.child
					section := &section{top: stepParent}
					for node.section == nparent.section {
						node = nparent
						node.section = section
						node = node.parent
					}
				}
			} else {
				// linking to a root node, ie. parent is under the root of a chain
				parent.section.top = parent
			}
		}
		child.parent = parent
		child.section.bottom = child
	}
	// this needed if someone lied about the parent before
	child.knownParent = false

	parent.sectionUnlock()
	if child != nil {
		child.sectionUnlock()
	}
}

// this immediately kills the chain from node to end (inclusive) section by section
func (self *BlockPool) killChain(node *poolNode, end *poolNode) {
	poolLogger.Debugf("kill chain section with root node %v", node)

	node.sectionLock()
	node.section.abort()
	self.set(node.hash, nil)
	child := node.child
	top := node.section.top
	i := 1
	self.wg.Add(1)
	go func() {
		var quit bool
		for node != top && node != end && child != nil {
			node = child
			select {
			case <-self.quit:
				quit = true
				break
			default:
			}
			self.set(node.hash, nil)
			child = node.child
		}
		poolLogger.Debugf("killed chain section of %v blocks with root node %v", i, node)
		if !quit {
			if node == top {
				if node != end && child != nil && end != nil {
					//
					self.killChain(child, end)
				}
			} else {
				if child != nil {
					// delink rest of this section if ended midsection
					child.section.bottom = child
					child.parent = nil
				}
			}
		}
		node.section.bottom = nil
		node.sectionUnlock()
		self.wg.Done()
	}()
}

// structure to store long range links on chain to skip along
type section struct {
	lock     sync.RWMutex
	bottom   *poolNode
	top      *poolNode
	controlC chan bool
	resetC   chan bool
}

func (self *section) start() {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if self.controlC != nil {
		self.controlC <- true
	}
}

func (self *section) stop() {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if self.controlC != nil {
		self.controlC <- false
	}
}

func (self *section) reset() {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if self.controlC != nil {
		self.resetC <- true
		self.controlC <- false
	}
}

func (self *section) abort() {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.controlC != nil {
		close(self.controlC)
		self.controlC = nil
	}
}

func (self *section) done() bool {
	self.lock.Lock()
	defer self.lock.Unlock()
	if self.controlC != nil {
		return true
	}
	return false
}

func (self *BlockPool) get(hash []byte) (node *poolNode) {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.pool[string(hash)]
}

func (self *BlockPool) set(hash []byte, node *poolNode) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.pool[string(hash)] = node
}

// first time for block request, this iteration retrieves nodes of the chain
// from node up to top (all the way if nil) via child links
// copies the controller
// and feeds nodeC channel
// this is performed under section readlock to prevent top from going away
// when
func (self *BlockPool) foldUp(node *poolNode, nodeC chan *poolNode) {
	self.wg.Add(1)
	go func() {
		node.sectionRLock()
		defer node.sectionRUnlock()
		for node != nil {
			select {
			case <-self.quit:
				break
			case nodeC <- node:
				if node == node.section.top {
					break
				}
				node = node.child
			}
		}
		close(nodeC)
		self.wg.Done()
	}()
}

func (self *poolNode) Lock() {
	self.sectionLock()
	self.lock.Lock()
}

func (self *poolNode) Unlock() {
	self.lock.Unlock()
	self.sectionUnlock()
}

func (self *poolNode) RLock() {
	self.lock.RLock()
}

func (self *poolNode) RUnlock() {
	self.lock.RUnlock()
}

func (self *poolNode) sectionLock() {
	self.lock.RLock()
	defer self.lock.RUnlock()
	self.section.lock.Lock()
}

func (self *poolNode) sectionUnlock() {
	self.lock.RLock()
	defer self.lock.RUnlock()
	self.section.lock.Unlock()
}

func (self *poolNode) sectionRLock() {
	self.lock.RLock()
	defer self.lock.RUnlock()
	self.section.lock.RLock()
}

func (self *poolNode) sectionRUnlock() {
	self.lock.RLock()
	defer self.lock.RUnlock()
	self.section.lock.RUnlock()
}
