package downloader

import (
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/fatih/set.v0"
)

const maxBlockFetch = 256

type hashFetcherFn func(common.Hash) error
type blockFetcherFn func([]common.Hash) error
type hashCheckFn func(common.Hash) bool
type chainInsertFn func(types.Blocks) error
type hashIterFn func() (common.Hash, error)

// XXX make threadsafe!!!!
type peers map[string]*peer

func (p peers) get(state int) []*peer {
	var peers []*peer
	for _, peer := range p {
		peer.mu.RLock()
		if peer.state == state {
			peers = append(peers, peer)
		}
		peer.mu.RUnlock()
	}

	return peers
}

func (p peers) setState(id string, state int) {
	if peer, exist := p[id]; exist {
		peer.mu.Lock()
		defer peer.mu.Unlock()
		peer.state = state
	}
}

type Downloader struct {
	queue *queue

	hasBlock    hashCheckFn
	insertChain chainInsertFn

	mu    sync.RWMutex
	peers peers

	currentPeer *peer

	fetchingHashes    int32
	downloadingBlocks int32

	newPeerCh    chan *peer
	selectPeerCh chan *peer
	HashCh       chan []common.Hash
	blockCh      chan blockPack
	quit         chan struct{}
}

type blockPack struct {
	peerId string
	blocks []*types.Block
}

func New(hasBlock hashCheckFn, insertChain chainInsertFn) *Downloader {
	downloader := &Downloader{
		queue:        newqueue(),
		peers:        make(peers),
		hasBlock:     hasBlock,
		insertChain:  insertChain,
		newPeerCh:    make(chan *peer, 1),
		selectPeerCh: make(chan *peer, 1),
		HashCh:       make(chan []common.Hash, 1),
		blockCh:      make(chan blockPack, 1),
		quit:         make(chan struct{}),
	}
	go downloader.peerHandler()
	go downloader.update()

	return downloader
}

func (d *Downloader) RegisterPeer(id string, td *big.Int, hash common.Hash, getHashes hashFetcherFn, getBlocks blockFetcherFn) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	glog.V(logger.Detail).Infoln("Register peer", id)

	// Create a new peer and add it to the list of known peers
	peer := newPeer(id, td, hash, getHashes, getBlocks)
	// add peer to our peer set
	d.peers[id] = peer
	// broadcast new peer
	d.newPeerCh <- peer

	return nil
}

func (d *Downloader) UnregisterPeer(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	glog.V(logger.Detail).Infoln("Unregister peer", id)

	delete(d.peers, id)
}

func (d *Downloader) peerHandler() {
	// Fields defined here so we can reduce the amount of locking
	// that needs to be done
	var highestTd = new(big.Int)
out:
	for {
		select {
		case newPeer := <-d.newPeerCh:
			// Check if TD of peer is higher than our current
			if newPeer.td.Cmp(highestTd) > 0 {
				glog.V(logger.Detail).Infoln("New peer with highest TD =", newPeer.td)

				highestTd.Set(newPeer.td)
				// select the peer for downloading
				d.selectPeerCh <- newPeer
			}
		case <-d.quit:
			break out
		}
	}
}

func (d *Downloader) update() {
out:
	for {
		select {
		case selectedPeer := <-d.selectPeerCh:
			// Make sure it's doing neither. Once done we can restart the
			// downloading process if the TD is higher. For now just get on
			// with whatever is going on. This prevents unecessary switching.
			if !(d.isFetchingHashes() || d.isDownloadingBlocks()) {
				glog.V(logger.Detail).Infoln("Selected new peer", selectedPeer.id)
				// Start the fetcher. This will block the update entirely
				// interupts need to be send to the appropriate channels
				// respectively.
				if err := d.startFetchingHashes(selectedPeer); err != nil {
					// handle error
					glog.V(logger.Debug).Infoln("Error fetching hashes:", err)
					// Reset
					break
				}

				// Start fetching blocks in paralel. The strategy is simple
				// take any available peers, seserve a chunk for each peer available,
				// let the peer deliver the chunkn and periodically check if a peer
				// has timedout. When done downloading, process blocks.
				if err := d.startFetchingBlocks(selectedPeer); err != nil {
					glog.V(logger.Debug).Infoln("Error downloading blocks:", err)
					// reset
					break
				}

				// XXX this will move when optimised
				// Sort the blocks by number. This bit needs much improvement. Right now
				// it assumes full honesty form peers (i.e. it's not checked when the blocks
				// link). We should at least check whihc queue match. This code could move
				// to a seperate goroutine where it periodically checks for linked pieces.
				types.BlockBy(types.Number).Sort(d.queue.blocks)
				blocks := d.queue.blocks

				glog.V(logger.Debug).Infoln("Inserting chain with", len(blocks), "blocks")
				// Loop untill we're out of queue
				for len(blocks) != 0 {
					max := int(math.Min(float64(len(blocks)), 256))
					// TODO check for parent error. When there's a parent error we should stop
					// processing and start requesting the `block.hash` so that it's parent and
					// grandparents can be requested and queued.
					d.insertChain(blocks[:max])
					blocks = blocks[max:]
				}
			}
		case <-d.quit:
			break out
		}
	}
}

func (d *Downloader) startFetchingHashes(p *peer) error {
	glog.V(logger.Debug).Infoln("Downloading hashes")

	start := time.Now()

	// Get the first batch of hashes
	p.getHashes(p.recentHash)
	atomic.StoreInt32(&d.fetchingHashes, 1)

out:
	for {
		select {
		case hashes := <-d.HashCh:
			var done bool // determines whether we're done fetching hashes (i.e. common hash found)
			hashSet := set.New()
			for _, hash := range hashes {
				if d.hasBlock(hash) {
					glog.V(logger.Debug).Infof("Found common hash %x\n", hash)

					done = true
					break
				}

				hashSet.Add(hash)
			}
			d.queue.put(hashSet)

			// Add hashes to the chunk set
			// Check if we're done fetching
			if !done {
				//fmt.Println("re-fetch. current =", d.queue.hashPool.Size())
				// Get the next set of hashes
				p.getHashes(hashes[len(hashes)-1])
				atomic.StoreInt32(&d.fetchingHashes, 1)
			} else {
				atomic.StoreInt32(&d.fetchingHashes, 0)
				break out
			}
		}
	}
	glog.V(logger.Detail).Infoln("Download hashes: done. Took", time.Since(start))

	return nil
}

func (d *Downloader) DeliverBlocks(id string, block []*types.Block) {
	d.blockCh <- blockPack{id, block}
}

func (d *Downloader) startFetchingBlocks(p *peer) error {
	glog.V(logger.Detail).Infoln("Downloading", d.queue.hashPool.Size(), "blocks")
	atomic.StoreInt32(&d.downloadingBlocks, 1)

	start := time.Now()

	// default ticker for re-fetching blocks everynow and then
	ticker := time.NewTicker(20 * time.Millisecond)
out:
	for {
		select {
		case blockPack := <-d.blockCh:
			//fmt.Println("get for", blockPack.peerId)

			d.queue.deliver(blockPack.peerId, blockPack.blocks)
			d.peers.setState(blockPack.peerId, idleState)
		case <-ticker.C:
			// If there are unrequested hashes left start fetching
			// from the available peers.
			if d.queue.hashPool.Size() > 0 {
				availablePeers := d.peers.get(idleState)
				for _, peer := range availablePeers {
					// Get a possible chunk. If nil is returned no chunk
					// could be returned due to no hashes available.
					chunk := d.queue.get(peer, maxBlockFetch)
					if chunk != nil {
						//fmt.Println("fetching for", peer.id)
						// Fetch the chunk and check for error. If the peer was somehow
						// already fetching a chunk due to a bug, it will be returned to
						// the queue
						if err := peer.fetch(chunk); err != nil {
							// log for tracing
							glog.V(logger.Debug).Infof("peer %s received double work (state = %v)\n", peer.id, peer.state)
							d.queue.put(chunk.hashes)
						}
					}
				}
				atomic.StoreInt32(&d.downloadingBlocks, 1)
			} else if len(d.queue.fetching) == 0 {
				// Whene there are no more queue and no more `fetching`. We can
				// safely assume we're done. Another part of the process will  check
				// for parent errors and will re-request anything that's missing
				atomic.StoreInt32(&d.downloadingBlocks, 0)
				// Break out so that we can process with processing blocks
				break out
			} else {
				// Check for bad peers. Bad peers may indicate a peer not responding
				// to a `getBlocks` message. A timeout of 5 seconds is set. Peers
				// that badly or poorly behave are removed from the peer set (not banned).
				// Bad peers are excluded from the available peer set and therefor won't be
				// reused. XXX We could re-introduce peers after X time.
				d.queue.mu.Lock()
				var badPeers []string
				for pid, chunk := range d.queue.fetching {
					if time.Since(chunk.itime) > 5*time.Second {
						badPeers = append(badPeers, pid)
						// remove peer as good peer from peer list
						d.UnregisterPeer(pid)
					}
				}
				d.queue.mu.Unlock()

				for _, pid := range badPeers {
					// A nil chunk is delivered so that the chunk's hashes are given
					// back to the queue objects. When hashes are put back in the queue
					// other (decent) peers can pick them up.
					// XXX We could make use of a reputation system here ranking peers
					// in their performance
					// 1) Time for them to respond;
					// 2) Measure their speed;
					// 3) Amount and availability.
					d.queue.deliver(pid, nil)
				}

			}
			//fmt.Println(d.queue.hashPool.Size(), len(d.queue.fetching))
		}
	}

	glog.V(logger.Detail).Infoln("Download blocks: done. Took", time.Since(start))

	return nil
}

func (d *Downloader) isFetchingHashes() bool {
	return atomic.LoadInt32(&d.fetchingHashes) == 1
}

func (d *Downloader) isDownloadingBlocks() bool {
	return atomic.LoadInt32(&d.downloadingBlocks) == 1
}
