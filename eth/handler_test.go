// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"maps"
	"math/big"
	"math/rand"
	"sort"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
)

var (
	// testKey is a private key to use for funding a tester account.
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

	// testAddr is the Ethereum address of the tester account.
	testAddr = crypto.PubkeyToAddress(testKey.PublicKey)
)

// testTxPool is a mock transaction pool that blindly accepts all transactions.
// Its goal is to get around setting up a valid statedb for the balance and nonce
// checks.
type testTxPool struct {
	pool map[common.Hash]*types.Transaction // Hash map of collected transactions

	txFeed event.Feed   // Notification feed to allow waiting for inclusion
	lock   sync.RWMutex // Protects the transaction pool
}

// newTestTxPool creates a mock transaction pool.
func newTestTxPool() *testTxPool {
	return &testTxPool{
		pool: make(map[common.Hash]*types.Transaction),
	}
}

// Has returns an indicator whether txpool has a transaction
// cached with the given hash.
func (p *testTxPool) Has(hash common.Hash) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.pool[hash] != nil
}

// Get retrieves the transaction from local txpool with given
// tx hash.
func (p *testTxPool) Get(hash common.Hash) *types.Transaction {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.pool[hash]
}

// Get retrieves the transaction from local txpool with given
// tx hash.
func (p *testTxPool) GetRLP(hash common.Hash) []byte {
	p.lock.Lock()
	defer p.lock.Unlock()

	tx := p.pool[hash]
	if tx != nil {
		blob, _ := rlp.EncodeToBytes(tx)
		return blob
	}
	return nil
}

// GetMetadata returns the transaction type and transaction size with the given
// hash.
func (p *testTxPool) GetMetadata(hash common.Hash) *txpool.TxMetadata {
	p.lock.Lock()
	defer p.lock.Unlock()

	tx := p.pool[hash]
	if tx != nil {
		return &txpool.TxMetadata{
			Type: tx.Type(),
			Size: tx.Size(),
		}
	}
	return nil
}

// Add appends a batch of transactions to the pool, and notifies any
// listeners if the addition channel is non nil
func (p *testTxPool) Add(txs []*types.Transaction, sync bool) []error {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, tx := range txs {
		p.pool[tx.Hash()] = tx
	}
	p.txFeed.Send(core.NewTxsEvent{Txs: txs})
	return make([]error, len(txs))
}

// Pending returns all the transactions known to the pool
func (p *testTxPool) Pending(filter txpool.PendingFilter) map[common.Address][]*txpool.LazyTransaction {
	p.lock.RLock()
	defer p.lock.RUnlock()

	batches := make(map[common.Address][]*types.Transaction)
	for _, tx := range p.pool {
		from, _ := types.Sender(types.HomesteadSigner{}, tx)
		batches[from] = append(batches[from], tx)
	}
	for _, batch := range batches {
		sort.Sort(types.TxByNonce(batch))
	}
	pending := make(map[common.Address][]*txpool.LazyTransaction)
	for addr, batch := range batches {
		for _, tx := range batch {
			pending[addr] = append(pending[addr], &txpool.LazyTransaction{
				Hash:      tx.Hash(),
				Tx:        tx,
				Time:      tx.Time(),
				GasFeeCap: uint256.MustFromBig(tx.GasFeeCap()),
				GasTipCap: uint256.MustFromBig(tx.GasTipCap()),
				Gas:       tx.Gas(),
				BlobGas:   tx.BlobGas(),
			})
		}
	}
	return pending
}

// SubscribeTransactions should return an event subscription of NewTxsEvent and
// send events to the given channel.
func (p *testTxPool) SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription {
	return p.txFeed.Subscribe(ch)
}

// testHandler is a live implementation of the Ethereum protocol handler, just
// preinitialized with some sane testing defaults and the transaction pool mocked
// out.
type testHandler struct {
	db      ethdb.Database
	chain   *core.BlockChain
	txpool  *testTxPool
	handler *handler
}

// newTestHandler creates a new handler for testing purposes with no blocks.
func newTestHandler() *testHandler {
	return newTestHandlerWithBlocks(0)
}

// newTestHandlerWithBlocks creates a new handler for testing purposes, with a
// given number of initial blocks.
func newTestHandlerWithBlocks(blocks int) *testHandler {
	// Create a database pre-initialize with a genesis block
	db := rawdb.NewMemoryDatabase()
	gspec := &core.Genesis{
		Config: params.TestChainConfig,
		Alloc:  types.GenesisAlloc{testAddr: {Balance: big.NewInt(1000000)}},
	}
	chain, _ := core.NewBlockChain(db, gspec, ethash.NewFaker(), nil)

	_, bs, _ := core.GenerateChainWithGenesis(gspec, ethash.NewFaker(), blocks, nil)
	if _, err := chain.InsertChain(bs); err != nil {
		panic(err)
	}
	txpool := newTestTxPool()

	handler, _ := newHandler(&handlerConfig{
		Database:   db,
		Chain:      chain,
		TxPool:     txpool,
		Network:    1,
		Sync:       ethconfig.SnapSync,
		BloomCache: 1,
	})
	handler.Start(1000)

	return &testHandler{
		db:      db,
		chain:   chain,
		txpool:  txpool,
		handler: handler,
	}
}

// close tears down the handler and all its internal constructs.
func (b *testHandler) close() {
	b.handler.Stop()
	b.chain.Stop()
}

func TestBroadcastChoice(t *testing.T) {
	self := enode.HexID("1111111111111111111111111111111111111111111111111111111111111111")
	choice49 := newBroadcastChoice(self, [16]byte{1})
	choice50 := newBroadcastChoice(self, [16]byte{1})

	// Create test peers and random tx sender addresses.
	rand := rand.New(rand.NewSource(33))
	txsenders := make([]common.Address, 400)
	for i := range txsenders {
		rand.Read(txsenders[i][:])
	}
	peers := createTestPeers(rand, 50)
	defer closePeers(peers)

	// Evaluate choice49 first.
	expectedCount := 7 // sqrt(49)
	var chosen49 = make([]map[*ethPeer]struct{}, len(txsenders))
	for i, txSender := range txsenders {
		set := choice49.choosePeers(peers[:49], txSender)
		chosen49[i] = maps.Clone(set)

		// Sanity check choices. Here we check that the function selects different peers
		// for different transaction senders.
		if len(set) != expectedCount {
			t.Fatalf("choice49 produced wrong count %d, want %d", len(set), expectedCount)
		}
		if i > 0 && maps.Equal(set, chosen49[i-1]) {
			t.Errorf("choice49 for tx %d is equal to tx %d", i, i-1)
		}
	}

	// Evaluate choice50 for the same peers and transactions. It should always yield more
	// peers than choice49, and the chosen set should be a superset of choice49's.
	for i, txSender := range txsenders {
		set := choice50.choosePeers(peers[:50], txSender)
		if len(set) < len(chosen49[i]) {
			t.Errorf("for tx %d, choice50 has less peers than choice49", i)
		}
		for p := range chosen49[i] {
			if _, ok := set[p]; !ok {
				t.Errorf("for tx %d, choice50 did not choose peer %v, but choice49 did", i, p.ID())
			}
		}
	}
}

func BenchmarkBroadcastChoice(b *testing.B) {
	b.Run("50", func(b *testing.B) {
		benchmarkBroadcastChoice(b, 50)
	})
	b.Run("200", func(b *testing.B) {
		benchmarkBroadcastChoice(b, 200)
	})
	b.Run("500", func(b *testing.B) {
		benchmarkBroadcastChoice(b, 500)
	})
}

// This measures the overhead of sending one transaction to N peers.
func benchmarkBroadcastChoice(b *testing.B, npeers int) {
	rand := rand.New(rand.NewSource(33))
	peers := createTestPeers(rand, npeers)
	defer closePeers(peers)

	txsenders := make([]common.Address, b.N)
	for i := range txsenders {
		rand.Read(txsenders[i][:])
	}

	self := enode.HexID("1111111111111111111111111111111111111111111111111111111111111111")
	choice := newBroadcastChoice(self, [16]byte{1})

	b.ResetTimer()
	for i := range b.N {
		set := choice.choosePeers(peers, txsenders[i])
		if len(set) == 0 {
			b.Fatal("empty result")
		}
	}
}

func createTestPeers(rand *rand.Rand, n int) []*ethPeer {
	peers := make([]*ethPeer, n)
	for i := range peers {
		var id enode.ID
		rand.Read(id[:])
		p2pPeer := p2p.NewPeer(id, "test", nil)
		ep := eth.NewPeer(eth.ETH69, p2pPeer, nil, nil)
		peers[i] = &ethPeer{Peer: ep}
	}
	return peers
}

func closePeers(peers []*ethPeer) {
	for _, p := range peers {
		p.Close()
	}
}
