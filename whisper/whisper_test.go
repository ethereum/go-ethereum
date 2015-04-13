package whisper

import (
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

type testNode struct {
	server *p2p.Server
	client *Whisper
}

func startNodes(n int) ([]*testNode, error) {
	// Start up the cluster of nodes
	cluster := make([]*testNode, 0, n)
	for i := 0; i < n; i++ {
		shh := New()

		// Generate the node identity
		key, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		name := common.MakeName(fmt.Sprintf("whisper-go-test-%d", i), "1.0")

		// Create an Ethereum server to communicate through
		server := &p2p.Server{
			PrivateKey: key,
			MaxPeers:   10,
			Name:       name,
			Protocols:  []p2p.Protocol{shh.Protocol()},
			ListenAddr: fmt.Sprintf(":%d", 30300+i),
			NAT:        nat.Any(),
		}
		if err := server.Start(); err != nil {
			return nil, err
		}
		// Peer online, store and iterate
		cluster = append(cluster, &testNode{
			server: server,
			client: shh,
		})
	}
	// Manually wire together the cluster nodes
	root := cluster[0].server.Self()
	for _, node := range cluster[1:] {
		node.server.SuggestPeer(root)
	}
	return cluster, nil
}

func stopNodes(cluster []*testNode) {
	for _, node := range cluster {
		node.server.Stop()
	}
}

func TestSelfMessage(t *testing.T) {
	// Start the single node cluster
	cluster, err := startNodes(1)
	if err != nil {
		t.Fatalf("failed to boot test cluster: %v", err)
	}
	defer stopNodes(cluster)

	client := cluster[0].client

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
	msg := NewMessage([]byte("hello whisper"))
	envelope, err := msg.Wrap(DefaultProofOfWork, Options{
		From: self,
		To:   &self.PublicKey,
		TTL:  DefaultTimeToLive,
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
	glog.SetV(6)
	glog.SetToStderr(true)

	// Start the sender-recipient cluster
	cluster, err := startNodes(2)
	if err != nil {
		t.Fatalf("failed to boot test cluster: %v", err)
	}
	defer stopNodes(cluster)

	sender := cluster[0].client
	senderId := sender.NewIdentity()

	recipient := cluster[1].client
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
	msg := NewMessage([]byte("hello whisper"))
	envelope, err := msg.Wrap(DefaultProofOfWork, Options{
		From: senderId,
		To:   &recipientId.PublicKey,
		TTL:  DefaultTimeToLive,
	})
	if err != nil {
		t.Fatalf("failed to wrap message: %v", err)
	}
	if err := sender.Send(envelope); err != nil {
		t.Fatalf("failed to send  direct message: %v", err)
	}
	// Wait for an arrival or a timeout
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("direct message receive timeout")
	}
}
