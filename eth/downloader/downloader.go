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
)

const (
	maxBlockFetch    = 128              // Amount of max blocks to be fetched per chunk
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
		queue:     newQueue(),
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
	return d.queue.Size()
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

// UnregisterPeer unregisters a peer. This will prevent any action from the specified peer.
func (d *Downloader) UnregisterPeer(id string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	glog.V(logger.Detail).Infoln("Unregister peer", id)

	delete(d.peers, id)
}

// SynchroniseWithPeer will select the peer and use it for synchronizing. If an empty string is given
// it will use the best peer possible and synchronize if it's TD is higher than our own. If any of the
// checks fail an error will be returned. This method is synchronous
func (d *Downloader) Synchronise(id string, hash common.Hash) error {
	// Make sure it's doing neither. Once done we can restart the
	// downloading process if the TD is higher. For now just get on
	// with whatever is going on. This prevents unnecessary switching.
	if d.isBusy() {
		return errBusy
	}

	// When a synchronization attempt is made while the queue still
	// contains items we abort the sync attempt
	if done, pend := d.queue.Size(); done+pend > 0 {
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
	d.queue.Done()
}

// TakeBlocks takes blocks from the queue and yields them to the blockTaker handler
// it's possible it yields no blocks
func (d *Downloader) TakeBlocks() types.Blocks {
	// Check that there are blocks available and its parents are known
	head := d.queue.GetHeadBlock()
	if head == nil || !d.hasBlock(head.ParentHash()) {
		return nil
	}
	// Retrieve a full batch of blocks
	return d.queue.TakeBlocks(head)
}

func (d *Downloader) Has(hash common.Hash) bool {
	return d.queue.Has(hash)
}

func (d *Downloader) getFromPeer(p *peer, hash common.Hash, ignoreInitial bool) (err error) {
	d.activePeer = p.id
	defer func() {
		// reset on error
		if err != nil {
			d.queue.Reset()
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

	if d.queue.Has(h) { // TODO: Is this possible? Shouldn't queue be empty for startFetchingHashes to be even called?
		return errAlreadyInPool
	}

	glog.V(logger.Debug).Infof("Downloading hashes (%x) from %s", h[:4], p.id)

	start := time.Now()

	// We ignore the initial hash in some cases (e.g. we received a block without it's parent)
	// In such circumstances we don't need to download the block so don't add it to the queue.
	if !ignoreInitial {
		// Add the hash to the queue first
		d.queue.Insert([]common.Hash{h})
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
			// Make sure the active peer is giving us the hashes
			if hashPack.peerId != activePeer.id {
				glog.V(logger.Debug).Infof("Received hashes from incorrect peer(%s)\n", hashPack.peerId)
				break
			}

			failureResponseTimer.Reset(hashTtl)

			// Make sure the peer actually gave something valid
			if len(hashPack.hashes) == 0 {
				glog.V(logger.Debug).Infof("Peer (%s) responded with empty hash set\n", activePeer.id)
				d.queue.Reset()

				return errEmptyHashSet
			}
			// Determine if we're done fetching hashes (queue up all pending), and continue if not done
			done, index := false, 0
			for index, hash = range hashPack.hashes {
				if d.hasBlock(hash) || d.queue.GetBlock(hash) != nil {
					glog.V(logger.Debug).Infof("Found common hash %x\n", hash[:4])
					hashPack.hashes = hashPack.hashes[:index]
					done = true
					break
				}
			}
			d.queue.Insert(hashPack.hashes)

			if !done {
				activePeer.getHashes(hash)
				continue
			}
			// We're done, allocate the download cache and proceed pulling the blocks
			offset := 0
			if block := d.getBlock(hash); block != nil {
				offset = int(block.NumberU64() + 1)
			}
			d.queue.Alloc(offset)
			break out

		case <-failureResponseTimer.C:
			glog.V(logger.Debug).Infof("Peer (%s) didn't respond in time for hash request\n", p.id)

			var p *peer // p will be set if a peer can be found
			// Attempt to find a new peer by checking inclusion of peers best hash in our
			// already fetched hash list. This can't guarantee 100% correctness but does
			// a fair job. This is always either correct or false incorrect.
			for id, peer := range d.peers {
				if d.queue.Has(peer.recentHash) && !attemptedPeers[id] {
					p = peer
					break
				}
			}

			// if all peers have been tried, abort the process entirely or if the hash is
			// the zero hash.
			if p == nil || (hash == common.Hash{}) {
				d.queue.Reset()
				return errTimeout
			}

			// set p to the active peer. this will invalidate any hashes that may be returned
			// by our previous (delayed) peer.
			activePeer = p
			p.getHashes(hash)
			glog.V(logger.Debug).Infof("Hash fetching switched to new peer(%s)\n", p.id)
		}
	}
	glog.V(logger.Detail).Infof("Downloaded hashes (%d) in %v\n", d.queue.Pending(), time.Since(start))

	return nil
}

func (d *Downloader) startFetchingBlocks(p *peer) error {
	glog.V(logger.Detail).Infoln("Downloading", d.queue.Pending(), "block(s)")

	atomic.StoreInt32(&d.downloadingBlocks, 1)
	defer atomic.StoreInt32(&d.downloadingBlocks, 0)
	// Defer the peer reset. This will empty the peer requested set
	// and makes sure there are no lingering peers with an incorrect
	// state
	defer d.peers.reset()

	start := time.Now()

	// default ticker for re-fetching blocks every now and then
	ticker := time.NewTicker(20 * time.Millisecond)
out:
	for {
		select {
		case blockPack := <-d.blockCh:
			// If the peer was previously banned and failed to deliver it's pack
			// in a reasonable time frame, ignore it's message.
			if d.peers[blockPack.peerId] != nil {
				err := d.queue.Deliver(blockPack.peerId, blockPack.blocks)
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
			// after removing bad peers make sure we actually have sufficient peer left to keep downloading
			if len(d.peers) == 0 {
				d.queue.Reset()

				return errNoPeers
			}

			// If there are unrequested hashes left start fetching
			// from the available peers.
			if d.queue.Pending() > 0 {
				// Throttle the download if block cache is full and waiting processing
				if d.queue.Throttle() {
					continue
				}

				availablePeers := d.peers.get(idleState)
				for _, peer := range availablePeers {
					// Get a possible chunk. If nil is returned no chunk
					// could be returned due to no hashes available.
					request := d.queue.Reserve(peer, maxBlockFetch)
					if request == nil {
						continue
					}
					// XXX make fetch blocking.
					// Fetch the chunk and check for error. If the peer was somehow
					// already fetching a chunk due to a bug, it will be returned to
					// the queue
					if err := peer.fetch(request); err != nil {
						// log for tracing
						glog.V(logger.Debug).Infof("peer %s received double work (state = %v)\n", peer.id, peer.state)
						d.queue.Cancel(request)
					}
				}
				// make sure that we have peers available for fetching. If all peers have been tried
				// and all failed throw an error
				if d.queue.InFlight() == 0 {
					d.queue.Reset()

					return fmt.Errorf("%v peers avaialable = %d. total peers = %d. hashes needed = %d", errPeersUnavailable, len(availablePeers), len(d.peers), d.queue.Pending())
				}

			} else if d.queue.InFlight() == 0 {
				// When there are no more queue and no more in flight, We can
				// safely assume we're done. Another part of the process will  check
				// for parent errors and will re-request anything that's missing
				break out
			} else {
				// Check for bad peers. Bad peers may indicate a peer not responding
				// to a `getBlocks` message. A timeout of 5 seconds is set. Peers
				// that badly or poorly behave are removed from the peer set (not banned).
				// Bad peers are excluded from the available peer set and therefor won't be
				// reused. XXX We could re-introduce peers after X time.
				badPeers := d.queue.Expire(blockTtl)
				for _, pid := range badPeers {
					// XXX We could make use of a reputation system here ranking peers
					// in their performance
					// 1) Time for them to respond;
					// 2) Measure their speed;
					// 3) Amount and availability.
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
		glog.Infof("adding %d (T=%d) hashes [ %x / %x ] from: %s\n", len(hashes), d.queue.Pending(), from[:4], to[:4], id)
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
