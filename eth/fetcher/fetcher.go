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
)

const (
	arriveTimeout = 500 * time.Millisecond // Time allowance before an announced block is explicitly requested
	fetchTimeout  = 5 * time.Second        // Maximum alloted time to return an explicitly requested block
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

// announce is the hash notification of the availability of a new block in the
// network.
type announce struct {
	hash common.Hash // Hash of the block being announced
	time time.Time   // Timestamp of the announcement

	origin string           // Identifier of the peer originating the notification
	fetch  blockRequesterFn // Fetcher function to retrieve
}

// Fetcher is responsible for accumulating block announcements from various peers
// and scheduling them for retrieval.
type Fetcher struct {
	// Various event channels
	notify chan *announce
	filter chan chan []*types.Block
	quit   chan struct{}

	// Callbacks
	hasBlock    hashCheckFn     // Checks if a block is present in the chain
	importBlock blockImporterFn // Injects a block from an origin peer into the chain
}

// New creates a block fetcher to retrieve blocks based on hash announcements.
func New(hasBlock hashCheckFn, importBlock blockImporterFn) *Fetcher {
	return &Fetcher{
		notify:      make(chan *announce),
		filter:      make(chan chan []*types.Block),
		quit:        make(chan struct{}),
		hasBlock:    hasBlock,
		importBlock: importBlock,
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
				fetch.Reset(arriveTimeout)
			}
			announced[notification.hash] = append(announced[notification.hash], notification)

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
					if nearest.Before(announces[0].time) {
						nearest = announces[0].time
					}
				}
				fetch.Reset(arriveTimeout + time.Since(nearest))
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
			// Create a closure with the retrieved blocks and origin peers
			peers := make([]string, 0, len(explicit))
			blocks = make([]*types.Block, 0, len(explicit))
			for _, block := range explicit {
				hash := block.Hash()
				if announce := fetching[hash]; announce != nil {
					// Drop the block if it surely cannot fit
					if f.hasBlock(hash) || !f.hasBlock(block.ParentHash()) {
						// delete(fetching, hash) // if we drop, it will re-fetch it, wait for timeout?
						continue
					}
					// Otherwise accumulate for import
					peers = append(peers, announce.origin)
					blocks = append(blocks, block)
				}
			}
			// If any explicit fetches were replied to, import them
			if count := len(blocks); count > 0 {
				glog.V(logger.Debug).Infof("Importing %d explicitly fetched blocks", len(blocks))
				go func() {
					// Make sure all hashes are cleaned up
					for _, block := range blocks {
						hash := block.Hash()
						defer func() { done <- hash }()
					}
					// Try and actually import the blocks
					for i := 0; i < len(blocks); i++ {
						if err := f.importBlock(peers[i], blocks[i]); err != nil {
							glog.V(logger.Detail).Infof("Failed to import explicitly fetched block: %v", err)
							return
						}
					}
				}()
			}
		}
	}
}
