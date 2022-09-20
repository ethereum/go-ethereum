package kcpxfer

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"testing"

	"github.com/xtaci/kcp-go"
)

type wrappedConn struct {
	net.PacketConn
}

func TestKCP(t *testing.T) {
	s1 := listen(t, "127.0.0.1:0")
	s2 := listen(t, "127.0.0.1:0")

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		receive(t, s1)
		wg.Done()
	}()

	go func() {
		send(t, s2, s1.LocalAddr().String())
		wg.Done()
	}()

	wg.Wait()
}

func setupKCP(s *kcp.UDPSession) {
	s.SetMtu(1200)
	s.SetStreamMode(true)

	// https://github.com/skywind3000/kcp/blob/master/README.en.md#protocol-configuration
	// Normal Mode: ikcp_nodelay(kcp, 0, 40, 0, 0);
	// Turbo Mode: ikcp_nodelay(kcp, 1, 10, 2, 1);
	s.SetNoDelay(1, 10, 2, 1)
	// s.SetNoDelay(0, 40, 0, 0)
}

func listen(t *testing.T, addr string) *net.UDPConn {
	socket, err := net.ListenPacket("udp4", addr)
	if err != nil {
		t.Fatal(err)
	}
	return socket.(*net.UDPConn)
}

func send(t *testing.T, conn *net.UDPConn, raddr string) error {
	wconn := wrappedConn{conn}

	s, err := kcp.NewConn(raddr, nil, 1, 1, wconn)
	if err != nil {
		t.Log("Could not establish KCP session:", err)
		return err
	}
	defer s.Close()

	setupKCP(s)

	t.Log("Transmitting data")
	for i := 0; i < 200; i++ {
		msg := make([]byte, 120)
		rand.Read(msg)
		hexmsg := []byte(hex.EncodeToString(msg))

		_, err := s.Write(hexmsg)
		if err != nil {
			return err
		}
		// t.Logf("Sent data: n=%d err=%v", n, err)
	}
	t.Log("sent data")
	if _, err := s.Write([]byte("FIN")); err != nil {
		return fmt.Errorf("unable to close connection: %s", err)
	}

	return nil
}

func receive(t *testing.T, conn *net.UDPConn) error {
	wconn := wrappedConn{conn}

	l, err := kcp.ServeConn(nil, 1, 1, wconn)
	if err != nil {
		return err
	}

	t.Log("Waiting for KCP conn")
	s, err := l.AcceptKCP()
	if err != nil {
		t.Log("accept error:", err)
		return err
	}

	t.Log("KCP socket accepted")
	setupKCP(s)

	for {
		buf := make([]byte, 2048)
		n, err := s.Read(buf)
		if err != nil {
			return err
		}
		if string(buf[:n]) == "FIN" {
			t.Log("connection finished")
			return nil
		}

		//		t.Log("Read KCP data:", string(buf[:n]))
	}

	return nil
}
