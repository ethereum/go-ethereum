package downloader

import (
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/fatih/set.v0"
)

const (
	maxBlockFetch       = 256 // Amount of max blocks to be fetched per chunk
	minDesiredPeerCount = 3   // Amount of peers desired to start syncing
)

type hashCheckFn func(common.Hash) bool
type chainInsertFn func(types.Blocks) error
type hashIterFn func() (common.Hash, error)
type currentTdFn func() *big.Int

type Downloader struct {
	mu    sync.RWMutex
	queue *queue
	peers peers

	// Callbacks
	hasBlock    hashCheckFn
	insertChain chainInsertFn
	currentTd   currentTdFn

	// Status
	fetchingHashes    int32
	downloadingBlocks int32
	processingBlocks  int32

	// Channels
	newPeerCh chan *peer
	syncCh    chan syncPack
	HashCh    chan []common.Hash
	blockCh   chan blockPack
	quit      chan struct{}
}

type blockPack struct {
	peerId string
	blocks []*types.Block
}

type syncPack struct {
	peer *peer
	hash common.Hash
}

func New(hasBlock hashCheckFn, insertChain chainInsertFn, currentTd currentTdFn) *Downloader {
	downloader := &Downloader{
		queue:       newqueue(),
		peers:       make(peers),
		hasBlock:    hasBlock,
		insertChain: insertChain,
		currentTd:   currentTd,
		newPeerCh:   make(chan *peer, 1),
		syncCh:      make(chan syncPack, 1),
		HashCh:      make(chan []common.Hash, 1),
		blockCh:     make(chan blockPack, 1),
		quit:        make(chan struct{}),
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
	// itimer is used to determine when to start ignoring `minDesiredPeerCount`
	itimer := time.NewTicker(5 * time.Second)
out:
	for {
		select {
		case <-d.newPeerCh:
			// Meet the `minDesiredPeerCount` before we select our best peer
			if len(d.peers) < minDesiredPeerCount {
				break
			}
			d.selectPeer(d.peers.bestPeer())
		case <-itimer.C:
			// The timer will make sure that the downloader keeps an active state
			// in which it attempts to always check the network for highest td peers
			d.selectPeer(d.peers.bestPeer())
		case <-d.quit:
			break out
		}
	}
}

func (d *Downloader) selectPeer(p *peer) {
	// Make sure it's doing neither. Once done we can restart the
	// downloading process if the TD is higher. For now just get on
	// with whatever is going on. This prevents unecessary switching.
	if !(d.isFetchingHashes() || d.isDownloadingBlocks() || d.isProcessing()) {
		// selected peer must be better than our own
		// XXX we also check the peer's recent hash to make sure we
		// don't have it. Some peers report (i think) incorrect TD.
		if p.td.Cmp(d.currentTd()) <= 0 || d.hasBlock(p.recentHash) {
			return
		}

		glog.V(logger.Detail).Infoln("New peer with highest TD =", p.td)
		d.syncCh <- syncPack{p, p.recentHash}
	}
}

func (d *Downloader) update() {
out:
	for {
		select {
		case sync := <-d.syncCh:
			selectedPeer := sync.peer
			glog.V(logger.Detail).Infoln("Synchronising with network using:", selectedPeer.id)
			// Start the fetcher. This will block the update entirely
			// interupts need to be send to the appropriate channels
			// respectively.
			if err := d.startFetchingHashes(selectedPeer, sync.hash); err != nil {
				// handle error
				glog.V(logger.Debug).Infoln("Error fetching hashes:", err)
				// XXX Reset
				break
			}

			// Start fetching blocks in paralel. The strategy is simple
			// take any available peers, seserve a chunk for each peer available,
			// let the peer deliver the chunkn and periodically check if a peer
			// has timedout. When done downloading, process blocks.
			if err := d.startFetchingBlocks(selectedPeer); err != nil {
				glog.V(logger.Debug).Infoln("Error downloading blocks:", err)
				// XXX reset
				break
			}

			glog.V(logger.Detail).Infoln("Sync completed")

			d.process()
		case <-d.quit:
			break out
		}
	}
}

// XXX Make synchronous
func (d *Downloader) startFetchingHashes(p *peer, hash common.Hash) error {
	glog.V(logger.Debug).Infoln("Downloading hashes")

	start := time.Now()

	// Get the first batch of hashes
	p.getHashes(hash)
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
					if chunk == nil {
						continue
					}

					//fmt.Println("fetching for", peer.id)
					// XXX make fetch blocking.
					// Fetch the chunk and check for error. If the peer was somehow
					// already fetching a chunk due to a bug, it will be returned to
					// the queue
					if err := peer.fetch(chunk); err != nil {
						// log for tracing
						glog.V(logger.Debug).Infof("peer %s received double work (state = %v)\n", peer.id, peer.state)
						d.queue.put(chunk.hashes)
					}
				}
				atomic.StoreInt32(&d.downloadingBlocks, 1)
			} else if len(d.queue.fetching) == 0 {
				// When there are no more queue and no more `fetching`. We can
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

// Add an (unrequested) block to the downloader. This is usually done through the
// NewBlockMsg by the protocol handler.
func (d *Downloader) AddBlock(id string, block *types.Block, td *big.Int) {
	hash := block.Hash()

	if d.hasBlock(hash) {
		return
	}

	glog.V(logger.Detail).Infoln("Inserting new block from:", id)
	d.queue.addBlock(id, block, td)

	// if the peer is in our healthy list of peers; update the td
	// here is a good chance to add the peer back to the list
	if peer := d.peers.getPeer(id); peer != nil {
		peer.mu.Lock()
		peer.td = td
		peer.recentHash = block.Hash()
		peer.mu.Unlock()
	}

	// if neither go ahead to process
	if !(d.isFetchingHashes() || d.isDownloadingBlocks()) {
		d.process()
	}
}

// Deliver a chunk to the downloader. This is usually done through the BlocksMsg by
// the protocol handler.
func (d *Downloader) DeliverChunk(id string, blocks []*types.Block) {
	d.blockCh <- blockPack{id, blocks}
}

func (d *Downloader) process() error {
	atomic.StoreInt32(&d.processingBlocks, 1)
	defer atomic.StoreInt32(&d.processingBlocks, 0)

	// XXX this will move when optimised
	// Sort the blocks by number. This bit needs much improvement. Right now
	// it assumes full honesty form peers (i.e. it's not checked when the blocks
	// link). We should at least check whihc queue match. This code could move
	// to a seperate goroutine where it periodically checks for linked pieces.
	types.BlockBy(types.Number).Sort(d.queue.blocks)
	blocks := d.queue.blocks

	glog.V(logger.Debug).Infoln("Inserting chain with", len(blocks), "blocks")

	var err error
	// Loop untill we're out of blocks
	for len(blocks) != 0 {
		max := int(math.Min(float64(len(blocks)), 256))
		// TODO check for parent error. When there's a parent error we should stop
		// processing and start requesting the `block.hash` so that it's parent and
		// grandparents can be requested and queued.
		err = d.insertChain(blocks[:max])
		if err != nil && core.IsParentErr(err) {
			glog.V(logger.Debug).Infoln("Aborting process due to missing parent. Fetching hashes")

			// TODO change this. This shite
			for i, block := range blocks[:max] {
				if !d.hasBlock(block.ParentHash()) {
					d.syncCh <- syncPack{d.peers.bestPeer(), block.Hash()}
					// remove processed blocks
					blocks = blocks[i:]

					break
				}
			}
			break
		}
		blocks = blocks[max:]
	}

	// This will allow the GC to remove the in memory blocks
	if len(blocks) == 0 {
		d.queue.blocks = nil
	} else {
		d.queue.blocks = blocks
	}
	return err
}

func (d *Downloader) isFetchingHashes() bool {
	return atomic.LoadInt32(&d.fetchingHashes) == 1
}

func (d *Downloader) isDownloadingBlocks() bool {
	return atomic.LoadInt32(&d.downloadingBlocks) == 1
}

func (d *Downloader) isProcessing() bool {
	return atomic.LoadInt32(&d.processingBlocks) == 1
}
