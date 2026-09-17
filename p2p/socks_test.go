// Copyright 2026 The go-ethereum Authors
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

package p2p

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/p2p/enode"
)

// TestSocksDialer checks that an outbound peer connection is carried by the proxy
// and arrives at the address taken from the enode record.
func TestSocksDialer(t *testing.T) {
	t.Parallel()

	dest := newEchoServer(t)
	ps := newSocksProxyServer(t, nil)

	dialer, err := newSocksDialer("socks5://"+ps.addr, &net.Dialer{})
	if err != nil {
		t.Fatal("can't create dialer:", err)
	}
	conn, err := dialer.Dial(context.Background(), nodeWithTCPEndpoint(dest))
	if err != nil {
		t.Fatal("dial through proxy failed:", err)
	}
	defer conn.Close()

	if got := roundtrip(t, conn); got != "ping" {
		t.Fatalf("echoed %q, want %q", got, "ping")
	}
	if got := ps.lastTarget(); got != dest {
		t.Fatalf("proxy asked to connect to %q, want %q", got, dest)
	}
}

// TestSocksDialerAuth checks that credentials in the proxy URL are used.
func TestSocksDialerAuth(t *testing.T) {
	t.Parallel()

	dest := newEchoServer(t)
	creds := &proxyCredentials{user: "geth", password: "hunter2"}
	ps := newSocksProxyServer(t, creds)

	dialer, err := newSocksDialer("socks5://geth:hunter2@"+ps.addr, &net.Dialer{})
	if err != nil {
		t.Fatal("can't create dialer:", err)
	}
	conn, err := dialer.Dial(context.Background(), nodeWithTCPEndpoint(dest))
	if err != nil {
		t.Fatal("dial through authenticated proxy failed:", err)
	}
	defer conn.Close()

	if got := roundtrip(t, conn); got != "ping" {
		t.Fatalf("echoed %q, want %q", got, "ping")
	}
}

// TestSocksDialerWrongAuth checks that a dial fails when the proxy rejects the
// supplied credentials, rather than falling back to a direct connection.
func TestSocksDialerWrongAuth(t *testing.T) {
	t.Parallel()

	dest := newEchoServer(t)
	ps := newSocksProxyServer(t, &proxyCredentials{user: "geth", password: "hunter2"})

	dialer, err := newSocksDialer("socks5://geth:wrong@"+ps.addr, &net.Dialer{})
	if err != nil {
		t.Fatal("can't create dialer:", err)
	}
	if conn, err := dialer.Dial(context.Background(), nodeWithTCPEndpoint(dest)); err == nil {
		conn.Close()
		t.Fatal("dial succeeded with wrong credentials")
	}
}

// TestSocksDialerNoTCPEndpoint checks that a node without a TCP port is rejected
// the same way the direct dialer rejects it.
func TestSocksDialerNoTCPEndpoint(t *testing.T) {
	t.Parallel()

	ps := newSocksProxyServer(t, nil)
	dialer, err := newSocksDialer("socks5://"+ps.addr, &net.Dialer{})
	if err != nil {
		t.Fatal("can't create dialer:", err)
	}
	node := enode.NewV4(&newkey().PublicKey, nil, 0, 0)
	if _, err := dialer.Dial(context.Background(), node); !errors.Is(err, errNoPort) {
		t.Fatalf("got error %v, want %v", err, errNoPort)
	}
}

func TestNewSocksDialerURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url     string
		wantErr string
	}{
		{url: "socks5://127.0.0.1:9050"},
		{url: "socks5h://127.0.0.1:9050"},
		{url: "socks5://user:password@127.0.0.1:1080"},
		{url: "socks5://[::1]:9050"},
		{url: "", wantErr: "missing scheme"},
		{url: "127.0.0.1:9050", wantErr: "missing scheme"},
		{url: "http://127.0.0.1:8080", wantErr: "unsupported scheme"},
		{url: "socks4://127.0.0.1:1080", wantErr: "unsupported scheme"},
		{url: "socks5://127.0.0.1", wantErr: "host:port"},
		{url: "socks5://", wantErr: "host:port"},
		{url: "socks5://%zz", wantErr: "invalid SOCKS proxy URL"},
	}
	for _, test := range tests {
		_, err := newSocksDialer(test.url, &net.Dialer{})
		switch {
		case test.wantErr == "" && err != nil:
			t.Errorf("newSocksDialer(%q) failed: %v", test.url, err)
		case test.wantErr != "" && err == nil:
			t.Errorf("newSocksDialer(%q) succeeded, want error %q", test.url, test.wantErr)
		case test.wantErr != "" && !strings.Contains(err.Error(), test.wantErr):
			t.Errorf("newSocksDialer(%q) error %q, want it to contain %q", test.url, err, test.wantErr)
		}
	}
}

// TestServerSocksProxyRequiresNoDiscovery checks that the server refuses to start
// with a proxy while discovery is enabled. Discovery is UDP-only and would leak the
// real address of the node, so this must fail loudly instead of starting.
func TestServerSocksProxyRequiresNoDiscovery(t *testing.T) {
	t.Parallel()

	srv := &Server{Config: Config{
		PrivateKey:  newkey(),
		SocksProxy:  "socks5://127.0.0.1:9050",
		NoDiscovery: false,
		NoDial:      true,
		MaxPeers:    1,
	}}
	err := srv.Start()
	if err == nil {
		srv.Stop()
		t.Fatal("server started with a proxy while discovery was enabled")
	}
	if !errors.Is(err, errDiscoveryNotProxied) {
		t.Fatalf("got error %v, want %v", err, errDiscoveryNotProxied)
	}
}

// TestServerSocksProxyConflictsWithDialer checks the two mutually exclusive ways of
// overriding the dialer are rejected together rather than one silently winning.
func TestServerSocksProxyConflictsWithDialer(t *testing.T) {
	t.Parallel()

	srv := &Server{Config: Config{
		PrivateKey:  newkey(),
		SocksProxy:  "socks5://127.0.0.1:9050",
		NoDiscovery: true,
		NoDial:      true,
		MaxPeers:    1,
		Dialer:      tcpDialer{&net.Dialer{}},
	}}
	err := srv.Start()
	if err == nil {
		srv.Stop()
		t.Fatal("server started with both Dialer and SocksProxy set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("got error %v, want it to mention that the options are mutually exclusive", err)
	}
}

// TestServerSocksProxyUsedForDialing checks that a server configured with a proxy
// actually dials peers through it.
func TestServerSocksProxyUsedForDialing(t *testing.T) {
	t.Parallel()

	dest := newEchoServer(t)
	ps := newSocksProxyServer(t, nil)

	srv := &Server{Config: Config{
		PrivateKey:  newkey(),
		SocksProxy:  "socks5://" + ps.addr,
		NoDiscovery: true,
		MaxPeers:    1,
	}}
	if err := srv.Start(); err != nil {
		t.Fatal("can't start server:", err)
	}
	defer srv.Stop()

	conn, err := srv.nodeDialer.Dial(context.Background(), nodeWithTCPEndpoint(dest))
	if err != nil {
		t.Fatal("dial through server dialer failed:", err)
	}
	defer conn.Close()

	if got := ps.lastTarget(); got != dest {
		t.Fatalf("proxy asked to connect to %q, want %q", got, dest)
	}
}

// --- test helpers ---

func roundtrip(t *testing.T, conn net.Conn) string {
	t.Helper()
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal("write failed:", err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatal("read failed:", err)
	}
	return string(buf)
}

func nodeWithTCPEndpoint(addr string) *enode.Node {
	host, port, _ := net.SplitHostPort(addr)
	p := 0
	fmt.Sscanf(port, "%d", &p)
	return enode.NewV4(&newkey().PublicKey, net.ParseIP(host), p, 0)
}

// newEchoServer starts a TCP server which echoes everything back, and returns its
// host:port address.
func newEchoServer(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("can't listen:", err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				io.Copy(conn, conn)
			}()
		}
	}()
	return l.Addr().String()
}

type proxyCredentials struct{ user, password string }

// socksProxyServer is a minimal SOCKS5 server supporting the CONNECT command, used
// to check that connections really are tunneled rather than made directly.
type socksProxyServer struct {
	addr string

	mu     sync.Mutex
	target string
}

func newSocksProxyServer(t *testing.T, creds *proxyCredentials) *socksProxyServer {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("can't listen:", err)
	}
	t.Cleanup(func() { l.Close() })

	ps := &socksProxyServer{addr: l.Addr().String()}
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				conn.SetDeadline(time.Now().Add(10 * time.Second))
				if err := ps.serve(conn, creds); err != nil {
					t.Logf("proxy connection failed: %v", err)
				}
			}()
		}
	}()
	return ps
}

func (ps *socksProxyServer) lastTarget() string {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.target
}

// serve implements the server half of RFC 1928 (and RFC 1929 for authentication),
// restricted to the CONNECT command.
func (ps *socksProxyServer) serve(conn net.Conn, creds *proxyCredentials) error {
	// Method negotiation.
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	if header[0] != 5 {
		return fmt.Errorf("bad SOCKS version %d", header[0])
	}
	methods := make([]byte, header[1])
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}
	wantMethod := byte(0) // no authentication
	if creds != nil {
		wantMethod = 2 // username/password
	}
	if !contains(methods, wantMethod) {
		conn.Write([]byte{5, 0xff})
		return errors.New("no acceptable authentication method offered")
	}
	if _, err := conn.Write([]byte{5, wantMethod}); err != nil {
		return err
	}
	if creds != nil {
		if err := ps.authenticate(conn, creds); err != nil {
			return err
		}
	}

	// CONNECT request.
	request := make([]byte, 4)
	if _, err := io.ReadFull(conn, request); err != nil {
		return err
	}
	if request[1] != 1 {
		return fmt.Errorf("unsupported command %d", request[1])
	}
	var host string
	switch request[3] {
	case 1: // IPv4
		buf := make([]byte, 4)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return err
		}
		host = net.IP(buf).String()
	case 3: // domain name
		length := make([]byte, 1)
		if _, err := io.ReadFull(conn, length); err != nil {
			return err
		}
		buf := make([]byte, length[0])
		if _, err := io.ReadFull(conn, buf); err != nil {
			return err
		}
		host = string(buf)
	case 4: // IPv6
		buf := make([]byte, 16)
		if _, err := io.ReadFull(conn, buf); err != nil {
			return err
		}
		host = net.IP(buf).String()
	default:
		return fmt.Errorf("unsupported address type %d", request[3])
	}
	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return err
	}
	target := net.JoinHostPort(host, fmt.Sprint(binary.BigEndian.Uint16(portBuf)))

	ps.mu.Lock()
	ps.target = target
	ps.mu.Unlock()

	// Connect onwards and relay.
	remote, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		conn.Write([]byte{5, 1, 0, 1, 0, 0, 0, 0, 0, 0}) // general failure
		return err
	}
	defer remote.Close()
	if _, err := conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0}); err != nil {
		return err
	}
	conn.SetDeadline(time.Time{})
	remote.SetDeadline(time.Time{})

	done := make(chan struct{})
	go func() {
		io.Copy(remote, conn)
		remote.Close()
		close(done)
	}()
	io.Copy(conn, remote)
	<-done
	return nil
}

func (ps *socksProxyServer) authenticate(conn net.Conn, creds *proxyCredentials) error {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return err
	}
	if header[0] != 1 {
		return fmt.Errorf("bad auth version %d", header[0])
	}
	user := make([]byte, header[1])
	if _, err := io.ReadFull(conn, user); err != nil {
		return err
	}
	length := make([]byte, 1)
	if _, err := io.ReadFull(conn, length); err != nil {
		return err
	}
	password := make([]byte, length[0])
	if _, err := io.ReadFull(conn, password); err != nil {
		return err
	}
	if string(user) != creds.user || string(password) != creds.password {
		conn.Write([]byte{1, 1})
		return errors.New("bad credentials")
	}
	_, err := conn.Write([]byte{1, 0})
	return err
}

func contains(haystack []byte, needle byte) bool {
	for _, b := range haystack {
		if b == needle {
			return true
		}
	}
	return false
}
