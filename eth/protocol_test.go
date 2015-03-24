package eth

import (
	"log"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/errs"
	ethlogger "github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

var logsys = ethlogger.NewStdLogSystem(os.Stdout, log.LstdFlags, ethlogger.LogLevel(ethlogger.DebugDetailLevel))

var ini = false

func logInit() {
	if !ini {
		ethlogger.AddLogSystem(logsys)
		ini = true
	}
}

type testTxPool struct {
	getTransactions func() []*types.Transaction
	addTransactions func(txs []*types.Transaction)
}

type testChainManager struct {
	getBlockHashes func(hash common.Hash, amount uint64) (hashes []common.Hash)
	getBlock       func(hash common.Hash) *types.Block
	status         func() (td *big.Int, currentBlock common.Hash, genesisBlock common.Hash)
}

type testBlockPool struct {
	addBlockHashes func(next func() (common.Hash, bool), peerId string)
	addBlock       func(block *types.Block, peerId string) (err error)
	addPeer        func(td *big.Int, currentBlock common.Hash, peerId string, requestHashes func(common.Hash) error, requestBlocks func([]common.Hash) error, peerError func(*errs.Error)) (best bool, suspended bool)
	removePeer     func(peerId string)
}

func (self *testTxPool) AddTransactions(txs []*types.Transaction) {
	if self.addTransactions != nil {
		self.addTransactions(txs)
	}
}

func (self *testTxPool) GetTransactions() types.Transactions { return nil }

func (self *testChainManager) GetBlockHashesFromHash(hash common.Hash, amount uint64) (hashes []common.Hash) {
	if self.getBlockHashes != nil {
		hashes = self.getBlockHashes(hash, amount)
	}
	return
}

func (self *testChainManager) Status() (td *big.Int, currentBlock common.Hash, genesisBlock common.Hash) {
	if self.status != nil {
		td, currentBlock, genesisBlock = self.status()
	}
	return
}

func (self *testChainManager) GetBlock(hash common.Hash) (block *types.Block) {
	if self.getBlock != nil {
		block = self.getBlock(hash)
	}
	return
}

func (self *testBlockPool) AddBlockHashes(next func() (common.Hash, bool), peerId string) {
	if self.addBlockHashes != nil {
		self.addBlockHashes(next, peerId)
	}
}

func (self *testBlockPool) AddBlock(block *types.Block, peerId string) {
	if self.addBlock != nil {
		self.addBlock(block, peerId)
	}
}

func (self *testBlockPool) AddPeer(td *big.Int, currentBlock common.Hash, peerId string, requestBlockHashes func(common.Hash) error, requestBlocks func([]common.Hash) error, peerError func(*errs.Error)) (best bool, suspended bool) {
	if self.addPeer != nil {
		best, suspended = self.addPeer(td, currentBlock, peerId, requestBlockHashes, requestBlocks, peerError)
	}
	return
}

func (self *testBlockPool) RemovePeer(peerId string) {
	if self.removePeer != nil {
		self.removePeer(peerId)
	}
}

func testPeer() *p2p.Peer {
	var id discover.NodeID
	pk := crypto.GenerateNewKeyPair().PublicKey
	copy(id[:], pk)
	return p2p.NewPeer(id, "test peer", []p2p.Cap{})
}

type ethProtocolTester struct {
	p2p.MsgReadWriter // writing to the tester feeds the protocol

	quit         chan error
	pipe         *p2p.MsgPipeRW    // the protocol read/writes on this end
	txPool       *testTxPool       // txPool
	chainManager *testChainManager // chainManager
	blockPool    *testBlockPool    // blockPool
	t            *testing.T
}

func newEth(t *testing.T) *ethProtocolTester {
	p1, p2 := p2p.MsgPipe()
	return &ethProtocolTester{
		MsgReadWriter: p1,
		quit:          make(chan error, 1),
		pipe:          p2,
		txPool:        &testTxPool{},
		chainManager:  &testChainManager{},
		blockPool:     &testBlockPool{},
		t:             t,
	}
}

func (self *ethProtocolTester) reset() {
	self.pipe.Close()

	p1, p2 := p2p.MsgPipe()
	self.MsgReadWriter = p1
	self.pipe = p2
	self.quit = make(chan error, 1)
}

func (self *ethProtocolTester) checkError(expCode int, delay time.Duration) (err error) {
	var timer = time.After(delay)
	select {
	case err = <-self.quit:
	case <-timer:
		self.t.Errorf("no error after %v, expected %v", delay, expCode)
		return
	}
	perr, ok := err.(*errs.Error)
	if ok && perr != nil {
		if code := perr.Code; code != expCode {
			self.t.Errorf("expected protocol error (code %v), got %v (%v)", expCode, code, err)
		}
	} else {
		self.t.Errorf("expected protocol error (code %v), got %v", expCode, err)
	}
	return
}

func (self *ethProtocolTester) run() {
	err := runEthProtocol(ProtocolVersion, NetworkId, self.txPool, self.chainManager, self.blockPool, testPeer(), self.pipe)
	self.quit <- err
}

func TestStatusMsgErrors(t *testing.T) {
	logInit()
	eth := newEth(t)
	td := common.Big1
	currentBlock := common.Hash{1}
	genesis := common.Hash{2}
	eth.chainManager.status = func() (*big.Int, common.Hash, common.Hash) { return td, currentBlock, genesis }
	go eth.run()

	tests := []struct {
		code          uint64
		data          interface{}
		wantErrorCode int
	}{
		{
			code: TxMsg, data: []interface{}{},
			wantErrorCode: ErrNoStatusMsg,
		},
		{
			code: StatusMsg, data: statusMsgData{10, NetworkId, td, currentBlock, genesis},
			wantErrorCode: ErrProtocolVersionMismatch,
		},
		{
			code: StatusMsg, data: statusMsgData{ProtocolVersion, 999, td, currentBlock, genesis},
			wantErrorCode: ErrNetworkIdMismatch,
		},
		{
			code: StatusMsg, data: statusMsgData{ProtocolVersion, NetworkId, td, currentBlock, common.Hash{3}},
			wantErrorCode: ErrGenesisBlockMismatch,
		},
	}
	for _, test := range tests {
		// first outgoing msg should be StatusMsg.
		err := p2p.ExpectMsg(eth, StatusMsg, &statusMsgData{
			ProtocolVersion: ProtocolVersion,
			NetworkId:       NetworkId,
			TD:              td,
			CurrentBlock:    currentBlock,
			GenesisBlock:    genesis,
		})
		if err != nil {
			t.Fatalf("incorrect outgoing status: %v", err)
		}

		// the send call might hang until reset because
		// the protocol might not read the payload.
		go p2p.Send(eth, test.code, test.data)
		eth.checkError(test.wantErrorCode, 1*time.Second)

		eth.reset()
		go eth.run()
	}
}
