package eth

import (
	"io"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
)

type testMsgReadWriter struct {
	in  chan p2p.Msg
	out chan p2p.Msg
}

func (self *testMsgReadWriter) In(msg p2p.Msg) {
	self.in <- msg
}

func (self *testMsgReadWriter) Out(msg p2p.Msg) {
	self.in <- msg
}

func (self *testMsgReadWriter) WriteMsg(msg p2p.Msg) error {
	self.out <- msg
	return nil
}

func (self *testMsgReadWriter) EncodeMsg(code uint64, data ...interface{}) error {
	return self.WriteMsg(p2p.NewMsg(code, data))
}

func (self *testMsgReadWriter) ReadMsg() (p2p.Msg, error) {
	msg, ok := <-self.in
	if !ok {
		return msg, io.EOF
	}
	return msg, nil
}

func errorCheck(t *testing.T, expCode int, err error) {
	perr, ok := err.(*protocolError)
	if ok && perr != nil {
		if code := perr.Code; code != expCode {
			ok = false
		}
	}
	if !ok {
		t.Errorf("expected error code %v, got %v", ErrNoStatusMsg, err)
	}
}

type TestBackend struct {
	getTransactions func() []*types.Transaction
	addTransactions func(txs []*types.Transaction)
	getBlockHashes  func(hash []byte, amount uint32) (hashes [][]byte)
<<<<<<< HEAD
	addBlockHashes  func(next func() ([]byte, bool), peerId string)
	getBlock        func(hash []byte) *types.Block
	addBlock        func(block *types.Block, peerId string) (err error)
	addPeer         func(td *big.Int, currentBlock []byte, peerId string, requestHashes func([]byte) error, requestBlocks func([][]byte) error, invalidBlock func(error)) (best bool)
	removePeer      func(peerId string)
=======
	addHash         func(hash []byte, peer *p2p.Peer) (more bool)
	getBlock        func(hash []byte) *types.Block
	addBlock        func(td *big.Int, block *types.Block, peer *p2p.Peer) (fetchHashes bool, err error)
	addPeer         func(td *big.Int, currentBlock []byte, peer *p2p.Peer) (fetchHashes bool)
>>>>>>> initial commit for eth-p2p integration
	status          func() (td *big.Int, currentBlock []byte, genesisBlock []byte)
}

func (self *TestBackend) GetTransactions() (txs []*types.Transaction) {
	if self.getTransactions != nil {
		txs = self.getTransactions()
	}
	return
}

func (self *TestBackend) AddTransactions(txs []*types.Transaction) {
	if self.addTransactions != nil {
		self.addTransactions(txs)
	}
}

func (self *TestBackend) GetBlockHashes(hash []byte, amount uint32) (hashes [][]byte) {
	if self.getBlockHashes != nil {
		hashes = self.getBlockHashes(hash, amount)
	}
	return
}

<<<<<<< HEAD
func (self *TestBackend) AddBlockHashes(next func() ([]byte, bool), peerId string) {
	if self.addBlockHashes != nil {
		self.addBlockHashes(next, peerId)
	}
}

=======
func (self *TestBackend) AddHash(hash []byte, peer *p2p.Peer) (more bool) {
	if self.addHash != nil {
		more = self.addHash(hash, peer)
	}
	return
}
>>>>>>> initial commit for eth-p2p integration
func (self *TestBackend) GetBlock(hash []byte) (block *types.Block) {
	if self.getBlock != nil {
		block = self.getBlock(hash)
	}
	return
}

<<<<<<< HEAD
func (self *TestBackend) AddBlock(block *types.Block, peerId string) (err error) {
	if self.addBlock != nil {
		err = self.addBlock(block, peerId)
=======
func (self *TestBackend) AddBlock(td *big.Int, block *types.Block, peer *p2p.Peer) (fetchHashes bool, err error) {
	if self.addBlock != nil {
		fetchHashes, err = self.addBlock(td, block, peer)
>>>>>>> initial commit for eth-p2p integration
	}
	return
}

<<<<<<< HEAD
func (self *TestBackend) AddPeer(td *big.Int, currentBlock []byte, peerId string, requestBlockHashes func([]byte) error, requestBlocks func([][]byte) error, invalidBlock func(error)) (best bool) {
	if self.addPeer != nil {
		best = self.addPeer(td, currentBlock, peerId, requestBlockHashes, requestBlocks, invalidBlock)
=======
func (self *TestBackend) AddPeer(td *big.Int, currentBlock []byte, peer *p2p.Peer) (fetchHashes bool) {
	if self.addPeer != nil {
		fetchHashes = self.addPeer(td, currentBlock, peer)
>>>>>>> initial commit for eth-p2p integration
	}
	return
}

<<<<<<< HEAD
func (self *TestBackend) RemovePeer(peerId string) {
	if self.removePeer != nil {
		self.removePeer(peerId)
	}
}

=======
>>>>>>> initial commit for eth-p2p integration
func (self *TestBackend) Status() (td *big.Int, currentBlock []byte, genesisBlock []byte) {
	if self.status != nil {
		td, currentBlock, genesisBlock = self.status()
	}
	return
}

<<<<<<< HEAD
// TODO: refactor this into p2p/client_identity
type peerId struct {
	pubkey []byte
}

func (self *peerId) String() string {
	return "test peer"
}

func (self *peerId) Pubkey() (pubkey []byte) {
	pubkey = self.pubkey
	if len(pubkey) == 0 {
		pubkey = crypto.GenerateNewKeyPair().PublicKey
		self.pubkey = pubkey
	}
	return
}

func testPeer() *p2p.Peer {
	return p2p.NewPeer(&peerId{}, []p2p.Cap{})
}

func TestErrNoStatusMsg(t *testing.T) {
=======
func TestEth(t *testing.T) {
>>>>>>> initial commit for eth-p2p integration
	quit := make(chan bool)
	rw := &testMsgReadWriter{make(chan p2p.Msg, 10), make(chan p2p.Msg, 10)}
	testBackend := &TestBackend{}
	var err error
	go func() {
<<<<<<< HEAD
		err = runEthProtocol(testBackend, testPeer(), rw)
=======
		err = runEthProtocol(testBackend, nil, rw)
>>>>>>> initial commit for eth-p2p integration
		close(quit)
	}()
	statusMsg := p2p.NewMsg(4)
	rw.In(statusMsg)
	<-quit
	errorCheck(t, ErrNoStatusMsg, err)
	// read(t, remote, []byte("hello, world"), nil)
}
