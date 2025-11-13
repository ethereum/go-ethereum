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
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
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
		common.HexToHash("0xaa"),
		common.HexToHash("0xbb"),
		common.HexToHash("0xcc"),
		common.HexToHash("0xdd"),
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

	delivery := &ReceiptsPacket70{
		RequestId:           req.id,
		LastBlockIncomplete: true,
		List: []*ReceiptList69{
			{
				items: []Receipt{
					{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 1))},
				},
			},
			{
				items: []Receipt{
					{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 2))},
				},
			},
		},
	}
	if _, err := peer.ReconstructReceiptsPacket(delivery); err != nil {
		t.Fatalf("first ReconstructReceiptsPacket failed: %v", err)
	}

	if err := peer.RequestPartialReceipts(req.id); err != nil {
		t.Fatalf("RequestPartialReceipts failed: %v", err)
	}

	var rereq *GetReceiptsPacket70
	select {
	case rereq = <-packetCh:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout waiting for re-request packet")
	}

	if _, ok := peer.receiptBuffer[req.id]; ok {
		t.Fatalf("receiptBuffer has stale request id")
	}
	if _, ok := peer.requestedReceipts[req.id]; ok {
		t.Fatalf("requestedReceipts has stale request id")
	}

	buffer, ok := peer.receiptBuffer[rereq.RequestId]
	if !ok {
		t.Fatalf("receiptBuffer should buffer incomplete receipts")
	}
	if rereq.FirstBlockReceiptIndex != uint64(len(buffer.list.items)) {
		t.Fatalf("unexpected FirstBlockReceiptIndex, got %d want %d", rereq.FirstBlockReceiptIndex, len(buffer.list.items))
	}
	if _, ok := peer.requestedReceipts[rereq.RequestId]; !ok {
		t.Fatalf("requestedReceipts should buffer receipt hashes")
	}

	delivery = &ReceiptsPacket70{
		RequestId:           rereq.RequestId,
		LastBlockIncomplete: false,
		List: []*ReceiptList69{
			{
				items: []Receipt{
					{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 1))},
				},
			},
			{
				items: []Receipt{
					{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 1))},
				},
			},
			{
				items: []Receipt{
					{GasUsed: 21_000, Logs: rlp.RawValue(make([]byte, 1))},
				},
			},
		},
	}
	if _, err := peer.ReconstructReceiptsPacket(delivery); err != nil {
		t.Fatalf("second ReconstructReceiptsPacket failed: %v", err)
	}
	if _, ok := peer.receiptBuffer[rereq.RequestId]; ok {
		t.Fatalf("receiptBuffer should be cleared after delivery")
	}
	if _, ok := peer.requestedReceipts[rereq.RequestId]; ok {
		t.Fatalf("requestedReceipts should be cleared after delivery")
	}
}
