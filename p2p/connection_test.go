package p2p

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

type TestNetworkConnection struct {
	in      chan []byte
	current []byte
	Out     [][]byte
	addr    net.Addr
}

func NewTestNetworkConnection(addr net.Addr) *TestNetworkConnection {
	return &TestNetworkConnection{
		in:      make(chan []byte),
		current: []byte{},
		Out:     [][]byte{},
		addr:    addr,
	}
}

func (self *TestNetworkConnection) In(latency time.Duration, packets ...[]byte) {
	time.Sleep(latency)
	for _, s := range packets {
		self.in <- s
	}
}

func (self *TestNetworkConnection) Read(buff []byte) (n int, err error) {
	if len(self.current) == 0 {
		select {
		case self.current = <-self.in:
		default:
			return 0, io.EOF
		}
	}
	length := len(self.current)
	if length > len(buff) {
		copy(buff[:], self.current[:len(buff)])
		self.current = self.current[len(buff):]
		return len(buff), nil
	} else {
		copy(buff[:length], self.current[:])
		self.current = []byte{}
		return length, io.EOF
	}
}

func (self *TestNetworkConnection) Write(buff []byte) (n int, err error) {
	self.Out = append(self.Out, buff)
	fmt.Printf("net write %v\n%v\n", len(self.Out), buff)
	return len(buff), nil
}

func (self *TestNetworkConnection) Close() (err error) {
	return
}

func (self *TestNetworkConnection) LocalAddr() (addr net.Addr) {
	return
}

func (self *TestNetworkConnection) RemoteAddr() (addr net.Addr) {
	return self.addr
}

func (self *TestNetworkConnection) SetDeadline(t time.Time) (err error) {
	return
}

func (self *TestNetworkConnection) SetReadDeadline(t time.Time) (err error) {
	return
}

func (self *TestNetworkConnection) SetWriteDeadline(t time.Time) (err error) {
	return
}

func setupConnection() (*Connection, *TestNetworkConnection) {
	addr := &TestAddr{"test:30303"}
	net := NewTestNetworkConnection(addr)
	conn := NewConnection(net, NewPeerErrorChannel())
	conn.Open()
	return conn, net
}

func TestReadingNilPacket(t *testing.T) {
	conn, net := setupConnection()
	go net.In(0, []byte{})
	// time.Sleep(10 * time.Millisecond)
	select {
	case packet := <-conn.Read():
		t.Errorf("read %v", packet)
	case err := <-conn.Error():
		t.Errorf("incorrect error %v", err)
	default:
	}
	conn.Close()
}

func TestReadingShortPacket(t *testing.T) {
	conn, net := setupConnection()
	go net.In(0, []byte{0})
	select {
	case packet := <-conn.Read():
		t.Errorf("read %v", packet)
	case err := <-conn.Error():
		if err.Code != PacketTooShort {
			t.Errorf("incorrect error %v, expected %v", err.Code, PacketTooShort)
		}
	}
	conn.Close()
}

func TestReadingInvalidPacket(t *testing.T) {
	conn, net := setupConnection()
	go net.In(0, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	select {
	case packet := <-conn.Read():
		t.Errorf("read %v", packet)
	case err := <-conn.Error():
		if err.Code != MagicTokenMismatch {
			t.Errorf("incorrect error %v, expected %v", err.Code, MagicTokenMismatch)
		}
	}
	conn.Close()
}

func TestReadingInvalidPayload(t *testing.T) {
	conn, net := setupConnection()
	go net.In(0, []byte{34, 64, 8, 145, 0, 0, 0, 2, 0})
	select {
	case packet := <-conn.Read():
		t.Errorf("read %v", packet)
	case err := <-conn.Error():
		if err.Code != PayloadTooShort {
			t.Errorf("incorrect error %v, expected %v", err.Code, PayloadTooShort)
		}
	}
	conn.Close()
}

func TestReadingEmptyPayload(t *testing.T) {
	conn, net := setupConnection()
	go net.In(0, []byte{34, 64, 8, 145, 0, 0, 0, 0})
	time.Sleep(10 * time.Millisecond)
	select {
	case packet := <-conn.Read():
		t.Errorf("read %v", packet)
	default:
	}
	select {
	case err := <-conn.Error():
		code := err.Code
		if code != EmptyPayload {
			t.Errorf("incorrect error, expected EmptyPayload, got %v", code)
		}
	default:
		t.Errorf("no error, expected EmptyPayload")
	}
	conn.Close()
}

func TestReadingCompletePacket(t *testing.T) {
	conn, net := setupConnection()
	go net.In(0, []byte{34, 64, 8, 145, 0, 0, 0, 1, 1})
	time.Sleep(10 * time.Millisecond)
	select {
	case packet := <-conn.Read():
		if bytes.Compare(packet, []byte{1}) != 0 {
			t.Errorf("incorrect payload read")
		}
	case err := <-conn.Error():
		t.Errorf("incorrect error %v", err)
	default:
		t.Errorf("nothing read")
	}
	conn.Close()
}

func TestReadingTwoCompletePackets(t *testing.T) {
	conn, net := setupConnection()
	go net.In(0, []byte{34, 64, 8, 145, 0, 0, 0, 1, 0, 34, 64, 8, 145, 0, 0, 0, 1, 1})

	for i := 0; i < 2; i++ {
		time.Sleep(10 * time.Millisecond)
		select {
		case packet := <-conn.Read():
			if bytes.Compare(packet, []byte{byte(i)}) != 0 {
				t.Errorf("incorrect payload read")
			}
		case err := <-conn.Error():
			t.Errorf("incorrect error %v", err)
		default:
			t.Errorf("nothing read")
		}
	}
	conn.Close()
}

func TestWriting(t *testing.T) {
	conn, net := setupConnection()
	conn.Write() <- []byte{0}
	time.Sleep(10 * time.Millisecond)
	if len(net.Out) == 0 {
		t.Errorf("no output")
	} else {
		out := net.Out[0]
		if bytes.Compare(out, []byte{34, 64, 8, 145, 0, 0, 0, 1, 0}) != 0 {
			t.Errorf("incorrect packet %v", out)
		}
	}
	conn.Close()
}

// hello packet with client id ABC: 0x22 40 08 91 00 00 00 08 84 00 00 00 43414243
