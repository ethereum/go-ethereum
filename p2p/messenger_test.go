package p2p

import (
	// "fmt"
	"bytes"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
)

func setupMessenger(handlers Handlers) (*TestNetworkConnection, chan *PeerError, *Messenger) {
	errchan := NewPeerErrorChannel()
	addr := &TestAddr{"test:30303"}
	net := NewTestNetworkConnection(addr)
	conn := NewConnection(net, errchan)
	mess := NewMessenger(nil, conn, errchan, handlers)
	mess.Start()
	return net, errchan, mess
}

type TestProtocol struct {
	Msgs []*Msg
}

func (self *TestProtocol) Start() {
}

func (self *TestProtocol) Stop() {
}

func (self *TestProtocol) Offset() MsgCode {
	return MsgCode(5)
}

func (self *TestProtocol) HandleIn(msg *Msg, response chan *Msg) {
	self.Msgs = append(self.Msgs, msg)
	close(response)
}

func (self *TestProtocol) HandleOut(msg *Msg) bool {
	if msg.Code() > 3 {
		return false
	} else {
		return true
	}
}

func (self *TestProtocol) Name() string {
	return "a"
}

func Packet(offset MsgCode, code MsgCode, params ...interface{}) []byte {
	msg, _ := NewMsg(code, params...)
	encoded := msg.Encode(offset)
	packet := []byte{34, 64, 8, 145}
	packet = append(packet, ethutil.NumberToBytes(uint32(len(encoded)), 32)...)
	return append(packet, encoded...)
}

func TestRead(t *testing.T) {
	handlers := make(Handlers)
	testProtocol := &TestProtocol{Msgs: []*Msg{}}
	handlers["a"] = func(p *Peer) Protocol { return testProtocol }
	net, _, mess := setupMessenger(handlers)
	mess.AddProtocols([]string{"a"})
	defer mess.Stop()
	wait := 1 * time.Millisecond
	packet := Packet(16, 1, uint32(1), "000")
	go net.In(0, packet)
	time.Sleep(wait)
	if len(testProtocol.Msgs) != 1 {
		t.Errorf("msg not relayed to correct protocol")
	} else {
		if testProtocol.Msgs[0].Code() != 1 {
			t.Errorf("incorrect msg code relayed to protocol")
		}
	}
}

func TestWrite(t *testing.T) {
	handlers := make(Handlers)
	testProtocol := &TestProtocol{Msgs: []*Msg{}}
	handlers["a"] = func(p *Peer) Protocol { return testProtocol }
	net, _, mess := setupMessenger(handlers)
	mess.AddProtocols([]string{"a"})
	defer mess.Stop()
	wait := 1 * time.Millisecond
	msg, _ := NewMsg(3, uint32(1), "000")
	err := mess.Write("b", msg)
	if err == nil {
		t.Errorf("expect error for unknown protocol")
	}
	err = mess.Write("a", msg)
	if err != nil {
		t.Errorf("expect no error for known protocol: %v", err)
	} else {
		time.Sleep(wait)
		if len(net.Out) != 1 {
			t.Errorf("msg not written")
		} else {
			out := net.Out[0]
			packet := Packet(16, 3, uint32(1), "000")
			if bytes.Compare(out, packet) != 0 {
				t.Errorf("incorrect packet %v", out)
			}
		}
	}
}

func TestPulse(t *testing.T) {
	net, _, mess := setupMessenger(make(Handlers))
	defer mess.Stop()
	ping := false
	timeout := false
	pingTimeout := 10 * time.Millisecond
	gracePeriod := 200 * time.Millisecond
	go mess.PingPong(pingTimeout, gracePeriod, func() { ping = true }, func() { timeout = true })
	net.In(0, Packet(0, 1))
	if ping {
		t.Errorf("ping sent too early")
	}
	time.Sleep(pingTimeout + 100*time.Millisecond)
	if !ping {
		t.Errorf("no ping sent after timeout")
	}
	if timeout {
		t.Errorf("timeout too early")
	}
	ping = false
	net.In(0, Packet(0, 1))
	time.Sleep(pingTimeout + 100*time.Millisecond)
	if !ping {
		t.Errorf("no ping sent after timeout")
	}
	if timeout {
		t.Errorf("timeout too early")
	}
	ping = false
	time.Sleep(gracePeriod)
	if ping {
		t.Errorf("ping called twice")
	}
	if !timeout {
		t.Errorf("no timeout after grace period")
	}
}
