package downloader

import (
	"errors"
	"fmt"
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
	maxBlockFetch    = 256              // Amount of max blocks to be fetched per chunk
	peerCountTimeout = 12 * time.Second // Amount of time it takes for the peer handler to ignore minDesiredPeerCount
	hashTtl          = 20 * time.Second // The amount of time it takes for a hash request to time out
)

var (
	minDesiredPeerCount = 5                // Amount of peers desired to start syncing
	blockTtl            = 20 * time.Second // The amount of time it takes for a block request to time out

	errLowTd            = errors.New("peer's TD is too low")
	errBusy             = errors.New("busy")
	errUnknownPeer      = errors.New("peer's unknown or unhealthy")
	ErrBadPeer          = errors.New("action from bad peer ignored")
	errTimeout          = errors.New("timeout")
	errEmptyHashSet     = errors.New("empty hash set by peer")
	errPeersUnavailable = errors.New("no peers available or all peers tried for block download process")
)

type hashCheckFn func(common.Hash) bool
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

type Downloader struct {
	mu         sync.RWMutex
	queue      *queue
	peers      peers
	activePeer string

	// Callbacks
	hasBlock    hashCheckFn
	insertChain chainInsertFn

	// Status
	fetchingHashes    int32
	downloadingBlocks int32
	processingBlocks  int32

	// Channels
	newPeerCh chan *peer
	hashCh    chan []common.Hash
	blockCh   chan blockPack
}

func New(hasBlock hashCheckFn, insertChain chainInsertFn) *Downloader {
	downloader := &Downloader{
		queue:       newqueue(),
		peers:       make(peers),
		hasBlock:    hasBlock,
		insertChain: insertChain,
		newPeerCh:   make(chan *peer, 1),
		hashCh:      make(chan []common.Hash, 1),
		blockCh:     make(chan blockPack, 1),
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

	return d.process(p)
}

func (d *Downloader) getFromPeer(p *peer, hash common.Hash, ignoreInitial bool) error {
	d.activePeer = p.id

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

	failureResponseTimer := time.NewTimer(hashTtl)

out:
	for {
		select {
		case hashes := <-d.hashCh:
			failureResponseTimer.Reset(hashTtl)

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

				return errEmptyHashSet
			} else if !done { // Check if we're done fetching
				// Get the next set of hashes
				p.getHashes(hashes[len(hashes)-1])
			} else { // we're done
				break out
			}
		case <-failureResponseTimer.C:
			glog.V(logger.Debug).Infof("Peer (%s) didn't respond in time for hash request\n", p.id)
			// TODO instead of reseting the queue select a new peer from which we can start downloading hashes.
			// 1. check for peer's best hash to be included in the current hash set;
			// 2. resume from last point (hashes[len(hashes)-1]) using the newly selected peer.
			d.queue.reset()

			return errTimeout
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
						//d.UnregisterPeer(pid)
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

	d.hashCh <- hashes

	return nil
}

// Add an (unrequested) block to the downloader. This is usually done through the
// NewBlockMsg by the protocol handler.
// Adding blocks is done synchronously. if there are missing blocks, blocks will be
// fetched first. If the downloader is busy or if some other processed failed an error
// will be returned.
func (d *Downloader) AddBlock(id string, block *types.Block, td *big.Int) error {
	hash := block.Hash()

	if d.hasBlock(hash) {
		return fmt.Errorf("known block %x", hash.Bytes()[:4])
	}

	peer := d.peers.getPeer(id)
	// if the peer is in our healthy list of peers; update the td
	// and add the block. Otherwise just ignore it
	if peer == nil {
		glog.V(logger.Detail).Infof("Ignored block from bad peer %s\n", id)
		return ErrBadPeer
	}

	peer.mu.Lock()
	peer.recentHash = block.Hash()
	peer.mu.Unlock()
	peer.promote()

	glog.V(logger.Detail).Infoln("Inserting new block from:", id)
	d.queue.addBlock(id, block)

	// if neither go ahead to process
	if d.isBusy() {
		return errBusy
	}

	// Check if the parent of the received block is known.
	// If the block is not know, request it otherwise, request.
	phash := block.ParentHash()
	if !d.hasBlock(phash) {
		glog.V(logger.Detail).Infof("Missing parent %x, requires fetching\n", phash.Bytes()[:4])

		// Get the missing hashes from the peer (synchronously)
		err := d.getFromPeer(peer, peer.recentHash, true)
		if err != nil {
			return err
		}
	}

	return d.process(peer)
}

func (d *Downloader) process(peer *peer) error {
	atomic.StoreInt32(&d.processingBlocks, 1)
	defer atomic.StoreInt32(&d.processingBlocks, 0)

	// XXX this will move when optimised
	// Sort the blocks by number. This bit needs much improvement. Right now
	// it assumes full honesty form peers (i.e. it's not checked when the blocks
	// link). We should at least check whihc queue match. This code could move
	// to a seperate goroutine where it periodically checks for linked pieces.
	types.BlockBy(types.Number).Sort(d.queue.blocks)
	if len(d.queue.blocks) == 0 {
		return nil
	}

	var blocks = d.queue.blocks
	glog.V(logger.Debug).Infof("Inserting chain with %d blocks (#%v - #%v)\n", len(blocks), blocks[0].Number(), blocks[len(blocks)-1].Number())

	// Loop untill we're out of blocks
	for len(blocks) != 0 {
		max := int(math.Min(float64(len(blocks)), 256))
		// TODO check for parent error. When there's a parent error we should stop
		// processing and start requesting the `block.hash` so that it's parent and
		// grandparents can be requested and queued.
		var i int
		i, err := d.insertChain(blocks[:max])
		if err != nil && core.IsParentErr(err) {
			// Ignore the missing blocks. Handler should take care of anything that's missing.
			glog.V(logger.Debug).Infof("Ignored block with missing parent (%d)\n", i)
			blocks = blocks[i+1:]

			continue
		} else if err != nil {
			// immediatly unregister the false peer but do not disconnect
			d.UnregisterPeer(d.activePeer)
			// Reset chain completely. This needs much, much improvement.
			// instead: check all blocks leading down to this block false block and remove it
			d.queue.blocks = nil

			return ErrBadPeer
		}

		// delete the blocks from the slice and let them be garbage collected
		// without this slice trick the blocks would stay in memory until nil
		// would be assigned to d.queue.blocks
		copy(blocks, blocks[max:])
		for k, n := len(blocks)-max, len(blocks); k < n; k++ {
			blocks[k] = nil
		}
		blocks = blocks[:len(blocks)-max]
	}

	if len(blocks) == 0 {
		d.queue.blocks = nil
	} else {
		d.queue.blocks = blocks
	}
	return nil
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

func (d *Downloader) IsBusy() bool {
	return d.isBusy()
}
