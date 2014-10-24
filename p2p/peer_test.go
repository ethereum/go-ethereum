package p2p

import (
	"bytes"
	"fmt"
	// "net"
	"testing"
	"time"
)

func TestPeer(t *testing.T) {
	handlers := make(Handlers)
	testProtocol := &TestProtocol{Msgs: []*Msg{}}
	handlers["aaa"] = func(p *Peer) Protocol { return testProtocol }
	handlers["ccc"] = func(p *Peer) Protocol { return testProtocol }
	addr := &TestAddr{"test:30"}
	conn := NewTestNetworkConnection(addr)
	_, server := SetupTestServer(handlers)
	server.Handshake()
	peer := NewPeer(conn, addr, true, server)
	// peer.Messenger().AddProtocols([]string{"aaa", "ccc"})
	peer.Start()
	defer peer.Stop()
	time.Sleep(2 * time.Millisecond)
	if len(conn.Out) != 1 {
		t.Errorf("handshake not sent")
	} else {
		out := conn.Out[0]
		packet := Packet(0, HandshakeMsg, P2PVersion, []byte(peer.server.identity.String()), []interface{}{peer.server.protocols}, peer.server.port, peer.server.identity.Pubkey()[1:])
		if bytes.Compare(out, packet) != 0 {
			t.Errorf("incorrect handshake packet %v != %v", out, packet)
		}
	}

	packet := Packet(0, HandshakeMsg, P2PVersion, []byte("peer"), []interface{}{"bbb", "aaa", "ccc"}, 30, []byte("0000000000000000000000000000000000000000000000000000000000000000"))
	conn.In(0, packet)
	time.Sleep(10 * time.Millisecond)

	pro, _ := peer.Messenger().protocols[0].(*BaseProtocol)
	if pro.state != handshakeReceived {
		t.Errorf("handshake not received")
	}
	if peer.Port != 30 {
		t.Errorf("port incorrectly set")
	}
	if peer.Id != "peer" {
		t.Errorf("id incorrectly set")
	}
	if string(peer.Pubkey) != "0000000000000000000000000000000000000000000000000000000000000000" {
		t.Errorf("pubkey incorrectly set")
	}
	fmt.Println(peer.Caps)
	if len(peer.Caps) != 3 || peer.Caps[0] != "aaa" || peer.Caps[1] != "bbb" || peer.Caps[2] != "ccc" {
		t.Errorf("protocols incorrectly set")
	}

	msg, _ := NewMsg(3)
	err := peer.Write("aaa", msg)
	if err != nil {
		t.Errorf("expect no error for known protocol: %v", err)
	} else {
		time.Sleep(1 * time.Millisecond)
		if len(conn.Out) != 2 {
			t.Errorf("msg not written")
		} else {
			out := conn.Out[1]
			packet := Packet(16, 3)
			if bytes.Compare(out, packet) != 0 {
				t.Errorf("incorrect packet %v != %v", out, packet)
			}
		}
	}

	msg, _ = NewMsg(2)
	err = peer.Write("ccc", msg)
	if err != nil {
		t.Errorf("expect no error for known protocol: %v", err)
	} else {
		time.Sleep(1 * time.Millisecond)
		if len(conn.Out) != 3 {
			t.Errorf("msg not written")
		} else {
			out := conn.Out[2]
			packet := Packet(21, 2)
			if bytes.Compare(out, packet) != 0 {
				t.Errorf("incorrect packet %v != %v", out, packet)
			}
		}
	}

	err = peer.Write("bbb", msg)
	time.Sleep(1 * time.Millisecond)
	if err == nil {
		t.Errorf("expect error for unknown protocol")
	}
}
