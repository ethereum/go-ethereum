// This file contains some shares testing functionality, common to  multiple
// different files and modules being tested.

package eth

import (
	"crypto/rand"
	"math/big"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

var (
	testBankKey, _  = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	testBankAddress = crypto.PubkeyToAddress(testBankKey.PublicKey)
	testBankFunds   = big.NewInt(1000000)
)

// newTestProtocolManager creates a new protocol manager for testing purposes,
// with the given number of blocks already known, and potential notification
// channels for different events.
func newTestProtocolManager(blocks int, generator func(int, *core.BlockGen), newtx chan<- []*types.Transaction) *ProtocolManager {
	var (
		evmux         = new(event.TypeMux)
		pow           = new(core.FakePow)
		db, _         = ethdb.NewMemDatabase()
		genesis       = core.WriteGenesisBlockForTesting(db, core.GenesisAccount{testBankAddress, testBankFunds})
		blockchain, _ = core.NewBlockChain(db, pow, evmux)
		blockproc     = core.NewBlockProcessor(db, pow, blockchain, evmux)
	)
	blockchain.SetProcessor(blockproc)
	if _, err := blockchain.InsertChain(core.GenerateChain(genesis, db, blocks, generator)); err != nil {
		panic(err)
	}
	pm := NewProtocolManager(NetworkId, evmux, &testTxPool{added: newtx}, pow, blockchain, db)
	pm.Start()
	return pm
}

// testTxPool is a fake, helper transaction pool for testing purposes
type testTxPool struct {
	pool  []*types.Transaction        // Collection of all transactions
	added chan<- []*types.Transaction // Notification channel for new transactions

	lock sync.RWMutex // Protects the transaction pool
}

// AddTransactions appends a batch of transactions to the pool, and notifies any
// listeners if the addition channel is non nil
func (p *testTxPool) AddTransactions(txs []*types.Transaction) {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.pool = append(p.pool, txs...)
	if p.added != nil {
		p.added <- txs
	}
}

// GetTransactions returns all the transactions known to the pool
func (p *testTxPool) GetTransactions() types.Transactions {
	p.lock.RLock()
	defer p.lock.RUnlock()

	txs := make([]*types.Transaction, len(p.pool))
	copy(txs, p.pool)

	return txs
}

// newTestTransaction create a new dummy transaction.
func newTestTransaction(from *crypto.Key, nonce uint64, datasize int) *types.Transaction {
	tx := types.NewTransaction(nonce, common.Address{}, big.NewInt(0), big.NewInt(100000), big.NewInt(0), make([]byte, datasize))
	tx, _ = tx.SignECDSA(from.PrivateKey)

	return tx
}

// testPeer is a simulated peer to allow testing direct network calls.
type testPeer struct {
	net p2p.MsgReadWriter // Network layer reader/writer to simulate remote messaging
	app *p2p.MsgPipeRW    // Application layer reader/writer to simulate the local side
	*peer
}

// newTestPeer creates a new peer registered at the given protocol manager.
func newTestPeer(name string, version int, pm *ProtocolManager, shake bool) (*testPeer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Generate a random id and create the peer
	var id discover.NodeID
	rand.Read(id[:])

	peer := pm.newPeer(version, NetworkId, p2p.NewPeer(id, name, nil), net)

	// Start the peer on a new thread
	errc := make(chan error, 1)
	go func() {
		pm.newPeerCh <- peer
		errc <- pm.handle(peer)
	}()
	tp := &testPeer{
		app:  app,
		net:  net,
		peer: peer,
	}
	// Execute any implicitly requested handshakes and return
	if shake {
		td, head, genesis := pm.blockchain.Status()
		tp.handshake(nil, td, head, genesis)
	}
	return tp, errc
}

// handshake simulates a trivial handshake that expects the same state from the
// remote side as we are simulating locally.
func (p *testPeer) handshake(t *testing.T, td *big.Int, head common.Hash, genesis common.Hash) {
	msg := &statusData{
		ProtocolVersion: uint32(p.version),
		NetworkId:       uint32(NetworkId),
		TD:              td,
		CurrentBlock:    head,
		GenesisBlock:    genesis,
	}
	if err := p2p.ExpectMsg(p.app, StatusMsg, msg); err != nil {
		t.Fatalf("status recv: %v", err)
	}
	if err := p2p.Send(p.app, StatusMsg, msg); err != nil {
		t.Fatalf("status send: %v", err)
	}
}

// close terminates the local side of the peer, notifying the remote protocol
// manager of termination.
func (p *testPeer) close() {
	p.app.Close()
}
