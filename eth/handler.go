package eth

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

// main entrypoint, wrappers starting a server running the eth protocol
// use this constructor to attach the protocol ("class") to server caps
// the Dev p2p layer then runs the protocol instance on each peer
func EthProtocol(protocolVersion, networkId int, txPool txPool, chainManager chainManager, downloader *downloader.Downloader) p2p.Protocol {
	protocol := newProtocolManager(txPool, chainManager, downloader)

	return p2p.Protocol{
		Name:    "eth",
		Version: uint(protocolVersion),
		Length:  ProtocolLength,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			//return runEthProtocol(protocolVersion, networkId, txPool, chainManager, downloader, p, rw)
			peer := protocol.newPeer(protocolVersion, networkId, p, rw)
			err := protocol.handle(peer)
			glog.V(logger.Detail).Infof("[%s]: %v\n", peer.id, err)

			return err
		},
	}
}

type hashFetcherFn func(common.Hash) error
type blockFetcherFn func([]common.Hash) error

// extProt is an interface which is passed around so we can expose GetHashes and GetBlock without exposing it to the rest of the protocol
// extProt is passed around to peers which require to GetHashes and GetBlocks
type extProt struct {
	getHashes hashFetcherFn
	getBlocks blockFetcherFn
}

func (ep extProt) GetHashes(hash common.Hash) error    { return ep.getHashes(hash) }
func (ep extProt) GetBlock(hashes []common.Hash) error { return ep.getBlocks(hashes) }

type EthProtocolManager struct {
	protVer, netId int
	txpool         txPool
	chainman       chainManager
	downloader     *downloader.Downloader

	pmu   sync.Mutex
	peers map[string]*peer
}

func newProtocolManager(txpool txPool, chainman chainManager, downloader *downloader.Downloader) *EthProtocolManager {
	return &EthProtocolManager{
		txpool:     txpool,
		chainman:   chainman,
		downloader: downloader,
		peers:      make(map[string]*peer),
	}
}

func (pm *EthProtocolManager) newPeer(pv, nv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	pm.pmu.Lock()
	defer pm.pmu.Unlock()

	td, current, genesis := pm.chainman.Status()

	peer := newPeer(pv, nv, genesis, current, td, p, rw)
	pm.peers[peer.id] = peer

	return peer
}

func (pm *EthProtocolManager) handle(p *peer) error {
	if err := p.handleStatus(); err != nil {
		return err
	}

	pm.downloader.RegisterPeer(p.id, p.td, p.currentHash, p.requestHashes, p.requestBlocks)
	defer pm.downloader.UnregisterPeer(p.id)

	// propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	if err := p.sendTransactions(pm.txpool.GetTransactions()); err != nil {
		return err
	}

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			return err
		}
	}

	return nil
}

func (self *EthProtocolManager) handleMsg(p *peer) error {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// make sure that the payload has been fully consumed
	defer msg.Discard()

	switch msg.Code {
	case GetTxMsg: // ignore
	case StatusMsg:
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	case TxMsg:
		// TODO: rework using lazy RLP stream
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for i, tx := range txs {
			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			jsonlogger.LogJson(&logger.EthTxReceived{
				TxHash:   tx.Hash().Hex(),
				RemoteId: p.ID().String(),
			})
		}
		self.txpool.AddTransactions(txs)

	case GetBlockHashesMsg:
		var request getBlockHashesMsgData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "->msg %v: %v", msg, err)
		}

		if request.Amount > maxHashes {
			request.Amount = maxHashes
		}
		hashes := self.chainman.GetBlockHashesFromHash(request.Hash, request.Amount)
		return p.sendBlockHashes(hashes)
	case BlockHashesMsg:
		msgStream := rlp.NewStream(msg.Payload)

		var hashes []common.Hash
		if err := msgStream.Decode(&hashes); err != nil {
			break
		}
		self.downloader.HashCh <- hashes

	case GetBlocksMsg:
		msgStream := rlp.NewStream(msg.Payload)
		if _, err := msgStream.List(); err != nil {
			return err
		}

		var blocks []*types.Block
		var i int
		for {
			i++
			var hash common.Hash
			err := msgStream.Decode(&hash)
			if err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}

			block := self.chainman.GetBlock(hash)
			if block != nil {
				blocks = append(blocks, block)
			}
			if i == maxBlocks {
				break
			}
		}
		return p.sendBlocks(blocks)
	case BlocksMsg:
		msgStream := rlp.NewStream(msg.Payload)

		var blocks []*types.Block
		if err := msgStream.Decode(&blocks); err != nil {
			glog.V(logger.Detail).Infoln("Decode error", err)
			fmt.Println("decode error", err)
			blocks = nil
		}
		self.downloader.DeliverChunk(p.id, blocks)

	case NewBlockMsg:
		var request newBlockMsgData
		if err := msg.Decode(&request); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		if err := request.Block.ValidateFields(); err != nil {
			return errResp(ErrDecode, "block validation %v: %v", msg, err)
		}
		hash := request.Block.Hash()
		_, chainHead, _ := self.chainman.Status()

		jsonlogger.LogJson(&logger.EthChainReceivedNewBlock{
			BlockHash:     hash.Hex(),
			BlockNumber:   request.Block.Number(), // this surely must be zero
			ChainHeadHash: chainHead.Hex(),
			BlockPrevHash: request.Block.ParentHash().Hex(),
			RemoteId:      p.ID().String(),
		})
		self.downloader.AddBlock(p.id, request.Block, request.TD)

	default:
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}
