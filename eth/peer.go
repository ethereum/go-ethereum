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

type statusMsgData struct {
	ProtocolVersion uint32
	NetworkId       uint32
	TD              *big.Int
	CurrentBlock    common.Hash
	GenesisBlock    common.Hash
}

type getBlockHashesMsgData struct {
	Hash   common.Hash
	Amount uint64
}

type peer struct {
	*p2p.Peer

	rw p2p.MsgReadWriter

	protv, netid int

	recentHash common.Hash
	id         string
	td         *big.Int

	genesis, ourHash common.Hash
	ourTd            *big.Int

	txHashes    *set.Set
	blockHashes *set.Set
}

func newPeer(protv, netid int, genesis, recentHash common.Hash, td *big.Int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	id := p.ID()

	return &peer{
		Peer:        p,
		rw:          rw,
		genesis:     genesis,
		ourHash:     recentHash,
		ourTd:       td,
		protv:       protv,
		netid:       netid,
		id:          fmt.Sprintf("%x", id[:8]),
		txHashes:    set.New(),
		blockHashes: set.New(),
	}
}

// sendTransactions sends transactions to the peer and includes the hashes
// in it's tx hash set for future reference. The tx hash will allow the
// manager to check whether the peer has already received this particular
// transaction
func (p *peer) sendTransactions(txs types.Transactions) error {
	for _, tx := range txs {
		p.txHashes.Add(tx.Hash())
	}

	return p2p.Send(p.rw, TxMsg, txs)
}

func (p *peer) sendBlockHashes(hashes []common.Hash) error {
	return p2p.Send(p.rw, BlockHashesMsg, hashes)
}

func (p *peer) sendBlocks(blocks []*types.Block) error {
	return p2p.Send(p.rw, BlocksMsg, blocks)
}

func (p *peer) sendNewBlockHashes(hashes []common.Hash) error {
	return p2p.Send(p.rw, NewBlockHashesMsg, hashes)
}

func (p *peer) sendNewBlock(block *types.Block) error {
	p.blockHashes.Add(block.Hash())

	return p2p.Send(p.rw, NewBlockMsg, []interface{}{block, block.Td})
}

func (p *peer) sendTransaction(tx *types.Transaction) error {
	p.txHashes.Add(tx.Hash())

	return p2p.Send(p.rw, TxMsg, []*types.Transaction{tx})
}

func (p *peer) requestHashes(from common.Hash) error {
	glog.V(logger.Debug).Infof("[%s] fetching hashes (%d) %x...\n", p.id, downloader.MaxHashFetch, from[:4])
	return p2p.Send(p.rw, GetBlockHashesMsg, getBlockHashesMsgData{from, downloader.MaxHashFetch})
}

func (p *peer) requestBlocks(hashes []common.Hash) error {
	glog.V(logger.Debug).Infof("[%s] fetching %v blocks\n", p.id, len(hashes))
	return p2p.Send(p.rw, GetBlocksMsg, hashes)
}

func (p *peer) handleStatus() error {
	errc := make(chan error, 1)
	go func() {
		errc <- p2p.Send(p.rw, StatusMsg, &statusMsgData{
			ProtocolVersion: uint32(p.protv),
			NetworkId:       uint32(p.netid),
			TD:              p.ourTd,
			CurrentBlock:    p.ourHash,
			GenesisBlock:    p.genesis,
		})
	}()

	// read and handle remote status
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

	var status statusMsgData
	if err := msg.Decode(&status); err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}

	if status.GenesisBlock != p.genesis {
		return errResp(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock, p.genesis)
	}

	if int(status.NetworkId) != p.netid {
		return errResp(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, p.netid)
	}

	if int(status.ProtocolVersion) != p.protv {
		return errResp(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.protv)
	}
	// Set the total difficulty of the peer
	p.td = status.TD
	// set the best hash of the peer
	p.recentHash = status.CurrentBlock

	return <-errc
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
		if !p.blockHashes.Has(hash) {
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
		if !p.txHashes.Has(hash) {
			list = append(list, p)
		}
	}
	return list
}

// BestPeer retrieves the known peer with the currently highest total difficulty.
func (ps *peerSet) BestPeer() *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	var best *peer
	for _, p := range ps.peers {
		if best == nil || p.td.Cmp(best.td) > 0 {
			best = p
		}
	}
	return best
}
