package blockpool

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/errs"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/pow"
)

var (
	// max number of block hashes sent in one request
	blockHashesBatchSize = 256
	// max number of blocks sent in one request
	blockBatchSize = 64
	// interval between two consecutive block checks (and requests)
	blocksRequestInterval = 3 * time.Second
	// level of redundancy in block requests sent
	blocksRequestRepetition = 1
	// interval between two consecutive block hash checks (and requests)
	blockHashesRequestInterval = 3 * time.Second
	// max number of idle iterations, ie., check through a section without new blocks coming in
	blocksRequestMaxIdleRounds = 20
	// timeout interval: max time allowed for peer without sending a block hash
	blockHashesTimeout = 60 * time.Second
	// timeout interval: max time allowed for peer without sending a block
	blocksTimeout = 60 * time.Second
	// timeout interval: max time allowed for best peer to remain idle (not send new block after sync complete)
	idleBestPeerTimeout = 60 * time.Second
	// duration of suspension after peer fatal error during which peer is not allowed to reconnect
	peerSuspensionInterval = 300 * time.Second
	// status is logged every statusUpdateInterval
	statusUpdateInterval = 3 * time.Second
	//
	nodeCacheSize = 1000
)

// blockpool config, values default to constants
type Config struct {
	BlockHashesBatchSize       int
	BlockBatchSize             int
	BlocksRequestRepetition    int
	BlocksRequestMaxIdleRounds int
	NodeCacheSize              int
	BlockHashesRequestInterval time.Duration
	BlocksRequestInterval      time.Duration
	BlockHashesTimeout         time.Duration
	BlocksTimeout              time.Duration
	IdleBestPeerTimeout        time.Duration
	PeerSuspensionInterval     time.Duration
	StatusUpdateInterval       time.Duration
}

// blockpool errors
const (
	ErrInvalidBlock = iota
	ErrInvalidPoW
	ErrInsufficientChainInfo
	ErrIdleTooLong
	ErrIncorrectTD
	ErrUnrequestedBlock
)

// error descriptions
var errorToString = map[int]string{
	ErrInvalidBlock:          "Invalid block",              // fatal
	ErrInvalidPoW:            "Invalid PoW",                // fatal
	ErrInsufficientChainInfo: "Insufficient chain info",    // fatal
	ErrIdleTooLong:           "Idle too long",              // fatal
	ErrIncorrectTD:           "Incorrect Total Difficulty", // should be fatal, not now temporarily
	ErrUnrequestedBlock:      "Unrequested block",
}

// error severity
func severity(code int) logger.LogLevel {
	switch code {
	case ErrIncorrectTD:
		return logger.WarnLevel
	case ErrUnrequestedBlock:
		return logger.WarnLevel
	default:
		return logger.ErrorLevel
	}
}

// init initialises the Config, zero values fall back to constants
func (self *Config) init() {
	if self.BlockHashesBatchSize == 0 {
		self.BlockHashesBatchSize = blockHashesBatchSize
	}
	if self.BlockBatchSize == 0 {
		self.BlockBatchSize = blockBatchSize
	}
	if self.BlocksRequestRepetition == 0 {
		self.BlocksRequestRepetition = blocksRequestRepetition
	}
	if self.BlocksRequestMaxIdleRounds == 0 {
		self.BlocksRequestMaxIdleRounds = blocksRequestMaxIdleRounds
	}
	if self.BlockHashesRequestInterval == 0 {
		self.BlockHashesRequestInterval = blockHashesRequestInterval
	}
	if self.BlocksRequestInterval == 0 {
		self.BlocksRequestInterval = blocksRequestInterval
	}
	if self.BlockHashesTimeout == 0 {
		self.BlockHashesTimeout = blockHashesTimeout
	}
	if self.BlocksTimeout == 0 {
		self.BlocksTimeout = blocksTimeout
	}
	if self.IdleBestPeerTimeout == 0 {
		self.IdleBestPeerTimeout = idleBestPeerTimeout
	}
	if self.PeerSuspensionInterval == 0 {
		self.PeerSuspensionInterval = peerSuspensionInterval
	}
	if self.NodeCacheSize == 0 {
		self.NodeCacheSize = nodeCacheSize
	}
	if self.StatusUpdateInterval == 0 {
		self.StatusUpdateInterval = statusUpdateInterval
	}
}

// node is the basic unit of the internal model of block chain/tree in the blockpool
type node struct {
	lock    sync.RWMutex
	hash    common.Hash
	block   *types.Block
	hashBy  string
	blockBy string
	peers   map[string]bool
	td      *big.Int
}

type index struct {
	int
}

// entry is the struct kept and indexed in the pool
type entry struct {
	node    *node
	section *section
	index   *index
}

type BlockPool struct {
	Config *Config

	// the minimal interface with blockchain manager
	hasBlock    func(hash common.Hash) bool // query if block is known
	insertChain func(types.Blocks) error    // add section to blockchain
	verifyPoW   func(pow.Block) bool        // soft PoW verification
	chainEvents *event.TypeMux              // ethereum eventer for chainEvents

	tdSub event.Subscription // subscription to core.ChainHeadEvent
	td    *big.Int           // our own total difficulty

	pool  map[common.Hash]*entry // the actual blockpool
	peers *peers                 // peers manager in peers.go

	status *status // info about blockpool (UI interface) in status.go

	lock      sync.RWMutex
	chainLock sync.RWMutex
	// alloc-easy pool of hash slices
	hashSlicePool chan []common.Hash

	nodeCache     map[common.Hash]*node
	nodeCacheLock sync.RWMutex
	nodeCacheList []common.Hash

	// waitgroup is used in tests to wait for result-critical routines
	// as well as in determining idle / syncing status
	wg      sync.WaitGroup //
	quit    chan bool      // chan used for quitting parallel routines
	running bool           //
}

// public constructor
// after blockpool returned, config can be set
// BlockPool.Start will call Config.init to set missing values
func New(
	hasBlock func(hash common.Hash) bool,
	insertChain func(types.Blocks) error,
	verifyPoW func(pow.Block) bool,
	chainEvents *event.TypeMux,
	td *big.Int,
) *BlockPool {

	return &BlockPool{
		Config:      &Config{},
		hasBlock:    hasBlock,
		insertChain: insertChain,
		verifyPoW:   verifyPoW,
		chainEvents: chainEvents,
		td:          td,
	}
}

// allows restart
func (self *BlockPool) Start() {
	self.lock.Lock()
	defer self.lock.Unlock()

	if self.running {
		return
	}

	// set missing values
	self.Config.init()

	self.hashSlicePool = make(chan []common.Hash, 150)
	self.nodeCache = make(map[common.Hash]*node)
	self.status = newStatus()
	self.quit = make(chan bool)
	self.pool = make(map[common.Hash]*entry)
	self.running = true

	self.peers = &peers{
		errors: &errs.Errors{
			Package: "Blockpool",
			Errors:  errorToString,
			Level:   severity,
		},
		peers:     make(map[string]*peer),
		blacklist: make(map[string]time.Time),
		status:    self.status,
		bp:        self,
	}

	// subscribe and listen to core.ChainHeadEvent{} for uptodate TD
	self.tdSub = self.chainEvents.Subscribe(core.ChainHeadEvent{})

	// status update interval
	timer := time.NewTicker(self.Config.StatusUpdateInterval)
	go func() {
		for {
			select {
			case <-self.quit:
				return
			case event := <-self.tdSub.Chan():
				if ev, ok := event.(core.ChainHeadEvent); ok {
					td := ev.Block.Td
					var height *big.Int
					if (ev.Block.HeaderHash == common.Hash{}) {
						height = ev.Block.Header().Number
					}
					glog.V(logger.Detail).Infof("ChainHeadEvent: height: %v, td: %v, hash: %s", height, td, hex(ev.Block.Hash()))
					self.setTD(td)
					self.peers.lock.Lock()

					if best := self.peers.best; best != nil {
						// only switch if we strictly go above otherwise we may stall if only
						if td.Cmp(best.td) > 0 {
							self.peers.best = nil
							self.switchPeer(best, nil)
						}
					}
					self.peers.lock.Unlock()
				}
			case <-timer.C:
				glog.V(logger.Detail).Infof("status:\n%v", self.Status())
			}
		}
	}()
	glog.V(logger.Info).Infoln("Blockpool started")
}

func (self *BlockPool) Stop() {
	self.lock.Lock()
	if !self.running {
		self.lock.Unlock()
		return
	}
	self.running = false

	self.lock.Unlock()

	glog.V(logger.Info).Infoln("Stopping...")

	self.tdSub.Unsubscribe()
	close(self.quit)

	self.lock.Lock()
	self.peers = nil
	self.pool = nil
	self.lock.Unlock()

	glog.V(logger.Info).Infoln("Stopped")
}

// Wait blocks until active processes finish
func (self *BlockPool) Wait(t time.Duration) {
	self.lock.Lock()
	if !self.running {
		self.lock.Unlock()
		return
	}
	self.lock.Unlock()

	glog.V(logger.Info).Infoln("Waiting for processes to complete...")
	w := make(chan bool)
	go func() {
		self.wg.Wait()
		close(w)
	}()

	select {
	case <-w:
		glog.V(logger.Info).Infoln("Processes complete")
	case <-time.After(t):
		glog.V(logger.Warn).Infoln("Timeout")
	}
}

/*
AddPeer is called by the eth protocol instance running on the peer after
the status message has been received with total difficulty and current block hash

Called a second time with the same peer id, it is used to update chain info for a peer.
This is used when a new (mined) block message is received.

RemovePeer needs to be called when the peer disconnects.

Peer info is currently not persisted across disconnects (or sessions) except for suspension

*/
func (self *BlockPool) AddPeer(

	td *big.Int, currentBlockHash common.Hash,
	peerId string,
	requestBlockHashes func(common.Hash) error,
	requestBlocks func([]common.Hash) error,
	peerError func(*errs.Error),

) (best bool, suspended bool) {

	return self.peers.addPeer(td, currentBlockHash, peerId, requestBlockHashes, requestBlocks, peerError)
}

// RemovePeer needs to be called when the peer disconnects
func (self *BlockPool) RemovePeer(peerId string) {
	self.peers.removePeer(peerId, true)
}

/*
AddBlockHashes

Entry point for eth protocol to add block hashes received via BlockHashesMsg

Only hashes from the best peer are handled

Initiates further hash requests until a known parent is reached (unless cancelled by a peerSwitch event, i.e., when a better peer becomes best peer)
Launches all block request processes on each chain section

The first argument is an iterator function. Using this block hashes are decoded from the rlp message payload on demand. As a result, AddBlockHashes needs to run synchronously for one peer since the message is discarded if the caller thread returns.
*/
func (self *BlockPool) AddBlockHashes(next func() (common.Hash, bool), peerId string) {

	bestpeer, best := self.peers.getPeer(peerId)
	if !best {
		return
	}
	// bestpeer is still the best peer

	self.wg.Add(1)
	defer func() { self.wg.Done() }()

	self.status.lock.Lock()
	self.status.activePeers[bestpeer.id]++
	self.status.lock.Unlock()

	var n int
	var hash common.Hash
	var ok, headSection, peerswitch bool
	var sec, child, parent *section
	var entry *entry
	var nodes []*node

	hash, ok = next()
	bestpeer.lock.RLock()

	glog.V(logger.Debug).Infof("AddBlockHashes: peer <%s> starting from [%s] (peer head: %s)", peerId, hex(bestpeer.parentHash), hex(bestpeer.currentBlockHash))

	// first check if we are building the head section of a peer's chain
	if bestpeer.parentHash == hash {
		if self.hasBlock(bestpeer.currentBlockHash) {
			bestpeer.lock.RUnlock()
			return
		}
		/*
		 When peer is promoted in switchPeer, a new header section process is launched.
		 Once the head section skeleton is actually created here, it is signaled to the process
		 so that it can quit.
		 In the special case that the node for parent of the head block is found in the blockpool
		 (with or without fetched block), a singleton section containing only the head block node is created.
		*/
		headSection = true
		if entry := self.get(bestpeer.currentBlockHash); entry == nil {
			glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s) head section starting from [%s] ", peerId, hex(bestpeer.currentBlockHash), hex(bestpeer.parentHash))
			// if head block is not yet in the pool, create entry and start node list for section
			self.nodeCacheLock.Lock()
			n := self.findOrCreateNode(bestpeer.currentBlockHash, peerId)
			n.block = bestpeer.currentBlock
			n.blockBy = peerId
			n.td = bestpeer.td
			self.nodeCacheLock.Unlock()

			// nodes is a list of nodes in one section ordered top-bottom (old to young)
			nodes = append(nodes, n)
		} else {
			// otherwise set child section iff found node is the root of a section
			// this is a possible scenario when a singleton head section was created
			// on an earlier occasion when this peer or another with the same block was best peer
			if entry.node == entry.section.bottom {
				child = entry.section
				glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s>: connects to child section root %s", peerId, hex(bestpeer.currentBlockHash))
			}
		}
	} else {
		// otherwise : we are not building the head section of the peer
		glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s) section starting from [%s] ", peerId, hex(bestpeer.currentBlockHash), hex(hash))
	}
	// the switch channel signals peerswitch event
	bestpeer.lock.RUnlock()

	// iterate over hashes coming from peer (first round we have hash set above)
LOOP:
	for ; ok; hash, ok = next() {
		n++
		select {
		case <-self.quit:
			// global quit for blockpool
			return

		case <-bestpeer.switchC:
			// if the peer is demoted, no more hashes read
			glog.V(logger.Detail).Infof("AddBlockHashes: demoted peer <%s> (head: %s)", peerId, hex(bestpeer.currentBlockHash), hex(hash))
			peerswitch = true
			break LOOP
		default:
		}

		// if we reach the blockchain we stop reading further blockhashes
		if self.hasBlock(hash) {
			// check if known block connecting the downloaded chain to our blockchain
			glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s) found block %s in the blockchain", peerId, hex(bestpeer.currentBlockHash), hex(hash))
			if len(nodes) == 1 {
				glog.V(logger.Detail).Infof("AddBlockHashes: singleton section pushed to blockchain peer <%s> (head: %s) found block %s in the blockchain", peerId, hex(bestpeer.currentBlockHash), hex(hash))

				// create new section if needed and push it to the blockchain
				sec = self.newSection(nodes)
				sec.addSectionToBlockChain(bestpeer)
			} else {

				/*
					 not added hash yet but according to peer child section built
					earlier chain connects with blockchain
					this maybe a potential vulnarability
					the root block arrives (or already there but its parenthash was not pointing to known block in the blockchain)
					we start inserting -> error -> remove the entire chain
					instead of punishing this peer
					solution: when switching peers always make sure best peers own head block
					and td together with blockBy are recorded on the node
				*/
				if len(nodes) == 0 && child != nil {
					glog.V(logger.Detail).Infof("AddBlockHashes: child section [%s] pushed to blockchain peer <%s> (head: %s) found block %s in the blockchain", sectionhex(child), peerId, hex(bestpeer.currentBlockHash), hex(hash))

					child.addSectionToBlockChain(bestpeer)
				}
			}
			break LOOP
		}

		// look up node in the pool
		entry = self.get(hash)
		if entry != nil {
			// reached a known chain in the pool
			if entry.node == entry.section.bottom && n == 1 {
				/*
					The first block hash received is an orphan node in the pool

					This also supports clients that (despite the spec) include <from> hash in their
					response to hashes request. Note that by providing <from> we can link sections
					without having to wait for the root block of the child section to arrive, so it allows for superior performance.
				*/
				glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s) found head block [%s] as root of connecting child section [%s] skipping", peerId, hex(bestpeer.currentBlockHash), hex(hash), sectionhex(entry.section))
				// record the entry's chain section as child section
				child = entry.section
				continue LOOP
			}
			// otherwise record entry's chain section as parent connecting it to the pool
			glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s) found block [%s] in section [%s]. Connected to pool.", peerId, hex(bestpeer.currentBlockHash), hex(hash), sectionhex(entry.section))
			parent = entry.section
			break LOOP
		}

		// finally if node for block hash does not exist, create it and append node to section nodes
		self.nodeCacheLock.Lock()
		nodes = append(nodes, self.findOrCreateNode(hash, peerId))
		self.nodeCacheLock.Unlock()
	} //for

	/*
		we got here if
		- run out of hashes (parent = nil) sent by our best peer
		- our peer is demoted (peerswitch = true)
		- reached blockchain or blockpool
		- quitting
	*/
	self.chainLock.Lock()

	glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s): %v nodes in new section", peerId, hex(bestpeer.currentBlockHash), len(nodes))
	/*
	  Handle forks where connecting node is mid-section by splitting section at fork.
	  No splitting needed if connecting node is head of a section.
	*/
	if parent != nil && entry != nil && entry.node != parent.top && len(nodes) > 0 {
		glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s): fork after %s", peerId, hex(bestpeer.currentBlockHash), hex(hash))

		self.splitSection(parent, entry)

		self.status.lock.Lock()
		self.status.values.Forks++
		self.status.lock.Unlock()
	}

	// If new section is created, link it to parent/child sections.
	sec = self.linkSections(nodes, parent, child)

	if sec != nil {
		glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s): section [%s] created", peerId, hex(bestpeer.currentBlockHash), sectionhex(sec))
	}

	self.chainLock.Unlock()

	/*
		If a blockpool node is reached (parent section is not nil),
		activate section (unless our peer is demoted by now).
		This can be the bottom half of a newly split section in case of a fork.

		bestPeer is nil if we got here after our peer got demoted while processing.
		In this case no activation should happen
	*/
	if parent != nil && !peerswitch {
		glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s): parent section [%s]", peerId, hex(bestpeer.currentBlockHash), sectionhex(parent))
		self.activateChain(parent, bestpeer, bestpeer.switchC, nil)
	}

	/*
	  If a new section was created, register section iff head section or no child known
	  Activate it with this peer.
	*/
	if sec != nil {
		// switch on section process (it is paused by switchC)
		if !peerswitch {
			if headSection || child == nil {
				bestpeer.lock.Lock()
				bestpeer.sections = append(bestpeer.sections, sec.top.hash)
				bestpeer.lock.Unlock()
			}
			/*
			  Request another batch of older block hashes for parent section here.
			  But only once, repeating only when the section's root block arrives.
			  Otherwise no way to check if it arrived.
			*/
			bestpeer.requestBlockHashes(sec.bottom.hash)
			glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s): start requesting blocks for section [%s]", peerId, hex(bestpeer.currentBlockHash), sectionhex(sec))
			sec.activate(bestpeer)
		} else {
			glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s) no longer best: delay requesting blocks for section [%s]", peerId, hex(bestpeer.currentBlockHash), sectionhex(sec))
			sec.deactivate()
		}
	}

	// If we are processing peer's head section, signal it to headSection process that it is created.

	if headSection {
		glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s) head section registered on head section process", peerId, hex(bestpeer.currentBlockHash))

		var headSec *section
		switch {
		case sec != nil:
			headSec = sec
		case child != nil:
			headSec = child
		default:
			headSec = parent
		}
		if !peerswitch {
			glog.V(logger.Detail).Infof("AddBlockHashes: peer <%s> (head: %s) head section [%s] created signalled to head section process", peerId, hex(bestpeer.currentBlockHash), sectionhex(headSec))
			bestpeer.headSectionC <- headSec
		}
	}
}

/*
	AddBlock is the entry point for the eth protocol to call when blockMsg is received.

	It has a strict interpretation of the protocol in that if the block received has not been requested, it results in an error.

	At the same time it is opportunistic in that if a requested block may be provided by any peer.

	The received block is checked for PoW. Only the first PoW-valid block for a hash is considered legit.

	If the block received is the head block of the current best peer, signal it to the head section process
*/
func (self *BlockPool) AddBlock(block *types.Block, peerId string) {

	self.status.lock.Lock()
	self.status.activePeers[peerId]++
	self.status.lock.Unlock()

	hash := block.Hash()

	// check if block is already inserted in the blockchain
	if self.hasBlock(hash) {
		return
	}

	sender, _ := self.peers.getPeer(peerId)
	if sender == nil {
		return
	}
	sender.lock.Lock()
	tdFromCurrentHead, currentBlockHash := sender.setChainInfoFromBlock(block)

	entry := self.get(hash)

	/* @zelig !!!
	   requested 5 hashes from both A & B. A responds sooner then B, process blocks. Close section.
	   delayed B sends you block ... UNREQUESTED. Blocked
	     if entry == nil {
	       glog.V(logger.Detail).Infof("AddBlock: unrequested block %s received from peer <%s> (head: %s)", hex(hash), peerId, hex(sender.currentBlockHash))
	       sender.addError(ErrUnrequestedBlock, "%x", hash)

	       self.status.lock.Lock()
	       self.status.badPeers[peerId]++
	       self.status.lock.Unlock()
	       return
	     }
	*/

	var bnode *node
	if entry == nil {
		self.nodeCacheLock.Lock()
		bnode = self.findOrCreateNode(currentBlockHash, peerId)
		self.nodeCacheLock.Unlock()
	} else {
		bnode = entry.node
	}

	bnode.lock.Lock()

	// check if block already received
	if bnode.block != nil {
		glog.V(logger.Detail).Infof("AddBlock: block %s from peer <%s> (head: %s) already sent by <%s> ", hex(hash), peerId, hex(sender.currentBlockHash), bnode.blockBy)
		// register peer on node as source
		if bnode.peers == nil {
			bnode.peers = make(map[string]bool)
		}
		foundBlockCurrentHead, found := bnode.peers[sender.id]
		if !found || foundBlockCurrentHead {
			// if found but not FoundBlockCurrentHead, then no update
			// necessary (||)
			bnode.peers[sender.id] = (currentBlockHash == hash)
			// for those that are false, TD will update their head
			// for those that are true, TD is checked !
			// this is checked at the time of TD calculation in checkTD
		}
		sender.setChainInfoFromNode(bnode)
	} else {
		/*
			@zelig needs discussing
			Viktor: pow check can be delayed in a go routine and therefore cache
			creation is not blocking
				// validate block for PoW
				if !self.verifyPoW(block) {
					glog.V(logger.Warn).Warnf("AddBlock: invalid PoW on block %s from peer  <%s> (head: %s)", hex(hash), peerId, hex(sender.currentBlockHash))
					sender.addError(ErrInvalidPoW, "%x", hash)

					self.status.lock.Lock()
					self.status.badPeers[peerId]++
					self.status.lock.Unlock()

					return
				}
		*/
		bnode.block = block
		bnode.blockBy = peerId
		glog.V(logger.Detail).Infof("AddBlock: set td on node %s from peer <%s> (head: %s) to %v (was %v) ", hex(hash), peerId, hex(sender.currentBlockHash), bnode.td, tdFromCurrentHead)
		bnode.td = tdFromCurrentHead
		self.status.lock.Lock()
		self.status.values.Blocks++
		self.status.values.BlocksInPool++
		self.status.lock.Unlock()
	}
	bnode.lock.Unlock()
	currentBlockC := sender.currentBlockC
	switchC := sender.switchC
	sender.lock.Unlock()

	// this must be called without peerlock.
	// peerlock held can halt the loop and block on select forever
	if tdFromCurrentHead != nil {
		select {
		case currentBlockC <- block:
		case <-switchC: // peer is not best peer
		}
	}
}

func (self *BlockPool) findOrCreateNode(hash common.Hash, peerId string) (bnode *node) {
	bnode, _ = self.nodeCache[hash]
	if bnode == nil {
		bnode = &node{
			hash:   hash,
			hashBy: peerId,
		}
		self.nodeCache[hash] = bnode
		// purge oversize cache
		if len(self.nodeCache) > self.Config.NodeCacheSize {
			delete(self.nodeCache, self.nodeCacheList[0])
			self.nodeCacheList = append(self.nodeCacheList[1:], hash)
		} else {
			self.nodeCacheList = append(self.nodeCacheList, hash)
		}

		self.status.lock.Lock()
		self.status.values.BlockHashes++
		self.status.lock.Unlock()
	}
	return
}

/*
  activateChain iterates down a chain section by section.
  It activates the section process on incomplete sections with peer.
  It relinks orphaned sections with their parent if root block (and its parent hash) is known.
*/
func (self *BlockPool) activateChain(sec *section, p *peer, switchC chan bool, connected map[common.Hash]*section) {

	var i int

LOOP:
	for sec != nil {
		parent := sec.parent
		glog.V(logger.Detail).Infof("activateChain: section [%s] activated by peer <%s>", sectionhex(sec), p.id)
		sec.activate(p)
		if i > 0 && connected != nil {
			connected[sec.top.hash] = sec
		}
		/*
		  Need to relink both complete and incomplete sections
		  An incomplete section could have been blockHashesRequestsComplete before being delinked from its parent.
		*/
		if parent == nil {
			if sec.bottom.block != nil {
				if entry := self.get(sec.bottom.block.ParentHash()); entry != nil {
					parent = entry.section
					glog.V(logger.Detail).Infof("activateChain: [%s]-[%s] link", sectionhex(parent), sectionhex(sec))
					link(parent, sec)
				}
			} else {
				glog.V(logger.Detail).Infof("activateChain: section [%s] activated by peer <%s> has missing root block", sectionhex(sec), p.id)
			}
		}
		sec = parent

		// stop if peer got demoted or global quit
		select {
		case <-switchC:
			break LOOP
		case <-self.quit:
			break LOOP
		default:
		}
	}
}

// check if block's actual TD (calculated after successful insertChain) is identical to TD advertised for peer's head block.
func (self *BlockPool) checkTD(nodes ...*node) {
	for _, n := range nodes {
		// skip check if queued future block
		n.lock.RLock()
		if n.td != nil && !n.block.Queued() {
			glog.V(logger.Detail).Infof("peer td %v =?= block td %v", n.td, n.block.Td)
			// @zelig: Commented out temp untill the rest of the network has been fixed.
			if n.td.Cmp(n.block.Td) != 0 {
				self.peers.peerError(n.blockBy, ErrIncorrectTD, "on block %x peer td %v =?= block td %v", n.hash, n.td, n.block.Td)
				self.status.lock.Lock()
				self.status.badPeers[n.blockBy]++
				self.status.lock.Unlock()
			}
		}
		n.lock.RUnlock()
	}
}

// requestBlocks must run in separate go routine, otherwise
// switchpeer -> activateChain -> activate deadlocks on section process select and peers.lock
func (self *BlockPool) requestBlocks(attempts int, hashes []common.Hash) {
	self.wg.Add(1)
	go func() {
		self.peers.requestBlocks(attempts, hashes)
		self.wg.Done()
	}()
}

// convenience methods to access adjacent sections
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

// accessor and setter for entries in the pool
func (self *BlockPool) get(hash common.Hash) *entry {
	self.lock.RLock()
	defer self.lock.RUnlock()
	return self.pool[hash]
}

func (self *BlockPool) set(hash common.Hash, e *entry) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.pool[hash] = e
}

// accessor and setter for total difficulty
func (self *BlockPool) getTD() *big.Int {
	self.lock.RLock()
	defer self.lock.RUnlock()
	return self.td
}

func (self *BlockPool) setTD(td *big.Int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.td = td
}

func (self *BlockPool) remove(sec *section) {
	// delete node entries from pool index under pool lock
	self.lock.Lock()
	defer self.lock.Unlock()

	for _, node := range sec.nodes {
		delete(self.pool, node.hash)
	}
	if sec.initialised && sec.poolRootIndex != 0 {
		self.status.lock.Lock()
		self.status.values.BlocksInPool -= len(sec.nodes) - sec.missing
		self.status.lock.Unlock()
	}
}

// get/put for optimised allocation similar to sync.Pool
func (self *BlockPool) getHashSlice() (s []common.Hash) {
	select {
	case s = <-self.hashSlicePool:
	default:
		s = make([]common.Hash, self.Config.BlockBatchSize)
	}
	return
}

func (self *BlockPool) putHashSlice(s []common.Hash) {
	if len(s) == self.Config.BlockBatchSize {
		select {
		case self.hashSlicePool <- s:
		default:
		}
	}
}

// pretty prints hash (byte array) with first 4 bytes in hex
func hex(hash common.Hash) (name string) {
	if (hash == common.Hash{}) {
		name = ""
	} else {
		name = fmt.Sprintf("%x", hash[:4])
	}
	return
}

// pretty prints a section using first 4 bytes in hex of bottom and top blockhash of the section
func sectionhex(section *section) (name string) {
	if section == nil {
		name = ""
	} else {
		name = fmt.Sprintf("%x-%x", section.bottom.hash[:4], section.top.hash[:4])
	}
	return
}
