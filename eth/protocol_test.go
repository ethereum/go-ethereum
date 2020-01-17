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
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

func init() {
	// log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))
}

var testAccount, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

// Tests that handshake failures are detected and reported correctly.
func TestStatusMsgErrors63(t *testing.T) {
	pm, _ := newTestProtocolManagerMust(t, downloader.FullSync, 0, nil, nil)
	var (
		genesis = pm.blockchain.Genesis()
		head    = pm.blockchain.CurrentHeader()
		td      = pm.blockchain.GetTd(head.Hash(), head.Number.Uint64())
	)
	defer pm.Stop()

	tests := []struct {
		code      uint64
		data      interface{}
		wantError error
	}{
		{
			code: TxMsg, data: []interface{}{},
			wantError: errResp(ErrNoStatusMsg, "first msg has code 2 (!= 0)"),
		},
		{
			code: StatusMsg, data: statusData63{10, DefaultConfig.NetworkId, td, head.Hash(), genesis.Hash()},
			wantError: errResp(ErrProtocolVersionMismatch, "10 (!= %d)", 63),
		},
		{
			code: StatusMsg, data: statusData63{63, 999, td, head.Hash(), genesis.Hash()},
			wantError: errResp(ErrNetworkIDMismatch, "999 (!= %d)", DefaultConfig.NetworkId),
		},
		{
			code: StatusMsg, data: statusData63{63, DefaultConfig.NetworkId, td, head.Hash(), common.Hash{3}},
			wantError: errResp(ErrGenesisMismatch, "0300000000000000 (!= %x)", genesis.Hash().Bytes()[:8]),
		},
	}
	for i, test := range tests {
		p, errc := newTestPeer("peer", 63, pm, false)
		// The send call might hang until reset because
		// the protocol might not read the payload.
		go p2p.Send(p.app, test.code, test.data)

		select {
		case err := <-errc:
			if err == nil {
				t.Errorf("test %d: protocol returned nil error, want %q", i, test.wantError)
			} else if err.Error() != test.wantError.Error() {
				t.Errorf("test %d: wrong error: got %q, want %q", i, err, test.wantError)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("protocol did not shut down within 2 seconds")
		}
		p.close()
	}
}

func TestStatusMsgErrors64(t *testing.T) {
	pm, _ := newTestProtocolManagerMust(t, downloader.FullSync, 0, nil, nil)
	var (
		genesis = pm.blockchain.Genesis()
		head    = pm.blockchain.CurrentHeader()
		td      = pm.blockchain.GetTd(head.Hash(), head.Number.Uint64())
		forkID  = forkid.NewID(pm.blockchain)
	)
	defer pm.Stop()

	tests := []struct {
		code      uint64
		data      interface{}
		wantError error
	}{
		{
			code: TxMsg, data: []interface{}{},
			wantError: errResp(ErrNoStatusMsg, "first msg has code 2 (!= 0)"),
		},
		{
			code: StatusMsg, data: statusData{10, DefaultConfig.NetworkId, td, head.Hash(), genesis.Hash(), forkID},
			wantError: errResp(ErrProtocolVersionMismatch, "10 (!= %d)", 64),
		},
		{
			code: StatusMsg, data: statusData{64, 999, td, head.Hash(), genesis.Hash(), forkID},
			wantError: errResp(ErrNetworkIDMismatch, "999 (!= %d)", DefaultConfig.NetworkId),
		},
		{
			code: StatusMsg, data: statusData{64, DefaultConfig.NetworkId, td, head.Hash(), common.Hash{3}, forkID},
			wantError: errResp(ErrGenesisMismatch, "0300000000000000000000000000000000000000000000000000000000000000 (!= %x)", genesis.Hash()),
		},
		{
			code: StatusMsg, data: statusData{64, DefaultConfig.NetworkId, td, head.Hash(), genesis.Hash(), forkid.ID{Hash: [4]byte{0x00, 0x01, 0x02, 0x03}}},
			wantError: errResp(ErrForkIDRejected, forkid.ErrLocalIncompatibleOrStale.Error()),
		},
	}
	for i, test := range tests {
		p, errc := newTestPeer("peer", 64, pm, false)
		// The send call might hang until reset because
		// the protocol might not read the payload.
		go p2p.Send(p.app, test.code, test.data)

		select {
		case err := <-errc:
			if err == nil {
				t.Errorf("test %d: protocol returned nil error, want %q", i, test.wantError)
			} else if err.Error() != test.wantError.Error() {
				t.Errorf("test %d: wrong error: got %q, want %q", i, err, test.wantError)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("protocol did not shut down within 2 seconds")
		}
		p.close()
	}
}

func TestForkIDSplit(t *testing.T) {
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

		chainNoFork, _  = core.NewBlockChain(dbNoFork, nil, configNoFork, engine, vm.Config{}, nil)
		chainProFork, _ = core.NewBlockChain(dbProFork, nil, configProFork, engine, vm.Config{}, nil)

		blocksNoFork, _  = core.GenerateChain(configNoFork, genesisNoFork, engine, dbNoFork, 2, nil)
		blocksProFork, _ = core.GenerateChain(configProFork, genesisProFork, engine, dbProFork, 2, nil)

		ethNoFork, _  = NewProtocolManager(configNoFork, nil, downloader.FullSync, 1, new(event.TypeMux), new(testTxPool), engine, chainNoFork, dbNoFork, 1, nil)
		ethProFork, _ = NewProtocolManager(configProFork, nil, downloader.FullSync, 1, new(event.TypeMux), new(testTxPool), engine, chainProFork, dbProFork, 1, nil)
	)
	ethNoFork.Start(1000)
	ethProFork.Start(1000)

	// Both nodes should allow the other to connect (same genesis, next fork is the same)
	p2pNoFork, p2pProFork := p2p.MsgPipe()
	peerNoFork := newPeer(64, p2p.NewPeer(enode.ID{1}, "", nil), p2pNoFork)
	peerProFork := newPeer(64, p2p.NewPeer(enode.ID{2}, "", nil), p2pProFork)

	errc := make(chan error, 2)
	go func() { errc <- ethNoFork.handle(peerProFork) }()
	go func() { errc <- ethProFork.handle(peerNoFork) }()

	select {
	case err := <-errc:
		t.Fatalf("frontier nofork <-> profork failed: %v", err)
	case <-time.After(250 * time.Millisecond):
		p2pNoFork.Close()
		p2pProFork.Close()
	}
	// Progress into Homestead. Fork's match, so we don't care what the future holds
	chainNoFork.InsertChain(blocksNoFork[:1])
	chainProFork.InsertChain(blocksProFork[:1])

	p2pNoFork, p2pProFork = p2p.MsgPipe()
	peerNoFork = newPeer(64, p2p.NewPeer(enode.ID{1}, "", nil), p2pNoFork)
	peerProFork = newPeer(64, p2p.NewPeer(enode.ID{2}, "", nil), p2pProFork)

	errc = make(chan error, 2)
	go func() { errc <- ethNoFork.handle(peerProFork) }()
	go func() { errc <- ethProFork.handle(peerNoFork) }()

	select {
	case err := <-errc:
		t.Fatalf("homestead nofork <-> profork failed: %v", err)
	case <-time.After(250 * time.Millisecond):
		p2pNoFork.Close()
		p2pProFork.Close()
	}
	// Progress into Spurious. Forks mismatch, signalling differing chains, reject
	chainNoFork.InsertChain(blocksNoFork[1:2])
	chainProFork.InsertChain(blocksProFork[1:2])

	p2pNoFork, p2pProFork = p2p.MsgPipe()
	peerNoFork = newPeer(64, p2p.NewPeer(enode.ID{1}, "", nil), p2pNoFork)
	peerProFork = newPeer(64, p2p.NewPeer(enode.ID{2}, "", nil), p2pProFork)

	errc = make(chan error, 2)
	go func() { errc <- ethNoFork.handle(peerProFork) }()
	go func() { errc <- ethProFork.handle(peerNoFork) }()

	select {
	case err := <-errc:
		if want := errResp(ErrForkIDRejected, forkid.ErrLocalIncompatibleOrStale.Error()); err.Error() != want.Error() {
			t.Fatalf("fork ID rejection error mismatch: have %v, want %v", err, want)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("split peers not rejected")
	}
}

// This test checks that received transactions are added to the local pool.
func TestRecvTransactions63(t *testing.T) { testRecvTransactions(t, 63) }
func TestRecvTransactions64(t *testing.T) { testRecvTransactions(t, 64) }

func testRecvTransactions(t *testing.T, protocol int) {
	txAdded := make(chan []*types.Transaction)
	pm, _ := newTestProtocolManagerMust(t, downloader.FullSync, 0, nil, txAdded)
	pm.acceptTxs = 1 // mark synced to accept transactions
	p, _ := newTestPeer("peer", protocol, pm, true)
	defer pm.Stop()
	defer p.close()

	tx := newTestTransaction(testAccount, 0, 0)
	if err := p2p.Send(p.app, TxMsg, []interface{}{tx}); err != nil {
		t.Fatalf("send error: %v", err)
	}
	select {
	case added := <-txAdded:
		if len(added) != 1 {
			t.Errorf("wrong number of added transactions: got %d, want 1", len(added))
		} else if added[0].Hash() != tx.Hash() {
			t.Errorf("added wrong tx hash: got %v, want %v", added[0].Hash(), tx.Hash())
		}
	case <-time.After(2 * time.Second):
		t.Errorf("no NewTxsEvent received within 2 seconds")
	}
}

// This test checks that pending transactions are sent.
func TestSendTransactions63(t *testing.T) { testSendTransactions(t, 63) }
func TestSendTransactions64(t *testing.T) { testSendTransactions(t, 64) }

func testSendTransactions(t *testing.T, protocol int) {
	pm, _ := newTestProtocolManagerMust(t, downloader.FullSync, 0, nil, nil)
	defer pm.Stop()

	// Fill the pool with big transactions.
	const txsize = txsyncPackSize / 10
	alltxs := make([]*types.Transaction, 100)
	for nonce := range alltxs {
		alltxs[nonce] = newTestTransaction(testAccount, uint64(nonce), txsize)
	}
	pm.txpool.AddRemotes(alltxs)

	// Connect several peers. They should all receive the pending transactions.
	var wg sync.WaitGroup
	checktxs := func(p *testPeer) {
		defer wg.Done()
		defer p.close()
		seen := make(map[common.Hash]bool)
		for _, tx := range alltxs {
			seen[tx.Hash()] = false
		}
		for n := 0; n < len(alltxs) && !t.Failed(); {
			var txs []*types.Transaction
			msg, err := p.app.ReadMsg()
			if err != nil {
				t.Errorf("%v: read error: %v", p.Peer, err)
			} else if msg.Code != TxMsg {
				t.Errorf("%v: got code %d, want TxMsg", p.Peer, msg.Code)
			}
			if err := msg.Decode(&txs); err != nil {
				t.Errorf("%v: %v", p.Peer, err)
			}
			for _, tx := range txs {
				hash := tx.Hash()
				seentx, want := seen[hash]
				if seentx {
					t.Errorf("%v: got tx more than once: %x", p.Peer, hash)
				}
				if !want {
					t.Errorf("%v: got unexpected tx: %x", p.Peer, hash)
				}
				seen[hash] = true
				n++
			}
		}
	}
	for i := 0; i < 3; i++ {
		p, _ := newTestPeer(fmt.Sprintf("peer #%d", i), protocol, pm, true)
		wg.Add(1)
		go checktxs(p)
	}
	wg.Wait()
}

// Tests that the custom union field encoder and decoder works correctly.
func TestGetBlockHeadersDataEncodeDecode(t *testing.T) {
	// Create a "random" hash for testing
	var hash common.Hash
	for i := range hash {
		hash[i] = byte(i)
	}
	// Assemble some table driven tests
	tests := []struct {
		packet *getBlockHeadersData
		fail   bool
	}{
		// Providing the origin as either a hash or a number should both work
		{fail: false, packet: &getBlockHeadersData{Origin: hashOrNumber{Number: 314}}},
		{fail: false, packet: &getBlockHeadersData{Origin: hashOrNumber{Hash: hash}}},

		// Providing arbitrary query field should also work
		{fail: false, packet: &getBlockHeadersData{Origin: hashOrNumber{Number: 314}, Amount: 314, Skip: 1, Reverse: true}},
		{fail: false, packet: &getBlockHeadersData{Origin: hashOrNumber{Hash: hash}, Amount: 314, Skip: 1, Reverse: true}},

		// Providing both the origin hash and origin number must fail
		{fail: true, packet: &getBlockHeadersData{Origin: hashOrNumber{Hash: hash, Number: 314}}},
	}
	// Iterate over each of the tests and try to encode and then decode
	for i, tt := range tests {
		bytes, err := rlp.EncodeToBytes(tt.packet)
		if err != nil && !tt.fail {
			t.Fatalf("test %d: failed to encode packet: %v", i, err)
		} else if err == nil && tt.fail {
			t.Fatalf("test %d: encode should have failed", i)
		}
		if !tt.fail {
			packet := new(getBlockHeadersData)
			if err := rlp.DecodeBytes(bytes, packet); err != nil {
				t.Fatalf("test %d: failed to decode packet: %v", i, err)
			}
			if packet.Origin.Hash != tt.packet.Origin.Hash || packet.Origin.Number != tt.packet.Origin.Number || packet.Amount != tt.packet.Amount ||
				packet.Skip != tt.packet.Skip || packet.Reverse != tt.packet.Reverse {
				t.Fatalf("test %d: encode decode mismatch: have %+v, want %+v", i, packet, tt.packet)
			}
		}
	}
}
