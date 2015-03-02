package blockpool

import (
	"bytes"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/errs"
	"github.com/ethereum/go-ethereum/ethutil"
)

type peer struct {
	lock sync.RWMutex

	// last known blockchain status
	td               *big.Int
	currentBlockHash []byte
	currentBlock     *types.Block
	parentHash       []byte
	headSection      *section

	id string

	// peer callbacks
	requestBlockHashes func([]byte) error
	requestBlocks      func([][]byte) error
	peerError          func(*errs.Error)
	errors             *errs.Errors

	sections [][]byte

	// channels to push new head block and head section for peer a
	currentBlockC chan *types.Block
	headSectionC  chan *section

	// channels to signal peers witch and peer quit
	idleC   chan bool
	switchC chan bool

	quit chan bool
	bp   *BlockPool

	// timers for head section process
	blockHashesRequestTimer <-chan time.Time
	blocksRequestTimer      <-chan time.Time
	suicide                 <-chan time.Time

	idle bool
}

// peers is the component keeping a record of peers in a hashmap
//
type peers struct {
	lock sync.RWMutex

	bp     *BlockPool
	errors *errs.Errors
	peers  map[string]*peer
	best   *peer
	status *status
}

// peer constructor
func (self *peers) newPeer(
	td *big.Int,
	currentBlockHash []byte,
	id string,
	requestBlockHashes func([]byte) error,
	requestBlocks func([][]byte) error,
	peerError func(*errs.Error),
) (p *peer) {

	p = &peer{
		errors:             self.errors,
		td:                 td,
		currentBlockHash:   currentBlockHash,
		id:                 id,
		requestBlockHashes: requestBlockHashes,
		requestBlocks:      requestBlocks,
		peerError:          peerError,
		currentBlockC:      make(chan *types.Block),
		headSectionC:       make(chan *section),
		bp:                 self.bp,
		idle:               true,
	}
	// at creation the peer is recorded in the peer pool
	self.peers[id] = p
	return
}

// dispatches an error to a peer if still connected
func (self *peers) peerError(id string, code int, format string, params ...interface{}) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	peer, ok := self.peers[id]
	if ok {
		peer.addError(code, format, params)
	}
	// blacklisting comes here
}

func (self *peer) addError(code int, format string, params ...interface{}) {
	err := self.errors.New(code, format, params...)
	self.peerError(err)
}

func (self *peer) setChainInfo(td *big.Int, c []byte) {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.td = td
	self.currentBlockHash = c

	self.currentBlock = nil
	self.parentHash = nil
	self.headSection = nil
}

func (self *peer) setChainInfoFromBlock(block *types.Block) {
	self.lock.Lock()
	defer self.lock.Unlock()
	// use the optional TD to update peer td, this helps second best peer selection
	// in case best peer is lost
	if block.Td != nil && block.Td.Cmp(self.td) > 0 {
		plog.DebugDetailf("setChainInfoFromBlock: update <%s> - head: %v->%v - TD: %v->%v", self.id, hex(self.currentBlockHash), hex(block.Hash()), self.td, block.Td)
		self.td = block.Td
		self.currentBlockHash = block.Hash()
		self.parentHash = block.ParentHash()
		self.currentBlock = block
		self.headSection = nil
	}
	self.bp.wg.Add(1)
	go func() {
		self.currentBlockC <- block
		self.bp.wg.Done()
	}()
}

func (self *peers) requestBlocks(attempts int, hashes [][]byte) {
	// distribute block request among known peers
	self.lock.RLock()
	defer self.lock.RUnlock()
	peerCount := len(self.peers)
	// on first attempt use the best peer
	if attempts == 0 {
		plog.DebugDetailf("request %v missing blocks from best peer <%s>", len(hashes), self.best.id)
		self.best.requestBlocks(hashes)
		return
	}
	repetitions := self.bp.Config.BlocksRequestRepetition
	if repetitions > peerCount {
		repetitions = peerCount
	}
	i := 0
	indexes := rand.Perm(peerCount)[0:repetitions]
	sort.Ints(indexes)

	plog.DebugDetailf("request %v missing blocks from %v/%v peers", len(hashes), repetitions, peerCount)
	for _, peer := range self.peers {
		if i == indexes[0] {
			plog.DebugDetailf("request length: %v", len(hashes))
			plog.DebugDetailf("request %v missing blocks [%x/%x] from peer <%s>", len(hashes), hashes[0][:4], hashes[len(hashes)-1][:4], peer.id)
			peer.requestBlocks(hashes)
			indexes = indexes[1:]
			if len(indexes) == 0 {
				break
			}
		}
		i++
	}
	self.bp.putHashSlice(hashes)
}

// addPeer implements the logic for blockpool.AddPeer
// returns true iff peer is promoted as best peer in the pool
func (self *peers) addPeer(
	td *big.Int,
	currentBlockHash []byte,
	id string,
	requestBlockHashes func([]byte) error,
	requestBlocks func([][]byte) error,
	peerError func(*errs.Error),
) (best bool) {

	var previousBlockHash []byte
	self.lock.Lock()
	p, found := self.peers[id]
	if found {
		if !bytes.Equal(p.currentBlockHash, currentBlockHash) {
			previousBlockHash = p.currentBlockHash
			plog.Debugf("addPeer: Update peer <%s> with td %v and current block %s (was %v)", id, td, hex(currentBlockHash), hex(previousBlockHash))
			p.setChainInfo(td, currentBlockHash)
			self.status.lock.Lock()
			self.status.values.NewBlocks++
			self.status.lock.Unlock()
		}
	} else {
		p = self.newPeer(td, currentBlockHash, id, requestBlockHashes, requestBlocks, peerError)

		self.status.lock.Lock()

		self.status.peers[id]++
		self.status.values.NewBlocks++
		self.status.lock.Unlock()

		plog.Debugf("addPeer: add new peer <%v> with td %v and current block %s", id, td, hex(currentBlockHash))
	}
	self.lock.Unlock()

	// check peer current head
	if self.bp.hasBlock(currentBlockHash) {
		// peer not ahead
		plog.Debugf("addPeer: peer <%v> with td %v and current block %s is behind", id, td, hex(currentBlockHash))
		return false
	}

	if self.best == p {
		// new block update for active current best peer -> request hashes
		plog.Debugf("addPeer: <%s> already the best peer. Request new head section info from %s", id, hex(currentBlockHash))

		if previousBlockHash != nil {
			if entry := self.bp.get(previousBlockHash); entry != nil {
				p.headSectionC <- nil
				self.bp.activateChain(entry.section, p, nil)
				p.sections = append(p.sections, previousBlockHash)
			}
		}
		best = true
	} else {
		currentTD := ethutil.Big0
		if self.best != nil {
			currentTD = self.best.td
		}
		if td.Cmp(currentTD) > 0 {
			self.status.lock.Lock()
			self.status.bestPeers[p.id]++
			self.status.lock.Unlock()
			plog.Debugf("addPeer: peer <%v> promoted best peer", id)
			self.bp.switchPeer(self.best, p)
			self.best = p
			best = true
		}
	}
	return
}

// removePeer is called (via RemovePeer) by the eth protocol when the peer disconnects
func (self *peers) removePeer(id string) {
	self.lock.Lock()
	defer self.lock.Unlock()

	p, found := self.peers[id]
	if !found {
		return
	}

	delete(self.peers, id)
	plog.Debugf("addPeer: remove peer <%v>", id)

	// if current best peer is removed, need to find a better one
	if self.best == p {
		var newp *peer
		// FIXME: own TD
		max := ethutil.Big0
		// peer with the highest self-acclaimed TD is chosen
		for _, pp := range self.peers {
			if pp.td.Cmp(max) > 0 {
				max = pp.td
				newp = pp
			}
		}
		if newp != nil {
			self.status.lock.Lock()
			self.status.bestPeers[p.id]++
			self.status.lock.Unlock()
			plog.Debugf("addPeer: peer <%v> with td %v promoted best peer", newp.id, newp.td)
		} else {
			plog.Warnln("addPeer: no suitable peers found")
		}
		self.best = newp
		self.bp.switchPeer(p, newp)
	}
}

// switchPeer launches section processes based on information about
// shared interest and legacy of peers
func (self *BlockPool) switchPeer(oldp, newp *peer) {

	// first quit AddBlockHashes, requestHeadSection and activateChain
	if oldp != nil {
		plog.DebugDetailf("<%s> quit peer processes", oldp.id)
		close(oldp.switchC)
	}
	if newp != nil {
		newp.idleC = make(chan bool)
		newp.switchC = make(chan bool)
		// if new best peer has no head section yet, create it and run it
		// otherwise head section is an element of peer.sections
		if newp.headSection == nil {
			plog.DebugDetailf("[%s] head section for [%s] not created, requesting info", newp.id, hex(newp.currentBlockHash))

			if newp.idle {
				self.wg.Add(1)
				newp.idle = false
				self.syncing()
			}

			go func() {
				newp.run()
				if !newp.idle {
					self.wg.Done()
					newp.idle = true
				}
			}()

		}

		var connected = make(map[string]*section)
		var sections [][]byte
		for _, hash := range newp.sections {
			plog.DebugDetailf("activate chain starting from section [%s]", hex(hash))
			// if section not connected (ie, top of a contiguous sequence of sections)
			if connected[string(hash)] == nil {
				// if not deleted, then reread from pool (it can be orphaned top half of a split section)
				if entry := self.get(hash); entry != nil {
					self.activateChain(entry.section, newp, connected)
					connected[string(hash)] = entry.section
					sections = append(sections, hash)
				}
			}
		}
		plog.DebugDetailf("<%s> section processes (%v non-contiguous sequences, was %v before)", newp.id, len(sections), len(newp.sections))
		// need to lock now that newp is exposed to section processes
		newp.lock.Lock()
		newp.sections = sections
		newp.lock.Unlock()
	}
	// finally deactivate section process for sections where newp didnt activate
	// newp activating section process changes the quit channel for this reason
	if oldp != nil {
		plog.DebugDetailf("<%s> quit section processes", oldp.id)
		//
		close(oldp.idleC)
	}
}

func (self *peers) getPeer(id string) (p *peer, best bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	if self.best != nil && self.best.id == id {
		return self.best, true
	}
	p = self.peers[id]
	return
}

func (self *peer) handleSection(sec *section) {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.headSection = sec
	self.blockHashesRequestTimer = nil

	if sec == nil {
		if self.idle {
			self.idle = false
			self.bp.wg.Add(1)
			self.bp.syncing()
		}

		self.suicide = time.After(self.bp.Config.BlockHashesTimeout)

		plog.DebugDetailf("HeadSection: <%s> head block hash changed (mined block received). New head %s", self.id, hex(self.currentBlockHash))
	} else {
		if !self.idle {
			self.idle = true
			self.suicide = nil
			self.bp.wg.Done()
		}
		plog.DebugDetailf("HeadSection: <%s> head section [%s] created", self.id, sectionhex(sec))
	}
}

func (self *peer) getCurrentBlock(currentBlock *types.Block) {
	// called by update or after AddBlock signals that head block of current peer is received
	if currentBlock == nil {
		if entry := self.bp.get(self.currentBlockHash); entry != nil {
			entry.node.lock.Lock()
			currentBlock = entry.node.block
			entry.node.lock.Unlock()
		}
		if currentBlock != nil {
			plog.DebugDetailf("HeadSection: <%s> head block %s found in blockpool", self.id, hex(self.currentBlockHash))
		} else {
			plog.DebugDetailf("HeadSection: <%s> head block %s not found... requesting it", self.id, hex(self.currentBlockHash))
			self.requestBlocks([][]byte{self.currentBlockHash})
			self.blocksRequestTimer = time.After(self.bp.Config.BlocksRequestInterval)
			return
		}
	} else {
		plog.DebugDetailf("HeadSection: <%s> head block %s received (parent: %s)", self.id, hex(self.currentBlockHash), hex(currentBlock.ParentHash()))
	}

	self.lock.Lock()
	defer self.lock.Unlock()
	self.currentBlock = currentBlock
	self.parentHash = currentBlock.ParentHash()
	plog.DebugDetailf("HeadSection: <%s> head block %s found (parent: [%s])... requesting  hashes", self.id, hex(self.currentBlockHash), hex(self.parentHash))
	self.blockHashesRequestTimer = time.After(0)
	self.blocksRequestTimer = nil
}

func (self *peer) getBlockHashes() {
	//if connecting parent is found
	if self.bp.hasBlock(self.parentHash) {
		plog.DebugDetailf("HeadSection: <%s> parent block %s found in blockchain", self.id, hex(self.parentHash))
		err := self.bp.insertChain(types.Blocks([]*types.Block{self.currentBlock}))
		if err != nil {
			self.addError(ErrInvalidBlock, "%v", err)

			self.bp.status.lock.Lock()
			self.bp.status.badPeers[self.id]++
			self.bp.status.lock.Unlock()
		}
	} else {
		if parent := self.bp.get(self.parentHash); parent != nil {
			if self.bp.get(self.currentBlockHash) == nil {
				plog.DebugDetailf("HeadSection: <%s> connecting parent %s found in pool... creating singleton section", self.id, hex(self.parentHash))
				n := &node{
					hash:    self.currentBlockHash,
					block:   self.currentBlock,
					hashBy:  self.id,
					blockBy: self.id,
				}
				self.bp.newSection([]*node{n}).activate(self)
			} else {
				plog.DebugDetailf("HeadSection: <%s> connecting parent %s found in pool...head section [%s] exists...not requesting hashes", self.id, hex(self.parentHash), sectionhex(parent.section))
				self.bp.activateChain(parent.section, self, nil)
			}
		} else {
			plog.DebugDetailf("HeadSection: <%s> section [%s] requestBlockHashes", self.id, sectionhex(self.headSection))
			self.requestBlockHashes(self.currentBlockHash)
			self.blockHashesRequestTimer = time.After(self.bp.Config.BlockHashesRequestInterval)
			return
		}
	}
	self.blockHashesRequestTimer = nil
	if !self.idle {
		self.idle = true
		self.suicide = nil
		self.bp.wg.Done()
	}
}

// main loop for head section process
func (self *peer) run() {

	self.lock.RLock()
	switchC := self.switchC
	currentBlockHash := self.currentBlockHash
	self.lock.RUnlock()

	self.blockHashesRequestTimer = nil

	self.blocksRequestTimer = time.After(0)
	self.suicide = time.After(self.bp.Config.BlockHashesTimeout)

	var quit chan bool

	var ping = time.NewTicker(5 * time.Second)

LOOP:
	for {
		select {
		case <-ping.C:
			plog.Debugf("HeadSection: <%s> section with head %s, idle: %v", self.id, hex(self.currentBlockHash), self.idle)

		// signal from AddBlockHashes that head section for current best peer is created
		// if sec == nil, it signals that chain info has updated (new block message)
		case sec := <-self.headSectionC:
			self.handleSection(sec)
			// local var quit channel is linked to sections suicide channel so that
			if sec == nil {
				quit = nil
			} else {
				quit = sec.suicideC
			}

		// periodic check for block hashes or parent block/section
		case <-self.blockHashesRequestTimer:
			self.getBlockHashes()

		// signal from AddBlock that head block of current best peer has been received
		case currentBlock := <-self.currentBlockC:
			self.getCurrentBlock(currentBlock)

		// keep requesting until found or timed out
		case <-self.blocksRequestTimer:
			self.getCurrentBlock(nil)

		// quitting on timeout
		case <-self.suicide:
			self.peerError(self.bp.peers.errors.New(ErrInsufficientChainInfo, "timed out without providing block hashes or head block %x", currentBlockHash))

			self.bp.status.lock.Lock()
			self.bp.status.badPeers[self.id]++
			self.bp.status.lock.Unlock()
			// there is no persistence here, so GC will just take care of cleaning up
			break LOOP

		// signal for peer switch, quit
		case <-switchC:
			var complete = "incomplete "
			if self.idle {
				complete = "complete"
			}
			plog.Debugf("HeadSection: <%s> section with head %s %s... quit request loop due to peer switch", self.id, hex(self.currentBlockHash), complete)
			break LOOP

		// global quit for blockpool
		case <-self.bp.quit:
			break LOOP

		// quit
		case <-quit:
			break LOOP
		}
	}
	if !self.idle {
		self.idle = true
		self.bp.wg.Done()
	}
}
