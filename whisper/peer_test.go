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

package whisper

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

type testPeer struct {
	client *Whisper
	stream *p2p.MsgPipeRW
	termed chan struct{}
}

func startTestPeer() *testPeer {
	// Create a simulated P2P remote peer and data streams to it
	remote := p2p.NewPeer(discover.NodeID{}, "", nil)
	tester, tested := p2p.MsgPipe()

	// Create a whisper client and connect with it to the tester peer
	client := New()
	client.Start()

	termed := make(chan struct{})
	go func() {
		defer client.Stop()
		defer close(termed)
		defer tested.Close()

		client.handlePeer(remote, tested)
	}()

	return &testPeer{
		client: client,
		stream: tester,
		termed: termed,
	}
}

func startTestPeerInited() (*testPeer, error) {
	peer := startTestPeer()

	if err := p2p.ExpectMsg(peer.stream, statusCode, []uint64{protocolVersion}); err != nil {
		peer.stream.Close()
		return nil, err
	}
	if err := p2p.SendItems(peer.stream, statusCode, protocolVersion); err != nil {
		peer.stream.Close()
		return nil, err
	}
	return peer, nil
}

func TestPeerStatusMessage(t *testing.T) {
	tester := startTestPeer()

	// Wait for the handshake status message and check it
	if err := p2p.ExpectMsg(tester.stream, statusCode, []uint64{protocolVersion}); err != nil {
		t.Fatalf("status message mismatch: %v", err)
	}
	// Terminate the node
	tester.stream.Close()

	select {
	case <-tester.termed:
	case <-time.After(time.Second):
		t.Fatalf("local close timed out")
	}
}

func TestPeerHandshakeFail(t *testing.T) {
	tester := startTestPeer()

	// Wait for and check the handshake
	if err := p2p.ExpectMsg(tester.stream, statusCode, []uint64{protocolVersion}); err != nil {
		t.Fatalf("status message mismatch: %v", err)
	}
	// Send an invalid handshake status and verify disconnect
	if err := p2p.SendItems(tester.stream, messagesCode); err != nil {
		t.Fatalf("failed to send malformed status: %v", err)
	}
	select {
	case <-tester.termed:
	case <-time.After(time.Second):
		t.Fatalf("remote close timed out")
	}
}

func TestPeerHandshakeSuccess(t *testing.T) {
	tester := startTestPeer()

	// Wait for and check the handshake
	if err := p2p.ExpectMsg(tester.stream, statusCode, []uint64{protocolVersion}); err != nil {
		t.Fatalf("status message mismatch: %v", err)
	}
	// Send a valid handshake status and make sure connection stays live
	if err := p2p.SendItems(tester.stream, statusCode, protocolVersion); err != nil {
		t.Fatalf("failed to send status: %v", err)
	}
	select {
	case <-tester.termed:
		t.Fatalf("valid handshake disconnected")

	case <-time.After(100 * time.Millisecond):
	}
	// Clean up the test
	tester.stream.Close()

	select {
	case <-tester.termed:
	case <-time.After(time.Second):
		t.Fatalf("local close timed out")
	}
}

func TestPeerSend(t *testing.T) {
	// Start a tester and execute the handshake
	tester, err := startTestPeerInited()
	if err != nil {
		t.Fatalf("failed to start initialized peer: %v", err)
	}
	defer tester.stream.Close()

	// Construct a message and inject into the tester
	message := NewMessage([]byte("peer broadcast test message"))
	envelope, err := message.Wrap(DefaultPoW, Options{
		TTL: DefaultTTL,
	})
	if err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	if err := tester.client.Send(envelope); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
	// Check that the message is eventually forwarded
	payload := []interface{}{envelope}
	if err := p2p.ExpectMsg(tester.stream, messagesCode, payload); err != nil {
		t.Fatalf("message mismatch: %v", err)
	}
	// Make sure that even with a re-insert, an empty batch is received
	if err := tester.client.Send(envelope); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
	if err := p2p.ExpectMsg(tester.stream, messagesCode, []interface{}{}); err != nil {
		t.Fatalf("message mismatch: %v", err)
	}
}

func TestPeerDeliver(t *testing.T) {
	// Start a tester and execute the handshake
	tester, err := startTestPeerInited()
	if err != nil {
		t.Fatalf("failed to start initialized peer: %v", err)
	}
	defer tester.stream.Close()

	// Watch for all inbound messages
	arrived := make(chan struct{}, 1)
	tester.client.Watch(Filter{
		Fn: func(message *Message) {
			arrived <- struct{}{}
		},
	})
	// Construct a message and deliver it to the tester peer
	message := NewMessage([]byte("peer broadcast test message"))
	envelope, err := message.Wrap(DefaultPoW, Options{
		TTL: DefaultTTL,
	})
	if err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	if err := p2p.Send(tester.stream, messagesCode, []*Envelope{envelope}); err != nil {
		t.Fatalf("failed to transfer message: %v", err)
	}
	// Check that the message is delivered upstream
	select {
	case <-arrived:
	case <-time.After(time.Second):
		t.Fatalf("message delivery timeout")
	}
	// Check that a resend is not delivered
	if err := p2p.Send(tester.stream, messagesCode, []*Envelope{envelope}); err != nil {
		t.Fatalf("failed to transfer message: %v", err)
	}
	select {
	case <-time.After(2 * transmissionCycle):
	case <-arrived:
		t.Fatalf("repeating message arrived")
	}
}

func TestPeerMessageExpiration(t *testing.T) {
	// Start a tester and execute the handshake
	tester, err := startTestPeerInited()
	if err != nil {
		t.Fatalf("failed to start initialized peer: %v", err)
	}
	defer tester.stream.Close()

	// Fetch the peer instance for later inspection
	tester.client.peerMu.RLock()
	if peers := len(tester.client.peers); peers != 1 {
		t.Fatalf("peer pool size mismatch: have %v, want %v", peers, 1)
	}
	var peer *peer
	for peer, _ = range tester.client.peers {
		break
	}
	tester.client.peerMu.RUnlock()

	// Construct a message and pass it through the tester
	message := NewMessage([]byte("peer test message"))
	envelope, err := message.Wrap(DefaultPoW, Options{
		TTL: time.Second,
	})
	if err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	if err := tester.client.Send(envelope); err != nil {
		t.Fatalf("failed to send message: %v", err)
	}
	payload := []interface{}{envelope}
	if err := p2p.ExpectMsg(tester.stream, messagesCode, payload); err != nil {
		t.Fatalf("message mismatch: %v", err)
	}
	// Check that the message is inside the cache
	if !peer.known.Has(envelope.Hash()) {
		t.Fatalf("message not found in cache")
	}
	// Discard messages until expiration and check cache again
	exp := time.Now().Add(time.Second + expirationCycle)
	for time.Now().Before(exp) {
		if err := p2p.ExpectMsg(tester.stream, messagesCode, []interface{}{}); err != nil {
			t.Fatalf("message mismatch: %v", err)
		}
	}
	if peer.known.Has(envelope.Hash()) {
		t.Fatalf("message not expired from cache")
	}
}
