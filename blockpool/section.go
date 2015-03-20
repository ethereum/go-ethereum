package blockpool

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

/*
  section is the worker on each chain section in the block pool
  - remove the section if there are blocks missing after an absolute time
  - remove the section if there are maxIdleRounds of idle rounds of block requests with no response
  - periodically polls the chain section for missing blocks which are then requested from peers
  - registers the process controller on the peer so that if the peer is promoted as best peer the second time (after a disconnect of a better one), all active processes are switched back on unless they removed (inserted in blockchain, invalid or expired)
  - when turned off (if peer disconnects and new peer connects with alternative chain), no blockrequests are made but absolute expiry timer is ticking
  - when turned back on it recursively calls itself on the root of the next chain section
*/
type section struct {
	lock sync.RWMutex

	parent *section // connecting section back in time towards blockchain
	child  *section // connecting section forward in time

	top    *node // the topmost node = head node = youngest node within the chain section
	bottom *node // the bottom node = root node = oldest node within the chain section
	nodes  []*node

	peer       *peer
	parentHash common.Hash

	blockHashes []common.Hash

	poolRootIndex int

	bp *BlockPool

	controlC  chan *peer     // to (de)register the current best peer
	poolRootC chan *peer     // indicate connectedness to blockchain (well, known blocks)
	offC      chan bool      // closed if process terminated
	suicideC  chan bool      // initiate suicide on the section
	quitInitC chan bool      // to signal end of initialisation
	forkC     chan chan bool // freeze section process while splitting
	switchC   chan bool      // switching
	idleC     chan bool      // channel to indicate thai food
	processC  chan *node     //
	missingC  chan *node     //

	blocksRequestTimer      <-chan time.Time
	blockHashesRequestTimer <-chan time.Time
	suicideTimer            <-chan time.Time

	blocksRequests      int
	blockHashesRequests int

	blocksRequestsComplete      bool
	blockHashesRequestsComplete bool
	ready                       bool
	same                        bool
	initialised                 bool
	active                      bool

	step        int
	idle        int
	missing     int
	lastMissing int
	depth       int
	invalid     bool
	poolRoot    bool
}

//
func (self *BlockPool) newSection(nodes []*node) *section {
	sec := &section{
		bottom:        nodes[len(nodes)-1],
		top:           nodes[0],
		nodes:         nodes,
		poolRootIndex: len(nodes),
		bp:            self,
		controlC:      make(chan *peer),
		poolRootC:     make(chan *peer),
		offC:          make(chan bool),
	}

	for i, n := range nodes {
		entry := &entry{node: n, section: sec, index: &index{i}}
		self.set(n.hash, entry)
	}

	plog.DebugDetailf("[%s] setup section process", sectionhex(sec))

	go sec.run()
	return sec
}

func (self *section) addSectionToBlockChain(p *peer) {
	self.bp.wg.Add(1)
	go func() {

		self.lock.Lock()
		defer self.lock.Unlock()
		defer func() {
			self.bp.wg.Done()
		}()

		var nodes []*node
		var n *node
		var keys []common.Hash
		var blocks []*types.Block
		for self.poolRootIndex > 0 {
			n = self.nodes[self.poolRootIndex-1]
			n.lock.RLock()
			block := n.block
			n.lock.RUnlock()
			if block == nil {
				break
			}
			self.poolRootIndex--
			keys = append(keys, n.hash)
			blocks = append(blocks, block)
			nodes = append(nodes, n)
		}

		if len(blocks) == 0 {
			return
		}

		self.bp.lock.Lock()
		for _, key := range keys {
			delete(self.bp.pool, key)
		}
		self.bp.lock.Unlock()

		plog.Infof("[%s] insert %v blocks [%v/%v] into blockchain", sectionhex(self), len(blocks), hex(blocks[0].Hash()), hex(blocks[len(blocks)-1].Hash()))
		err := self.bp.insertChain(blocks)
		if err != nil {
			self.invalid = true
			self.bp.peers.peerError(n.blockBy, ErrInvalidBlock, "%v", err)
			plog.Warnf("invalid block %x", n.hash)
			plog.Warnf("penalise peers %v (hash), %v (block)", n.hashBy, n.blockBy)

			// or invalid block and the entire chain needs to be removed
			self.removeChain()
		} else {
			// check tds
			self.bp.wg.Add(1)
			go func() {
				plog.DebugDetailf("checking td")
				self.bp.checkTD(nodes...)
				self.bp.wg.Done()
			}()
			// if all blocks inserted in this section
			// then need to try to insert blocks in child section
			if self.poolRootIndex == 0 {
				// if there is a child section, then recursively call itself:
				// also if section process is not terminated,
				// then signal blockchain connectivity with poolRootC
				if child := self.bp.getChild(self); child != nil {
					select {
					case <-child.offC:
						plog.DebugDetailf("[%s] add complete child section [%s] to the blockchain", sectionhex(self), sectionhex(child))
					case child.poolRootC <- p:
						plog.DebugDetailf("[%s] add incomplete child section [%s] to the blockchain", sectionhex(self), sectionhex(child))
					}
					child.addSectionToBlockChain(p)
				} else {
					plog.DebugDetailf("[%s] no child section in pool", sectionhex(self))
				}
				plog.DebugDetailf("[%s] section completely inserted to blockchain - remove", sectionhex(self))
				// complete sections are removed. if called from within section process,
				// this must run in its own go routine to avoid deadlock
				self.remove()
			}
		}

		self.bp.status.lock.Lock()
		if err == nil {
			headKey := blocks[0].ParentHash().Str()
			height := self.bp.status.chain[headKey] + len(blocks)
			self.bp.status.chain[blocks[len(blocks)-1].Hash().Str()] = height
			if height > self.bp.status.values.LongestChain {
				self.bp.status.values.LongestChain = height
			}
			delete(self.bp.status.chain, headKey)
		}
		self.bp.status.values.BlocksInChain += len(blocks)
		self.bp.status.values.BlocksInPool -= len(blocks)
		if err != nil {
			self.bp.status.badPeers[n.blockBy]++
		}
		self.bp.status.lock.Unlock()

	}()

}

func (self *section) run() {

	// absolute time after which sub-chain is killed if not complete (some blocks are missing)
	self.suicideC = make(chan bool)
	self.forkC = make(chan chan bool)
	self.suicideTimer = time.After(self.bp.Config.BlocksTimeout)

	// node channels for the section
	// container for missing block hashes
	var checking bool
	var ping = time.NewTicker(5 * time.Second)

LOOP:
	for !self.blockHashesRequestsComplete || !self.blocksRequestsComplete {

		select {
		case <-ping.C:
			var name = "no peer"
			if self.peer != nil {
				name = self.peer.id
			}
			plog.DebugDetailf("[%s] peer <%s> active: %v", sectionhex(self), name, self.active)

		// global quit from blockpool
		case <-self.bp.quit:
			break LOOP

		// pause for peer switching
		case <-self.switchC:
			self.switchC = nil

		case p := <-self.poolRootC:
			// signal on pool root channel indicates that the blockpool is
			// connected to the blockchain, insert the longest chain of blocks
			// ignored in idle mode to avoid inserting chain sections of non-live peers
			self.poolRoot = true
			// switch off hash requests in case they were on
			self.blockHashesRequestTimer = nil
			self.blockHashesRequestsComplete = true
			self.switchOn(p)

		// peer quit or demoted, put section in idle mode
		case <-self.idleC:
			// peer quit or demoted, put section in idle mode
			plog.Debugf("[%s] peer <%s> quit or demoted", sectionhex(self), self.peer.id)
			self.switchOff()
			self.idleC = nil

		// timebomb - if section is not complete in time, nuke the entire chain
		case <-self.suicideTimer:
			self.removeChain()
			plog.Debugf("[%s] timeout. (%v total attempts): missing %v/%v/%v...suicide", sectionhex(self), self.blocksRequests, self.missing, self.lastMissing, self.depth)
			self.suicideTimer = nil
			break LOOP

		// closing suicideC triggers section suicide: removes section nodes from pool and terminates section process
		case <-self.suicideC:
			plog.DebugDetailf("[%s] quit", sectionhex(self))
			break LOOP

		// alarm for checking blocks in the section
		case <-self.blocksRequestTimer:
			plog.DebugDetailf("[%s] alarm: block request time", sectionhex(self))
			self.processC = self.missingC

		// alarm for checking parent of the section or sending out hash requests
		case <-self.blockHashesRequestTimer:
			plog.DebugDetailf("[%s] alarm: hash request time", sectionhex(self))
			self.blockHashesRequest()

		// activate this section process with a peer
		case p := <-self.controlC:
			if p == nil {
				self.switchOff()
			} else {
				self.switchOn(p)
			}
			self.bp.wg.Done()
		// blocks the process until section is split at the fork
		case waiter := <-self.forkC:
			<-waiter
			self.initialised = false
			self.quitInitC = nil

		//
		case n, ok := <-self.processC:
			// channel closed, first iteration finished
			if !ok && !self.initialised {
				plog.DebugDetailf("[%s] section initalised: missing %v/%v/%v", sectionhex(self), self.missing, self.lastMissing, self.depth)
				self.initialised = true
				self.processC = nil
				// self.processC = make(chan *node, self.missing)
				self.checkRound()
				checking = false
				break
			}
			// plog.DebugDetailf("[%s] section proc step %v: missing %v/%v/%v", sectionhex(self), self.step, self.missing, self.lastMissing, self.depth)
			if !checking {
				self.step = 0
				self.missing = 0
				checking = true
			}
			self.step++

			n.lock.RLock()
			block := n.block
			n.lock.RUnlock()

			// if node has no block, request it (buffer it for batch request)
			// feed it to missingC channel for the next round
			if block == nil {
				pos := self.missing % self.bp.Config.BlockBatchSize
				if pos == 0 {
					if self.missing != 0 {
						self.bp.requestBlocks(self.blocksRequests, self.blockHashes[:])
					}
					self.blockHashes = self.bp.getHashSlice()
				}
				self.blockHashes[pos] = n.hash
				self.missing++
				self.missingC <- n
			} else {
				// checking for parent block
				if self.poolRoot {
					// if node has got block (received via async AddBlock call from protocol)
					if self.step == self.lastMissing {
						// current root of the pool
						plog.DebugDetailf("[%s] received block for current pool root %s", sectionhex(self), hex(n.hash))
						self.addSectionToBlockChain(self.peer)
					}
				} else {
					if (self.parentHash == common.Hash{}) && n == self.bottom {
						self.parentHash = block.ParentHash()
						plog.DebugDetailf("[%s] got parent head block hash %s...checking", sectionhex(self), hex(self.parentHash))
						self.blockHashesRequest()
					}
				}
			}
			if self.initialised && self.step == self.lastMissing {
				plog.DebugDetailf("[%s] check if new blocks arrived (attempt %v): missing %v/%v/%v", sectionhex(self), self.blocksRequests, self.missing, self.lastMissing, self.depth)
				self.checkRound()
				checking = false
			}
		} // select
	} // for

	close(self.offC)
	if self.peer != nil {
		self.active = false
		self.bp.wg.Done()
	}

	plog.DebugDetailf("[%s] section process terminated: %v blocks retrieved (%v attempts), hash requests complete on root (%v attempts).", sectionhex(self), self.depth, self.blocksRequests, self.blockHashesRequests)

}

func (self *section) switchOn(newpeer *peer) {

	oldpeer := self.peer
	// reset switchC/switchC to current best peer
	self.idleC = newpeer.idleC
	self.switchC = newpeer.switchC
	self.peer = newpeer

	if oldpeer != newpeer {
		oldp := "no peer"
		newp := "no peer"
		if oldpeer != nil {
			oldp = oldpeer.id
		}
		if newpeer != nil {
			newp = newpeer.id
		}

		plog.DebugDetailf("[%s] active mode <%s> -> <%s>", sectionhex(self), oldp, newp)
	}

	// activate section with current peer
	if oldpeer == nil {
		self.bp.wg.Add(1)
		self.active = true

		if !self.blockHashesRequestsComplete {
			self.blockHashesRequestTimer = time.After(0)
		}
		if !self.blocksRequestsComplete {
			if !self.initialised {
				if self.quitInitC != nil {
					<-self.quitInitC
				}
				self.missingC = make(chan *node, self.bp.Config.BlockHashesBatchSize)
				self.processC = make(chan *node, self.bp.Config.BlockHashesBatchSize)
				self.quitInitC = make(chan bool)

				self.step = 0
				self.missing = 0
				self.depth = len(self.nodes)
				self.lastMissing = self.depth

				self.feedNodes()
			} else {
				self.blocksRequestTimer = time.After(0)
			}
		}
	}
}

// put the section to idle mode
func (self *section) switchOff() {
	// active -> idle
	if self.peer != nil {
		oldp := "no peer"
		oldpeer := self.peer
		if oldpeer != nil {
			oldp = oldpeer.id
		}
		plog.DebugDetailf("[%s] idle mode peer <%s> -> <> (%v total attempts): missing %v/%v/%v", sectionhex(self), oldp, self.blocksRequests, self.missing, self.lastMissing, self.depth)

		self.active = false
		self.peer = nil
		// turn off timers
		self.blocksRequestTimer = nil
		self.blockHashesRequestTimer = nil

		if self.quitInitC != nil {
			<-self.quitInitC
			self.quitInitC = nil
		}
		self.processC = nil
		self.bp.wg.Done()
	}
}

// iterates through nodes of a section to feed processC
// used to initialise chain section
func (self *section) feedNodes() {
	// if not run at least once fully, launch iterator
	self.bp.wg.Add(1)
	go func() {
		self.lock.Lock()
		defer self.lock.Unlock()
		defer func() {
			self.bp.wg.Done()
		}()
		var n *node
	INIT:
		for _, n = range self.nodes {
			select {
			case self.processC <- n:
			case <-self.bp.quit:
				break INIT
			}
		}
		close(self.processC)
		close(self.quitInitC)
	}()
}

func (self *section) blockHashesRequest() {

	if self.switchC != nil {
		self.bp.chainLock.Lock()
		parentSection := self.parent

		if parentSection == nil {

			// only link to new parent if not switching peers
			// this protects against synchronisation issue where during switching
			// a demoted peer's fork will be chosen over the best peer's chain
			// because relinking the correct chain (activateChain) is overwritten here in
			// demoted peer's section process just before the section is put to idle mode
			if (self.parentHash != common.Hash{}) {
				if parent := self.bp.get(self.parentHash); parent != nil {
					parentSection = parent.section
					plog.DebugDetailf("[%s] blockHashesRequest: parent section [%s] linked\n", sectionhex(self), sectionhex(parentSection))
					link(parentSection, self)
				} else {
					if self.bp.hasBlock(self.parentHash) {
						self.poolRoot = true
						plog.DebugDetailf("[%s] blockHashesRequest: parentHash known ... inserting section in blockchain", sectionhex(self))
						self.addSectionToBlockChain(self.peer)
						self.blockHashesRequestTimer = nil
						self.blockHashesRequestsComplete = true
					}
				}
			}
		}
		self.bp.chainLock.Unlock()

		if !self.poolRoot {
			if parentSection != nil {
				//  activate parent section with this peer
				// but only if not during switch mode
				plog.DebugDetailf("[%s] parent section [%s] activated\n", sectionhex(self), sectionhex(parentSection))
				self.bp.activateChain(parentSection, self.peer, nil)
				// if not root of chain, switch off
				plog.DebugDetailf("[%s] parent found, hash requests deactivated (after %v total attempts)\n", sectionhex(self), self.blockHashesRequests)
				self.blockHashesRequestTimer = nil
				self.blockHashesRequestsComplete = true
			} else {
				self.blockHashesRequests++
				plog.DebugDetailf("[%s] hash request on root (%v total attempts)\n", sectionhex(self), self.blockHashesRequests)
				self.peer.requestBlockHashes(self.bottom.hash)
				self.blockHashesRequestTimer = time.After(self.bp.Config.BlockHashesRequestInterval)
			}
		}
	}
}

// checks number of missing blocks after each round of request and acts accordingly
func (self *section) checkRound() {
	if self.missing == 0 {
		// no missing blocks
		plog.DebugDetailf("[%s] section checked: got all blocks. process complete (%v total blocksRequests): missing %v/%v/%v", sectionhex(self), self.blocksRequests, self.missing, self.lastMissing, self.depth)
		self.blocksRequestsComplete = true
		self.blocksRequestTimer = nil
	} else {
		// some missing blocks
		plog.DebugDetailf("[%s] section checked: missing %v/%v/%v", sectionhex(self), self.missing, self.lastMissing, self.depth)
		self.blocksRequests++
		pos := self.missing % self.bp.Config.BlockBatchSize
		if pos == 0 {
			pos = self.bp.Config.BlockBatchSize
		}
		self.bp.requestBlocks(self.blocksRequests, self.blockHashes[:pos])

		// handle idle rounds
		if self.missing == self.lastMissing {
			// idle round
			if self.same {
				// more than once
				self.idle++
				// too many idle rounds
				if self.idle >= self.bp.Config.BlocksRequestMaxIdleRounds {
					plog.DebugDetailf("[%s] block requests had %v idle rounds (%v total attempts): missing %v/%v/%v\ngiving up...", sectionhex(self), self.idle, self.blocksRequests, self.missing, self.lastMissing, self.depth)
					self.removeChain()
				}
			} else {
				self.idle = 0
			}
			self.same = true
		} else {
			self.same = false
		}
		self.lastMissing = self.missing
		// put processC offline
		self.processC = nil
		self.blocksRequestTimer = time.After(self.bp.Config.BlocksRequestInterval)
	}
}

/*
 link connects two sections via parent/child fields
 creating a doubly linked list
 caller must hold BlockPool chainLock
*/
func link(parent *section, child *section) {
	if parent != nil {
		exChild := parent.child
		parent.child = child
		if exChild != nil && exChild != child {
			if child != nil {
				// if child is nil it is not a real fork
				plog.DebugDetailf("[%s] chain fork [%s] -> [%s]", sectionhex(parent), sectionhex(exChild), sectionhex(child))
			}
			exChild.parent = nil
		}
	}
	if child != nil {
		exParent := child.parent
		if exParent != nil && exParent != parent {
			if parent != nil {
				// if parent is nil it is not a real fork, but suicide delinking section
				plog.DebugDetailf("[%s] chain reverse fork [%s] -> [%s]", sectionhex(child), sectionhex(exParent), sectionhex(parent))
			}
			exParent.child = nil
		}
		child.parent = parent
	}
}

/*
  handle forks where connecting node is mid-section
  by splitting section at fork
  no splitting needed if connecting node is head of a section
  caller must hold chain lock
*/
func (self *BlockPool) splitSection(parent *section, entry *entry) {
	plog.DebugDetailf("[%s] split section at fork", sectionhex(parent))
	parent.deactivate()
	waiter := make(chan bool)
	parent.wait(waiter)
	chain := parent.nodes
	parent.nodes = chain[entry.index.int:]
	parent.top = parent.nodes[0]
	parent.poolRootIndex -= entry.index.int
	orphan := self.newSection(chain[0:entry.index.int])
	link(orphan, parent.child)
	close(waiter)
	orphan.deactivate()
}

func (self *section) wait(waiter chan bool) {
	self.forkC <- waiter
}

func (self *BlockPool) linkSections(nodes []*node, parent, child *section) (sec *section) {
	// if new section is created, link it to parent/child sections
	// and launch section process fetching block and further hashes
	if len(nodes) > 0 {
		sec = self.newSection(nodes)
		plog.Debugf("[%s]->[%s](%v)->[%s] new chain section", sectionhex(parent), sectionhex(sec), len(nodes), sectionhex(child))
		link(parent, sec)
		link(sec, child)
	} else {
		if parent != nil && child != nil {
			// now this can only happen if we allow response to hash request to include <from> hash
			// in this case we just link parent and child (without needing root block of child section)
			plog.Debugf("[%s]->[%s] connecting known sections", sectionhex(parent), sectionhex(child))
			link(parent, child)
		}
	}
	return
}

func (self *section) activate(p *peer) {
	self.bp.wg.Add(1)
	select {
	case <-self.offC:
		plog.DebugDetailf("[%s] completed section process. cannot activate for peer <%s>", sectionhex(self), p.id)
		self.bp.wg.Done()
	case self.controlC <- p:
		plog.DebugDetailf("[%s] activate section process for peer <%s>", sectionhex(self), p.id)
	}
}

func (self *section) deactivate() {
	self.bp.wg.Add(1)
	self.controlC <- nil
}

// removes this section exacly
func (self *section) remove() {
	select {
	case <-self.offC:
		close(self.suicideC)
		plog.DebugDetailf("[%s] remove: suicide", sectionhex(self))
	case <-self.suicideC:
		plog.DebugDetailf("[%s] remove: suicided already", sectionhex(self))
	default:
		plog.DebugDetailf("[%s] remove: suicide", sectionhex(self))
		close(self.suicideC)
	}
	self.unlink()
	self.bp.remove(self)
	plog.DebugDetailf("[%s] removed section.", sectionhex(self))

}

// remove a section and all its descendents from the pool
func (self *section) removeChain() {
	// need to get the child before removeSection delinks the section
	self.bp.chainLock.RLock()
	child := self.child
	self.bp.chainLock.RUnlock()

	plog.DebugDetailf("[%s] remove chain", sectionhex(self))
	self.remove()
	if child != nil {
		child.removeChain()
	}
}

// unlink a section from its parent/child
func (self *section) unlink() {
	// first delink from child and parent under chainlock
	self.bp.chainLock.Lock()
	link(nil, self)
	link(self, nil)
	self.bp.chainLock.Unlock()
}
