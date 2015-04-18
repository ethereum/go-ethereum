package downloader

import (
	"errors"
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
	maxBlockFetch       = 256              // Amount of max blocks to be fetched per chunk
	minDesiredPeerCount = 3                // Amount of peers desired to start syncing
	blockTtl            = 15 * time.Second // The amount of time it takes for a request to time out
)

var (
	errLowTd       = errors.New("peer's TD is too low")
	errBusy        = errors.New("busy")
	errUnknownPeer = errors.New("peer's unknown or unhealthy")
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
	peer          *peer
	hash          common.Hash
	ignoreInitial bool
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

	glog.V(logger.Detail).Infoln("Register peer", id, "TD =", td)

	// Create a new peer and add it to the list of known peers
	peer := newPeer(id, td, hash, getHashes, getBlocks)
	// add peer to our peer set
	d.peers[id] = peer
	// broadcast new peer
	d.newPeerCh <- peer

	return nil
}

// UnregisterPeer unregister's a peer. This will prevent any action from the specified peer.
func (d *Downloader) UnregisterPeer(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	glog.V(logger.Detail).Infoln("Unregister peer", id)

	delete(d.peers, id)
}

// SynchroniseWithPeer will select the peer and use it for synchronising. If an empty string is given
// it will use the best peer possible and synchronise if it's TD is higher than our own. If any of the
// checks fail an error will be returned. This method is synchronous
func (d *Downloader) SynchroniseWithPeer(id string) (types.Blocks, error) {
	// Check if we're busy
	if d.isBusy() {
		return nil, errBusy
	}

	// Attempt to select a peer. This can either be nothing, which returns, best peer
	// or selected peer. If no peer could be found an error will be returned
	var p *peer
	if len(id) == 0 {
		p = d.peers[id]
		if p == nil {
			return nil, errUnknownPeer
		}
	} else {
		p = d.peers.bestPeer()
	}

	// Make sure our td is lower than the peer's td
	if p.td.Cmp(d.currentTd()) <= 0 || d.hasBlock(p.recentHash) {
		return nil, errLowTd
	}

	// Get the hash from the peer and initiate the downloading progress.
	err := d.getFromPeer(p, p.recentHash, false)
	if err != nil {
		return nil, err
	}

	return d.queue.blocks, nil
}

// Synchronise will synchronise using the best peer.
func (d *Downloader) Synchronise() (types.Blocks, error) {
	return d.SynchroniseWithPeer("")
}

func (d *Downloader) getFromPeer(p *peer, hash common.Hash, ignoreInitial bool) error {
	glog.V(logger.Detail).Infoln("Synchronising with the network using:", p.id)
	// Start the fetcher. This will block the update entirely
	// interupts need to be send to the appropriate channels
	// respectively.
	if err := d.startFetchingHashes(p, hash, ignoreInitial); err != nil {
		// handle error
		glog.V(logger.Debug).Infoln("Error fetching hashes:", err)
		// XXX Reset
		return err
	}

	// Start fetching blocks in paralel. The strategy is simple
	// take any available peers, seserve a chunk for each peer available,
	// let the peer deliver the chunkn and periodically check if a peer
	// has timedout. When done downloading, process blocks.
	if err := d.startFetchingBlocks(p); err != nil {
		glog.V(logger.Debug).Infoln("Error downloading blocks:", err)
		// XXX reset
		return err
	}

	glog.V(logger.Detail).Infoln("Sync completed")

	return nil
}

func (d *Downloader) peerHandler() {
	// itimer is used to determine when to start ignoring `minDesiredPeerCount`
	//itimer := time.NewTicker(5 * time.Second)
	itimer := time.NewTimer(5 * time.Second)
out:
	for {
		select {
		case <-d.newPeerCh:
			itimer.Stop()
			// Meet the `minDesiredPeerCount` before we select our best peer
			if len(d.peers) < minDesiredPeerCount {
				break
			}

			d.selectPeer(d.peers.bestPeer())
		case <-itimer.C:
			// The timer will make sure that the downloader keeps an active state
			// in which it attempts to always check the network for highest td peers
			// Either select the peer or restart the timer if no peers could
			// be selected.
			if peer := d.peers.bestPeer(); peer != nil {
				d.selectPeer(d.peers.bestPeer())
			} else {
				itimer.Reset(5 * time.Second)
			}
		case <-d.quit:
			break out
		}
	}
}

func (d *Downloader) selectPeer(p *peer) {
	// Make sure it's doing neither. Once done we can restart the
	// downloading process if the TD is higher. For now just get on
	// with whatever is going on. This prevents unecessary switching.
	if !d.isBusy() {
		// selected peer must be better than our own
		// XXX we also check the peer's recent hash to make sure we
		// don't have it. Some peers report (i think) incorrect TD.
		if p.td.Cmp(d.currentTd()) <= 0 || d.hasBlock(p.recentHash) {
			return
		}

		glog.V(logger.Detail).Infoln("New peer with highest TD =", p.td)
		d.syncCh <- syncPack{p, p.recentHash, false}
	}

}

func (d *Downloader) update() {
out:
	for {
		select {
		case sync := <-d.syncCh:
			selectedPeer := sync.peer
			glog.V(logger.Detail).Infoln("Synchronising with the network using:", selectedPeer.id)
			// Start the fetcher. This will block the update entirely
			// interupts need to be send to the appropriate channels
			// respectively.
			if err := d.startFetchingHashes(selectedPeer, sync.hash, sync.ignoreInitial); err != nil {
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
func (d *Downloader) startFetchingHashes(p *peer, hash common.Hash, ignoreInitial bool) error {
	atomic.StoreInt32(&d.fetchingHashes, 1)
	defer atomic.StoreInt32(&d.fetchingHashes, 0)

	glog.V(logger.Debug).Infof("Downloading hashes (%x) from %s", hash.Bytes()[:4], p.id)

	start := time.Now()

	// We ignore the initial hash in some cases (e.g. we received a block without it's parent)
	// In such circumstances we don't need to download the block so don't add it to the queue.
	if !ignoreInitial {
		// Add the hash to the queue first
		d.queue.hashPool.Add(hash)
	}
	// Get the first batch of hashes
	p.getHashes(hash)

out:
	for {
		select {
		case hashes := <-d.HashCh:
			var done bool // determines whether we're done fetching hashes (i.e. common hash found)
			hashSet := set.New()
			for _, hash := range hashes {
				if d.hasBlock(hash) {
					glog.V(logger.Debug).Infof("Found common hash %x\n", hash[:4])

					done = true
					break
				}

				hashSet.Add(hash)
			}
			d.queue.put(hashSet)

			// Add hashes to the chunk set
			if len(hashes) == 0 { // Make sure the peer actually gave you something valid
				glog.V(logger.Debug).Infof("Peer (%s) responded with empty hash set\n", p.id)
				d.queue.reset()

				break out
			} else if !done { // Check if we're done fetching
				//fmt.Println("re-fetch. current =", d.queue.hashPool.Size())
				// Get the next set of hashes
				p.getHashes(hashes[len(hashes)-1])
			} else { // we're done
				break out
			}
		}
	}
	glog.V(logger.Detail).Infof("Downloaded hashes (%d). Took %v\n", d.queue.hashPool.Size(), time.Since(start))

	return nil
}

func (d *Downloader) startFetchingBlocks(p *peer) error {
	glog.V(logger.Detail).Infoln("Downloading", d.queue.hashPool.Size(), "blocks")
	atomic.StoreInt32(&d.downloadingBlocks, 1)
	defer atomic.StoreInt32(&d.downloadingBlocks, 0)

	start := time.Now()

	// default ticker for re-fetching blocks everynow and then
	ticker := time.NewTicker(20 * time.Millisecond)
out:
	for {
		select {
		case blockPack := <-d.blockCh:
			// If the peer was previously banned and failed to deliver it's pack
			// in a reasonable time frame, ignore it's message.
			if d.peers[blockPack.peerId] != nil {
				d.peers[blockPack.peerId].promote()
				d.queue.deliver(blockPack.peerId, blockPack.blocks)
				d.peers.setState(blockPack.peerId, idleState)
			}
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
					if time.Since(chunk.itime) > blockTtl {
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
					if peer := d.peers[pid]; peer != nil {
						peer.demote()
					}
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

	peer := d.peers.getPeer(id)
	// if the peer is in our healthy list of peers; update the td
	// and add the block. Otherwise just ignore it
	if peer == nil {
		glog.V(logger.Detail).Infof("Ignored block from bad peer %s\n", id)
		return
	}

	peer.mu.Lock()
	peer.td = td
	peer.recentHash = block.Hash()
	peer.mu.Unlock()
	peer.promote()

	glog.V(logger.Detail).Infoln("Inserting new block from:", id)
	d.queue.addBlock(id, block, td)

	// if neither go ahead to process
	if !d.isBusy() {
		// Check if the parent of the received block is known.
		// If the block is not know, request it otherwise, request.
		phash := block.ParentHash()
		if !d.hasBlock(phash) {
			glog.V(logger.Detail).Infof("Missing parent %x, requires fetching\n", phash.Bytes()[:4])
			d.syncCh <- syncPack{peer, peer.recentHash, true}
		} else {
			d.process()
		}
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
	if len(blocks) == 0 {
		return nil
	}

	glog.V(logger.Debug).Infof("Inserting chain with %d blocks (#%v - #%v)\n", len(blocks), blocks[0].Number(), blocks[len(blocks)-1].Number())

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
					d.syncCh <- syncPack{d.peers.bestPeer(), block.Hash(), true}
					// remove processed blocks
					blocks = blocks[i:]

					break
				}
			}
			break
		} else if err != nil {
			// Reset chain completely. This needs much, much improvement.
			// instead: check all blocks leading down to this block false block and remove it
			blocks = nil
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

func (d *Downloader) isBusy() bool {
	return d.isFetchingHashes() || d.isDownloadingBlocks() || d.isProcessing()
}
