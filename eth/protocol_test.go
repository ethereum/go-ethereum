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
	"crypto/rand"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func init() {
	// glog.SetToStderr(true)
	// glog.SetV(6)
}

var testAccount = crypto.NewKey(rand.Reader)

func TestStatusMsgErrors(t *testing.T) {
	pm := newProtocolManagerForTesting(nil)
	td, currentBlock, genesis := pm.chainman.Status()
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
			code: StatusMsg, data: statusData{10, NetworkId, td, currentBlock, genesis},
			wantError: errResp(ErrProtocolVersionMismatch, "10 (!= 0)"),
		},
		{
			code: StatusMsg, data: statusData{uint32(ProtocolVersions[0]), 999, td, currentBlock, genesis},
			wantError: errResp(ErrNetworkIdMismatch, "999 (!= 1)"),
		},
		{
			code: StatusMsg, data: statusData{uint32(ProtocolVersions[0]), NetworkId, td, currentBlock, common.Hash{3}},
			wantError: errResp(ErrGenesisBlockMismatch, "0300000000000000000000000000000000000000000000000000000000000000 (!= %x)", genesis),
		},
	}

	for i, test := range tests {
		p, errc := newTestPeer(pm)
		// The send call might hang until reset because
		// the protocol might not read the payload.
		go p2p.Send(p, test.code, test.data)

		select {
		case err := <-errc:
			if err == nil {
				t.Errorf("test %d: protocol returned nil error, want %q", test.wantError)
			} else if err.Error() != test.wantError.Error() {
				t.Errorf("test %d: wrong error: got %q, want %q", i, err, test.wantError)
			}
		case <-time.After(2 * time.Second):
			t.Errorf("protocol did not shut down withing 2 seconds")
		}
		p.close()
	}
}

// This test checks that received transactions are added to the local pool.
func TestRecvTransactions(t *testing.T) {
	txAdded := make(chan []*types.Transaction)
	pm := newProtocolManagerForTesting(txAdded)
	p, _ := newTestPeer(pm)
	defer pm.Stop()
	defer p.close()
	p.handshake(t)

	tx := newtx(testAccount, 0, 0)
	if err := p2p.Send(p, TxMsg, []interface{}{tx}); err != nil {
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
		t.Errorf("no TxPreEvent received within 2 seconds")
	}
}

// This test checks that pending transactions are sent.
func TestSendTransactions(t *testing.T) {
	pm := newProtocolManagerForTesting(nil)
	defer pm.Stop()

	// Fill the pool with big transactions.
	const txsize = txsyncPackSize / 10
	alltxs := make([]*types.Transaction, 100)
	for nonce := range alltxs {
		alltxs[nonce] = newtx(testAccount, uint64(nonce), txsize)
	}
	pm.txpool.AddTransactions(alltxs)

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
			msg, err := p.ReadMsg()
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
		p, _ := newTestPeer(pm)
		p.handshake(t)
		wg.Add(1)
		go checktxs(p)
	}
	wg.Wait()
}

// testPeer wraps all peer-related data for tests.
type testPeer struct {
	p2p.MsgReadWriter                // writing to the test peer feeds the protocol
	pipe              *p2p.MsgPipeRW // the protocol read/writes on this end
	pm                *ProtocolManager
	*peer
}

func newProtocolManagerForTesting(txAdded chan<- []*types.Transaction) *ProtocolManager {
	db, _ := ethdb.NewMemDatabase()
	core.WriteTestNetGenesisBlock(db, 0)
	var (
		em       = new(event.TypeMux)
		chain, _ = core.NewChainManager(db, core.FakePow{}, em)
		txpool   = &fakeTxPool{added: txAdded}
		pm       = NewProtocolManager(NetworkId, em, txpool, core.FakePow{}, chain)
	)
	pm.Start()
	return pm
}

func newTestPeer(pm *ProtocolManager) (*testPeer, <-chan error) {
	var id discover.NodeID
	rand.Read(id[:])
	rw1, rw2 := p2p.MsgPipe()
	peer := pm.newPeer(pm.protVer, pm.netId, p2p.NewPeer(id, "test peer", nil), rw2)
	errc := make(chan error, 1)
	go func() {
		pm.newPeerCh <- peer
		errc <- pm.handle(peer)
	}()
	return &testPeer{rw1, rw2, pm, peer}, errc
}

func (p *testPeer) handshake(t *testing.T) {
	td, currentBlock, genesis := p.pm.chainman.Status()
	msg := &statusData{
		ProtocolVersion: uint32(p.pm.protVer),
		NetworkId:       uint32(p.pm.netId),
		TD:              td,
		CurrentBlock:    currentBlock,
		GenesisBlock:    genesis,
	}
	if err := p2p.ExpectMsg(p, StatusMsg, msg); err != nil {
		t.Fatalf("status recv: %v", err)
	}
	if err := p2p.Send(p, StatusMsg, msg); err != nil {
		t.Fatalf("status send: %v", err)
	}
}

func (p *testPeer) close() {
	p.pipe.Close()
}

type fakeTxPool struct {
	// all transactions are collected.
	mu  sync.Mutex
	all []*types.Transaction
	// if added is non-nil, it receives added transactions.
	added chan<- []*types.Transaction
}

func (pool *fakeTxPool) AddTransactions(txs []*types.Transaction) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.all = append(pool.all, txs...)
	if pool.added != nil {
		pool.added <- txs
	}
}

func (pool *fakeTxPool) GetTransactions() types.Transactions {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	txs := make([]*types.Transaction, len(pool.all))
	copy(txs, pool.all)
	return types.Transactions(txs)
}

func newtx(from *crypto.Key, nonce uint64, datasize int) *types.Transaction {
	data := make([]byte, datasize)
	tx := types.NewTransaction(nonce, common.Address{}, big.NewInt(0), big.NewInt(100000), big.NewInt(0), data)
	tx, _ = tx.SignECDSA(from.PrivateKey)
	return tx
}
