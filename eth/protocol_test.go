package eth

import (
	"bytes"
	"io"
	"log"
	"math/big"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/errs"
	"github.com/ethereum/go-ethereum/ethutil"
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

type testMsgReadWriter struct {
	in  chan p2p.Msg
	out []p2p.Msg
}

func (self *testMsgReadWriter) In(msg p2p.Msg) {
	self.in <- msg
}

func (self *testMsgReadWriter) Out() (msg p2p.Msg, ok bool) {
	if len(self.out) > 0 {
		msg = self.out[0]
		self.out = self.out[1:]
		ok = true
	}
	return
}

func (self *testMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	self.out = append(self.out, msg)
	return nil
}

func (self *testMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	msg, ok := <-self.in
	if !ok {
		return msg, io.EOF
	}
	return msg, nil
}

type testTxPool struct {
	getTransactions func() []*types.Transaction
	addTransactions func(txs []*types.Transaction)
}

type testChainManager struct {
	getBlockHashes func(hash []byte, amount uint64) (hashes [][]byte)
	getBlock       func(hash []byte) *types.Block
	status         func() (td *big.Int, currentBlock []byte, genesisBlock []byte)
}

type testBlockPool struct {
	addBlockHashes func(next func() ([]byte, bool), peerId string)
	addBlock       func(block *types.Block, peerId string) (err error)
	addPeer        func(td *big.Int, currentBlock []byte, peerId string, requestHashes func([]byte) error, requestBlocks func([][]byte) error, peerError func(*errs.Error)) (best bool)
	removePeer     func(peerId string)
}

// func (self *testTxPool) GetTransactions() (txs []*types.Transaction) {
// 	if self.getTransactions != nil {
// 		txs = self.getTransactions()
// 	}
// 	return
// }

func (self *testTxPool) AddTransactions(txs []*types.Transaction) {
	if self.addTransactions != nil {
		self.addTransactions(txs)
	}
}

func (self *testTxPool) GetTransactions() types.Transactions { return nil }

func (self *testChainManager) GetBlockHashesFromHash(hash []byte, amount uint64) (hashes [][]byte) {
	if self.getBlockHashes != nil {
		hashes = self.getBlockHashes(hash, amount)
	}
	return
}

func (self *testChainManager) Status() (td *big.Int, currentBlock []byte, genesisBlock []byte) {
	if self.status != nil {
		td, currentBlock, genesisBlock = self.status()
	}
	return
}

func (self *testChainManager) GetBlock(hash []byte) (block *types.Block) {
	if self.getBlock != nil {
		block = self.getBlock(hash)
	}
	return
}

func (self *testBlockPool) AddBlockHashes(next func() ([]byte, bool), peerId string) {
	if self.addBlockHashes != nil {
		self.addBlockHashes(next, peerId)
	}
}

func (self *testBlockPool) AddBlock(block *types.Block, peerId string) {
	if self.addBlock != nil {
		self.addBlock(block, peerId)
	}
}

func (self *testBlockPool) AddPeer(td *big.Int, currentBlock []byte, peerId string, requestBlockHashes func([]byte) error, requestBlocks func([][]byte) error, peerError func(*errs.Error)) (best bool) {
	if self.addPeer != nil {
		best = self.addPeer(td, currentBlock, peerId, requestBlockHashes, requestBlocks, peerError)
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
	quit         chan error
	rw           *testMsgReadWriter // p2p.MsgReadWriter
	txPool       *testTxPool        // txPool
	chainManager *testChainManager  // chainManager
	blockPool    *testBlockPool     // blockPool
	t            *testing.T
}

func newEth(t *testing.T) *ethProtocolTester {
	return &ethProtocolTester{
		quit:         make(chan error),
		rw:           &testMsgReadWriter{in: make(chan p2p.Msg, 10)},
		txPool:       &testTxPool{},
		chainManager: &testChainManager{},
		blockPool:    &testBlockPool{},
		t:            t,
	}
}

func (self *ethProtocolTester) reset() {
	self.rw = &testMsgReadWriter{in: make(chan p2p.Msg, 10)}
	self.quit = make(chan error)
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

func (self *ethProtocolTester) In(msg p2p.Msg) {
	self.rw.In(msg)
}

func (self *ethProtocolTester) Out() (p2p.Msg, bool) {
	return self.rw.Out()
}

func (self *ethProtocolTester) checkMsg(i int, code uint64, val interface{}) (msg p2p.Msg) {
	if i >= len(self.rw.out) {
		self.t.Errorf("expected at least %v msgs, got %v", i, len(self.rw.out))
		return
	}
	msg = self.rw.out[i]
	if msg.Code != code {
		self.t.Errorf("expected msg code %v, got %v", code, msg.Code)
	}
	if val != nil {
		if err := msg.Decode(val); err != nil {
			self.t.Errorf("rlp encoding error: %v", err)
		}
	}
	return
}

func (self *ethProtocolTester) run() {
	err := runEthProtocol(self.txPool, self.chainManager, self.blockPool, testPeer(), self.rw)
	self.quit <- err
}

func TestStatusMsgErrors(t *testing.T) {
	logInit()
	eth := newEth(t)
	td := ethutil.Big1
	currentBlock := []byte{1}
	genesis := []byte{2}
	eth.chainManager.status = func() (*big.Int, []byte, []byte) { return td, currentBlock, genesis }
	go eth.run()
	statusMsg := p2p.NewMsg(4)
	eth.In(statusMsg)
	delay := 1 * time.Second
	eth.checkError(ErrNoStatusMsg, delay)
	var status statusMsgData
	eth.checkMsg(0, StatusMsg, &status) // first outgoing msg should be StatusMsg
	if status.TD.Cmp(td) != 0 ||
		status.ProtocolVersion != ProtocolVersion ||
		status.NetworkId != NetworkId ||
		status.TD.Cmp(td) != 0 ||
		bytes.Compare(status.CurrentBlock, currentBlock) != 0 ||
		bytes.Compare(status.GenesisBlock, genesis) != 0 {
		t.Errorf("incorrect outgoing status")
	}

	eth.reset()
	go eth.run()
	statusMsg = p2p.NewMsg(0, uint32(48), uint32(0), td, currentBlock, genesis)
	eth.In(statusMsg)
	eth.checkError(ErrProtocolVersionMismatch, delay)

	eth.reset()
	go eth.run()
	statusMsg = p2p.NewMsg(0, uint32(49), uint32(1), td, currentBlock, genesis)
	eth.In(statusMsg)
	eth.checkError(ErrNetworkIdMismatch, delay)

	eth.reset()
	go eth.run()
	statusMsg = p2p.NewMsg(0, uint32(49), uint32(0), td, currentBlock, []byte{3})
	eth.In(statusMsg)
	eth.checkError(ErrGenesisBlockMismatch, delay)

}
