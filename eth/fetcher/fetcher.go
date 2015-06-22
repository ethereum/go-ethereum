// Package fetcher contains the block announcement based synchonisation.
package fetcher

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"
)

const (
	arriveTimeout = 500 * time.Millisecond // Time allowance before an announced block is explicitly requested
	gatherSlack   = 100 * time.Millisecond // Interval used to collate almost-expired announces with fetches
	fetchTimeout  = 5 * time.Second        // Maximum alloted time to return an explicitly requested block
	maxUncleDist  = 7                      // Maximum allowed backward distance from the chain head
	maxQueueDist  = 32                     // Maximum allowed distance from the chain head to queue
	hashLimit     = 256                    // Maximum number of unique blocks a peer may have announced
	blockLimit    = 64                     // Maximum number of unique blocks a per may have delivered
)

var (
	errTerminated = errors.New("terminated")
)

// blockRetrievalFn is a callback type for retrieving a block from the local chain.
type blockRetrievalFn func(common.Hash) *types.Block

// blockRequesterFn is a callback type for sending a block retrieval request.
type blockRequesterFn func([]common.Hash) error

// blockValidatorFn is a callback type to verify a block's header for fast propagation.
type blockValidatorFn func(block *types.Block, parent *types.Block) error

// blockBroadcasterFn is a callback type for broadcasting a block to connected peers.
type blockBroadcasterFn func(block *types.Block, propagate bool)

// chainHeightFn is a callback type to retrieve the current chain height.
type chainHeightFn func() uint64

// chainInsertFn is a callback type to insert a batch of blocks into the local chain.
type chainInsertFn func(types.Blocks) (int, error)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

// announce is the hash notification of the availability of a new block in the
// network.
type announce struct {
	hash common.Hash // Hash of the block being announced
	time time.Time   // Timestamp of the announcement

	origin string           // Identifier of the peer originating the notification
	fetch  blockRequesterFn // Fetcher function to retrieve
}

// inject represents a schedules import operation.
type inject struct {
	origin string
	block  *types.Block
}

// Fetcher is responsible for accumulating block announcements from various peers
// and scheduling them for retrieval.
type Fetcher struct {
	// Various event channels
	notify chan *announce
	inject chan *inject
	filter chan chan []*types.Block
	done   chan common.Hash
	quit   chan struct{}

	// Announce states
	announces map[string]int              // Per peer announce counts to prevent memory exhaustion
	announced map[common.Hash][]*announce // Announced blocks, scheduled for fetching
	fetching  map[common.Hash]*announce   // Announced blocks, currently fetching

	// Block cache
	queue  *prque.Prque            // Queue containing the import operations (block number sorted)
	queues map[string]int          // Per peer block counts to prevent memory exhaustion
	queued map[common.Hash]*inject // Set of already queued blocks (to dedup imports)

	// Callbacks
	getBlock       blockRetrievalFn   // Retrieves a block from the local chain
	validateBlock  blockValidatorFn   // Checks if a block's headers have a valid proof of work
	broadcastBlock blockBroadcasterFn // Broadcasts a block to connected peers
	chainHeight    chainHeightFn      // Retrieves the current chain's height
	insertChain    chainInsertFn      // Injects a batch of blocks into the chain
	dropPeer       peerDropFn         // Drops a peer for misbehaving

	// Testing hooks
	fetchingHook func([]common.Hash) // Method to call upon starting a block fetch
	importedHook func(*types.Block)  // Method to call upon successful block import
}

// New creates a block fetcher to retrieve blocks based on hash announcements.
func New(getBlock blockRetrievalFn, validateBlock blockValidatorFn, broadcastBlock blockBroadcasterFn, chainHeight chainHeightFn, insertChain chainInsertFn, dropPeer peerDropFn) *Fetcher {
	return &Fetcher{
		notify:         make(chan *announce),
		inject:         make(chan *inject),
		filter:         make(chan chan []*types.Block),
		done:           make(chan common.Hash),
		quit:           make(chan struct{}),
		announces:      make(map[string]int),
		announced:      make(map[common.Hash][]*announce),
		fetching:       make(map[common.Hash]*announce),
		queue:          prque.New(),
		queues:         make(map[string]int),
		queued:         make(map[common.Hash]*inject),
		getBlock:       getBlock,
		validateBlock:  validateBlock,
		broadcastBlock: broadcastBlock,
		chainHeight:    chainHeight,
		insertChain:    insertChain,
		dropPeer:       dropPeer,
	}
}

// Start boots up the announcement based synchoniser, accepting and processing
// hash notifications and block fetches until termination requested.
func (f *Fetcher) Start() {
	go f.loop()
}

// Stop terminates the announcement based synchroniser, canceling all pending
// operations.
func (f *Fetcher) Stop() {
	close(f.quit)
}

// Notify announces the fetcher of the potential availability of a new block in
// the network.
func (f *Fetcher) Notify(peer string, hash common.Hash, time time.Time, fetcher blockRequesterFn) error {
	block := &announce{
		hash:   hash,
		time:   time,
		origin: peer,
		fetch:  fetcher,
	}
	select {
	case f.notify <- block:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Enqueue tries to fill gaps the the fetcher's future import queue.
func (f *Fetcher) Enqueue(peer string, block *types.Block) error {
	op := &inject{
		origin: peer,
		block:  block,
	}
	select {
	case f.inject <- op:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Filter extracts all the blocks that were explicitly requested by the fetcher,
// returning those that should be handled differently.
func (f *Fetcher) Filter(blocks types.Blocks) types.Blocks {
	// Send the filter channel to the fetcher
	filter := make(chan []*types.Block)

	select {
	case f.filter <- filter:
	case <-f.quit:
		return nil
	}
	// Request the filtering of the block list
	select {
	case filter <- blocks:
	case <-f.quit:
		return nil
	}
	// Retrieve the blocks remaining after filtering
	select {
	case blocks := <-filter:
		return blocks
	case <-f.quit:
		return nil
	}
}

// Loop is the main fetcher loop, checking and processing various notification
// events.
func (f *Fetcher) loop() {
	// Iterate the block fetching until a quit is requested
	fetch := time.NewTimer(0)
	for {
		// Clean up any expired block fetches
		for hash, announce := range f.fetching {
			if time.Since(announce.time) > fetchTimeout {
				f.forgetHash(hash)
			}
		}
		// Import any queued blocks that could potentially fit
		height := f.chainHeight()
		for !f.queue.Empty() {
			op := f.queue.PopItem().(*inject)

			// If too high up the chain or phase, continue later
			number := op.block.NumberU64()
			if number > height+1 {
				f.queue.Push(op, -float32(op.block.NumberU64()))
				break
			}
			// Otherwise if fresh and still unknown, try and import
			hash := op.block.Hash()
			if number+maxUncleDist < height || f.getBlock(hash) != nil {
				f.forgetBlock(hash)
				continue
			}
			f.insert(op.origin, op.block)
		}
		// Wait for an outside event to occur
		select {
		case <-f.quit:
			// Fetcher terminating, abort all operations
			return

		case notification := <-f.notify:
			// A block was announced, make sure the peer isn't DOSing us
			count := f.announces[notification.origin] + 1
			if count > hashLimit {
				glog.V(logger.Debug).Infof("Peer %s: exceeded outstanding announces (%d)", notification.origin, hashLimit)
				break
			}
			// All is well, schedule the announce if block's not yet downloading
			if _, ok := f.fetching[notification.hash]; ok {
				break
			}
			f.announces[notification.origin] = count
			f.announced[notification.hash] = append(f.announced[notification.hash], notification)
			if len(f.announced) == 1 {
				f.reschedule(fetch)
			}

		case op := <-f.inject:
			// A direct block insertion was requested, try and fill any pending gaps
			f.enqueue(op.origin, op.block)

		case hash := <-f.done:
			// A pending import finished, remove all traces of the notification
			f.forgetHash(hash)
			f.forgetBlock(hash)

		case <-fetch.C:
			// At least one block's timer ran out, check for needing retrieval
			request := make(map[string][]common.Hash)

			for hash, announces := range f.announced {
				if time.Since(announces[0].time) > arriveTimeout-gatherSlack {
					// Pick a random peer to retrieve from, reset all others
					announce := announces[rand.Intn(len(announces))]
					f.forgetHash(hash)

					// If the block still didn't arrive, queue for fetching
					if f.getBlock(hash) == nil {
						request[announce.origin] = append(request[announce.origin], hash)
						f.fetching[hash] = announce
					}
				}
			}
			// Send out all block requests
			for peer, hashes := range request {
				if glog.V(logger.Detail) && len(hashes) > 0 {
					list := "["
					for _, hash := range hashes {
						list += fmt.Sprintf("%x, ", hash[:4])
					}
					list = list[:len(list)-2] + "]"

					glog.V(logger.Detail).Infof("Peer %s: fetching %s", peer, list)
				}
				hashes := hashes // closure!
				go func() {
					if f.fetchingHook != nil {
						f.fetchingHook(hashes)
					}
					f.fetching[hashes[0]].fetch(hashes)
				}()
			}
			// Schedule the next fetch if blocks are still pending
			f.reschedule(fetch)

		case filter := <-f.filter:
			// Blocks arrived, extract any explicit fetches, return all else
			var blocks types.Blocks
			select {
			case blocks = <-filter:
			case <-f.quit:
				return
			}

			explicit, download := []*types.Block{}, []*types.Block{}
			for _, block := range blocks {
				hash := block.Hash()

				// Filter explicitly requested blocks from hash announcements
				if _, ok := f.fetching[hash]; ok {
					// Discard if already imported by other means
					if f.getBlock(hash) == nil {
						explicit = append(explicit, block)
					} else {
						f.forgetHash(hash)
					}
				} else {
					download = append(download, block)
				}
			}

			select {
			case filter <- download:
			case <-f.quit:
				return
			}
			// Schedule the retrieved blocks for ordered import
			for _, block := range explicit {
				if announce := f.fetching[block.Hash()]; announce != nil {
					f.enqueue(announce.origin, block)
				}
			}
		}
	}
}

// reschedule resets the specified fetch timer to the next announce timeout.
func (f *Fetcher) reschedule(fetch *time.Timer) {
	// Short circuit if no blocks are announced
	if len(f.announced) == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	for _, announces := range f.announced {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	fetch.Reset(arriveTimeout - time.Since(earliest))
}

// enqueue schedules a new future import operation, if the block to be imported
// has not yet been seen.
func (f *Fetcher) enqueue(peer string, block *types.Block) {
	hash := block.Hash()

	// Ensure the peer isn't DOSing us
	count := f.queues[peer] + 1
	if count > blockLimit {
		glog.V(logger.Debug).Infof("Peer %s: discarded block #%d [%x], exceeded allowance (%d)", peer, block.NumberU64(), hash.Bytes()[:4], blockLimit)
		return
	}
	// Discard any past or too distant blocks
	if dist := int64(block.NumberU64()) - int64(f.chainHeight()); dist < -maxUncleDist || dist > maxQueueDist {
		glog.V(logger.Debug).Infof("Peer %s: discarded block #%d [%x], distance %d", peer, block.NumberU64(), hash.Bytes()[:4], dist)
		return
	}
	// Schedule the block for future importing
	if _, ok := f.queued[hash]; !ok {
		op := &inject{
			origin: peer,
			block:  block,
		}
		f.queues[peer] = count
		f.queued[hash] = op
		f.queue.Push(op, -float32(block.NumberU64()))

		if glog.V(logger.Debug) {
			glog.Infof("Peer %s: queued block #%d [%x], total %v", peer, block.NumberU64(), hash.Bytes()[:4], f.queue.Size())
		}
	}
}

// insert spawns a new goroutine to run a block insertion into the chain. If the
// block's number is at the same height as the current import phase, if updates
// the phase states accordingly.
func (f *Fetcher) insert(peer string, block *types.Block) {
	hash := block.Hash()

	// Run the import on a new thread
	glog.V(logger.Debug).Infof("Peer %s: importing block #%d [%x]", peer, block.NumberU64(), hash[:4])
	go func() {
		defer func() { f.done <- hash }()

		// If the parent's unknown, abort insertion
		parent := f.getBlock(block.ParentHash())
		if parent == nil {
			return
		}
		// Quickly validate the header and propagate the block if it passes
		if err := f.validateBlock(block, parent); err != nil {
			glog.V(logger.Debug).Infof("Peer %s: block #%d [%x] verification failed: %v", peer, block.NumberU64(), hash[:4], err)
			f.dropPeer(peer)
			return
		}
		go f.broadcastBlock(block, true)

		// Run the actual import and log any issues
		if _, err := f.insertChain(types.Blocks{block}); err != nil {
			glog.V(logger.Warn).Infof("Peer %s: block #%d [%x] import failed: %v", peer, block.NumberU64(), hash[:4], err)
			return
		}
		// If import succeeded, broadcast the block
		go f.broadcastBlock(block, false)

		// Invoke the testing hook if needed
		if f.importedHook != nil {
			f.importedHook(block)
		}
	}()
}

// forgetHash removes all traces of a block announcement from the fetcher's
// internal state.
func (f *Fetcher) forgetHash(hash common.Hash) {
	// Remove all pending announces and decrement DOS counters
	for _, announce := range f.announced[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.announced, hash)

	// Remove any pending fetches and decrement the DOS counters
	if announce := f.fetching[hash]; announce != nil {
		f.announces[announce.origin]--
		if f.announces[announce.origin] == 0 {
			delete(f.announces, announce.origin)
		}
		delete(f.fetching, hash)
	}
}

// forgetBlock removes all traces of a queued block frmo the fetcher's internal
// state.
func (f *Fetcher) forgetBlock(hash common.Hash) {
	if insert := f.queued[hash]; insert != nil {
		f.queues[insert.origin]--
		if f.queues[insert.origin] == 0 {
			delete(f.queues, insert.origin)
		}
		delete(f.queued, hash)
	}
}
