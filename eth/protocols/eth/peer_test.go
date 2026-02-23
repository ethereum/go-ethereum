// Copyright 2020 The go-ethereum Authors
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

// This file contains some shares testing functionality, common to  multiple
// different files and modules being tested.

package eth

import (
	"crypto/rand"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/tracker"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// testPeer is a simulated peer to allow testing direct network calls.
type testPeer struct {
	*Peer

	net p2p.MsgReadWriter // Network layer reader/writer to simulate remote messaging
	app *p2p.MsgPipeRW    // Application layer reader/writer to simulate the local side
}

// newTestPeer creates a new peer registered at the given data backend.
func newTestPeer(name string, version uint, backend Backend) (*testPeer, <-chan error) {
	// Create a message pipe to communicate through
	app, net := p2p.MsgPipe()

	// Start the peer on a new thread
	var id enode.ID
	rand.Read(id[:])

	peer := NewPeer(version, p2p.NewPeer(id, name, nil), net, backend.TxPool())
	errc := make(chan error, 1)
	go func() {
		defer app.Close()

		errc <- backend.RunPeer(peer, func(peer *Peer) error {
			return Handle(backend, peer)
		})
	}()
	return &testPeer{app: app, net: net, Peer: peer}, errc
}

// close terminates the local side of the peer, notifying the remote protocol
// manager of termination.
func (p *testPeer) close() {
	p.Peer.Close()
	p.app.Close()
}

func TestPeerSet(t *testing.T) {
	size := 5
	s := newKnownCache(size)

	// add 10 items
	for i := 0; i < size*2; i++ {
		s.Add(common.Hash{byte(i)})
	}

	if s.Cardinality() != size {
		t.Fatalf("wrong size, expected %d but found %d", size, s.Cardinality())
	}

	vals := []common.Hash{}
	for i := 10; i < 20; i++ {
		vals = append(vals, common.Hash{byte(i)})
	}

	// add item in batch
	s.Add(vals...)
	if s.Cardinality() < size {
		t.Fatalf("bad size")
	}
}

func TestPartialReceipt(t *testing.T) {
	gen := func(_ int, g *core.BlockGen) {
		signer := types.HomesteadSigner{}

		for range 4 {
			tx, _ := types.SignTx(types.NewTransaction(g.TxNonce(testAddr), testAddr, big.NewInt(10), params.TxGas, g.BaseFee(), nil), signer, testKey)
			g.AddTx(tx)
		}
	}

	backend := newTestBackendWithGenerator(4, true, true, gen)
	defer backend.close()

	app, net := p2p.MsgPipe()
	var id enode.ID
	if _, err := rand.Read(id[:]); err != nil {
		t.Fatalf("failed to create random peer: %v", err)
	}

	peer := NewPeer(ETH70, p2p.NewPeer(id, "peer", nil), net, nil)

	packetCh := make(chan *GetReceiptsPacket70, 1)
	go func() {
		for {
			msg, err := app.ReadMsg()
			if err != nil {
				return
			}
			if msg.Code == GetReceiptsMsg {
				var pkt GetReceiptsPacket70
				if err := msg.Decode(&pkt); err == nil {
					select {
					case packetCh <- &pkt:
					default:
					}
				}
			}
			msg.Discard()
		}
	}()

	hashes := []common.Hash{
		backend.chain.GetBlockByNumber(1).Hash(),
		backend.chain.GetBlockByNumber(2).Hash(),
		backend.chain.GetBlockByNumber(3).Hash(),
		backend.chain.GetBlockByNumber(4).Hash(),
	}

	sink := make(chan *Response, 1)
	req, err := peer.RequestReceipts(hashes, sink)
	if err != nil {
		t.Fatalf("RequestReceipts failed: %v", err)
	}
	select {
	case _ = <-packetCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for request packet")
	}

	receipts := []Receipt{
		{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 1))},
	}
	logReceipts := []Receipt{
		{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 1))},
		{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 1))},
		{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 1))},
	}
	delivery := &ReceiptsPacket70{
		RequestId:           req.id,
		LastBlockIncomplete: true,
		List: encodeRL([]*ReceiptList69{
			{
				items: encodeRL(receipts),
			},
			{
				items: encodeRL(receipts),
			},
		}),
	}

	tresp := tracker.Response{ID: delivery.RequestId, MsgCode: ReceiptsMsg, Size: delivery.List.Len()}
	if err := peer.tracker.Fulfil(tresp); err != nil {
		t.Fatalf("Tracker failed: %v", err)
	}

	receiptList, _ := delivery.List.Items()
	if err := peer.bufferReceipts(delivery.RequestId, receiptList, delivery.LastBlockIncomplete, backend); err != nil {
		t.Fatalf("first bufferReceipts failed: %v", err)
	}

	if err := peer.requestPartialReceipts(req.id); err != nil {
		t.Fatalf("requestPartialReceipts failed: %v", err)
	}

	var rereq *GetReceiptsPacket70
	select {
	case rereq = <-packetCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for re-request packet")
	}

	buffer, ok := peer.receiptBuffer[rereq.RequestId]
	if !ok {
		t.Fatalf("receiptBuffer should buffer incomplete receipts")
	}
	if rereq.FirstBlockReceiptIndex != uint64(buffer.list[len(buffer.list)-1].items.Len()) {
		t.Fatalf("unexpected FirstBlockReceiptIndex, got %d want %d", rereq.FirstBlockReceiptIndex, buffer.list[len(buffer.list)-1].items.Len())
	}

	delivery = &ReceiptsPacket70{
		RequestId:           req.id,
		LastBlockIncomplete: true,
		List: encodeRL([]*ReceiptList69{
			{
				items: encodeRL(receipts),
			},
		}),
	}
	tresp = tracker.Response{ID: delivery.RequestId, MsgCode: ReceiptsMsg, Size: delivery.List.Len()}
	if err := peer.tracker.Fulfil(tresp); err != nil {
		t.Fatalf("Tracker failed: %v", err)
	}
	receiptLists, _ := delivery.List.Items()
	if err := peer.bufferReceipts(delivery.RequestId, receiptLists, delivery.LastBlockIncomplete, backend); err != nil {
		t.Fatalf("second bufferReceipts failed: %v", err)
	}

	if err := peer.requestPartialReceipts(req.id); err != nil {
		t.Fatalf("requestPartialReceipts failed: %v", err)
	}

	select {
	case rereq = <-packetCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for re-request packet")
	}

	buffer, ok = peer.receiptBuffer[rereq.RequestId]
	if !ok {
		t.Fatalf("receiptBuffer should buffer incomplete receipts")
	}
	if rereq.FirstBlockReceiptIndex != uint64(buffer.list[len(buffer.list)-1].items.Len()) {
		t.Fatalf("unexpected FirstBlockReceiptIndex, got %d want %d", rereq.FirstBlockReceiptIndex, buffer.list[len(buffer.list)-1].items.Len())
	}
	if len(rereq.GetReceiptsRequest) != 3 {
		t.Fatalf("wrong partial request range, got %d want %d", len(rereq.GetReceiptsRequest), 3)
	}

	delivery = &ReceiptsPacket70{
		RequestId:           rereq.RequestId,
		LastBlockIncomplete: false,
		List: encodeRL([]*ReceiptList69{
			{
				items: encodeRL(receipts),
			},
			{
				items: encodeRL(receipts),
			},
			{
				items: encodeRL(receipts),
			},
		}),
	}

	tresp = tracker.Response{ID: delivery.RequestId, MsgCode: ReceiptsMsg, Size: delivery.List.Len()}
	if err := peer.tracker.Fulfil(tresp); err != nil {
		t.Fatalf("Tracker failed: %v", err)
	}
	receiptList, _ = delivery.List.Items()
	if err := peer.bufferReceipts(delivery.RequestId, receiptList, delivery.LastBlockIncomplete, backend); err != nil {
		t.Fatalf("third bufferReceipts failed: %v", err)
	}
	merged := peer.flushReceipts(rereq.RequestId)
	if merged == nil {
		t.Fatalf("flushReceipts should return merged receipt lists")
	}
	if _, ok := peer.receiptBuffer[rereq.RequestId]; ok {
		t.Fatalf("receiptBuffer should be cleared after flush")
	}
	for i, list := range merged {
		if i == 1 {
			if list.items.Len() != len(logReceipts) {
				t.Fatalf("wrong response buffering, got %d want %d", list.items.Len(), len(logReceipts))
			}
		} else {
			if list.items.Len() != len(receipts) {
				t.Fatalf("wrong response buffering, got %d want %d", list.items.Len(), len(receipts))
			}
		}
	}
}

func TestPartialReceiptFailure(t *testing.T) {
	gen := func(_ int, g *core.BlockGen) {
		signer := types.HomesteadSigner{}

		for range 4 {
			tx, _ := types.SignTx(types.NewTransaction(g.TxNonce(testAddr), testAddr, big.NewInt(10), params.TxGas, g.BaseFee(), nil), signer, testKey)
			g.AddTx(tx)
		}
	}

	backend := newTestBackendWithGenerator(4, true, true, gen)
	defer backend.close()

	app, net := p2p.MsgPipe()
	var id enode.ID
	if _, err := rand.Read(id[:]); err != nil {
		t.Fatalf("failed to create random peer: %v", err)
	}

	peer := NewPeer(ETH70, p2p.NewPeer(id, "peer", nil), net, nil)

	packetCh := make(chan *GetReceiptsPacket70, 1)
	go func() {
		for {
			msg, err := app.ReadMsg()
			if err != nil {
				return
			}
			if msg.Code == GetReceiptsMsg {
				var pkt GetReceiptsPacket70
				if err := msg.Decode(&pkt); err == nil {
					select {
					case packetCh <- &pkt:
					default:
					}
				}
			}
			msg.Discard()
		}
	}()

	// If a peer delivers response which is never requested, it should fail the validation
	delivery := &ReceiptsPacket70{
		RequestId:           66,
		LastBlockIncomplete: true,
		List: encodeRL([]*ReceiptList69{
			{
				items: encodeRL([]Receipt{
					{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 1))},
				}),
			},
			{
				items: encodeRL([]Receipt{
					{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 2))},
				}),
			},
		}),
	}
	receiptList, _ := delivery.List.Items()
	err := peer.bufferReceipts(delivery.RequestId, receiptList, delivery.LastBlockIncomplete, backend)
	if err == nil {
		t.Fatal("Unknown response should be dropped")
	}

	// If a peer deliverse excessive amount of receipts, it should also fail the validation
	hashes := []common.Hash{
		backend.chain.GetBlockByNumber(1).Hash(),
		backend.chain.GetBlockByNumber(2).Hash(),
		backend.chain.GetBlockByNumber(3).Hash(),
		backend.chain.GetBlockByNumber(4).Hash(),
	}

	// Case 1 ) The number of receipts exceeds maximum tx count
	sink := make(chan *Response, 1)
	req, err := peer.RequestReceipts(hashes, sink)
	if err != nil {
		t.Fatalf("RequestReceipts failed: %v", err)
	}
	select {
	case _ = <-packetCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for request packet")
	}

	maxTxCount := backend.chain.GetBlockByNumber(1).GasUsed() / 21_000
	excessiveReceipts := []Receipt{{Logs: rlp.RawValue(make([]byte, 1))}}
	for range maxTxCount {
		excessiveReceipts = append(excessiveReceipts, Receipt{Logs: rlp.RawValue(make([]byte, 1))})
	}
	delivery = &ReceiptsPacket70{
		RequestId:           req.id,
		LastBlockIncomplete: true,
		List: encodeRL([]*ReceiptList69{{
			items: encodeRL(excessiveReceipts),
		}}),
	}
	receiptList, _ = delivery.List.Items()
	err = peer.bufferReceipts(delivery.RequestId, receiptList, delivery.LastBlockIncomplete, backend)
	if err == nil {
		t.Fatal("Response with the excessive number of receipts should fail the validation")
	}

	// Case 2 ) Total receipt size exceeds the block gas limit
	req, err = peer.RequestReceipts(hashes, sink)
	if err != nil {
		t.Fatalf("RequestReceipts failed: %v", err)
	}
	select {
	case _ = <-packetCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for request packet")
	}
	maxReceiptSize := backend.chain.GetBlockByNumber(1).GasUsed() / params.LogDataGas
	delivery = &ReceiptsPacket70{
		RequestId:           req.id,
		LastBlockIncomplete: true,
		List: encodeRL([]*ReceiptList69{{
			items: encodeRL([]Receipt{
				{Logs: rlp.RawValue(make([]byte, maxReceiptSize+1))},
			}),
		}}),
	}
	receiptList, _ = delivery.List.Items()
	err = peer.bufferReceipts(delivery.RequestId, receiptList, delivery.LastBlockIncomplete, backend)
	if err == nil {
		t.Fatal("Response with the large log size should fail the validation")
	}
}
