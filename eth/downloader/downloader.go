package downloader

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/fatih/set.v0"
)

const (
	maxBlockFetch    = 256              // Amount of max blocks to be fetched per chunk
	peerCountTimeout = 12 * time.Second // Amount of time it takes for the peer handler to ignore minDesiredPeerCount
	hashTtl          = 20 * time.Second // The amount of time it takes for a hash request to time out
)

var (
	minDesiredPeerCount = 5                // Amount of peers desired to start syncing
	blockTtl            = 20 * time.Second // The amount of time it takes for a block request to time out

	errLowTd               = errors.New("peer's TD is too low")
	errBusy                = errors.New("busy")
	errUnknownPeer         = errors.New("peer's unknown or unhealthy")
	ErrBadPeer             = errors.New("action from bad peer ignored")
	errNoPeers             = errors.New("no peers to keep download active")
	errPendingQueue        = errors.New("pending items in queue")
	errTimeout             = errors.New("timeout")
	errEmptyHashSet        = errors.New("empty hash set by peer")
	errPeersUnavailable    = errors.New("no peers available or all peers tried for block download process")
	errAlreadyInPool       = errors.New("hash already in pool")
	errBlockNumberOverflow = errors.New("received block which overflows")
)

type hashCheckFn func(common.Hash) bool
type getBlockFn func(common.Hash) *types.Block
type chainInsertFn func(types.Blocks) (int, error)
type hashIterFn func() (common.Hash, error)

type blockPack struct {
	peerId string
	blocks []*types.Block
}

type syncPack struct {
	peer          *peer
	hash          common.Hash
	ignoreInitial bool
}

type hashPack struct {
	peerId string
	hashes []common.Hash
}

type Downloader struct {
	mu         sync.RWMutex
	queue      *queue
	peers      peers
	activePeer string

	// Callbacks
	hasBlock hashCheckFn
	getBlock getBlockFn

	// Status
	fetchingHashes    int32
	downloadingBlocks int32

	// Channels
	newPeerCh chan *peer
	hashCh    chan hashPack
	blockCh   chan blockPack
}

func New(hasBlock hashCheckFn, getBlock getBlockFn) *Downloader {
	downloader := &Downloader{
		queue:     newqueue(),
		peers:     make(peers),
		hasBlock:  hasBlock,
		getBlock:  getBlock,
		newPeerCh: make(chan *peer, 1),
		hashCh:    make(chan hashPack, 1),
		blockCh:   make(chan blockPack, 1),
	}

	return downloader
}

func (d *Downloader) Stats() (current int, max int) {
	return d.queue.blockHashes.Size(), d.queue.fetchPool.Size() + d.queue.hashPool.Size()
}

func (d *Downloader) RegisterPeer(id string, hash common.Hash, getHashes hashFetcherFn, getBlocks blockFetcherFn) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	glog.V(logger.Detail).Infoln("Register peer", id)

	// Create a new peer and add it to the list of known peers
	peer := newPeer(id, hash, getHashes, getBlocks)
	// add peer to our peer set
	d.peers[id] = peer
	// broadcast new peer

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
func (d *Downloader) Synchronise(id string, hash common.Hash) error {
	// Make sure it's doing neither. Once done we can restart the
	// downloading process if the TD is higher. For now just get on
	// with whatever is going on. This prevents unecessary switching.
	if d.isBusy() {
		return errBusy
	}

	// When a synchronisation attempt is made while the queue stil
	// contains items we abort the sync attempt
	if d.queue.size() > 0 {
		return errPendingQueue
	}

	// Fetch the peer using the id or throw an error if the peer couldn't be found
	p := d.peers[id]
	if p == nil {
		return errUnknownPeer
	}

	// Get the hash from the peer and initiate the downloading progress.
	err := d.getFromPeer(p, hash, false)
	if err != nil {
		return err
	}

	return nil
}

// Done lets the downloader know that whatever previous hashes were taken
// are processed. If the block count reaches zero and done is called
// we reset the queue for the next batch of incoming hashes and blocks.
func (d *Downloader) Done() {
	d.queue.mu.Lock()
	defer d.queue.mu.Unlock()

	if len(d.queue.blocks) == 0 {
		d.queue.resetNoTS()
	}
}

// TakeBlocks takes blocks from the queue and yields them to the blockTaker handler
// it's possible it yields no blocks
func (d *Downloader) TakeBlocks() types.Blocks {
	d.queue.mu.Lock()
	defer d.queue.mu.Unlock()

	var blocks types.Blocks
	if len(d.queue.blocks) > 0 {
		// Make sure the parent hash is known
		if d.queue.blocks[0] != nil && !d.hasBlock(d.queue.blocks[0].ParentHash()) {
			return nil
		}

		for _, block := range d.queue.blocks {
			if block == nil {
				break
			}

			blocks = append(blocks, block)
		}
		d.queue.blockOffset += len(blocks)
		// delete the blocks from the slice and let them be garbage collected
		// without this slice trick the blocks would stay in memory until nil
		// would be assigned to d.queue.blocks
		copy(d.queue.blocks, d.queue.blocks[len(blocks):])
		for k, n := len(d.queue.blocks)-len(blocks), len(d.queue.blocks); k < n; k++ {
			d.queue.blocks[k] = nil
		}
		d.queue.blocks = d.queue.blocks[:len(d.queue.blocks)-len(blocks)]

		//d.queue.blocks = d.queue.blocks[len(blocks):]
		if len(d.queue.blocks) == 0 {
			d.queue.blocks = nil
		}

	}

	return blocks
}

func (d *Downloader) Has(hash common.Hash) bool {
	return d.queue.has(hash)
}

func (d *Downloader) getFromPeer(p *peer, hash common.Hash, ignoreInitial bool) (err error) {
	d.activePeer = p.id
	defer func() {
		// reset on error
		if err != nil {
			d.queue.reset()
		}
	}()

	glog.V(logger.Detail).Infoln("Synchronising with the network using:", p.id)
	// Start the fetcher. This will block the update entirely
	// interupts need to be send to the appropriate channels
	// respectively.
	if err = d.startFetchingHashes(p, hash, ignoreInitial); err != nil {
		return err
	}

	// Start fetching blocks in paralel. The strategy is simple
	// take any available peers, seserve a chunk for each peer available,
	// let the peer deliver the chunkn and periodically check if a peer
	// has timedout.
	if err = d.startFetchingBlocks(p); err != nil {
		return err
	}

	glog.V(logger.Detail).Infoln("Sync completed")

	return nil
}

// XXX Make synchronous
func (d *Downloader) startFetchingHashes(p *peer, h common.Hash, ignoreInitial bool) error {
	atomic.StoreInt32(&d.fetchingHashes, 1)
	defer atomic.StoreInt32(&d.fetchingHashes, 0)

	if d.queue.has(h) {
		return errAlreadyInPool
	}

	glog.V(logger.Debug).Infof("Downloading hashes (%x) from %s", h[:4], p.id)

	start := time.Now()

	// We ignore the initial hash in some cases (e.g. we received a block without it's parent)
	// In such circumstances we don't need to download the block so don't add it to the queue.
	if !ignoreInitial {
		// Add the hash to the queue first
		d.queue.hashPool.Add(h)
	}
	// Get the first batch of hashes
	p.getHashes(h)

	var (
		failureResponseTimer = time.NewTimer(hashTtl)
		attemptedPeers       = make(map[string]bool) // attempted peers will help with retries
		activePeer           = p                     // active peer will help determine the current active peer
		hash                 common.Hash             // common and last hash
	)
	attemptedPeers[p.id] = true

out:
	for {
		select {
		case hashPack := <-d.hashCh:
			// make sure the active peer is giving us the hashes
			if hashPack.peerId != activePeer.id {
				glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)\n", hashPack.peerId)
				break
			}

			failureResponseTimer.Reset(hashTtl)

			var (
				hashes = hashPack.hashes
				done   bool // determines whether we're done fetching hashes (i.e. common hash found)
			)
			hashSet := set.New()
			for _, hash = range hashes {
				if d.hasBlock(hash) || d.queue.blockHashes.Has(hash) {
					glog.V(logger.Debug).Infof("Found common hash %x\n", hash[:4])

					done = true
					break
				}

				hashSet.Add(hash)
			}
			d.queue.put(hashSet)

			// Add hashes to the chunk set
			if len(hashes) == 0 { // Make sure the peer actually gave you something valid
				glog.V(logger.Debug).Infof("Peer (%s) responded with empty hash set\n", activePeer.id)
				d.queue.reset()

				return errEmptyHashSet
			} else if !done { // Check if we're done fetching
				// Get the next set of hashes
				activePeer.getHashes(hash)
			} else { // we're done
				// The offset of the queue is determined by the highest known block
				var offset int
				if block := d.getBlock(hash); block != nil {
					offset = int(block.NumberU64() + 1)
				}
				// allocate proper size for the queueue
				d.queue.alloc(offset, d.queue.hashPool.Size())

				break out
			}
		case <-failureResponseTimer.C:
			glog.V(logger.Debug).Infof("Peer (%s) didn't respond in time for hash request\n", p.id)

			var p *peer // p will be set if a peer can be found
			// Attempt to find a new peer by checking inclusion of peers best hash in our
			// already fetched hash list. This can't guarantee 100% correctness but does
			// a fair job. This is always either correct or false incorrect.
			for id, peer := range d.peers {
				if d.queue.hashPool.Has(peer.recentHash) && !attemptedPeers[id] {
					p = peer
					break
				}
			}

			// if all peers have been tried, abort the process entirely or if the hash is
			// the zero hash.
			if p == nil || (hash == common.Hash{}) {
				d.queue.reset()
				return errTimeout
			}

			// set p to the active peer. this will invalidate any hashes that may be returned
			// by our previous (delayed) peer.
			activePeer = p
			p.getHashes(hash)
			glog.V(logger.Debug).Infof("Hash fetching switched to new peer(%s)\n", p.id)
		}
	}
	glog.V(logger.Detail).Infof("Downloaded hashes (%d) in %v\n", d.queue.hashPool.Size(), time.Since(start))

	return nil
}

func (d *Downloader) startFetchingBlocks(p *peer) error {
	glog.V(logger.Detail).Infoln("Downloading", d.queue.hashPool.Size(), "block(s)")
	atomic.StoreInt32(&d.downloadingBlocks, 1)
	defer atomic.StoreInt32(&d.downloadingBlocks, 0)
	// Defer the peer reset. This will empty the peer requested set
	// and makes sure there are no lingering peers with an incorrect
	// state
	defer d.peers.reset()

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
				err := d.queue.deliver(blockPack.peerId, blockPack.blocks)
				if err != nil {
					glog.V(logger.Debug).Infof("deliver failed for peer %s: %v\n", blockPack.peerId, err)
					// FIXME d.UnregisterPeer(blockPack.peerId)
					break
				}

				if glog.V(logger.Debug) {
					glog.Infof("adding %d blocks from: %s\n", len(blockPack.blocks), blockPack.peerId)
				}
				d.peers[blockPack.peerId].promote()
				d.peers.setState(blockPack.peerId, idleState)
			}
		case <-ticker.C:
			// after removing bad peers make sure we actually have suffucient peer left to keep downlading
			if len(d.peers) == 0 {
				d.queue.reset()

				return errNoPeers
			}

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

				// make sure that we have peers available for fetching. If all peers have been tried
				// and all failed throw an error
				if len(d.queue.fetching) == 0 {
					d.queue.reset()

					return fmt.Errorf("%v peers avaialable = %d. total peers = %d. hashes needed = %d", errPeersUnavailable, len(availablePeers), len(d.peers), d.queue.hashPool.Size())
				}

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
						// FIXME d.UnregisterPeer(pid)
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
						peer.reset()
					}
				}

			}
		}
	}

	glog.V(logger.Detail).Infoln("Downloaded block(s) in", time.Since(start))

	return nil
}

// Deliver a chunk to the downloader. This is usually done through the BlocksMsg by
// the protocol handler.
func (d *Downloader) DeliverChunk(id string, blocks []*types.Block) {
	d.blockCh <- blockPack{id, blocks}
}

func (d *Downloader) AddHashes(id string, hashes []common.Hash) error {
	// make sure that the hashes that are being added are actually from the peer
	// that's the current active peer. hashes that have been received from other
	// peers are dropped and ignored.
	if d.activePeer != id {
		return fmt.Errorf("received hashes from %s while active peer is %s", id, d.activePeer)
	}

	if glog.V(logger.Detail) && len(hashes) != 0 {
		from, to := hashes[0], hashes[len(hashes)-1]
		glog.Infof("adding %d (T=%d) hashes [ %x / %x ] from: %s\n", len(hashes), d.queue.hashPool.Size(), from[:4], to[:4], id)
	}

	d.hashCh <- hashPack{id, hashes}

	return nil
}

func (d *Downloader) isFetchingHashes() bool {
	return atomic.LoadInt32(&d.fetchingHashes) == 1
}

func (d *Downloader) isDownloadingBlocks() bool {
	return atomic.LoadInt32(&d.downloadingBlocks) == 1
}

func (d *Downloader) isBusy() bool {
	return d.isFetchingHashes() || d.isDownloadingBlocks()
}

func (d *Downloader) IsBusy() bool {
	return d.isBusy()
}
