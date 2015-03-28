package eth

import (
	"fmt"
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
	} else {
		td = common.Big1
		currentBlock = common.Hash{1}
		genesisBlock = common.Hash{2}
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

func (self *ethProtocolTester) handshake(t *testing.T, mock bool) {
	td, currentBlock, genesis := self.chainManager.Status()
	// first outgoing msg should be StatusMsg.
	err := p2p.ExpectMsg(self, StatusMsg, &statusMsgData{
		ProtocolVersion: ProtocolVersion,
		NetworkId:       NetworkId,
		TD:              *td,
		CurrentBlock:    currentBlock,
		GenesisBlock:    genesis,
	})
	if err != nil {
		t.Fatalf("incorrect outgoing status: %v", err)
	}
	if mock {
		go p2p.Send(self, StatusMsg, &statusMsgData{ProtocolVersion, NetworkId, *td, currentBlock, genesis})
	}
}

func TestStatusMsgErrors(t *testing.T) {
	logInit()
	eth := newEth(t)
	go eth.run()
	td, currentBlock, genesis := eth.chainManager.Status()

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
			code: StatusMsg, data: statusMsgData{10, NetworkId, *td, currentBlock, genesis},
			wantErrorCode: ErrProtocolVersionMismatch,
		},
		{
			code: StatusMsg, data: statusMsgData{ProtocolVersion, 999, *td, currentBlock, genesis},
			wantErrorCode: ErrNetworkIdMismatch,
		},
		{
			code: StatusMsg, data: statusMsgData{ProtocolVersion, NetworkId, *td, currentBlock, common.Hash{3}},
			wantErrorCode: ErrGenesisBlockMismatch,
		},
	}
	for _, test := range tests {
		eth.handshake(t, false)
		// the send call might hang until reset because
		// the protocol might not read the payload.
		go p2p.Send(eth, test.code, test.data)
		eth.checkError(test.wantErrorCode, 1*time.Second)

		eth.reset()
		go eth.run()
	}
}

func TestNewBlockMsg(t *testing.T) {
	logInit()
	eth := newEth(t)
	eth.blockPool.addBlock = func(block *types.Block, peerId string) (err error) {
		fmt.Printf("Add Block: %v\n", block)
		return
	}

	var disconnected bool
	eth.blockPool.removePeer = func(peerId string) {
		fmt.Printf("peer <%s> is disconnected\n", peerId)
		disconnected = true
	}

	go eth.run()

	eth.handshake(t, true)
	err := p2p.ExpectMsg(eth, TxMsg, []interface{}{})
	if err != nil {
		t.Errorf("transactions expected, got %v", err)
	}

	var tds = make(chan *big.Int)
	eth.blockPool.addPeer = func(td *big.Int, currentBlock common.Hash, peerId string, requestHashes func(common.Hash) error, requestBlocks func([]common.Hash) error, peerError func(*errs.Error)) (best bool, suspended bool) {
		tds <- td
		return
	}

	var delay = 1 * time.Second
	// eth.reset()
	block := types.NewBlock(common.Hash{1}, common.Address{1}, common.Hash{1}, common.Big1, 1, "extra")

	go p2p.Send(eth, NewBlockMsg, &newBlockMsgData{Block: block})
	timer := time.After(delay)

	select {
	case td := <-tds:
		if td.Cmp(common.Big0) != 0 {
			t.Errorf("incorrect td %v, expected %v", td, common.Big0)
		}
	case <-timer:
		t.Errorf("no td recorded after %v", delay)
		return
	case err := <-eth.quit:
		t.Errorf("no error expected, got %v", err)
		return
	}

	go p2p.Send(eth, NewBlockMsg, &newBlockMsgData{block, common.Big2})
	timer = time.After(delay)

	select {
	case td := <-tds:
		if td.Cmp(common.Big2) != 0 {
			t.Errorf("incorrect td %v, expected %v", td, common.Big2)
		}
	case <-timer:
		t.Errorf("no td recorded after %v", delay)
		return
	case err := <-eth.quit:
		t.Errorf("no error expected, got %v", err)
		return
	}

	go p2p.Send(eth, NewBlockMsg, []interface{}{})
	// Block.DecodeRLP: validation failed: header is nil
	eth.checkError(ErrDecode, delay)

}
