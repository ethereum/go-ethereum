package whisper

import (
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
)

type testPeer struct {
	client *Whisper
	stream *p2p.MsgPipeRW
	termed chan struct{}
}

func startTestPeer() *testPeer {
	// Create a simulated P2P remote peer and data streams to it
	remote := p2p.NewPeer(randomNodeID(), randomNodeName(), whisperCaps())
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
	// Assemble and return the test peer
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
	case <-time.After(2 * transmissionTicks):
	case <-arrived:
		t.Fatalf("repeating message arrived")
	}
}
