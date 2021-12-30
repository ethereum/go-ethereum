// Copyright 2014 The go-ethereum Authors
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
	"fmt"
	"math/big"
	"math/rand"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// testEthHandler is a mock event handler to listen for inbound network requests
// on the `eth` protocol and convert them into a more easily testable form.
type testEthHandler struct {
	blockBroadcasts event.Feed
	txAnnounces     event.Feed
	txBroadcasts    event.Feed
}

func (h *testEthHandler) Chain() *core.BlockChain              { panic("no backing chain") }
func (h *testEthHandler) TxPool() eth.TxPool                   { panic("no backing tx pool") }
func (h *testEthHandler) AcceptTxs() bool                      { return true }
func (h *testEthHandler) RunPeer(*eth.Peer, eth.Handler) error { panic("not used in tests") }
func (h *testEthHandler) PeerInfo(enode.ID) interface{}        { panic("not used in tests") }

func (h *testEthHandler) Handle(peer *eth.Peer, packet eth.Packet) error {
	switch packet := packet.(type) {
	case *eth.NewBlockPacket:
		h.blockBroadcasts.Send(packet.Block)
		return nil

	case *eth.NewPooledTransactionHashesPacket:
		h.txAnnounces.Send(([]common.Hash)(*packet))
		return nil

	case *eth.TransactionsPacket:
		h.txBroadcasts.Send(([]*types.Transaction)(*packet))
		return nil

	case *eth.PooledTransactionsPacket:
		h.txBroadcasts.Send(([]*types.Transaction)(*packet))
		return nil

	default:
		panic(fmt.Sprintf("unexpected eth packet type in tests: %T", packet))
	}
}

// Tests that peers are correctly accepted (or rejected) based on the advertised
// fork IDs in the protocol handshake.
func TestForkIDSplit66(t *testing.T) { testForkIDSplit(t, eth.ETH66) }

func testForkIDSplit(t *testing.T, protocol uint) {
	t.Parallel()

	var (
		engine = ethash.NewFaker()

		configNoFork  = &params.ChainConfig{HomesteadBlock: big.NewInt(1)}
		configProFork = &params.ChainConfig{
			HomesteadBlock: big.NewInt(1),
			EIP150Block:    big.NewInt(2),
			EIP155Block:    big.NewInt(2),
			EIP158Block:    big.NewInt(2),
			ByzantiumBlock: big.NewInt(3),
		}
		dbNoFork  = rawdb.NewMemoryDatabase()
		dbProFork = rawdb.NewMemoryDatabase()

		gspecNoFork  = &core.Genesis{Config: configNoFork}
		gspecProFork = &core.Genesis{Config: configProFork}

		genesisNoFork  = gspecNoFork.MustCommit(dbNoFork)
		genesisProFork = gspecProFork.MustCommit(dbProFork)

		chainNoFork, _  = core.NewBlockChain(dbNoFork, nil, configNoFork, engine, vm.Config{}, nil, nil)
		chainProFork, _ = core.NewBlockChain(dbProFork, nil, configProFork, engine, vm.Config{}, nil, nil)

		blocksNoFork, _  = core.GenerateChain(configNoFork, genesisNoFork, engine, dbNoFork, 2, nil)
		blocksProFork, _ = core.GenerateChain(configProFork, genesisProFork, engine, dbProFork, 2, nil)

		ethNoFork, _ = newHandler(&handlerConfig{
			Database:   dbNoFork,
			Chain:      chainNoFork,
			TxPool:     newTestTxPool(),
			Merger:     consensus.NewMerger(rawdb.NewMemoryDatabase()),
			Network:    1,
			Sync:       downloader.FullSync,
			BloomCache: 1,
		})
		ethProFork, _ = newHandler(&handlerConfig{
			Database:   dbProFork,
			Chain:      chainProFork,
			TxPool:     newTestTxPool(),
			Merger:     consensus.NewMerger(rawdb.NewMemoryDatabase()),
			Network:    1,
			Sync:       downloader.FullSync,
			BloomCache: 1,
		})
	)
	ethNoFork.Start(1000)
	ethProFork.Start(1000)

	// Clean up everything after ourselves
	defer chainNoFork.Stop()
	defer chainProFork.Stop()

	defer ethNoFork.Stop()
	defer ethProFork.Stop()

	// Both nodes should allow the other to connect (same genesis, next fork is the same)
	p2pNoFork, p2pProFork := p2p.MsgPipe()
	defer p2pNoFork.Close()
	defer p2pProFork.Close()

	peerNoFork := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pNoFork), p2pNoFork, nil)
	peerProFork := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pProFork), p2pProFork, nil)
	defer peerNoFork.Close()
	defer peerProFork.Close()

	errc := make(chan error, 2)
	go func(errc chan error) {
		errc <- ethNoFork.runEthPeer(peerProFork, func(peer *eth.Peer) error { return nil })
	}(errc)
	go func(errc chan error) {
		errc <- ethProFork.runEthPeer(peerNoFork, func(peer *eth.Peer) error { return nil })
	}(errc)

	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("frontier nofork <-> profork failed: %v", err)
			}
		case <-time.After(250 * time.Millisecond):
			t.Fatalf("frontier nofork <-> profork handler timeout")
		}
	}
	// Progress into Homestead. Fork's match, so we don't care what the future holds
	chainNoFork.InsertChain(blocksNoFork[:1])
	chainProFork.InsertChain(blocksProFork[:1])

	p2pNoFork, p2pProFork = p2p.MsgPipe()
	defer p2pNoFork.Close()
	defer p2pProFork.Close()

	peerNoFork = eth.NewPeer(protocol, p2p.NewPeer(enode.ID{1}, "", nil), p2pNoFork, nil)
	peerProFork = eth.NewPeer(protocol, p2p.NewPeer(enode.ID{2}, "", nil), p2pProFork, nil)
	defer peerNoFork.Close()
	defer peerProFork.Close()

	errc = make(chan error, 2)
	go func(errc chan error) {
		errc <- ethNoFork.runEthPeer(peerProFork, func(peer *eth.Peer) error { return nil })
	}(errc)
	go func(errc chan error) {
		errc <- ethProFork.runEthPeer(peerNoFork, func(peer *eth.Peer) error { return nil })
	}(errc)

	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				t.Fatalf("homestead nofork <-> profork failed: %v", err)
			}
		case <-time.After(250 * time.Millisecond):
			t.Fatalf("homestead nofork <-> profork handler timeout")
		}
	}
	// Progress into Spurious. Forks mismatch, signalling differing chains, reject
	chainNoFork.InsertChain(blocksNoFork[1:2])
	chainProFork.InsertChain(blocksProFork[1:2])

	p2pNoFork, p2pProFork = p2p.MsgPipe()
	defer p2pNoFork.Close()
	defer p2pProFork.Close()

	peerNoFork = eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pNoFork), p2pNoFork, nil)
	peerProFork = eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pProFork), p2pProFork, nil)
	defer peerNoFork.Close()
	defer peerProFork.Close()

	errc = make(chan error, 2)
	go func(errc chan error) {
		errc <- ethNoFork.runEthPeer(peerProFork, func(peer *eth.Peer) error { return nil })
	}(errc)
	go func(errc chan error) {
		errc <- ethProFork.runEthPeer(peerNoFork, func(peer *eth.Peer) error { return nil })
	}(errc)

	var successes int
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err == nil {
				successes++
				if successes == 2 { // Only one side disconnects
					t.Fatalf("fork ID rejection didn't happen")
				}
			}
		case <-time.After(250 * time.Millisecond):
			t.Fatalf("split peers not rejected")
		}
	}
}

// Tests that received transactions are added to the local pool.
func TestRecvTransactions66(t *testing.T) { testRecvTransactions(t, eth.ETH66) }

func testRecvTransactions(t *testing.T, protocol uint) {
	t.Parallel()

	// Create a message handler, configure it to accept transactions and watch them
	handler := newTestHandler()
	defer handler.close()

	handler.handler.acceptTxs = 1 // mark synced to accept transactions

	txs := make(chan core.NewTxsEvent)
	sub := handler.txpool.SubscribeNewTxsEvent(txs)
	defer sub.Unsubscribe()

	// Create a source peer to send messages through and a sink handler to receive them
	p2pSrc, p2pSink := p2p.MsgPipe()
	defer p2pSrc.Close()
	defer p2pSink.Close()

	src := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pSrc), p2pSrc, handler.txpool)
	sink := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pSink), p2pSink, handler.txpool)
	defer src.Close()
	defer sink.Close()

	go handler.handler.runEthPeer(sink, func(peer *eth.Peer) error {
		return eth.Handle((*ethHandler)(handler.handler), peer)
	})
	// Run the handshake locally to avoid spinning up a source handler
	var (
		genesis = handler.chain.Genesis()
		head    = handler.chain.CurrentBlock()
		td      = handler.chain.GetTd(head.Hash(), head.NumberU64())
	)
	if err := src.Handshake(1, td, head.Hash(), genesis.Hash(), forkid.NewIDWithChain(handler.chain), forkid.NewFilter(handler.chain)); err != nil {
		t.Fatalf("failed to run protocol handshake")
	}
	// Send the transaction to the sink and verify that it's added to the tx pool
	tx := types.NewTransaction(0, common.Address{}, big.NewInt(0), 100000, big.NewInt(0), nil)
	tx, _ = types.SignTx(tx, types.HomesteadSigner{}, testKey)

	if err := src.SendTransactions([]*types.Transaction{tx}); err != nil {
		t.Fatalf("failed to send transaction: %v", err)
	}
	select {
	case event := <-txs:
		if len(event.Txs) != 1 {
			t.Errorf("wrong number of added transactions: got %d, want 1", len(event.Txs))
		} else if event.Txs[0].Hash() != tx.Hash() {
			t.Errorf("added wrong tx hash: got %v, want %v", event.Txs[0].Hash(), tx.Hash())
		}
	case <-time.After(2 * time.Second):
		t.Errorf("no NewTxsEvent received within 2 seconds")
	}
}

// This test checks that pending transactions are sent.
func TestSendTransactions66(t *testing.T) { testSendTransactions(t, eth.ETH66) }

func testSendTransactions(t *testing.T, protocol uint) {
	t.Parallel()

	// Create a message handler and fill the pool with big transactions
	handler := newTestHandler()
	defer handler.close()

	insert := make([]*types.Transaction, 100)
	for nonce := range insert {
		tx := types.NewTransaction(uint64(nonce), common.Address{}, big.NewInt(0), 100000, big.NewInt(0), make([]byte, 10240))
		tx, _ = types.SignTx(tx, types.HomesteadSigner{}, testKey)

		insert[nonce] = tx
	}
	go handler.txpool.AddRemotes(insert) // Need goroutine to not block on feed
	time.Sleep(250 * time.Millisecond)   // Wait until tx events get out of the system (can't use events, tx broadcaster races with peer join)

	// Create a source handler to send messages through and a sink peer to receive them
	p2pSrc, p2pSink := p2p.MsgPipe()
	defer p2pSrc.Close()
	defer p2pSink.Close()

	src := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pSrc), p2pSrc, handler.txpool)
	sink := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pSink), p2pSink, handler.txpool)
	defer src.Close()
	defer sink.Close()

	go handler.handler.runEthPeer(src, func(peer *eth.Peer) error {
		return eth.Handle((*ethHandler)(handler.handler), peer)
	})
	// Run the handshake locally to avoid spinning up a source handler
	var (
		genesis = handler.chain.Genesis()
		head    = handler.chain.CurrentBlock()
		td      = handler.chain.GetTd(head.Hash(), head.NumberU64())
	)
	if err := sink.Handshake(1, td, head.Hash(), genesis.Hash(), forkid.NewIDWithChain(handler.chain), forkid.NewFilter(handler.chain)); err != nil {
		t.Fatalf("failed to run protocol handshake")
	}
	// After the handshake completes, the source handler should stream the sink
	// the transactions, subscribe to all inbound network events
	backend := new(testEthHandler)

	anns := make(chan []common.Hash)
	annSub := backend.txAnnounces.Subscribe(anns)
	defer annSub.Unsubscribe()

	bcasts := make(chan []*types.Transaction)
	bcastSub := backend.txBroadcasts.Subscribe(bcasts)
	defer bcastSub.Unsubscribe()

	go eth.Handle(backend, sink)

	// Make sure we get all the transactions on the correct channels
	seen := make(map[common.Hash]struct{})
	for len(seen) < len(insert) {
		switch protocol {
		case 66:
			select {
			case hashes := <-anns:
				for _, hash := range hashes {
					if _, ok := seen[hash]; ok {
						t.Errorf("duplicate transaction announced: %x", hash)
					}
					seen[hash] = struct{}{}
				}
			case <-bcasts:
				t.Errorf("initial tx broadcast received on post eth/66")
			}

		default:
			panic("unsupported protocol, please extend test")
		}
	}
	for _, tx := range insert {
		if _, ok := seen[tx.Hash()]; !ok {
			t.Errorf("missing transaction: %x", tx.Hash())
		}
	}
}

// Tests that transactions get propagated to all attached peers, either via direct
// broadcasts or via announcements/retrievals.
func TestTransactionPropagation66(t *testing.T) { testTransactionPropagation(t, eth.ETH66) }

func testTransactionPropagation(t *testing.T, protocol uint) {
	t.Parallel()

	// Create a source handler to send transactions from and a number of sinks
	// to receive them. We need multiple sinks since a one-to-one peering would
	// broadcast all transactions without announcement.
	source := newTestHandler()
	source.handler.snapSync = 0 // Avoid requiring snap, otherwise some will be dropped below
	defer source.close()

	sinks := make([]*testHandler, 10)
	for i := 0; i < len(sinks); i++ {
		sinks[i] = newTestHandler()
		defer sinks[i].close()

		sinks[i].handler.acceptTxs = 1 // mark synced to accept transactions
	}
	// Interconnect all the sink handlers with the source handler
	for i, sink := range sinks {
		sink := sink // Closure for gorotuine below

		sourcePipe, sinkPipe := p2p.MsgPipe()
		defer sourcePipe.Close()
		defer sinkPipe.Close()

		sourcePeer := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{byte(i + 1)}, "", nil, sourcePipe), sourcePipe, source.txpool)
		sinkPeer := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{0}, "", nil, sinkPipe), sinkPipe, sink.txpool)
		defer sourcePeer.Close()
		defer sinkPeer.Close()

		go source.handler.runEthPeer(sourcePeer, func(peer *eth.Peer) error {
			return eth.Handle((*ethHandler)(source.handler), peer)
		})
		go sink.handler.runEthPeer(sinkPeer, func(peer *eth.Peer) error {
			return eth.Handle((*ethHandler)(sink.handler), peer)
		})
	}
	// Subscribe to all the transaction pools
	txChs := make([]chan core.NewTxsEvent, len(sinks))
	for i := 0; i < len(sinks); i++ {
		txChs[i] = make(chan core.NewTxsEvent, 1024)

		sub := sinks[i].txpool.SubscribeNewTxsEvent(txChs[i])
		defer sub.Unsubscribe()
	}
	// Fill the source pool with transactions and wait for them at the sinks
	txs := make([]*types.Transaction, 1024)
	for nonce := range txs {
		tx := types.NewTransaction(uint64(nonce), common.Address{}, big.NewInt(0), 100000, big.NewInt(0), nil)
		tx, _ = types.SignTx(tx, types.HomesteadSigner{}, testKey)

		txs[nonce] = tx
	}
	source.txpool.AddRemotes(txs)

	// Iterate through all the sinks and ensure they all got the transactions
	for i := range sinks {
		for arrived, timeout := 0, false; arrived < len(txs) && !timeout; {
			select {
			case event := <-txChs[i]:
				arrived += len(event.Txs)
			case <-time.After(time.Second):
				t.Errorf("sink %d: transaction propagation timed out: have %d, want %d", i, arrived, len(txs))
				timeout = true
			}
		}
	}
}

// Tests that post eth protocol handshake, clients perform a mutual checkpoint
// challenge to validate each other's chains. Hash mismatches, or missing ones
// during a fast sync should lead to the peer getting dropped.
func TestCheckpointChallenge(t *testing.T) {
	tests := []struct {
		syncmode   downloader.SyncMode
		checkpoint bool
		timeout    bool
		empty      bool
		match      bool
		drop       bool
	}{
		// If checkpointing is not enabled locally, don't challenge and don't drop
		{downloader.FullSync, false, false, false, false, false},
		{downloader.SnapSync, false, false, false, false, false},

		// If checkpointing is enabled locally and remote response is empty, only drop during fast sync
		{downloader.FullSync, true, false, true, false, false},
		{downloader.SnapSync, true, false, true, false, true}, // Special case, fast sync, unsynced peer

		// If checkpointing is enabled locally and remote response mismatches, always drop
		{downloader.FullSync, true, false, false, false, true},
		{downloader.SnapSync, true, false, false, false, true},

		// If checkpointing is enabled locally and remote response matches, never drop
		{downloader.FullSync, true, false, false, true, false},
		{downloader.SnapSync, true, false, false, true, false},

		// If checkpointing is enabled locally and remote times out, always drop
		{downloader.FullSync, true, true, false, true, true},
		{downloader.SnapSync, true, true, false, true, true},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("sync %v checkpoint %v timeout %v empty %v match %v", tt.syncmode, tt.checkpoint, tt.timeout, tt.empty, tt.match), func(t *testing.T) {
			testCheckpointChallenge(t, tt.syncmode, tt.checkpoint, tt.timeout, tt.empty, tt.match, tt.drop)
		})
	}
}

func testCheckpointChallenge(t *testing.T, syncmode downloader.SyncMode, checkpoint bool, timeout bool, empty bool, match bool, drop bool) {

	// Reduce the checkpoint handshake challenge timeout
	defer func(old time.Duration) { syncChallengeTimeout = old }(syncChallengeTimeout)
	syncChallengeTimeout = 250 * time.Millisecond

	// Create a test handler and inject a CHT into it. The injection is a bit
	// ugly, but it beats creating everything manually just to avoid reaching
	// into the internals a bit.
	handler := newTestHandler()
	defer handler.close()

	if syncmode == downloader.SnapSync {
		atomic.StoreUint32(&handler.handler.snapSync, 1)
	} else {
		atomic.StoreUint32(&handler.handler.snapSync, 0)
	}
	var response *types.Header
	if checkpoint {
		number := (uint64(rand.Intn(500))+1)*params.CHTFrequency - 1
		response = &types.Header{Number: big.NewInt(int64(number)), Extra: []byte("valid")}

		handler.handler.checkpointNumber = number
		handler.handler.checkpointHash = response.Hash()
	}

	// Create a challenger peer and a challenged one.
	p2pLocal, p2pRemote := p2p.MsgPipe()
	defer p2pLocal.Close()
	defer p2pRemote.Close()

	local := eth.NewPeer(eth.ETH66, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pLocal), p2pLocal, handler.txpool)
	remote := eth.NewPeer(eth.ETH66, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pRemote), p2pRemote, handler.txpool)
	defer local.Close()
	defer remote.Close()

	handlerDone := make(chan struct{})
	go func() {
		defer close(handlerDone)
		handler.handler.runEthPeer(local, func(peer *eth.Peer) error {
			return eth.Handle((*ethHandler)(handler.handler), peer)
		})
	}()

	// Run the handshake locally to avoid spinning up a remote handler.
	var (
		genesis = handler.chain.Genesis()
		head    = handler.chain.CurrentBlock()
		td      = handler.chain.GetTd(head.Hash(), head.NumberU64())
	)
	if err := remote.Handshake(1, td, head.Hash(), genesis.Hash(), forkid.NewIDWithChain(handler.chain), forkid.NewFilter(handler.chain)); err != nil {
		t.Fatalf("failed to run protocol handshake")
	}
	// Connect a new peer and check that we receive the checkpoint challenge.
	if checkpoint {
		msg, err := p2pRemote.ReadMsg()
		if err != nil {
			t.Fatalf("failed to read checkpoint challenge: %v", err)
		}
		request := new(eth.GetBlockHeadersPacket66)
		if err := msg.Decode(request); err != nil {
			t.Fatalf("failed to decode checkpoint challenge: %v", err)
		}
		query := request.GetBlockHeadersPacket
		if query.Origin.Number != response.Number.Uint64() || query.Amount != 1 || query.Skip != 0 || query.Reverse {
			t.Fatalf("challenge mismatch: have [%d, %d, %d, %v] want [%d, %d, %d, %v]",
				query.Origin.Number, query.Amount, query.Skip, query.Reverse,
				response.Number.Uint64(), 1, 0, false)
		}
		// Create a block to reply to the challenge if no timeout is simulated.
		if !timeout {
			if empty {
				if err := remote.ReplyBlockHeadersRLP(request.RequestId, []rlp.RawValue{}); err != nil {
					t.Fatalf("failed to answer challenge: %v", err)
				}
			} else if match {
				responseRlp, _ := rlp.EncodeToBytes(response)
				if err := remote.ReplyBlockHeadersRLP(request.RequestId, []rlp.RawValue{responseRlp}); err != nil {
					t.Fatalf("failed to answer challenge: %v", err)
				}
			} else {
				responseRlp, _ := rlp.EncodeToBytes(types.Header{Number: response.Number})
				if err := remote.ReplyBlockHeadersRLP(request.RequestId, []rlp.RawValue{responseRlp}); err != nil {
					t.Fatalf("failed to answer challenge: %v", err)
				}
			}
		}
	}
	// Wait until the test timeout passes to ensure proper cleanup
	time.Sleep(syncChallengeTimeout + 300*time.Millisecond)

	// Verify that the remote peer is maintained or dropped.
	if drop {
		<-handlerDone
		if peers := handler.handler.peers.len(); peers != 0 {
			t.Fatalf("peer count mismatch: have %d, want %d", peers, 0)
		}
	} else {
		if peers := handler.handler.peers.len(); peers != 1 {
			t.Fatalf("peer count mismatch: have %d, want %d", peers, 1)
		}
	}
}

// Tests that blocks are broadcast to a sqrt number of peers only.
func TestBroadcastBlock1Peer(t *testing.T)    { testBroadcastBlock(t, 1, 1) }
func TestBroadcastBlock2Peers(t *testing.T)   { testBroadcastBlock(t, 2, 1) }
func TestBroadcastBlock3Peers(t *testing.T)   { testBroadcastBlock(t, 3, 1) }
func TestBroadcastBlock4Peers(t *testing.T)   { testBroadcastBlock(t, 4, 2) }
func TestBroadcastBlock5Peers(t *testing.T)   { testBroadcastBlock(t, 5, 2) }
func TestBroadcastBlock8Peers(t *testing.T)   { testBroadcastBlock(t, 9, 3) }
func TestBroadcastBlock12Peers(t *testing.T)  { testBroadcastBlock(t, 12, 3) }
func TestBroadcastBlock16Peers(t *testing.T)  { testBroadcastBlock(t, 16, 4) }
func TestBroadcastBloc26Peers(t *testing.T)   { testBroadcastBlock(t, 26, 5) }
func TestBroadcastBlock100Peers(t *testing.T) { testBroadcastBlock(t, 100, 10) }

func testBroadcastBlock(t *testing.T, peers, bcasts int) {
	t.Parallel()

	// Create a source handler to broadcast blocks from and a number of sinks
	// to receive them.
	source := newTestHandlerWithBlocks(1)
	defer source.close()

	sinks := make([]*testEthHandler, peers)
	for i := 0; i < len(sinks); i++ {
		sinks[i] = new(testEthHandler)
	}
	// Interconnect all the sink handlers with the source handler
	var (
		genesis = source.chain.Genesis()
		td      = source.chain.GetTd(genesis.Hash(), genesis.NumberU64())
	)
	for i, sink := range sinks {
		sink := sink // Closure for gorotuine below

		sourcePipe, sinkPipe := p2p.MsgPipe()
		defer sourcePipe.Close()
		defer sinkPipe.Close()

		sourcePeer := eth.NewPeer(eth.ETH66, p2p.NewPeerPipe(enode.ID{byte(i)}, "", nil, sourcePipe), sourcePipe, nil)
		sinkPeer := eth.NewPeer(eth.ETH66, p2p.NewPeerPipe(enode.ID{0}, "", nil, sinkPipe), sinkPipe, nil)
		defer sourcePeer.Close()
		defer sinkPeer.Close()

		go source.handler.runEthPeer(sourcePeer, func(peer *eth.Peer) error {
			return eth.Handle((*ethHandler)(source.handler), peer)
		})
		if err := sinkPeer.Handshake(1, td, genesis.Hash(), genesis.Hash(), forkid.NewIDWithChain(source.chain), forkid.NewFilter(source.chain)); err != nil {
			t.Fatalf("failed to run protocol handshake")
		}
		go eth.Handle(sink, sinkPeer)
	}
	// Subscribe to all the transaction pools
	blockChs := make([]chan *types.Block, len(sinks))
	for i := 0; i < len(sinks); i++ {
		blockChs[i] = make(chan *types.Block, 1)
		defer close(blockChs[i])

		sub := sinks[i].blockBroadcasts.Subscribe(blockChs[i])
		defer sub.Unsubscribe()
	}
	// Initiate a block propagation across the peers
	time.Sleep(100 * time.Millisecond)
	source.handler.BroadcastBlock(source.chain.CurrentBlock(), true)

	// Iterate through all the sinks and ensure the correct number got the block
	done := make(chan struct{}, peers)
	for _, ch := range blockChs {
		ch := ch
		go func() {
			<-ch
			done <- struct{}{}
		}()
	}
	var received int
	for {
		select {
		case <-done:
			received++

		case <-time.After(100 * time.Millisecond):
			if received != bcasts {
				t.Errorf("broadcast count mismatch: have %d, want %d", received, bcasts)
			}
			return
		}
	}
}

// Tests that a propagated malformed block (uncles or transactions don't match
// with the hashes in the header) gets discarded and not broadcast forward.
func TestBroadcastMalformedBlock66(t *testing.T) { testBroadcastMalformedBlock(t, eth.ETH66) }

func testBroadcastMalformedBlock(t *testing.T, protocol uint) {
	t.Parallel()

	// Create a source handler to broadcast blocks from and a number of sinks
	// to receive them.
	source := newTestHandlerWithBlocks(1)
	defer source.close()

	// Create a source handler to send messages through and a sink peer to receive them
	p2pSrc, p2pSink := p2p.MsgPipe()
	defer p2pSrc.Close()
	defer p2pSink.Close()

	src := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pSrc), p2pSrc, source.txpool)
	sink := eth.NewPeer(protocol, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pSink), p2pSink, source.txpool)
	defer src.Close()
	defer sink.Close()

	go source.handler.runEthPeer(src, func(peer *eth.Peer) error {
		return eth.Handle((*ethHandler)(source.handler), peer)
	})
	// Run the handshake locally to avoid spinning up a sink handler
	var (
		genesis = source.chain.Genesis()
		td      = source.chain.GetTd(genesis.Hash(), genesis.NumberU64())
	)
	if err := sink.Handshake(1, td, genesis.Hash(), genesis.Hash(), forkid.NewIDWithChain(source.chain), forkid.NewFilter(source.chain)); err != nil {
		t.Fatalf("failed to run protocol handshake")
	}
	// After the handshake completes, the source handler should stream the sink
	// the blocks, subscribe to inbound network events
	backend := new(testEthHandler)

	blocks := make(chan *types.Block, 1)
	sub := backend.blockBroadcasts.Subscribe(blocks)
	defer sub.Unsubscribe()

	go eth.Handle(backend, sink)

	// Create various combinations of malformed blocks
	head := source.chain.CurrentBlock()

	malformedUncles := head.Header()
	malformedUncles.UncleHash[0]++
	malformedTransactions := head.Header()
	malformedTransactions.TxHash[0]++
	malformedEverything := head.Header()
	malformedEverything.UncleHash[0]++
	malformedEverything.TxHash[0]++

	// Try to broadcast all malformations and ensure they all get discarded
	for _, header := range []*types.Header{malformedUncles, malformedTransactions, malformedEverything} {
		block := types.NewBlockWithHeader(header).WithBody(head.Transactions(), head.Uncles())
		if err := src.SendNewBlock(block, big.NewInt(131136)); err != nil {
			t.Fatalf("failed to broadcast block: %v", err)
		}
		select {
		case <-blocks:
			t.Fatalf("malformed block forwarded")
		case <-time.After(100 * time.Millisecond):
		}
	}
}
