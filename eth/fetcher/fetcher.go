// Package fetcher contains the block announcement based synchonisation.
package fetcher

import (
	"errors"
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
	fetchTimeout  = 5 * time.Second        // Maximum alloted time to return an explicitly requested block
	maxQueueDist  = 256                    // Maximum allowed distance from the chain head to queue
)

var (
	errTerminated = errors.New("terminated")
)

// hashCheckFn is a callback type for verifying a hash's presence in the local chain.
type hashCheckFn func(common.Hash) bool

// blockRequesterFn is a callback type for sending a block retrieval request.
type blockRequesterFn func([]common.Hash) error

// blockImporterFn is a callback type for trying to inject a block into the local chain.
type blockImporterFn func(peer string, block *types.Block) error

// chainHeightFn is a callback type to retrieve the current chain height.
type chainHeightFn func() uint64

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
	insert chan *inject
	filter chan chan []*types.Block
	quit   chan struct{}

	// Callbacks
	hasBlock    hashCheckFn     // Checks if a block is present in the chain
	importBlock blockImporterFn // Injects a block from an origin peer into the chain
	chainHeight chainHeightFn   // Retrieves the current chain's height
}

// New creates a block fetcher to retrieve blocks based on hash announcements.
func New(hasBlock hashCheckFn, importBlock blockImporterFn, chainHeight chainHeightFn) *Fetcher {
	return &Fetcher{
		notify:      make(chan *announce),
		insert:      make(chan *inject),
		filter:      make(chan chan []*types.Block),
		quit:        make(chan struct{}),
		hasBlock:    hasBlock,
		importBlock: importBlock,
		chainHeight: chainHeight,
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
	case f.insert <- op:
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
	announced := make(map[common.Hash][]*announce)
	fetching := make(map[common.Hash]*announce)
	queued := prque.New()
	fetch := time.NewTimer(0)
	done := make(chan common.Hash)

	// Iterate the block fetching until a quit is requested
	for {
		// Clean up any expired block fetches
		for hash, announce := range fetching {
			if time.Since(announce.time) > fetchTimeout {
				delete(announced, hash)
				delete(fetching, hash)
			}
		}
		// Import any queued blocks that could potentially fit
		height := f.chainHeight()
		for !queued.Empty() {
			// Fetch the next block, and skip if already known
			op := queued.PopItem().(*inject)
			if f.hasBlock(op.block.Hash()) {
				continue
			}
			// If unknown, but too high up the chain, continue later
			if number := op.block.NumberU64(); number > height+1 {
				queued.Push(op, -float32(op.block.NumberU64()))
				break
			}
			// Block may just fit, try to import it
			glog.V(logger.Debug).Infof("Peer %s: importing block %x", op.origin, op.block.Hash().Bytes()[:4])
			go func() {
				defer func() { done <- op.block.Hash() }()

				if err := f.importBlock(op.origin, op.block); err != nil {
					glog.V(logger.Detail).Infof("Peer %s: block %x import failed: %v", op.origin, op.block.Hash().Bytes()[:4], err)
					return
				}
			}()
		}
		// Wait for an outside event to occur
		select {
		case <-f.quit:
			// Fetcher terminating, abort all operations
			return

		case notification := <-f.notify:
			// A block was announced, schedule if it's not yet downloading
			glog.V(logger.Debug).Infof("Peer %s: scheduling %x", notification.origin, notification.hash[:4])
			if _, ok := fetching[notification.hash]; ok {
				break
			}
			if len(announced) == 0 {
				glog.V(logger.Detail).Infof("Scheduling fetch in %v, at %v", arriveTimeout-time.Since(notification.time), notification.time.Add(arriveTimeout))
				fetch.Reset(arriveTimeout - time.Since(notification.time))
			}
			announced[notification.hash] = append(announced[notification.hash], notification)

		case op := <-f.insert:
			// A direct block insertion was requested, try and fill any pending gaps
			queued.Push(op, -float32(op.block.NumberU64()))
			glog.V(logger.Detail).Infof("Peer %s: filled block %x, total %v", op.origin, op.block.Hash().Bytes()[:4], queued.Size())

		case hash := <-done:
			// A pending import finished, remove all traces of the notification
			delete(announced, hash)
			delete(fetching, hash)

		case <-fetch.C:
			// At least one block's timer ran out, check for needing retrieval
			request := make(map[string][]common.Hash)

			for hash, announces := range announced {
				if time.Since(announces[0].time) > arriveTimeout {
					announce := announces[rand.Intn(len(announces))]
					if !f.hasBlock(hash) {
						request[announce.origin] = append(request[announce.origin], hash)
						fetching[hash] = announce
					}
					delete(announced, hash)
				}
			}
			// Send out all block requests
			for peer, hashes := range request {
				glog.V(logger.Debug).Infof("Peer %s: explicitly fetching %d blocks", peer, len(hashes))
				go fetching[hashes[0]].fetch(hashes)
			}
			// Schedule the next fetch if blocks are still pending
			if len(announced) > 0 {
				nearest := time.Now()
				for _, announces := range announced {
					if nearest.After(announces[0].time) {
						nearest = announces[0].time
					}
				}
				glog.V(logger.Detail).Infof("Rescheduling fetch in %v, at %v", arriveTimeout-time.Since(nearest), nearest.Add(arriveTimeout))
				fetch.Reset(arriveTimeout - time.Since(nearest))
			}

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
				if _, ok := fetching[hash]; ok {
					// Discard if already imported by other means
					if !f.hasBlock(hash) {
						explicit = append(explicit, block)
					} else {
						delete(fetching, hash)
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
			height := f.chainHeight()
			for _, block := range explicit {
				// Skip any blocks too far into the future
				if height+maxQueueDist < block.NumberU64() {
					continue
				}
				// Otherwise if the announce is still pending, schedule
				hash := block.Hash()
				if announce := fetching[hash]; announce != nil {
					queued.Push(&inject{origin: announce.origin, block: block}, -float32(block.NumberU64()))
					glog.V(logger.Detail).Infof("Peer %s: scheduled block %x, total %v", announce.origin, hash[:4], queued.Size())
				}
			}
		}
	}
}
