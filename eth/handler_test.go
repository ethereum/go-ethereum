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
	"cmp"
	"crypto/ecdsa"
	"fmt"
	"maps"
	"math"
	"math/big"
	"math/rand"
	"slices"
	"sort"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
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
	txPool   map[common.Hash]*types.Transaction // Hash map of collected transactions
	cellPool map[common.Hash][]kzg4844.Cell

	custody map[common.Hash]types.CustodyBitmap

	txFeed event.Feed   // Notification feed to allow waiting for inclusion
	lock   sync.RWMutex // Protects the transaction pool
}

// newTestTxPool creates a mock transaction pool.
func newTestTxPool() *testTxPool {
	return &testTxPool{
		txPool:   make(map[common.Hash]*types.Transaction),
		cellPool: make(map[common.Hash][]kzg4844.Cell),
		custody:  make(map[common.Hash]types.CustodyBitmap),
	}
}

// Has returns an indicator whether txpool has a transaction
// cached with the given hash.
func (p *testTxPool) Has(hash common.Hash) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.txPool[hash] != nil
}

// Has returns an indicator whether txpool has a transaction
// cached with the given hash.
func (p *testTxPool) HasPayload(hash common.Hash) bool {
	p.lock.Lock()
	defer p.lock.Unlock()

	return p.cellPool[hash] != nil
}

// Get retrieves the transaction from local txpool with given
// tx hash.
func (p *testTxPool) Get(hash common.Hash) *types.Transaction {
	p.lock.Lock()
	defer p.lock.Unlock()
	return p.txPool[hash]
}

// Get retrieves the transaction from local txpool with given
// tx hash.
func (p *testTxPool) GetRLP(hash common.Hash, _ uint) []byte {
	p.lock.Lock()
	defer p.lock.Unlock()

	tx := p.txPool[hash]
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

	tx := p.txPool[hash]
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
		p.txPool[tx.Hash()] = tx
	}
	p.txFeed.Send(core.NewTxsEvent{Txs: txs})
	return make([]error, len(txs))
}

func (p *testTxPool) Remove(txs ...*types.Transaction) {
	p.lock.Lock()
	defer p.lock.Unlock()

	for _, tx := range txs {
		delete(p.txPool, tx.Hash())
	}
}

// Pending returns all the transactions known to the pool
func (p *testTxPool) Pending(filter txpool.PendingFilter) (map[common.Address][]*txpool.LazyTransaction, int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	var count int
	batches := make(map[common.Address][]*types.Transaction)
	for _, tx := range p.txPool {
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
			count++
		}
	}
	return pending, count
}

// SubscribeTransactions should return an event subscription of NewTxsEvent and
// send events to the given channel.
func (p *testTxPool) SubscribeTransactions(ch chan<- core.NewTxsEvent, reorgs bool) event.Subscription {
	return p.txFeed.Subscribe(ch)
}
func (p *testTxPool) GetBlobHashes(hash common.Hash) []common.Hash {
	p.lock.RLock()
	defer p.lock.RUnlock()

	tx, exists := p.txPool[hash]
	if !exists {
		return nil
	}
	return tx.BlobHashes()
}

func (p *testTxPool) GetBlobCells(vhashes []common.Hash, mask types.CustodyBitmap) ([][]*kzg4844.Cell, [][]*kzg4844.Proof, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	requestedIndices := mask.Indices()
	cells := make([][]*kzg4844.Cell, len(vhashes))
	proofs := make([][]*kzg4844.Proof, len(vhashes))

	for i, vhash := range vhashes {
		// Find the tx containing this versioned hash
		var foundTx *types.Transaction
		var blobIdx int
		for _, tx := range p.txPool {
			for j, bh := range tx.BlobHashes() {
				if bh == vhash {
					foundTx = tx
					blobIdx = j
					break
				}
			}
			if foundTx != nil {
				break
			}
		}
		if foundTx == nil {
			continue
		}
		txCells, ok := p.cellPool[foundTx.Hash()]
		if !ok {
			continue
		}
		_ = blobIdx // cells in the mock are stored flat by cell index
		blobCells := make([]*kzg4844.Cell, len(requestedIndices))
		for j, idx := range requestedIndices {
			if int(idx) < len(txCells) {
				cell := txCells[idx]
				blobCells[j] = &cell
			}
		}
		cells[i] = blobCells
	}
	return cells, proofs, nil
}

func (p *testTxPool) GetCustody(hash common.Hash) *types.CustodyBitmap {
	p.lock.RLock()
	defer p.lock.RUnlock()
	mask, ok := p.custody[hash]
	if !ok {
		return nil
	}
	return &mask
}

// AddCells adds cells for a specific transaction hash (for testing)
func (p *testTxPool) AddCells(hash common.Hash, cells []kzg4844.Cell, mask types.CustodyBitmap) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.cellPool[hash] = cells
	p.custody[hash] = mask
}

func (p *testTxPool) AddPooledTx(pooledTx *blobpool.BlobTxForPool) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	hash := pooledTx.Tx.Hash()
	p.cellPool[hash] = pooledTx.CellSidecar.Cells
	p.txPool[hash] = pooledTx.Tx
	return nil
}

// FilterType should check whether the pool supports the given type of transactions.
func (p *testTxPool) FilterType(kind byte) bool {
	switch kind {
	case types.LegacyTxType, types.AccessListTxType, types.DynamicFeeTxType, types.BlobTxType, types.SetCodeTxType:
		return true
	}
	return false
}

func (p *testTxPool) ValidateTxBasics(_ *types.Transaction) error {
	return nil
}

// testHandler is a live implementation of the Ethereum protocol handler, just
// preinitialized with some sane testing defaults and the transaction pool mocked
// out.
type testHandler struct {
	db       ethdb.Database
	chain    *core.BlockChain
	txpool   *testTxPool
	blobpool *testTxPool
	handler  *handler
}

// newTestHandler creates a new handler for testing purposes with no blocks.
func newTestHandler(mode ethconfig.SyncMode) *testHandler {
	return newTestHandlerWithBlocks(0, mode)
}

// newTestHandlerWithBlocks creates a new handler for testing purposes, with a
// given number of initial blocks.
func newTestHandlerWithBlocks(blocks int, mode ethconfig.SyncMode) *testHandler {
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
		BlobPool:   txpool,
		Network:    1,
		Sync:       mode,
		BloomCache: 1,
	})
	handler.Start(1000)

	return &testHandler{
		db:       db,
		chain:    chain,
		txpool:   txpool,
		blobpool: txpool,
		handler:  handler,
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

func TestBroadcastChoiceMatchesSort(t *testing.T) {
	self := enode.HexID("1111111111111111111111111111111111111111111111111111111111111111")
	rand := rand.New(rand.NewSource(33))
	peers := createTestPeers(rand, 500)
	defer closePeers(peers)

	senders := make([]common.Address, 8)
	for i := range senders {
		rand.Read(senders[i][:])
	}
	for _, count := range []int{0, 1, 2, 3, 49, 50, 200, 500} {
		for i, sender := range senders {
			choice := newBroadcastChoice(self, [16]byte{1})
			got := maps.Clone(choice.choosePeers(peers[:count], sender))

			want := make(map[*ethPeer]struct{})
			scores := slices.Clone(choice.tmp)
			sortBroadcastPeersReference(scores)
			for _, peer := range scores[:int(math.Ceil(math.Sqrt(float64(count))))] {
				want[peer.p] = struct{}{}
			}
			if !maps.Equal(got, want) {
				t.Errorf("count %d sender %d: choice differs from full sort", count, i)
			}
		}
	}
}

func TestBroadcastChoiceTiedCutoff(t *testing.T) {
	scores := []broadcastPeer{
		{p: new(ethPeer), score: 3},
		{p: new(ethPeer), score: 5},
		{p: new(ethPeer), score: 3},
		{p: new(ethPeer), score: 1},
	}
	choice := &broadcastChoice{tmp: slices.Clone(scores)}
	got := choice.selectPeers(2)

	want := slices.Clone(scores)
	sortBroadcastPeersReference(want)
	for i := range want {
		if choice.tmp[i] != want[i] {
			t.Fatal("tied cutoff did not use the full-sort order")
		}
	}
	gotSet := make(map[*ethPeer]struct{}, len(got))
	wantSet := make(map[*ethPeer]struct{}, 2)
	for _, peer := range got {
		gotSet[peer.p] = struct{}{}
	}
	for _, peer := range want[:2] {
		wantSet[peer.p] = struct{}{}
	}
	if !maps.Equal(gotSet, wantSet) {
		t.Fatal("tied cutoff selected different peers than full sort")
	}
}

func sortBroadcastPeersReference(peers []broadcastPeer) {
	slices.SortFunc(peers, func(a, b broadcastPeer) int {
		return cmp.Compare(a.score, b.score)
	})
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

func BenchmarkBroadcastTransactions(b *testing.B) {
	for _, peers := range []int{50, 200, 500} {
		b.Run(fmt.Sprintf("peers=%d", peers), func(b *testing.B) {
			for _, batch := range []int{1, 10, 100} {
				b.Run(fmt.Sprintf("batch=%d", batch), func(b *testing.B) {
					benchmarkBroadcastTransactions(b, peers, batch)
				})
			}
		})
	}
}

const (
	maxBroadcastEventsPerSample       = 32
	maxBroadcastTransactionsPerSample = 1600
)

// broadcastEventsPerSample bounds the source transactions in one sample to
// keep the asynchronous peer queues bounded while still amortizing scheduling.
func broadcastEventsPerSample(batch int) int {
	return min(maxBroadcastEventsPerSample, max(1, maxBroadcastTransactionsPerSample/batch))
}

type broadcastBenchmarkBatch struct {
	txs  types.Transactions
	list []common.Hash
}

func makeBroadcastBenchmarkBatches(b *testing.B, signer types.Signer, first, count, batch int) []broadcastBenchmarkBatch {
	// Give each transaction a distinct sender so the batch can produce distinct
	// direct and announcement queues for each peer.
	keys := make([]*ecdsa.PrivateKey, batch)
	for i := range keys {
		secret := make([]byte, 32)
		secret[len(secret)-1] = byte(i + 1)
		var err error
		keys[i], err = crypto.ToECDSA(secret)
		if err != nil {
			b.Fatal(err)
		}
	}

	batches := make([]broadcastBenchmarkBatch, count)
	for i := range batches {
		batches[i].txs = make(types.Transactions, batch)
		batches[i].list = make([]common.Hash, batch)
		for j := range batches[i].txs {
			tx := types.NewTransaction(uint64((first+i)*batch+j), common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(1), nil)
			var err error
			batches[i].txs[j], err = types.SignTx(tx, signer, keys[j])
			if err != nil {
				b.Fatal(err)
			}
			batches[i].list[j] = batches[i].txs[j].Hash()
		}
	}
	return batches
}

// benchmarkBroadcastTransactions measures the complete transaction-event path,
// including direct transaction and announcement packets. Each sample groups
// several events to amortize asynchronous peer scheduling and reports ns/op
// per source event. It builds each group while the timer is stopped instead
// of retaining inputs for every iteration. Run it with both one and four CPUs:
// the former gives a repeatable single-P result and the latter exercises
// concurrent peer writers.
func benchmarkBroadcastTransactions(b *testing.B, peers, batch int) {
	eventsPerSample := broadcastEventsPerSample(batch)
	source := newTestHandler(ethconfig.FullSync)
	peersToClose := make([]*eth.Peer, 0, peers)
	writers := make([]*broadcastBenchmarkPeerRW, 0, peers)
	b.Cleanup(func() {
		source.close()
		for _, peer := range peersToClose {
			peer.Close()
		}
	})

	delivered := make(chan struct{}, peers)
	errs := make(chan error, 1)
	for i := range peers {
		id := enode.ID{byte(i), byte(i >> 8)}
		writer := &broadcastBenchmarkPeerRW{delivered: delivered, errs: errs}
		peer := eth.NewPeer(eth.ETH69, p2p.NewPeer(id, "", nil), writer, source.txpool, source.blobpool, source.chain.Config())
		if err := source.handler.peers.registerPeer(peer, nil); err != nil {
			b.Fatal(err)
		}
		peersToClose = append(peersToClose, peer)
		writers = append(writers, writer)
	}

	signer := types.LatestSigner(source.chain.Config())
	b.ResetTimer()
	for i := range b.N {
		b.StopTimer()
		group := makeBroadcastBenchmarkBatches(b, signer, i*eventsPerSample, eventsPerSample, batch)
		hashes := make(map[common.Hash]struct{}, len(group)*batch)
		txs := make(types.Transactions, 0, len(group)*batch)
		for _, event := range group {
			for _, hash := range event.list {
				hashes[hash] = struct{}{}
			}
			txs = append(txs, event.txs...)
		}
		for _, writer := range writers {
			if err := writer.expect(hashes); err != nil {
				b.Fatal(err)
			}
		}
		b.StartTimer()
		for _, event := range group {
			source.txpool.Add(event.txs, false)
		}
		for range peers {
			select {
			case err := <-errs:
				b.Fatal(err)
			case <-delivered:
			}
		}
		b.StopTimer()
		var direct int
		for _, writer := range writers {
			direct += writer.directCount()
		}
		want := len(hashes) * int(math.Ceil(math.Sqrt(float64(peers))))
		if direct != want {
			b.Fatalf("got %d direct recipients, want %d", direct, want)
		}
		source.txpool.Remove(txs...)
	}
	b.ReportMetric(float64(b.Elapsed())/float64(b.N*eventsPerSample), "ns/op")
}

// broadcastBenchmarkPeerRW consumes the messages issued by the asynchronous
// peer broadcasters and verifies the transaction hashes they carry.
type broadcastBenchmarkPeerRW struct {
	lock      sync.Mutex
	expected  map[common.Hash]struct{}
	seen      map[common.Hash]struct{}
	direct    map[common.Hash]struct{}
	delivered chan<- struct{}
	errs      chan<- error
}

func (rw *broadcastBenchmarkPeerRW) ReadMsg() (p2p.Msg, error) {
	return p2p.Msg{}, p2p.ErrPipeClosed
}

func (rw *broadcastBenchmarkPeerRW) expect(hashes map[common.Hash]struct{}) error {
	rw.lock.Lock()
	defer rw.lock.Unlock()

	if rw.expected != nil {
		return fmt.Errorf("previous broadcast is still pending")
	}
	rw.expected = hashes
	rw.seen = make(map[common.Hash]struct{}, len(hashes))
	rw.direct = make(map[common.Hash]struct{}, len(hashes))
	return nil
}

func (rw *broadcastBenchmarkPeerRW) WriteMsg(msg p2p.Msg) error {
	rw.lock.Lock()
	defer rw.lock.Unlock()

	if rw.expected == nil {
		return rw.fail(fmt.Errorf("unexpected broadcast message %d", msg.Code))
	}
	switch msg.Code {
	case eth.TransactionsMsg:
		var txs types.Transactions
		if err := msg.Decode(&txs); err != nil {
			return rw.fail(err)
		}
		for _, tx := range txs {
			if _, ok := rw.expected[tx.Hash()]; !ok {
				return rw.fail(fmt.Errorf("unexpected broadcast transaction %v", tx.Hash()))
			}
			if _, ok := rw.seen[tx.Hash()]; ok {
				return rw.fail(fmt.Errorf("duplicate broadcast transaction %v", tx.Hash()))
			}
			rw.seen[tx.Hash()] = struct{}{}
			rw.direct[tx.Hash()] = struct{}{}
		}
	case eth.NewPooledTransactionHashesMsg:
		var packet eth.NewPooledTransactionHashesPacket71
		if err := msg.Decode(&packet); err != nil {
			return rw.fail(err)
		}
		for _, hash := range packet.Hashes {
			if _, ok := rw.expected[hash]; !ok {
				return rw.fail(fmt.Errorf("unexpected transaction announcement %v", hash))
			}
			if _, ok := rw.seen[hash]; ok {
				return rw.fail(fmt.Errorf("duplicate transaction announcement %v", hash))
			}
			rw.seen[hash] = struct{}{}
		}
	default:
		if err := msg.Discard(); err != nil {
			return rw.fail(err)
		}
		return rw.fail(fmt.Errorf("unexpected message code %d", msg.Code))
	}
	if len(rw.seen) == len(rw.expected) {
		rw.expected = nil
		rw.delivered <- struct{}{}
	}
	return nil
}

func (rw *broadcastBenchmarkPeerRW) directCount() int {
	rw.lock.Lock()
	defer rw.lock.Unlock()

	return len(rw.direct)
}

func (rw *broadcastBenchmarkPeerRW) fail(err error) error {
	select {
	case rw.errs <- err:
	default:
	}
	return err
}

func TestBroadcastBenchmarkPeerRW(t *testing.T) {
	tx := types.NewTransaction(1, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(1), nil)
	hashes := map[common.Hash]struct{}{tx.Hash(): {}}
	delivered := make(chan struct{}, 1)
	errs := make(chan error, 1)
	writer := &broadcastBenchmarkPeerRW{delivered: delivered, errs: errs}

	for _, message := range []struct {
		code uint64
		data any
	}{
		{eth.TransactionsMsg, types.Transactions{tx}},
		{eth.NewPooledTransactionHashesMsg, eth.NewPooledTransactionHashesPacket71{
			Types:  []byte{tx.Type()},
			Sizes:  []uint32{uint32(tx.Size())},
			Hashes: []common.Hash{tx.Hash()},
		}},
	} {
		if err := writer.expect(hashes); err != nil {
			t.Fatal(err)
		}
		if message.code == eth.TransactionsMsg {
			if err := writer.expect(hashes); err == nil {
				t.Fatal("accepted a new expected broadcast before delivering the previous one")
			}
		}
		if err := p2p.Send(writer, message.code, message.data); err != nil {
			t.Fatal(err)
		}
		select {
		case <-delivered:
		case err := <-errs:
			t.Fatal(err)
		default:
			t.Fatal("message was not delivered")
		}
		wantDirect := 0
		if message.code == eth.TransactionsMsg {
			wantDirect = 1
		}
		if got := writer.directCount(); got != wantDirect {
			t.Errorf("direct count %d, want %d", got, wantDirect)
		}
	}
}

func TestBroadcastBenchmarkPeerRWRejectsDuplicates(t *testing.T) {
	tx := types.NewTransaction(1, common.Address{}, big.NewInt(0), params.TxGas, big.NewInt(1), nil)
	hashes := map[common.Hash]struct{}{tx.Hash(): {}}

	for _, message := range []struct {
		code uint64
		data any
	}{
		{eth.TransactionsMsg, types.Transactions{tx, tx}},
		{eth.NewPooledTransactionHashesMsg, eth.NewPooledTransactionHashesPacket71{
			Types:  []byte{tx.Type(), tx.Type()},
			Sizes:  []uint32{uint32(tx.Size()), uint32(tx.Size())},
			Hashes: []common.Hash{tx.Hash(), tx.Hash()},
		}},
	} {
		writer := &broadcastBenchmarkPeerRW{
			delivered: make(chan struct{}, 1),
			errs:      make(chan error, 1),
		}
		if err := writer.expect(hashes); err != nil {
			t.Fatal(err)
		}
		if err := p2p.Send(writer, message.code, message.data); err == nil {
			t.Fatal("accepted duplicate transaction")
		}
	}
}

func createTestPeers(rand *rand.Rand, n int) []*ethPeer {
	peers := make([]*ethPeer, n)
	for i := range peers {
		var id enode.ID
		rand.Read(id[:])
		p2pPeer := p2p.NewPeer(id, "test", nil)
		ep := eth.NewPeer(eth.ETH69, p2pPeer, nil, nil, nil, nil)
		peers[i] = &ethPeer{Peer: ep}
	}
	return peers
}

func closePeers(peers []*ethPeer) {
	for _, p := range peers {
		p.Close()
	}
}
