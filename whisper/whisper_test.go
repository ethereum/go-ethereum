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

package whisper

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

func startTestCluster(n int) []*Whisper {
	// Create the batch of simulated peers
	nodes := make([]*p2p.Peer, n)
	for i := 0; i < n; i++ {
		nodes[i] = p2p.NewPeer(discover.NodeID{}, "", nil)
	}
	whispers := make([]*Whisper, n)
	for i := 0; i < n; i++ {
		whispers[i] = New()
		whispers[i].Start()
	}
	// Wire all the peers to the root one
	for i := 1; i < n; i++ {
		src, dst := p2p.MsgPipe()

		go whispers[0].handlePeer(nodes[i], src)
		go whispers[i].handlePeer(nodes[0], dst)
	}
	return whispers
}

func TestSelfMessage(t *testing.T) {
	// Start the single node cluster
	client := startTestCluster(1)[0]

	// Start watching for self messages, signal any arrivals
	self := client.NewIdentity()
	done := make(chan struct{})

	client.Watch(Filter{
		To: &self.PublicKey,
		Fn: func(msg *Message) {
			close(done)
		},
	})
	// Send a dummy message to oneself
	msg := NewMessage([]byte("self whisper"))
	envelope, err := msg.Wrap(DefaultPoW, Options{
		From: self,
		To:   &self.PublicKey,
		TTL:  DefaultTTL,
	})
	if err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	// Dump the message into the system and wait for it to pop back out
	if err := client.Send(envelope); err != nil {
		t.Fatalf("failed to send self-message: %v", err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("self-message receive timeout")
	}
}

func TestDirectMessage(t *testing.T) {
	// Start the sender-recipient cluster
	cluster := startTestCluster(2)

	sender := cluster[0]
	senderId := sender.NewIdentity()

	recipient := cluster[1]
	recipientId := recipient.NewIdentity()

	// Watch for arriving messages on the recipient
	done := make(chan struct{})
	recipient.Watch(Filter{
		To: &recipientId.PublicKey,
		Fn: func(msg *Message) {
			close(done)
		},
	})
	// Send a dummy message from the sender
	msg := NewMessage([]byte("direct whisper"))
	envelope, err := msg.Wrap(DefaultPoW, Options{
		From: senderId,
		To:   &recipientId.PublicKey,
		TTL:  DefaultTTL,
	})
	if err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	if err := sender.Send(envelope); err != nil {
		t.Fatalf("failed to send direct message: %v", err)
	}
	// Wait for an arrival or a timeout
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("direct message receive timeout")
	}
}

func TestAnonymousBroadcast(t *testing.T) {
	testBroadcast(true, t)
}

func TestIdentifiedBroadcast(t *testing.T) {
	testBroadcast(false, t)
}

func testBroadcast(anonymous bool, t *testing.T) {
	// Start the single sender multi recipient cluster
	cluster := startTestCluster(3)

	sender := cluster[1]
	targets := cluster[1:]
	for _, target := range targets {
		if !anonymous {
			target.NewIdentity()
		}
	}
	// Watch for arriving messages on the recipients
	dones := make([]chan struct{}, len(targets))
	for i := 0; i < len(targets); i++ {
		done := make(chan struct{}) // need for the closure
		dones[i] = done

		targets[i].Watch(Filter{
			Topics: NewFilterTopicsFromStringsFlat("broadcast topic"),
			Fn: func(msg *Message) {
				close(done)
			},
		})
	}
	// Send a dummy message from the sender
	msg := NewMessage([]byte("broadcast whisper"))
	envelope, err := msg.Wrap(DefaultPoW, Options{
		Topics: NewTopicsFromStrings("broadcast topic"),
		TTL:    DefaultTTL,
	})
	if err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	if err := sender.Send(envelope); err != nil {
		t.Fatalf("failed to send broadcast message: %v", err)
	}
	// Wait for an arrival on each recipient, or timeouts
	timeout := time.After(time.Second)
	for _, done := range dones {
		select {
		case <-done:
		case <-timeout:
			t.Fatalf("broadcast message receive timeout")
		}
	}
}

func TestMessageExpiration(t *testing.T) {
	// Start the single node cluster and inject a dummy message
	node := startTestCluster(1)[0]

	message := NewMessage([]byte("expiring message"))
	envelope, err := message.Wrap(DefaultPoW, Options{
		TTL: time.Second,
	})
	if err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	if err := node.Send(envelope); err != nil {
		t.Fatalf("failed to inject message: %v", err)
	}
	// Check that the message is inside the cache
	if _, ok := node.messages[envelope.Hash()]; !ok {
		t.Fatalf("message not found in cache")
	}
	// Wait for expiration and check cache again
	time.Sleep(time.Second)     // wait for expiration
	time.Sleep(expirationCycle) // wait for cleanup cycle
	if _, ok := node.messages[envelope.Hash()]; ok {
		t.Fatalf("message not expired from cache")
	}
}
