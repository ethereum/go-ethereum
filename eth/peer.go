package eth

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"gopkg.in/fatih/set.v0"
)

var (
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

const (
	maxKnownTxs    = 32768 // Maximum transactions hashes to keep in the known list (prevent DOS)
	maxKnownBlocks = 1024  // Maximum block hashes to keep in the known list (prevent DOS)
)

type peer struct {
	*p2p.Peer

	rw p2p.MsgReadWriter

	version int // Protocol version negotiated
	network int // Network ID being on

	id string

	head common.Hash
	td   *big.Int
	lock sync.RWMutex

	knownTxs    *set.Set // Set of transaction hashes known to be known by this peer
	knownBlocks *set.Set // Set of block hashes known to be known by this peer
}

func newPeer(version, network int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	id := p.ID()

	return &peer{
		Peer:        p,
		rw:          rw,
		version:     version,
		network:     network,
		id:          fmt.Sprintf("%x", id[:8]),
		knownTxs:    set.New(),
		knownBlocks: set.New(),
	}
}

// Head retrieves a copy of the current head (most recent) hash of the peer.
func (p *peer) Head() (hash common.Hash) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	copy(hash[:], p.head[:])
	return hash
}

// SetHead updates the head (most recent) hash of the peer.
func (p *peer) SetHead(hash common.Hash) {
	p.lock.Lock()
	defer p.lock.Unlock()

	copy(p.head[:], hash[:])
}

// Td retrieves the current total difficulty of a peer.
func (p *peer) Td() *big.Int {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return new(big.Int).Set(p.td)
}

// SetTd updates the current total difficulty of a peer.
func (p *peer) SetTd(td *big.Int) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.td.Set(td)
}

// MarkBlock marks a block as known for the peer, ensuring that the block will
// never be propagated to this particular peer.
func (p *peer) MarkBlock(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known block hash
	for p.knownBlocks.Size() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(hash)
}

// MarkTransaction marks a transaction as known for the peer, ensuring that it
// will never be propagated to this particular peer.
func (p *peer) MarkTransaction(hash common.Hash) {
	// If we reached the memory allowance, drop a previously known transaction hash
	for p.knownTxs.Size() >= maxKnownTxs {
		p.knownTxs.Pop()
	}
	p.knownTxs.Add(hash)
}

// SendTransactions sends transactions to the peer and includes the hashes
// in its transaction hash set for future reference.
func (p *peer) SendTransactions(txs types.Transactions) error {
	propTxnOutPacketsMeter.Mark(1)
	for _, tx := range txs {
		propTxnOutTrafficMeter.Mark(tx.Size().Int64())
		p.knownTxs.Add(tx.Hash())
	}
	return p2p.Send(p.rw, TxMsg, txs)
}

// SendBlockHashes sends a batch of known hashes to the remote peer.
func (p *peer) SendBlockHashes(hashes []common.Hash) error {
	reqHashOutPacketsMeter.Mark(1)
	reqHashOutTrafficMeter.Mark(int64(32 * len(hashes)))

	return p2p.Send(p.rw, BlockHashesMsg, hashes)
}

// SendBlocks sends a batch of blocks to the remote peer.
func (p *peer) SendBlocks(blocks []*types.Block) error {
	reqBlockOutPacketsMeter.Mark(1)
	for _, block := range blocks {
		reqBlockOutTrafficMeter.Mark(block.Size().Int64())
	}
	return p2p.Send(p.rw, BlocksMsg, blocks)
}

// SendNewBlockHashes announces the availability of a number of blocks through
// a hash notification.
func (p *peer) SendNewBlockHashes(hashes []common.Hash) error {
	propHashOutPacketsMeter.Mark(1)
	propHashOutTrafficMeter.Mark(int64(32 * len(hashes)))

	for _, hash := range hashes {
		p.knownBlocks.Add(hash)
	}
	return p2p.Send(p.rw, NewBlockHashesMsg, hashes)
}

// SendNewBlock propagates an entire block to a remote peer.
func (p *peer) SendNewBlock(block *types.Block) error {
	propBlockOutPacketsMeter.Mark(1)
	propBlockOutTrafficMeter.Mark(block.Size().Int64())

	p.knownBlocks.Add(block.Hash())
	return p2p.Send(p.rw, NewBlockMsg, []interface{}{block, block.Td})
}

// RequestHashes fetches a batch of hashes from a peer, starting at from, going
// towards the genesis block.
func (p *peer) RequestHashes(from common.Hash) error {
	glog.V(logger.Debug).Infof("Peer [%s] fetching hashes (%d) from %x...\n", p.id, downloader.MaxHashFetch, from[:4])
	return p2p.Send(p.rw, GetBlockHashesMsg, getBlockHashesData{from, uint64(downloader.MaxHashFetch)})
}

// RequestHashesFromNumber fetches a batch of hashes from a peer, starting at the
// requested block number, going upwards towards the genesis block.
func (p *peer) RequestHashesFromNumber(from uint64, count int) error {
	glog.V(logger.Debug).Infof("Peer [%s] fetching hashes (%d) from #%d...\n", p.id, count, from)
	return p2p.Send(p.rw, GetBlockHashesFromNumberMsg, getBlockHashesFromNumberData{from, uint64(count)})
}

// RequestBlocks fetches a batch of blocks corresponding to the specified hashes.
func (p *peer) RequestBlocks(hashes []common.Hash) error {
	glog.V(logger.Debug).Infof("[%s] fetching %v blocks\n", p.id, len(hashes))
	return p2p.Send(p.rw, GetBlocksMsg, hashes)
}

// Handshake executes the eth protocol handshake, negotiating version number,
// network IDs, difficulties, head and genesis blocks.
func (p *peer) Handshake(td *big.Int, head common.Hash, genesis common.Hash) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 1)
	go func() {
		errc <- p2p.Send(p.rw, StatusMsg, &statusData{
			ProtocolVersion: uint32(p.version),
			NetworkId:       uint32(p.network),
			TD:              td,
			CurrentBlock:    head,
			GenesisBlock:    genesis,
		})
	}()
	// In the mean time retrieve the remote status message
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != StatusMsg {
		return errResp(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	var status statusData
	if err := msg.Decode(&status); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return errResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock, genesis)
	}
	if int(status.NetworkId) != p.network {
		return errResp(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, p.network)
	}
	if int(status.ProtocolVersion) != p.version {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	// Configure the remote peer, and sanity check out handshake too
	p.td, p.head = status.TD, status.CurrentBlock
	return <-errc
}

// String implements fmt.Stringer.
func (p *peer) String() string {
	return fmt.Sprintf("Peer %s [%s]", p.id,
		fmt.Sprintf("eth/%2d", p.version),
	)
}

// peerSet represents the collection of active peers currently participating in
// the Ethereum sub-protocol.
type peerSet struct {
	peers map[string]*peer
	lock  sync.RWMutex
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if _, ok := ps.peers[id]; !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

// PeersWithoutBlock retrieves a list of peers that do not have a given block in
// their set of known hashes.
func (ps *peerSet) PeersWithoutBlock(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownBlocks.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

// PeersWithoutTx retrieves a list of peers that do not have a given transaction
// in their set of known hashes.
func (ps *peerSet) PeersWithoutTx(hash common.Hash) []*peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	list := make([]*peer, 0, len(ps.peers))
	for _, p := range ps.peers {
		if !p.knownTxs.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var (
		bestPeer *peer
		bestTd   *big.Int
	)
	for _, p := range ps.peers {
		if td := p.Td(); bestPeer == nil || td.Cmp(bestTd) > 0 {
			bestPeer, bestTd = p, td
		}
	}
	return bestPeer
}
