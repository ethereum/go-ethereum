package eth

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"gopkg.in/fatih/set.v0"
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

func getBestPeer(peers map[string]*peer) *peer {
	var peer *peer
	for _, cp := range peers {
		if peer == nil || cp.td.Cmp(peer.td) > 0 {
			peer = cp
		}
	}
	return peer
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

func (p *peer) sendNewBlock(block *types.Block) error {
	p.blockHashes.Add(block.Hash())

	return p2p.Send(p.rw, NewBlockMsg, []interface{}{block, block.Td})
}

func (p *peer) sendTransaction(tx *types.Transaction) error {
	p.txHashes.Add(tx.Hash())

	return p2p.Send(p.rw, TxMsg, []*types.Transaction{tx})
}

func (p *peer) requestHashes(from common.Hash) error {
	glog.V(logger.Debug).Infof("[%s] fetching hashes (%d) %x...\n", p.id, maxHashes, from[:4])
	return p2p.Send(p.rw, GetBlockHashesMsg, getBlockHashesMsgData{from, maxHashes})
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
