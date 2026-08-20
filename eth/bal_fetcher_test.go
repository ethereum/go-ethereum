// Copyright 2026 The go-ethereum Authors
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
	"bytes"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// balRigChain builds an Amsterdam chain whose blocks carry access lists.
func balRigChain(t *testing.T) (*core.Genesis, []*types.Block) {
	t.Helper()
	u64 := func(v uint64) *uint64 { return &v }
	config := *params.MergedTestChainConfig
	config.AmsterdamTime = u64(0)
	config.Ethash = nil

	var (
		recipient = common.Address{0xba, 0x1f}
		signer    = types.LatestSigner(&config)
		gasPrice  = big.NewInt(params.InitialBaseFee + 1)
	)
	gspec := &core.Genesis{
		Config:     &config,
		Difficulty: common.Big0,
		BaseFee:    big.NewInt(params.InitialBaseFee),
		GasLimit:   30_000_000,
		Alloc: types.GenesisAlloc{
			testAddr: {Balance: new(big.Int).Mul(big.NewInt(1000), big.NewInt(params.Ether))},
		},
	}
	_, blocks, _ := core.GenerateChainWithGenesis(gspec, beacon.New(ethash.NewFaker()), 6, func(i int, b *core.BlockGen) {
		b.AddTx(types.MustSignNewTx(testKey, signer, &types.LegacyTx{
			Nonce: uint64(i), Gas: 1_000_000, GasPrice: gasPrice, To: &recipient, Value: big.NewInt(1000),
		}))
	})
	return gspec, blocks
}

// newBALTestHandler is newTestHandler over the access-list rig chain.
func newBALTestHandler(t *testing.T, gspec *core.Genesis, blocks []*types.Block) *testHandler {
	t.Helper()
	db := rawdb.NewMemoryDatabase()
	chain, err := core.NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chain.InsertChain(blocks); err != nil {
		t.Fatal(err)
	}
	txpool := newTestTxPool()
	h, err := newHandler(&handlerConfig{
		Database:   db,
		Chain:      chain,
		TxPool:     txpool,
		BlobPool:   txpool,
		Network:    1,
		Sync:       ethconfig.FullSync,
		BloomCache: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	h.Start(1000)
	return &testHandler{db: db, chain: chain, txpool: txpool, blobpool: txpool, handler: h}
}

func waitPeers(t *testing.T, h *handler, n int) {
	t.Helper()
	for start := time.Now(); h.peers.len() < n; time.Sleep(10 * time.Millisecond) {
		if time.Since(start) > 3*time.Second {
			t.Fatal("peer never registered")
		}
	}
}

// TestBALFetcherFetchesMissing wires two live handlers over a pipe: the
// fetcher fills the local gaps from the honest remote and kicks the
// migration.
func TestBALFetcherFetchesMissing(t *testing.T) {
	gspec, blocks := balRigChain(t)
	local := newBALTestHandler(t, gspec, blocks)
	defer local.close()
	remote := newBALTestHandler(t, gspec, blocks)
	defer remote.close()

	var missing []core.BALRequest
	for _, b := range blocks[1:4] {
		if !rawdb.HasAccessList(remote.db, b.Hash(), b.NumberU64()) {
			t.Fatalf("rig: block %d has no stored list to serve", b.NumberU64())
		}
		missing = append(missing, core.BALRequest{Number: b.NumberU64(), Hash: b.Hash()})
		rawdb.DeleteAccessList(local.db, b.Hash(), b.NumberU64())
	}

	p2pLocal, p2pRemote := p2p.MsgPipe()
	defer p2pLocal.Close()
	defer p2pRemote.Close()
	peerAtLocal := eth.NewPeer(eth.ETH71, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pLocal), p2pLocal, local.txpool, local.txpool, nil)
	peerAtRemote := eth.NewPeer(eth.ETH71, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pRemote), p2pRemote, remote.txpool, remote.txpool, nil)
	defer peerAtLocal.Close()
	defer peerAtRemote.Close()
	go local.handler.runEthPeer(peerAtLocal, func(p *eth.Peer) error {
		return eth.Handle((*ethHandler)(local.handler), p)
	})
	go remote.handler.runEthPeer(peerAtRemote, func(p *eth.Peer) error {
		return eth.Handle((*ethHandler)(remote.handler), p)
	})
	waitPeers(t, local.handler, 1)

	kicked := make(chan struct{}, 1)
	f := newBALFetcher(local.db, local.chain, local.handler.peers, local.handler.removePeer, func() {
		select {
		case kicked <- struct{}{}:
		default:
		}
	})
	defer f.stop()
	f.request(missing)

	for start := time.Now(); ; time.Sleep(10 * time.Millisecond) {
		landed := 0
		for _, r := range missing {
			if rawdb.HasAccessList(local.db, r.Hash, r.Number) {
				landed++
			}
		}
		if landed == len(missing) {
			break
		}
		if time.Since(start) > 5*time.Second {
			t.Fatalf("lists never landed: %d of %d", landed, len(missing))
		}
	}
	for _, r := range missing {
		got := rawdb.ReadAccessListRLP(local.db, r.Hash, r.Number)
		want := rawdb.ReadAccessListRLP(remote.db, r.Hash, r.Number)
		if !bytes.Equal(got, want) {
			t.Fatalf("block %d list differs from the served one", r.Number)
		}
	}
	select {
	case <-kicked:
	case <-time.After(time.Second):
		t.Fatal("fetcher never kicked the migration")
	}
}

// TestBALFetcherDropsForgingPeer serves syntactically valid lists of the
// wrong block: nothing may be stored and the peer must go.
func TestBALFetcherDropsForgingPeer(t *testing.T) {
	gspec, blocks := balRigChain(t)
	local := newBALTestHandler(t, gspec, blocks)
	defer local.close()

	target := blocks[2]
	forged := rawdb.ReadAccessListRLP(local.db, blocks[0].Hash(), blocks[0].NumberU64())
	if len(forged) == 0 {
		t.Fatal("rig: no list to forge with")
	}
	rawdb.DeleteAccessList(local.db, target.Hash(), target.NumberU64())

	p2pLocal, p2pRemote := p2p.MsgPipe()
	defer p2pLocal.Close()
	defer p2pRemote.Close()
	peerAtLocal := eth.NewPeer(eth.ETH71, p2p.NewPeerPipe(enode.ID{1}, "", nil, p2pLocal), p2pLocal, local.txpool, local.txpool, nil)
	remote := eth.NewPeer(eth.ETH71, p2p.NewPeerPipe(enode.ID{2}, "", nil, p2pRemote), p2pRemote, local.txpool, local.txpool, nil)
	defer peerAtLocal.Close()
	defer remote.Close()
	go local.handler.runEthPeer(peerAtLocal, func(p *eth.Peer) error {
		return eth.Handle((*ethHandler)(local.handler), p)
	})
	head := local.chain.CurrentBlock()
	if err := remote.Handshake(1, local.chain, eth.BlockRangeUpdatePacket{EarliestBlock: 0, LatestBlock: head.Number.Uint64(), LatestBlockHash: head.Hash()}); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	go func() {
		for {
			msg, err := p2pRemote.ReadMsg()
			if err != nil {
				return
			}
			if msg.Code != eth.GetBlockAccessListsMsg {
				msg.Discard()
				continue
			}
			var query eth.GetBlockAccessListsPacket
			if err := msg.Decode(&query); err != nil {
				return
			}
			var list rlp.RawList[rlp.RawValue]
			for range query.GetBlockAccessListsRequest {
				list.AppendRaw(forged)
			}
			remote.ReplyBlockAccessLists(query.RequestId, list)
		}
	}()
	waitPeers(t, local.handler, 1)

	dropped := make(chan struct{})
	var once sync.Once
	f := newBALFetcher(local.db, local.chain, local.handler.peers, func(id string) {
		once.Do(func() { close(dropped) })
		local.handler.removePeer(id)
	}, func() {})
	defer f.stop()
	f.request([]core.BALRequest{{Number: target.NumberU64(), Hash: target.Hash()}})

	select {
	case <-dropped:
	case <-time.After(5 * time.Second):
		t.Fatal("forging peer never dropped")
	}
	if rawdb.HasAccessList(local.db, target.Hash(), target.NumberU64()) {
		t.Fatal("a forged list was stored")
	}
	for start := time.Now(); local.handler.peers.len() != 0; time.Sleep(10 * time.Millisecond) {
		if time.Since(start) > 3*time.Second {
			t.Fatal("peer still registered after the drop")
		}
	}
}
