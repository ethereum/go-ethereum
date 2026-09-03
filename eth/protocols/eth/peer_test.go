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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/enode"
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

	peer := NewPeer(version, p2p.NewPeer(id, name, nil), net, backend.TxPool(), backend.BlobPool(), nil)
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

// TestBufferReceiptsNoProgress checks that an incomplete receipt response
// delivering no receipt is rejected, while an empty list at the end of a
// multi-block response is still accepted.
func TestBufferReceiptsNoProgress(t *testing.T) {
	p := &Peer{receiptBuffer: make(map[uint64]*receiptRequest)}

	newRequest := func(id uint64, blocks int) {
		req := new(receiptRequest)
		for i := 0; i < blocks; i++ {
			req.request = append(req.request, common.Hash{byte(i + 1)})
			req.gasUsed = append(req.gasUsed, 1_000_000)
			req.timestamps = append(req.timestamps, 0)
		}
		p.receiptBuffer[id] = req
	}

	// A single empty list with the incomplete flag makes no progress.
	newRequest(1, 1)
	if err := p.bufferReceipts(1, []*ReceiptList{NewReceiptList(nil)}, true); err == nil {
		t.Fatal("expected error for incomplete response without receipts")
	}
	if _, ok := p.receiptBuffer[1]; ok {
		t.Fatal("buffer entry not removed after invalid response")
	}

	// An empty list preceded by other lists advances the request and is valid.
	newRequest(2, 2)
	lists := []*ReceiptList{
		NewReceiptList([]*types.Receipt{{Status: 1, CumulativeGasUsed: 21000}}),
		NewReceiptList(nil),
	}
	if err := p.bufferReceipts(2, lists, true); err != nil {
		t.Fatalf("unexpected error for multi-block incomplete response: %v", err)
	}
	if _, ok := p.receiptBuffer[2]; !ok {
		t.Fatal("buffer entry missing after valid incomplete response")
	}
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
