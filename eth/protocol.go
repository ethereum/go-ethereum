package eth

import (
	"bytes"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	ethlogger "github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
)

var logger = ethlogger.NewLogger("SERV")

// ethProtocol represents the ethereum wire protocol
// instance is running on each peer
type ethProtocol struct {
	eth  backend
	td   *big.Int
	peer *p2p.Peer
	rw   p2p.MsgReadWriter
}

// backend is the interface the ethereum protocol backend should implement
// used as an argument to EthProtocol
type backend interface {
	GetTransactions() (txs []*types.Transaction)
	AddTransactions(txs []*types.Transaction)
	GetBlockHashes(hash []byte, amount uint32) (hashes [][]byte)
	AddHash(hash []byte, peer *p2p.Peer) (more bool)
	GetBlock(hash []byte) (block *types.Block)
	AddBlock(td *big.Int, block *types.Block, peer *p2p.Peer) (fetchHashes bool, err error)
	AddPeer(td *big.Int, currentBlock []byte, peer *p2p.Peer) (fetchHashes bool)
	Status() (td *big.Int, currentBlock []byte, genesisBlock []byte)
}

const (
	ProtocolVersion = 43
	// 0x00 // PoC-1
	// 0x01 // PoC-2
	// 0x07 // PoC-3
	// 0x09 // PoC-4
	// 0x17 // PoC-5
	// 0x1c // PoC-6
	NetworkId          = 0
	ProtocolLength     = uint64(8)
	ProtocolMaxMsgSize = 10 * 1024 * 1024

	blockHashesBatchSize = 256
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

// message structs used for rlp decoding
type newBlockMsgData struct {
	Block *types.Block
	TD    *big.Int
}

type getBlockHashesMsgData struct {
	Hash   []byte
	Amount uint32
}

// main entrypoint, wrappers starting a server running the eth protocol
// use this constructor to attach the protocol (class) to server caps
func EthProtocol(eth backend) *p2p.Protocol {
	return &p2p.Protocol{
		Name:    "eth",
		Version: ProtocolVersion,
		Length:  ProtocolLength,
		Run: func(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
			return runEthProtocol(eth, peer, rw)
		},
	}
}

func runEthProtocol(eth backend, peer *p2p.Peer, rw p2p.MsgReadWriter) (err error) {
	self := &ethProtocol{
		eth:  eth,
		rw:   rw,
		peer: peer,
	}
	err = self.handleStatus()
	if err == nil {
		go func() {
			for {
				err = self.handle()
				if err != nil {
					break
				}
			}
		}()
	}
	return
}

func (self *ethProtocol) handle() error {
	msg, err := self.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return ProtocolError(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// make sure that the payload has been fully consumed
	defer msg.Discard()

	switch msg.Code {

	case StatusMsg:
		return ProtocolError(ErrExtraStatusMsg, "")

	case GetTxMsg:
		txs := self.eth.GetTransactions()
		// TODO: rewrite using rlp flat
		txsInterface := make([]interface{}, len(txs))
		for i, tx := range txs {
			txsInterface[i] = tx.RlpData()
		}
		return self.rw.EncodeMsg(TxMsg, txsInterface...)

	case TxMsg:
		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return ProtocolError(ErrDecode, "%v", err)
		}
		self.eth.AddTransactions(txs)

	case GetBlockHashesMsg:
		var request getBlockHashesMsgData
		if err := msg.Decode(&request); err != nil {
			return ProtocolError(ErrDecode, "%v", err)
		}
		hashes := self.eth.GetBlockHashes(request.Hash, request.Amount)
		return self.rw.EncodeMsg(BlockHashesMsg, ethutil.ByteSliceToInterface(hashes)...)

	case BlockHashesMsg:
		// TODO: redo using lazy decode , this way very inefficient on known chains
		// s := rlp.NewListStream(msg.Payload, uint64(msg.Size))
		var blockHashes [][]byte
		if err := msg.Decode(&blockHashes); err != nil {
			return ProtocolError(ErrDecode, "%v", err)
		}
		fetchMore := true
		for _, hash := range blockHashes {
			fetchMore = self.eth.AddHash(hash, self.peer)
			if !fetchMore {
				break
			}
		}
		if fetchMore {
			return self.FetchHashes(blockHashes[len(blockHashes)-1])
		}

	case GetBlocksMsg:
		// Limit to max 300 blocks
		var blockHashes [][]byte
		if err := msg.Decode(&blockHashes); err != nil {
			return ProtocolError(ErrDecode, "%v", err)
		}
		max := int(math.Min(float64(len(blockHashes)), 300.0))
		var blocks []interface{}
		for i, hash := range blockHashes {
			if i >= max {
				break
			}
			block := self.eth.GetBlock(hash)
			if block != nil {
				blocks = append(blocks, block.Value().Raw())
			}
		}
		return self.rw.EncodeMsg(BlocksMsg, blocks...)

	case BlocksMsg:
		var blocks []*types.Block
		if err := msg.Decode(&blocks); err != nil {
			return ProtocolError(ErrDecode, "%v", err)
		}
		for _, block := range blocks {
			fetchHashes, err := self.eth.AddBlock(nil, block, self.peer)
			if err != nil {
				return ProtocolError(ErrInvalidBlock, "%v", err)
			}
			if fetchHashes {
				if err := self.FetchHashes(block.Hash()); err != nil {
					return err
				}
			}
		}

	case NewBlockMsg:
		var request newBlockMsgData
		if err := msg.Decode(&request); err != nil {
			return ProtocolError(ErrDecode, "%v", err)
		}
		var fetchHashes bool
		// this should reset td and offer blockpool as candidate new peer?
		if fetchHashes, err = self.eth.AddBlock(request.TD, request.Block, self.peer); err != nil {
			return ProtocolError(ErrInvalidBlock, "%v", err)
		}
		if fetchHashes {
			return self.FetchHashes(request.Block.Hash())
		}

	default:
		return ProtocolError(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

type statusMsgData struct {
	ProtocolVersion uint
	NetworkId       uint
	TD              *big.Int
	CurrentBlock    []byte
	GenesisBlock    []byte
}

func (self *ethProtocol) statusMsg() p2p.Msg {
	td, currentBlock, genesisBlock := self.eth.Status()

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
		return ProtocolError(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, StatusMsg)
	}

	if msg.Size > ProtocolMaxMsgSize {
		return ProtocolError(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}

	var status statusMsgData
	if err := msg.Decode(&status); err != nil {
		return ProtocolError(ErrDecode, "%v", err)
	}

	_, _, genesisBlock := self.eth.Status()

	if bytes.Compare(status.GenesisBlock, genesisBlock) != 0 {
		return ProtocolError(ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock, genesisBlock)
	}

	if status.NetworkId != NetworkId {
		return ProtocolError(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, NetworkId)
	}

	if ProtocolVersion != status.ProtocolVersion {
		return ProtocolError(ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, ProtocolVersion)
	}

	logger.Infof("Peer is [eth] capable (%d/%d). TD = %v ~ %x", status.ProtocolVersion, status.NetworkId, status.CurrentBlock)

	if self.eth.AddPeer(status.TD, status.CurrentBlock, self.peer) {
		return self.FetchHashes(status.CurrentBlock)
	}

	return nil
}

func (self *ethProtocol) FetchHashes(from []byte) error {
	logger.Debugf("Fetching hashes (%d) %x...\n", blockHashesBatchSize, from[0:4])
	return self.rw.EncodeMsg(GetBlockHashesMsg, from, blockHashesBatchSize)
}
