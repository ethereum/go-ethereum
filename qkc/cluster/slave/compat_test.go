// Copyright 2026-2027, QuarkChain.

package slave

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

// startPythonPeer starts a Python protocol peer subprocess and returns the
// TCP port and a cleanup function. The peer listens on a random port (port=0)
// and prints "PORT:<port>" to stdout when ready.
func startPythonPeer(t *testing.T, extraArgs ...string) (int, func()) {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot get caller path")
	}
	pyScript := filepath.Join(filepath.Dir(filename), "testdata", "pyproto", "peer.py")

	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not found in PATH")
	}
	if _, err := os.Stat(pyScript); err != nil {
		t.Skipf("peer.py not found at %s", pyScript)
	}

	args := []string{pyScript, "--port", "0", "--id", "py", "--shards", "1"}
	args = append(args, extraArgs...)

	cmd := exec.Command("python3", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout pipe: %v", err)
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		t.Fatalf("start python peer: %v", err)
	}

	// Read PORT:<port> line from stdout.
	portCh := make(chan int, 1)
	errCh := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "PORT:") {
				var port int
				if _, err := fmt.Sscanf(line, "PORT:%d", &port); err == nil {
					portCh <- port
					return
				}
			}
		}
		errCh <- scanner.Err()
	}()

	var port int
	select {
	case port = <-portCh:
	case err := <-errCh:
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatalf("read port from python peer: %v", err)
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatal("timeout waiting for python peer port")
	}

	cleanup := func() {
		cmd.Process.Kill()
		cmd.Wait()
	}

	return port, cleanup
}

// dialPythonPeer starts a Python peer, dials its TCP port, wraps the
// connection in an XshardConn, and starts it. Returns the XshardConn and a
// cleanup function.
func dialPythonPeer(t *testing.T, extraArgs ...string) (*XshardConn, func()) {
	t.Helper()

	port, cleanupPy := startPythonPeer(t, extraArgs...)

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		cleanupPy()
		t.Fatalf("dial python peer: %v", err)
	}

	xc := NewXshardConnFromConn(conn, 0, []byte("go"), []uint32{1}, log.New())
	xc.Start()

	cleanup := func() {
		xc.Close()
		conn.Close()
		cleanupPy()
	}

	return xc, cleanup
}

// ---------------------------------------------------------------------------
// Test: Python → Go PING/PONG
//
// Validates: Python SlaveConnection.send_ping() initiator behavior.
// Python sends PING, Go XshardConn.handlePing() records identity and replies
// PONG. Tests that Go correctly receives and responds to a Python-initiated
// PING/PONG exchange.
// ---------------------------------------------------------------------------
func TestPythonCompat_PingPong_PythonToGo(t *testing.T) {
	xc, cleanup := dialPythonPeer(t, "--send-ping")
	defer cleanup()

	// Wait for Go side to receive PING from Python.
	// Python peer sends PING immediately after accept.
	if !xc.WaitUntilPingReceived() {
		t.Fatal("Go did not receive PING from Python peer")
	}

	// Verify Go recorded Python's identity from the PING.
	if got := string(xc.RemoteID()); got != "py" {
		t.Fatalf("RemoteID: got %q, want %q", got, "py")
	}
	shards := xc.RemoteFullShardIDList()
	if len(shards) != 1 || shards[0] != 1 {
		t.Fatalf("RemoteFullShardIDList: got %v, want [1]", shards)
	}
}

// ---------------------------------------------------------------------------
// Test: Go → Python PING/PONG
//
// Validates: Go XshardConn.SendPing() outbound PING/PONG exchange.
// Go sends PING, Python SlaveConnection.handle_ping() records identity and
// replies PONG. Tests that Go's SendPing() correctly parses Python's PONG
// response.
// ---------------------------------------------------------------------------
func TestPythonCompat_PingPong_GoToPython(t *testing.T) {
	xc, cleanup := dialPythonPeer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	id, shardList, err := xc.SendPing(ctx)
	if err != nil {
		t.Fatalf("SendPing: %v", err)
	}

	if string(id) != "py" {
		t.Fatalf("SendPing returned id %q, want %q", string(id), "py")
	}
	if len(shardList) != 1 || shardList[0] != 1 {
		t.Fatalf("SendPing returned shardList %v, want [1]", shardList)
	}
}

// ---------------------------------------------------------------------------
// Test: RPC request/response matching
//
// Validates: Python's echo-RPC behavior (opcode → opcode+1, same rpc_id,
// same payload). Verifies that Go's RPC ID generation, pending map lifecycle,
// and response matching work correctly when communicating with a Python peer.
// ---------------------------------------------------------------------------
func TestPythonCompat_RPCRequestResponse(t *testing.T) {
	xc, cleanup := dialPythonPeer(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send a request with opcode=0x10. Python echoes back opcode=0x11.
	payload := []byte("hello-rpc")
	resp, err := xc.SendRPC(ctx, 0x10, payload)
	if err != nil {
		t.Fatalf("SendRPC: %v", err)
	}

	if resp.Opcode != 0x11 {
		t.Fatalf("response opcode: got 0x%02x, want 0x11", resp.Opcode)
	}
	if string(resp.Payload) != string(payload) {
		t.Fatalf("response payload: got %q, want %q", string(resp.Payload), string(payload))
	}

	// Send a second RPC with a different payload to verify sequential RPCs.
	payload2 := []byte("second-rpc")
	resp2, err := xc.SendRPC(ctx, 0x10, payload2)
	if err != nil {
		t.Fatalf("second SendRPC: %v", err)
	}

	if resp2.Opcode != 0x11 {
		t.Fatalf("second response opcode: got 0x%02x, want 0x11", resp2.Opcode)
	}
	if string(resp2.Payload) != string(payload2) {
		t.Fatalf("second response payload: got %q, want %q", string(resp2.Payload), string(payload2))
	}

	// Verify RPC IDs are unique (each response matches its own request).
	if resp.RPCID == resp2.RPCID {
		t.Fatal("RPC IDs should be unique")
	}
}

// ---------------------------------------------------------------------------
// Test: Connection close propagation
//
// Validates: Python's SlaveConnection.close() behavior.
// When the Python peer disconnects, Go's readLoop must detect the TCP close
// and call Close(). After close, any RPC must fail with ErrConnectionClosed.
//
// Note: Testing mid-flight RPC wakeup is non-deterministic because Python
// echoes the response before the process is killed. This test verifies the
// deterministic post-close behavior instead.
// ---------------------------------------------------------------------------
func TestPythonCompat_ConnectionClosePropagation(t *testing.T) {
	port, cleanupPy := startPythonPeer(t)

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		cleanupPy()
		t.Fatalf("dial: %v", err)
	}

	xc := NewXshardConnFromConn(conn, 0, []byte("go"), []uint32{1}, log.New())
	xc.Start()
	defer xc.Close()

	// Kill the Python peer — this closes the TCP connection from the other end.
	cleanupPy()

	// Wait for Go to detect the connection close.
	select {
	case <-xc.WaitUntilClosed():
	case <-time.After(5 * time.Second):
		t.Fatal("Go did not detect connection close within 5 seconds")
	}

	if !xc.IsClosed() {
		t.Fatal("XshardConn should be closed after Python disconnect")
	}

	// Any RPC after close should fail with ErrConnectionClosed.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err = xc.SendRPC(ctx, 0x01, []byte("test"))
	if err != ErrConnectionClosed {
		t.Fatalf("expected ErrConnectionClosed after close, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test: Pool reconnect after Remove
//
// Validates: Python's SlaveConnectionManager.connect_to_slave() reconnection
// behavior. After a connection is removed from the pool and the slave ID is
// cleaned up, a new connection to a peer with the same identity must be
// accepted. Tests the XshardPool.Remove() → slaveIDs cleanup → reconnection
// invariant.
// ---------------------------------------------------------------------------
func TestPythonCompat_PoolReconnect(t *testing.T) {
	pool := NewXshardPool(log.New())
	defer pool.Close()

	// --- First connection ---
	xc1, cleanup1 := dialPythonPeer(t)
	defer cleanup1()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.VerifyAndAdd(ctx, 1, xc1, []byte("py"), []uint32{1}); err != nil {
		t.Fatalf("first VerifyAndAdd: %v", err)
	}
	if pool.OutboundSize() != 1 {
		t.Fatalf("pool size after add: got %d, want 1", pool.OutboundSize())
	}

	// Remove and verify the pool is empty.
	pool.Remove(1, xc1)
	if pool.OutboundSize() != 0 {
		t.Fatalf("pool size after remove: got %d, want 0", pool.OutboundSize())
	}

	// Clean up the first peer before starting the second.
	cleanup1()

	// --- Second connection (same identity, should be accepted) ---
	xc2, cleanup2 := dialPythonPeer(t)
	defer cleanup2()

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	if err := pool.VerifyAndAdd(ctx2, 1, xc2, []byte("py"), []uint32{1}); err != nil {
		t.Fatalf("second VerifyAndAdd (reconnect) failed: %v", err)
	}
	if pool.OutboundSize() != 1 {
		t.Fatalf("pool size after reconnect: got %d, want 1", pool.OutboundSize())
	}
}
