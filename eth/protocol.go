package eth

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	ProtocolVersion    = 54
	NetworkId          = 0
	ProtocolLength     = uint64(8)
	ProtocolMaxMsgSize = 10 * 1024 * 1024
)

// eth protocol message codes
const (
	StatusMsg = iota
	GetTxMsg  // unused
	TxMsg
	GetBlockHashesMsg
	BlockHashesMsg
	GetBlocksMsg
	BlocksMsg
	NewBlockMsg
)

// ethProtocol represents the ethereum wire protocol
// instance is running on each peer
type ethProtocol struct {
	txPool       txPool
	chainManager chainManager
	blockPool    blockPool
	peer         *p2p.Peer
	id           string
	rw           p2p.MsgReadWriter
}

// backend is the interface the ethereum protocol backend should implement
// used as an argument to EthProtocol
type txPool interface {
	AddTransactions([]*types.Transaction)
	GetTransactions() types.Transactions
}

type chainManager interface {
	GetBlockHashesFromHash(hash []byte, amount uint64) (hashes [][]byte)
	GetBlock(hash []byte) (block *types.Block)
	Status() (td *big.Int, currentBlock []byte, genesisBlock []byte)
}

type blockPool interface {
	AddBlockHashes(next func() ([]byte, bool), peerId string)
	AddBlock(block *types.Block, peerId string)
	AddPeer(td *big.Int, currentBlock []byte, peerId string, requestHashes func([]byte) error, requestBlocks func([][]byte) error, peerError func(int, string, ...interface{})) (best bool)
	RemovePeer(peerId string)
}

// message structs used for rlp decoding
type newBlockMsgData struct {
	Block *types.Block
	TD    *big.Int
}

const maxHashes = 255

type getBlockHashesMsgData struct {
	Hash   []byte
	Amount uint64
}

// main entrypoint, wrappers starting a server running the eth protocol
// use this constructor to attach the protocol ("class") to server caps
// the Dev p2p layer then runs the protocol instance on each peer
func EthProtocol(txPool txPool, chainManager chainManager, blockPool blockPool) p2p.Protocol {
	return p2p.Protocol{
		Name:    "eth",
		Version: ProtocolVersion,
		Length:  ProtocolLength,
		Run: func(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
			return runEthProtocol(txPool, chainManager, blockPool, peer, rw)
		},
	}
}

// the main loop that handles incoming messages
// note RemovePeer in the post-disconnect hook
func runEthProtocol(txPool txPool, chainManager chainManager, blockPool blockPool, peer *p2p.Peer, rw p2p.MsgReadWriter) (err error) {
	id := peer.ID()
	self := &ethProtocol{
		txPool:       txPool,
		chainManager: chainManager,
		blockPool:    blockPool,
		rw:           rw,
		peer:         peer,
		id:           fmt.Sprintf("%x", id[:8]),
	}
	err = self.handleStatus()
	if err == nil {
		self.propagateTxs()
		for {
			err = self.handle()
			if err != nil {
				self.blockPool.RemovePeer(self.id)
				break
			}
		}
	}
	return
}

func (self *ethProtocol) handle() error {
	msg, err := self.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return self.protoError(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// make sure that the payload has been fully consumed
	defer msg.Discard()

	switch msg.Code {
	case GetTxMsg: // ignore
	case StatusMsg:
		return self.protoError(ErrExtraStatusMsg, "")

	case TxMsg:
		// TODO: rework using lazy RLP stream
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return self.protoError(ErrDecode, "msg %v: %v", msg, err)
		}
		for _, tx := range txs {
			jsonlogger.LogJson(&logger.EthTxReceived{
				TxHash:   ethutil.Bytes2Hex(tx.Hash()),
				RemoteId: self.peer.ID().String(),
			})
		}
		self.txPool.AddTransactions(txs)

	case GetBlockHashesMsg:
		var request getBlockHashesMsgData
		if err := msg.Decode(&request); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}

		if request.Amount > maxHashes {
			request.Amount = maxHashes
		}
		hashes := self.chainManager.GetBlockHashesFromHash(request.Hash, request.Amount)
		return p2p.EncodeMsg(self.rw, BlockHashesMsg, ethutil.ByteSliceToInterface(hashes)...)

	case BlockHashesMsg:
		// TODO: redo using lazy decode , this way very inefficient on known chains
		msgStream := rlp.NewStream(msg.Payload)
		var err error
		var i int

		iter := func() (hash []byte, ok bool) {
			hash, err = msgStream.Bytes()
			if err == nil {
				i++
				ok = true
			} else {
				if err != io.EOF {
					self.protoError(ErrDecode, "msg %v: after %v hashes : %v", msg, i, err)
				}
			}
			return
		}

		self.blockPool.AddBlockHashes(iter, self.id)

	case GetBlocksMsg:
		msgStream := rlp.NewStream(msg.Payload)
		var blocks []interface{}
		var i int
		for {
			i++
			var hash []byte
			if err := msgStream.Decode(&hash); err != nil {
				if err == io.EOF {
					break
				} else {
					return self.protoError(ErrDecode, "msg %v: %v", msg, err)
				}
			}
			block := self.chainManager.GetBlock(hash)
			if block != nil {
				blocks = append(blocks, block)
			}
			if i == blockHashesBatchSize {
				break
			}
		}
		return p2p.EncodeMsg(self.rw, BlocksMsg, blocks...)

	case BlocksMsg:
		msgStream := rlp.NewStream(msg.Payload)
		for {
			var block types.Block
			if err := msgStream.Decode(&block); err != nil {
				if err == io.EOF {
					break
				} else {
					return self.protoError(ErrDecode, "msg %v: %v", msg, err)
				}
			}
			self.blockPool.AddBlock(&block, self.id)
		}

	case NewBlockMsg:
		var request newBlockMsgData
		if err := msg.Decode(&request); err != nil {
			return self.protoError(ErrDecode, "msg %v: %v", msg, err)
		}
		hash := request.Block.Hash()
		_, chainHead, _ := self.chainManager.Status()

		jsonlogger.LogJson(&logger.EthChainReceivedNewBlock{
			BlockHash:     ethutil.Bytes2Hex(hash),
			BlockNumber:   request.Block.Number(), // this surely must be zero
			ChainHeadHash: ethutil.Bytes2Hex(chainHead),
			BlockPrevHash: ethutil.Bytes2Hex(request.Block.ParentHash()),
			RemoteId:      self.peer.ID().String(),
		})
		// to simplify backend interface adding a new block
		// uses AddPeer followed by AddHashes, AddBlock only if peer is the best peer
		// (or selected as new best peer)
		if self.blockPool.AddPeer(request.TD, hash, self.id, self.requestBlockHashes, self.requestBlocks, self.protoErrorDisconnect) {
			self.blockPool.AddBlock(request.Block, self.id)
		}

	default:
		return self.protoError(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

type statusMsgData struct {
	ProtocolVersion uint32
	NetworkId       uint32
	TD              *big.Int
	CurrentBlock    []byte
	GenesisBlock    []byte
}

func (self *ethProtocol) statusMsg() p2p.Msg {
	td, currentBlock, genesisBlock := self.chainManager.Status()

	return p2p.NewMsg(StatusMsg,
		uint32(ProtocolVersion),
		uint32(NetworkId),
		td,
		currentBlock,
		genesisBlock,
	)
}

func (self *ethProtocol) handleStatus() error {
	// send precanned status message
	if err := self.rw.WriteMsg(self.statusMsg()); err != nil {
		return err
	}

	// read and handle remote status
	msg, err := self.rw.ReadMsg()
	if err != nil {
		return err
	}

	if msg.Code != StatusMsg {
		return self.protoError(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}

	if msg.Size > ProtocolMaxMsgSize {
		return self.protoError(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}

	var status statusMsgData
	if err := msg.Decode(&status); err != nil {
		return self.protoError(ErrDecode, "msg %v: %v", msg, err)
	}

	_, _, genesisBlock := self.chainManager.Status()

	if bytes.Compare(status.GenesisBlock, genesisBlock) != 0 {
		return self.protoError(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock, genesisBlock)
	}

	if status.NetworkId != NetworkId {
		return self.protoError(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, NetworkId)
	}

	if ProtocolVersion != status.ProtocolVersion {
		return self.protoError(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, ProtocolVersion)
	}

	self.peer.Infof("Peer is [eth] capable (%d/%d). TD=%v H=%x\n", status.ProtocolVersion, status.NetworkId, status.TD, status.CurrentBlock[:4])

	self.blockPool.AddPeer(status.TD, status.CurrentBlock, self.id, self.requestBlockHashes, self.requestBlocks, self.protoErrorDisconnect)

	return nil
}

func (self *ethProtocol) requestBlockHashes(from []byte) error {
	self.peer.Debugf("fetching hashes (%d) %x...\n", blockHashesBatchSize, from[0:4])
	return p2p.EncodeMsg(self.rw, GetBlockHashesMsg, interface{}(from), uint64(blockHashesBatchSize))
}

func (self *ethProtocol) requestBlocks(hashes [][]byte) error {
	self.peer.Debugf("fetching %v blocks", len(hashes))
	return p2p.EncodeMsg(self.rw, GetBlocksMsg, ethutil.ByteSliceToInterface(hashes)...)
}

func (self *ethProtocol) protoError(code int, format string, params ...interface{}) (err *protocolError) {
	err = ProtocolError(code, format, params...)
	if err.Fatal() {
		self.peer.Errorln("err %v", err)
		// disconnect
	} else {
		self.peer.Debugf("fyi %v", err)
	}
	return
}

func (self *ethProtocol) protoErrorDisconnect(code int, format string, params ...interface{}) {
	err := ProtocolError(code, format, params...)
	if err.Fatal() {
		self.peer.Errorln("err %v", err)
		// disconnect
	} else {
		self.peer.Debugf("fyi %v", err)
	}

}

func (self *ethProtocol) propagateTxs() {
	transactions := self.txPool.GetTransactions()
	iface := make([]interface{}, len(transactions))
	for i, transaction := range transactions {
		iface[i] = transaction
	}

	self.rw.WriteMsg(p2p.NewMsg(TxMsg, iface...))
}
