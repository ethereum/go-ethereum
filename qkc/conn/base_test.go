// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package conn

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"net"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/wire"
)

// -- fake transport -----------------------------------------------------------

type rpcResult struct {
	resp wire.SSZMessage
	err  error
}

type fakeFrameTransport struct {
	frames            chan *wire.ClusterFrame
	writes            chan *wire.ClusterFrame
	closed            chan struct{}
	closeOnce         sync.Once
	closeMu           sync.Mutex
	closeCount        int
	writeMu           sync.Mutex
	writing           bool
	closeWhileWriting bool
	writeStarted      chan struct{}
	writeOnce         sync.Once
	releaseWrite      chan struct{}
	writeErr          error
	writePanic        bool
}

// interruptibleFakeFrameTransport models a real net.Conn: Close releases a
// write parked inside WriteFrame (shutdown is interrupt-first).
type interruptibleFakeFrameTransport struct {
	*fakeFrameTransport
	interruptOnce sync.Once
}

func (t *interruptibleFakeFrameTransport) Close() error {
	err := t.fakeFrameTransport.Close()
	t.interruptOnce.Do(func() {
		if t.releaseWrite != nil {
			close(t.releaseWrite)
		}
	})
	return err
}

func newFakeFrameTransport() *fakeFrameTransport {
	return &fakeFrameTransport{
		frames: make(chan *wire.ClusterFrame, 8),
		writes: make(chan *wire.ClusterFrame, 8),
		closed: make(chan struct{}),
	}
}

func (t *fakeFrameTransport) ReadFrame() (*wire.ClusterFrame, error) {
	select {
	case frame := <-t.frames:
		return frame, nil
	case <-t.closed:
		return nil, io.EOF
	}
}

func (t *fakeFrameTransport) WriteFrame(frame *wire.ClusterFrame) error {
	if t.writePanic {
		panic("fake transport write panic")
	}
	if t.writeErr != nil {
		return t.writeErr
	}
	select {
	case t.writes <- frame:
		if t.releaseWrite != nil {
			t.writeMu.Lock()
			t.writing = true
			t.writeMu.Unlock()
			t.writeOnce.Do(func() { close(t.writeStarted) })
			<-t.releaseWrite
			t.writeMu.Lock()
			t.writing = false
			t.writeMu.Unlock()
		}
		return nil
	case <-t.closed:
		return errors.New("fake transport closed")
	}
}

func (t *fakeFrameTransport) Close() error {
	t.closeOnce.Do(func() {
		t.writeMu.Lock()
		t.closeWhileWriting = t.writing
		t.writeMu.Unlock()
		t.closeMu.Lock()
		t.closeCount++
		t.closeMu.Unlock()
		close(t.closed)
	})
	return nil
}

func (t *fakeFrameTransport) RemoteAddr() string {
	return "fake"
}

func (t *fakeFrameTransport) closes() int {
	t.closeMu.Lock()
	defer t.closeMu.Unlock()
	return t.closeCount
}

// staticReaderTransport feeds wire.ReadClusterFrame from a fixed byte stream (clean
// EOF vs truncated frame semantics).
type staticReaderTransport struct {
	reader    io.Reader
	closed    chan struct{}
	closeOnce sync.Once
}

type eofSignalReader struct {
	reader io.Reader
	done   chan struct{}
}

func (r *eofSignalReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	if err == io.EOF {
		close(r.done)
	}
	return n, err
}

func newStaticReaderTransport(r io.Reader) *staticReaderTransport {
	return &staticReaderTransport{reader: r, closed: make(chan struct{})}
}

func (t *staticReaderTransport) ReadFrame() (*wire.ClusterFrame, error) {
	return wire.ReadClusterFrame(t.reader, wire.MaxFramePayloadLen)
}

func (t *staticReaderTransport) WriteFrame(*wire.ClusterFrame) error { return nil }

func (t *staticReaderTransport) Close() error {
	t.closeOnce.Do(func() { close(t.closed) })
	return nil
}

func (t *staticReaderTransport) RemoteAddr() string { return "static" }

// -- test helpers ------------------------------------------------------------

// NonRPCSerializerFor creates a serializer for a message without a response.
// Its opcode must be included in Config.NonRPCOps.
func NonRPCSerializerFor[R any, RP sszPointer[R]]() *OpSerializer {
	return OpSerializerFor[R, R, RP, RP](0)
}

var pingSer = OpSerializerFor[wire.PingRequest, wire.PongResponse](byte(wire.ClusterOpPong))

var pingSerializers = map[byte]*OpSerializer{
	byte(wire.ClusterOpPing): pingSer,
}

func pongHandler() TypedHandler {
	return func(req wire.SSZMessage) (wire.SSZMessage, error) {
		return &wire.PongResponse{}, nil
	}
}

func validPongPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := (&wire.PongResponse{}).MarshalSSZ()
	if err != nil {
		t.Fatalf("serialize pong: %v", err)
	}
	return payload
}

func validPingPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := (&wire.PingRequest{ChainID: [wire.UInt256Length]byte{1}}).MarshalSSZ()
	if err != nil {
		t.Fatalf("serialize ping: %v", err)
	}
	return payload
}

func validHelloPayload(t *testing.T) []byte {
	t.Helper()
	payload, err := (&wire.HelloCommand{}).MarshalSSZ()
	if err != nil {
		t.Fatalf("serialize hello: %v", err)
	}
	return payload
}

func newConn(tr FrameTransport) *BaseConn {
	return NewBaseConn(Config{Transport: tr, Serializers: pingSerializers, Logger: log.New()})
}

func newConnWithHello(tr FrameTransport) *BaseConn {
	return NewBaseConn(Config{
		Transport: tr,
		Serializers: map[byte]*OpSerializer{
			byte(wire.ClusterOpPing):  pingSer,
			byte(wire.CommandOpHello): NonRPCSerializerFor[wire.HelloCommand](),
		},
		NonRPCOps: map[byte]struct{}{byte(wire.CommandOpHello): {}},
		Logger:    log.New(),
	})
}

func newPingConn(tr FrameTransport) *BaseConn {
	return NewBaseConn(Config{
		Transport:   tr,
		Serializers: pingSerializers,
		Logger:      log.New(),
	})
}

func newPingServerConn(tr FrameTransport) *BaseConn {
	return NewBaseConn(Config{
		Transport:   tr,
		Serializers: pingSerializers,
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): pongHandler(),
		},
		Logger: log.New(),
	})
}

// waitForGoroutines polls until the goroutine count drops back to the baseline.
func waitForGoroutines(t *testing.T, baseline int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= baseline {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("goroutine leak: %d goroutines, baseline %d", runtime.NumGoroutine(), baseline)
}

// receive waits for one value from ch and fails the test instead of hanging
// when it never arrives.
func receive[T any](t *testing.T, ch <-chan T, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(3 * time.Second):
		t.Fatalf("%s never arrived", what)
		var zero T
		return zero
	}
}

// feedFrame delivers a frame into the fake transport's inbound queue.
func feedFrame(t *testing.T, tr *fakeFrameTransport, f *wire.ClusterFrame) {
	t.Helper()
	select {
	case tr.frames <- f:
	case <-time.After(time.Second):
		t.Fatal("feeding frame blocked: transport queue full")
	}
}

// awaitClosed fails the test if the connection is not closed shortly.
func awaitClosed(t *testing.T, conn *BaseConn, what string) {
	t.Helper()
	select {
	case <-conn.WaitUntilClosed():
	case <-time.After(2 * time.Second):
		t.Fatalf("connection did not close after %s", what)
	}
	if !conn.IsClosed() {
		t.Fatalf("connection state is not Closed after %s", what)
	}
}

// -- Config validation tests --------------------------------------------------

// serializerMissingCallback returns a full ping serializer with one callback
// removed, for config validation tests.
func serializerMissingCallback(missing string) *OpSerializer {
	ser := OpSerializerFor[wire.PingRequest, wire.PongResponse](byte(wire.ClusterOpPong))
	switch missing {
	case "NewRequest":
		ser.NewRequest = nil
	case "NewResponse":
		ser.NewResponse = nil
	default:
		panic("unknown callback: " + missing)
	}
	return ser
}

// Keep the canonical serializer pairs consistent with the wire payload registry.
// The registry itself does not carry request/response pairing information.
func TestOpSerializerFor_WireRegistry(t *testing.T) {
	serializers := map[wire.ClusterOp]*OpSerializer{
		wire.ClusterOpPing:                         pingSer,
		wire.ClusterOpConnectToSlavesRequest:       OpSerializerFor[wire.ConnectToSlavesRequest, wire.ConnectToSlavesResponse](byte(wire.ClusterOpConnectToSlavesResponse)),
		wire.ClusterOpAddRootBlockRequest:          OpSerializerFor[wire.AddRootBlockRequest, wire.AddRootBlockResponse](byte(wire.ClusterOpAddRootBlockResponse)),
		wire.ClusterOpAddShardBlockHeaderRequest:   OpSerializerFor[wire.AddShardBlockHeaderRequest, wire.AddShardBlockHeaderResponse](byte(wire.ClusterOpAddShardBlockHeaderResponse)),
		wire.ClusterOpAddXshardTxListRequest:       OpSerializerFor[wire.AddXshardTxListRequest, wire.AddXshardTxListResponse](byte(wire.ClusterOpAddXshardTxListResponse)),
		wire.ClusterOpGetRootChainStakesRequest:    OpSerializerFor[wire.GetRootChainStakesRequest, wire.GetRootChainStakesResponse](byte(wire.ClusterOpGetRootChainStakesResponse)),
		wire.ClusterOpGetUnconfirmedHeadersRequest: OpSerializerFor[wire.GetUnconfirmedHeadersRequest, wire.GetUnconfirmedHeadersResponse](byte(wire.ClusterOpGetUnconfirmedHeadersResponse)),
	}
	covered := make(map[wire.ClusterOp]bool)
	for requestOp, ser := range serializers {
		for _, tc := range []struct {
			op      wire.ClusterOp
			message any
		}{
			{requestOp, ser.NewRequest()},
			{wire.ClusterOp(ser.ResponseOpCode), ser.NewResponse()},
		} {
			registered, err := wire.NewClusterMessage(tc.op)
			if err != nil {
				t.Fatal(err)
			}
			if reflect.TypeOf(tc.message) != reflect.TypeOf(registered) {
				t.Errorf("opcode 0x%x: serializer creates %T, wire registry creates %T", tc.op, tc.message, registered)
			}
			covered[tc.op] = true
		}
	}
	for op := int(wire.ClusterOpBase); op <= math.MaxUint8; op++ {
		if _, err := wire.NewClusterMessage(wire.ClusterOp(op)); err == nil && !covered[wire.ClusterOp(op)] {
			t.Errorf("opcode 0x%x has no serializer coverage", op)
		}
	}
	hello, err := wire.NewCommandMessage(wire.CommandOpHello)
	if err != nil {
		t.Fatal(err)
	}
	if request := NonRPCSerializerFor[wire.HelloCommand]().NewRequest(); reflect.TypeOf(request) != reflect.TypeOf(hello) {
		t.Errorf("Hello serializer creates %T, wire registry creates %T", request, hello)
	}
}

// TestConfig_ValidationPanics covers config validation panics.
func TestConfig_ValidationPanics(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"nil transport", Config{}, "Transport"},
		{"nil serializer", Config{Transport: newFakeFrameTransport(), Serializers: map[byte]*OpSerializer{0x01: nil}}, "serializer"},
		{"nil handler", Config{Transport: newFakeFrameTransport(), Handlers: map[byte]TypedHandler{0x01: nil}}, "handler"},
		{"handler without serializer", Config{Transport: newFakeFrameTransport(), Handlers: map[byte]TypedHandler{0x01: pongHandler()}}, "serializer for handler opcode"},
		{"non-RPC opcode without serializer", Config{Transport: newFakeFrameTransport(), NonRPCOps: map[byte]struct{}{0x01: {}}}, "serializer for non-RPC opcode"},
		{"response opcode marked as non-RPC", Config{Transport: newFakeFrameTransport(), Serializers: pingSerializers, NonRPCOps: map[byte]struct{}{byte(wire.ClusterOpPong): {}}}, "serializer for non-RPC opcode"},
		{"missing NewRequest callback", Config{Transport: newFakeFrameTransport(), Serializers: map[byte]*OpSerializer{0x01: serializerMissingCallback("NewRequest")}}, "missing callback"},
		{"missing NewResponse callback", Config{Transport: newFakeFrameTransport(), Serializers: map[byte]*OpSerializer{0x01: serializerMissingCallback("NewResponse")}}, "missing callback"},
		{"non-RPC missing NewRequest callback", Config{Transport: newFakeFrameTransport(), Serializers: map[byte]*OpSerializer{0x01: serializerMissingCallback("NewRequest")}, NonRPCOps: map[byte]struct{}{0x01: {}}}, "missing callback"},
		{"non-RPC missing NewResponse callback", Config{Transport: newFakeFrameTransport(), Serializers: map[byte]*OpSerializer{0x01: serializerMissingCallback("NewResponse")}, NonRPCOps: map[byte]struct{}{0x01: {}}}, "missing callback"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Fatal("expected panic for " + tt.name)
				}
				msg, _ := r.(string)
				if !strings.Contains(msg, tt.want) {
					t.Fatalf("panic %q does not contain %q", msg, tt.want)
				}
			}()
			NewBaseConn(tt.cfg)
		})
	}
}

// TestConfig_ResponseOpcodePanics: response opcodes must be unique and disjoint
// from every request opcode (RPC or non-RPC), so classification is unambiguous
// and response serializers cannot depend on map-iteration order.
func TestConfig_ResponseOpcodePanics(t *testing.T) {
	tests := []struct {
		name        string
		serializers map[byte]*OpSerializer
		nonRPC      map[byte]struct{}
	}{
		{
			name: "response opcode conflicts with rpc request opcode",
			serializers: map[byte]*OpSerializer{
				0x01: OpSerializerFor[wire.PingRequest, wire.PongResponse](0x02),
				0x02: OpSerializerFor[wire.PingRequest, wire.PongResponse](0x03),
			},
		},
		{
			name: "duplicate response opcode",
			serializers: map[byte]*OpSerializer{
				0x81: OpSerializerFor[wire.PingRequest, wire.PongResponse](0x90),
				0x82: OpSerializerFor[wire.PingRequest, wire.PongResponse](0x90),
			},
		},
		{
			name: "response opcode conflicts with non-rpc request opcode",
			serializers: map[byte]*OpSerializer{
				0x81: OpSerializerFor[wire.PingRequest, wire.PongResponse](0x90),
				0x90: OpSerializerFor[wire.PingRequest, wire.PingRequest](0x90),
			},
			nonRPC: map[byte]struct{}{0x90: {}},
		},
		{
			name:        "self-referencing rpc response opcode",
			serializers: map[byte]*OpSerializer{0x81: OpSerializerFor[wire.PingRequest, wire.PongResponse](0x81)},
		},
		{
			name:        "zero response opcode for rpc",
			serializers: map[byte]*OpSerializer{0x81: OpSerializerFor[wire.PingRequest, wire.PongResponse](0)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("expected panic for " + tt.name)
				}
			}()
			NewBaseConn(Config{
				Transport:   newFakeFrameTransport(),
				Serializers: tt.serializers,
				NonRPCOps:   tt.nonRPC,
				Logger:      log.New(),
			})
		})
	}
}

// -- BaseConn unit tests (fake transport) --------------------------------------

// TestBaseConn_CloseWithInFlightWrite verifies that shutdown does not wait for
// an in-flight WriteFrame. Shutdown completes pending RPCs and closes the
// transport (interrupt-first) while a blocked write may still be in progress.
func TestBaseConn_CloseWithInFlightWrite(t *testing.T) {
	t.Run("close does not wait for in-flight write", func(t *testing.T) {
		tr := newFakeFrameTransport()
		tr.writeStarted = make(chan struct{})
		tr.releaseWrite = make(chan struct{})
		conn := newConn(tr)
		conn.Start()

		result := make(chan error, 1)
		go func() {
			_, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
			result <- err
		}()

		select {
		case <-tr.writeStarted:
		case <-time.After(time.Second):
			t.Fatal("fake transport did not start writing")
		}

		closeDone := make(chan struct{})
		go func() {
			conn.Close()
			close(closeDone)
		}()
		select {
		case <-closeDone:
		case <-time.After(time.Second):
			t.Fatal("Close blocked waiting for in-flight write")
		}
		if !tr.closeWhileWriting {
			t.Fatal("expected interrupt-first shutdown: transport.Close must run while the write is still in flight")
		}

		close(tr.releaseWrite)
		select {
		case err := <-result:
			if err != ErrConnectionClosed {
				t.Fatalf("expected ErrConnectionClosed, got %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("RPC did not complete after write released")
		}
	})

	t.Run("interrupt: Close releases blocked writer", func(t *testing.T) {
		base := newFakeFrameTransport()
		base.writeStarted = make(chan struct{})
		base.releaseWrite = make(chan struct{})
		tr := &interruptibleFakeFrameTransport{fakeFrameTransport: base}
		conn := newConn(tr)
		conn.Start()

		result := make(chan error, 1)
		go func() {
			_, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
			result <- err
		}()
		select {
		case <-tr.writeStarted:
		case <-time.After(time.Second):
			t.Fatal("fake transport did not start writing")
		}

		closeDone := make(chan struct{})
		go func() {
			conn.Close()
			close(closeDone)
		}()
		select {
		case <-closeDone:
		case <-time.After(time.Second):
			t.Fatal("Close did not interrupt blocked writer")
		}
		if err := <-result; err != ErrConnectionClosed {
			t.Fatalf("expected ErrConnectionClosed, got %v", err)
		}
	})
}

func TestBaseConn_CleanShutdown(t *testing.T) {
	t.Run("local Close", func(t *testing.T) {
		conn := newConn(newFakeFrameTransport())
		conn.Start()
		conn.Close()
	})

	t.Run("clean peer EOF", func(t *testing.T) {
		conn := newConn(newStaticReaderTransport(bytes.NewReader(nil)))
		conn.Start()

		<-conn.WaitUntilClosed()
		if !conn.IsClosed() {
			t.Fatal("connection should be closed after clean EOF")
		}
	})
}

func TestBaseConn_ClosedConnectionDropsBufferedFrame(t *testing.T) {
	var input bytes.Buffer
	if err := wire.WriteClusterFrame(&input, &wire.ClusterFrame{Opcode: 0xff, RPCID: 1}); err != nil {
		t.Fatalf("encode invalid frame: %v", err)
	}
	if err := wire.WriteClusterFrame(&input, &wire.ClusterFrame{
		Opcode: byte(wire.ClusterOpPing), RPCID: 2, Payload: validPingPayload(t),
	}); err != nil {
		t.Fatalf("encode buffered request: %v", err)
	}

	reader := &eofSignalReader{reader: bytes.NewReader(input.Bytes()), done: make(chan struct{})}
	forwarded := make(chan struct{}, 1)
	handled := make(chan struct{}, 1)
	conn := NewBaseConn(Config{
		Transport:   newStaticReaderTransport(reader),
		Serializers: pingSerializers,
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): func(wire.SSZMessage) (wire.SSZMessage, error) {
				handled <- struct{}{}
				return &wire.PongResponse{}, nil
			},
		},
		Forwarder: func(f *wire.ClusterFrame) bool {
			if f.Opcode == byte(wire.ClusterOpPing) {
				forwarded <- struct{}{}
			}
			return false
		},
	})
	conn.Start()
	awaitClosed(t, conn, "invalid frame")
	select {
	case <-reader.done:
	case <-time.After(time.Second):
		t.Fatal("reader did not consume buffered frames")
	}
	select {
	case <-forwarded:
		t.Fatal("forwarder received a frame after connection close")
	default:
	}
	select {
	case <-handled:
		t.Fatal("handler ran after connection close")
	default:
	}
}

// TestBaseConn_CanceledRPCNotSent: an RPC whose context expires while waiting
// for writeMu must not allocate an rpc id, register pending, or write a frame
// once it acquires the lock — the peer must never execute an abandoned call.
func TestBaseConn_CanceledRPCNotSent(t *testing.T) {
	tr := newFakeFrameTransport()
	tr.writeStarted = make(chan struct{})
	tr.releaseWrite = make(chan struct{})
	conn := newPingConn(tr)
	conn.Start()

	// First RPC parks inside WriteFrame holding writeMu.
	first := make(chan error, 1)
	go func() {
		_, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
		first <- err
	}()
	select {
	case <-tr.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("first write did not block")
	}
	select {
	case <-tr.writes: // rpc_id=1 frame recorded; writer still parked
	case <-time.After(time.Second):
		t.Fatal("first write was not recorded")
	}

	// Second RPC's context expires while it waits for writeMu.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	second := make(chan error, 1)
	go func() {
		_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
		second <- err
	}()

	time.Sleep(30 * time.Millisecond)
	close(tr.releaseWrite) // unblock the writer; second RPC acquires writeMu

	select {
	case err := <-second:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected deadline exceeded, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled RPC did not return after writer unblocked")
	}

	// No frame was written for the canceled RPC.
	select {
	case f := <-tr.writes:
		t.Fatalf("canceled RPC wrote a frame (rpc_id=%d)", f.RPCID)
	default:
	}
	// Only the first RPC remains pending; no id was consumed.
	if n := conn.pendingLen(); n != 1 {
		t.Fatalf("pending = %d, want 1", n)
	}

	// The next RPC gets rpc_id 2, proving the canceled call allocated nothing.
	third := make(chan rpcResult, 1)
	go func() {
		resp, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
		third <- rpcResult{resp: resp, err: err}
	}()
	select {
	case f := <-tr.writes:
		if f.RPCID != 2 {
			t.Fatalf("next rpc id = %d, want 2", f.RPCID)
		}
	case <-time.After(time.Second):
		t.Fatal("next RPC did not write")
	}

	// Complete both RPCs and close.
	tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: 1, Payload: validPongPayload(t)}
	tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: 2, Payload: validPongPayload(t)}
	if err := <-first; err != nil {
		t.Fatalf("first RPC failed: %v", err)
	}
	if res := <-third; res.err != nil {
		t.Fatalf("third RPC failed: %v", res.err)
	}
	conn.Close()
}

// TestBaseConn_CancelErrorContract: SendRPC with an expired or canceled
// context fails immediately without writing.
func TestBaseConn_CancelErrorContract(t *testing.T) {
	tests := []struct {
		name      string
		newCtx    func() (context.Context, context.CancelFunc)
		wantError func(error) bool
	}{
		{
			name: "deadline exceeded",
			newCtx: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			},
			wantError: func(err error) bool { return errors.Is(err, context.DeadlineExceeded) },
		},
		{
			name: "already cancelled",
			newCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			wantError: func(err error) bool { return errors.Is(err, context.Canceled) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newFakeFrameTransport()
			conn := newConn(tr)
			conn.Start()
			defer conn.Close()

			ctx, cancel := tt.newCtx()
			defer cancel()
			_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
			if !tt.wantError(err) {
				t.Fatalf("SendRPC error = %v", err)
			}
			select {
			case <-tr.writes:
				t.Fatal("canceled RPC was sent")
			default:
			}
		})
	}
}

// TestBaseConn_TimeoutThenLateResponse: a caller-side timeout completes the
// RPC, and a late response does not affect subsequent RPCs or close the connection.
func TestBaseConn_TimeoutThenLateResponse(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := newPingConn(tr)
	conn.Start()
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
		result <- err
	}()
	request := receive(t, tr.writes, "ping request")

	err := receive(t, result, "timeout result")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	if conn.IsClosed() {
		t.Fatal("caller-side timeout must not shut the connection down")
	}
	if n := conn.pendingLen(); n != 0 {
		t.Fatalf("pending = %d after timeout, want 0", n)
	}
	// Late responses and duplicates for the timed-out id are both ignored.
	for i := 0; i < 2; i++ {
		feedFrame(t, tr, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: request.RPCID, Payload: validPongPayload(t)})
	}

	// Barrier: frames are consumed FIFO by the single readerLoop, so completing
	// this fresh RPC proves the late PONG was fully processed beforehand.
	next := make(chan rpcResult, 1)
	go func() {
		resp, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
		next <- rpcResult{resp: resp, err: err}
	}()
	request2 := receive(t, tr.writes, "second ping request")
	feedFrame(t, tr, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: request2.RPCID, Payload: validPongPayload(t)})

	res := receive(t, next, "second rpc result")
	if res.err != nil {
		t.Fatalf("rpc after late response failed: %v", res.err)
	}
	if conn.IsClosed() {
		t.Fatal("late response must not shut the connection down")
	}
}

func TestBaseConn_CanceledRPCsCleanPending(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := newPingConn(tr)
	conn.Start()
	defer conn.Close()

	for i := 0; i < 32; i++ {
		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
			result <- err
		}()
		receive(t, tr.writes, "request without a response")
		cancel()
		if err := receive(t, result, "canceled RPC result"); !errors.Is(err, context.Canceled) {
			t.Fatalf("iteration %d: error = %v, want context canceled", i, err)
		}
		if n := conn.pendingLen(); n != 0 {
			t.Fatalf("iteration %d: pending = %d after cancellation", i, n)
		}
	}
	if !conn.IsActive() {
		t.Fatal("canceling calls closed the connection")
	}
}

func TestBaseConn_DuplicateResponseDoesNotCompleteNextRPC(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := newPingConn(tr)
	conn.Start()
	defer conn.Close()

	var previous *wire.ClusterFrame
	for i := 0; i < 2; i++ {
		result := make(chan rpcResult, 1)
		go func() {
			resp, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
			result <- rpcResult{resp: resp, err: err}
		}()
		request := receive(t, tr.writes, "ping request")
		if previous != nil {
			feedFrame(t, tr, previous)
			feedFrame(t, tr, previous)
		}
		chainID := [wire.UInt256Length]byte{byte(i + 1)}
		payload, err := (&wire.PongResponse{ChainID: chainID}).MarshalSSZ()
		if err != nil {
			t.Fatal(err)
		}
		response := &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: request.RPCID, Payload: payload}
		feedFrame(t, tr, response)
		res := receive(t, result, "RPC response")
		if res.err != nil {
			t.Fatalf("RPC failed after duplicate responses: %v", res.err)
		}
		if resp, ok := res.resp.(*wire.PongResponse); !ok || resp.ChainID != chainID {
			t.Fatalf("received another RPC's response: %+v", res.resp)
		}
		previous = response
	}
	if !conn.IsActive() {
		t.Fatal("duplicate response closed the connection")
	}
}

// Registered request and command opcodes must never complete a pending RPC.
// A response opcode must match the response paired with that RPC's request.
func TestBaseConn_InvalidResponseOpcodeClosesConnection(t *testing.T) {
	rootOp := byte(wire.ClusterOpAddRootBlockRequest)
	rootSer := OpSerializerFor[wire.AddRootBlockRequest, wire.AddRootBlockResponse](byte(wire.ClusterOpAddRootBlockResponse))
	connectOp := byte(wire.ClusterOpConnectToSlavesRequest)
	connectSer := OpSerializerFor[wire.ConnectToSlavesRequest, wire.ConnectToSlavesResponse](byte(wire.ClusterOpConnectToSlavesResponse))
	helloOp := byte(wire.CommandOpHello)
	for _, tc := range []struct {
		name        string
		requestOp   byte
		inboundOp   byte
		payload     []byte
		serializers map[byte]*OpSerializer
		nonRPC      map[byte]struct{}
	}{
		{
			name:        "request opcode instead of root block acknowledgement",
			requestOp:   rootOp,
			inboundOp:   rootOp,
			payload:     mustSerialize(t, &wire.AddRootBlockResponse{}),
			serializers: map[byte]*OpSerializer{rootOp: rootSer},
		},
		{
			name:        "response opcode paired with a different request",
			requestOp:   byte(wire.ClusterOpPing),
			inboundOp:   byte(wire.ClusterOpConnectToSlavesResponse),
			payload:     mustSerialize(t, &wire.ConnectToSlavesResponse{}),
			serializers: map[byte]*OpSerializer{byte(wire.ClusterOpPing): pingSer, connectOp: connectSer},
		},
		{
			name:        "outbound-only request opcode",
			requestOp:   byte(wire.ClusterOpPing),
			inboundOp:   rootOp,
			payload:     mustSerialize(t, &wire.AddRootBlockRequest{}),
			serializers: map[byte]*OpSerializer{byte(wire.ClusterOpPing): pingSer, rootOp: rootSer},
		},
		{
			name:        "non-RPC command without handler and nonzero RPC ID",
			requestOp:   byte(wire.ClusterOpPing),
			inboundOp:   helloOp,
			payload:     validHelloPayload(t),
			serializers: map[byte]*OpSerializer{byte(wire.ClusterOpPing): pingSer, helloOp: NonRPCSerializerFor[wire.HelloCommand]()},
			nonRPC:      map[byte]struct{}{helloOp: {}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := newFakeFrameTransport()
			conn := NewBaseConn(Config{Transport: tr, Serializers: tc.serializers, NonRPCOps: tc.nonRPC})
			conn.Start()
			defer conn.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan rpcResult, 1)
			ser := tc.serializers[tc.requestOp]
			req := ser.NewRequest()
			go func() {
				resp, err := conn.SendRPC(ctx, tc.requestOp, req)
				result <- rpcResult{resp: resp, err: err}
			}()
			request := receive(t, tr.writes, "RPC request")
			feedFrame(t, tr, &wire.ClusterFrame{Opcode: tc.inboundOp, RPCID: request.RPCID, Payload: tc.payload})
			res := receive(t, result, "rejected response result")
			if res.err != ErrConnectionClosed || res.resp != nil {
				t.Fatalf("invalid response completed RPC: response=%T err=%v", res.resp, res.err)
			}
			awaitClosed(t, conn, "invalid response opcode")
			if n := conn.pendingLen(); n != 0 {
				t.Fatalf("pending = %d, want 0", n)
			}
		})
	}
}

func TestBaseConn_WriteFailureClosesConnection(t *testing.T) {
	tr := newFakeFrameTransport()
	writeErr := errors.New("write failed")
	tr.writeErr = writeErr
	conn := newConn(tr)
	conn.Start()

	_, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
	if !errors.Is(err, ErrConnectionClosed) || !errors.Is(err, writeErr) {
		t.Fatalf("expected ErrConnectionClosed, got %v", err)
	}
	<-conn.WaitUntilClosed()
	if !conn.IsClosed() {
		t.Fatal("connection should be closed after write failure")
	}
}

// stubSSZRequest supplies serialized bytes or an error for outbound validation tests.
type stubSSZRequest struct {
	wire.PingRequest
	payload []byte
	err     error
}

func (req *stubSSZRequest) MarshalSSZ() ([]byte, error) {
	return req.payload, req.err
}

func TestBaseConn_SendRPCInvalidRequestKeepsConnection(t *testing.T) {
	marshalErr := errors.New("marshal failed")
	for _, tc := range []struct {
		name    string
		req     wire.SSZMessage
		wantErr error
	}{
		{name: "nil request"},
		{name: "serialization error", req: &stubSSZRequest{err: marshalErr}, wantErr: marshalErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := newFakeFrameTransport()
			conn := newPingConn(tr)
			conn.Start()
			defer conn.Close()
			resp, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), tc.req)
			if err == nil || resp != nil || (tc.wantErr != nil && !errors.Is(err, tc.wantErr)) {
				t.Fatalf("invalid request: response=%v error=%v", resp, err)
			}
			if !conn.IsActive() || conn.pendingLen() != 0 || conn.nextRPCID != 0 {
				t.Fatalf("invalid request changed connection: state=%v pending=%d rpcid=%d", conn.State(), conn.pendingLen(), conn.nextRPCID)
			}
			select {
			case frame := <-tr.writes:
				t.Fatalf("invalid request wrote a frame: %+v", frame)
			default:
			}
		})
	}
}

func TestBaseConn_InvalidRPCFrameKeepsConnection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{name: "oversized payload", payload: make([]byte, wire.MaxFramePayloadLen+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local, peer := net.Pipe()
			defer peer.Close()
			if err := peer.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
				t.Fatal(err)
			}
			conn := newConn(newPipeTransport(t, local))
			conn.Start()
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), &stubSSZRequest{payload: tc.payload})
			if !errors.Is(err, wire.ErrInvalidClusterFrame) {
				t.Fatalf("invalid RPC frame error = %v, want validation error", err)
			}
			if !conn.IsActive() || conn.pendingLen() != 0 {
				t.Fatalf("after invalid RPC frame: state=%v pending=%d", conn.State(), conn.pendingLen())
			}
			read := make(chan error, 1)
			go func() { _, err := wire.ReadClusterFrame(peer, wire.MaxFramePayloadLen); read <- err }()
			if err := conn.WriteFrame(&wire.ClusterFrame{}); err != nil {
				t.Fatalf("write after validation failure: %v", err)
			}
			if err := receive(t, read, "frame after validation failure"); err != nil {
				t.Fatalf("read after validation failure: %v", err)
			}
		})
	}
}

func TestBaseConn_InvalidFrameKeepsConnection(t *testing.T) {
	for _, tc := range []struct {
		name  string
		frame *wire.ClusterFrame
	}{
		{name: "nil frame"},
		{name: "unsupported peer ID", frame: &wire.ClusterFrame{ClusterPeerID: 7}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			local, peer := net.Pipe()
			defer peer.Close()
			if err := peer.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
				t.Fatal(err)
			}
			conn := newConn(newPipeTransport(t, local))
			conn.Start()
			defer conn.Close()
			if err := conn.WriteFrame(tc.frame); !errors.Is(err, wire.ErrInvalidClusterFrame) {
				t.Fatalf("invalid frame error = %v, want validation error", err)
			}
			if !conn.IsActive() {
				t.Fatal("connection closed after invalid frame")
			}
			read := make(chan error, 1)
			go func() { _, err := wire.ReadClusterFrame(peer, wire.MaxFramePayloadLen); read <- err }()
			if err := conn.WriteFrame(&wire.ClusterFrame{}); err != nil {
				t.Fatalf("write after validation failure: %v", err)
			}
			if err := receive(t, read, "frame after validation failure"); err != nil {
				t.Fatalf("read after validation failure: %v", err)
			}
		})
	}
}

func TestBaseConn_InvalidRPCPreservesOtherPending(t *testing.T) {
	local, peer := net.Pipe()
	defer peer.Close()
	if err := peer.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}
	conn := newConn(newPipeTransport(t, local))
	conn.Start()
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result := make(chan rpcResult, 1)
	req := &wire.PingRequest{ChainID: [wire.UInt256Length]byte{1}}
	go func() {
		resp, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), req)
		result <- rpcResult{resp, err}
	}()
	request, err := wire.ReadClusterFrame(peer, wire.MaxFramePayloadLen)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.SendRPC(ctx, byte(wire.ClusterOpPing), &stubSSZRequest{payload: make([]byte, wire.MaxFramePayloadLen+1)})
	if !errors.Is(err, wire.ErrInvalidClusterFrame) || conn.pendingLen() != 1 {
		t.Fatalf("invalid RPC: error=%v pending=%d", err, conn.pendingLen())
	}
	payload, err := (&wire.PongResponse{ChainID: [wire.UInt256Length]byte{1}}).MarshalSSZ()
	if err != nil {
		t.Fatal(err)
	}
	if err := wire.WriteClusterFrame(peer, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: request.RPCID, Payload: payload}); err != nil {
		t.Fatal(err)
	}
	if got := receive(t, result, "pending RPC"); got.err != nil {
		t.Fatalf("pending RPC failed: %v", got.err)
	}
}

func TestBaseConn_InvalidResponseClosesConnection(t *testing.T) {
	tr := newFakeFrameTransport()
	tr.writeErr = wire.ErrInvalidClusterFrame
	conn := newPingServerConn(tr)
	conn.Start()
	defer conn.Close()
	tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: validPingPayload(t)}
	awaitClosed(t, conn, "invalid response frame")
}

// TestBaseConn_WriteFailureReleasesWriteMuAndShutsDown: the transport error
// path (not just the panic path) must preserve the transport error for the sender,
// shut the connection down (transport closed exactly once), and leave writeMu
// free: follow-up sends observe the Closed state instead of blocking forever.
func TestBaseConn_WriteFailureReleasesWriteMuAndShutsDown(t *testing.T) {
	writeBoom := errors.New("fake write boom")
	tr := newFakeFrameTransport()
	tr.writeErr = writeBoom
	conn := newConnWithHello(tr)
	conn.Start()

	errCh := make(chan error, 1)
	go func() { errCh <- conn.SendCommand(byte(wire.CommandOpHello), &wire.HelloCommand{}) }()
	if err := receive(t, errCh, "SendCommand result"); !errors.Is(err, ErrConnectionClosed) || !errors.Is(err, writeBoom) {
		t.Fatalf("SendCommand error = %v, want connection closed and transport error", err)
	}

	awaitClosed(t, conn, "transport write failure")
	if got := tr.closes(); got != 1 {
		t.Fatalf("transport closed %d times, want 1", got)
	}

	// Follow-up send must return promptly with ErrConnectionClosed.
	errCh2 := make(chan error, 1)
	go func() { errCh2 <- conn.SendCommand(byte(wire.CommandOpHello), &wire.HelloCommand{}) }()
	if err := receive(t, errCh2, "post-failure SendCommand"); err != ErrConnectionClosed {
		t.Fatalf("SendCommand after write failure = %v, want ErrConnectionClosed", err)
	}
}

func TestBaseConn_WriteFrameFailurePreservesCause(t *testing.T) {
	writeErr := errors.New("write failed")
	tr := newFakeFrameTransport()
	tr.writeErr = writeErr
	conn := newConn(tr)
	conn.Start()
	defer conn.Close()

	err := conn.WriteFrame(&wire.ClusterFrame{})
	if !errors.Is(err, ErrConnectionClosed) || !errors.Is(err, writeErr) {
		t.Fatalf("WriteFrame error = %v, want connection closed and transport error", err)
	}
}

func TestBaseConn_WriteFailureShutsDownBeforeNextWrite(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := newConnWithHello(tr)
	conn.Start()
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pending := make(chan error, 1)
	go func() {
		_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
		pending <- err
	}()
	receive(t, tr.writes, "pending RPC request")
	tr.writeErr = errors.New("write failed")

	// A failed write must complete shutdown before releasing writeMu.
	conn.writeMu.Lock()
	err := conn.writeFrameLocked(&wire.ClusterFrame{})
	conn.writeMu.Unlock()
	if err == nil {
		t.Fatal("expected transport write error")
	}
	if !conn.IsClosed() {
		t.Fatal("write failure did not close the connection")
	}
	if tr.closes() != 1 || conn.pendingLen() != 0 {
		t.Fatalf("after write failure: transport closes=%d pending=%d", tr.closes(), conn.pendingLen())
	}
	tr.writeErr = nil // Any unexpected transport write would now succeed.
	results := make(chan error, 6)
	for i := 0; i < cap(results); i++ {
		go func() {
			var err error
			switch i % 3 {
			case 0:
				_, err = conn.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
			case 1:
				err = conn.SendCommand(byte(wire.CommandOpHello), &wire.HelloCommand{})
			case 2:
				err = conn.WriteFrame(&wire.ClusterFrame{})
			}
			results <- err
		}()
	}
	for i := 0; i < cap(results); i++ {
		if err := receive(t, results, "send after shutdown"); err != ErrConnectionClosed {
			t.Fatalf("send after shutdown = %v, want ErrConnectionClosed", err)
		}
	}
	if len(tr.writes) != 0 {
		t.Fatal("transport received a write after failure")
	}
	conn.Close()
	conn.Close()
	awaitClosed(t, conn, "Close after write failure")
	if err := receive(t, pending, "pending RPC after Close"); err != ErrConnectionClosed {
		t.Fatalf("pending RPC = %v, want ErrConnectionClosed", err)
	}
	if tr.closes() != 1 || conn.pendingLen() != 0 {
		t.Fatalf("after Close: transport closes=%d pending=%d", tr.closes(), conn.pendingLen())
	}
}

// TestBaseConn_WriteFramePanicReleasesWriteMu: a panic from the transport's
// WriteFrame is caught by writeFrameLocked (SendRPC/writeFrame release
// writeMu via the returned error, not a defer). If writeMu were leaked, senders
// issued after the panic would block forever at writeMu.Lock() instead of
// observing the Closed state.
func TestBaseConn_WriteFramePanicReleasesWriteMu(t *testing.T) {
	tr := newFakeFrameTransport()
	tr.writePanic = true
	conn := newConnWithHello(tr)
	conn.Start()

	// The panic surfaces as an error from the write path, which triggers
	// shutdown rather than leaving writeMu held.
	result := make(chan error, 1)
	go func() {
		_, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
		result <- err
	}()
	select {
	case err := <-result:
		if !errors.Is(err, ErrConnectionClosed) {
			t.Fatalf("SendRPC after WriteFrame panic: expected ErrConnectionClosed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SendRPC blocked on writeMu after WriteFrame panic")
	}
	// A later sender (blocking on writeMu) must observe the Closed state, not
	// wait forever: proves writeMu was released.
	if err := conn.SendCommand(byte(wire.CommandOpHello), &wire.HelloCommand{}); err != ErrConnectionClosed {
		t.Fatalf("SendCommand after WriteFrame panic: expected ErrConnectionClosed, got %v", err)
	}
	<-conn.WaitUntilClosed()
}

func TestBaseConn_HandlerPanicShutsDownConnection(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := NewBaseConn(Config{
		Transport: tr,
		Serializers: map[byte]*OpSerializer{
			byte(wire.ClusterOpPing): pingSer,
		},
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): func(req wire.SSZMessage) (wire.SSZMessage, error) {
				panic("handler panic test")
			},
		},
		Logger: log.New(),
	})
	conn.Start()

	tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: validPingPayload(t)}
	select {
	case <-conn.WaitUntilClosed():
		if !conn.IsClosed() {
			t.Fatal("expected connection to close after handler panic")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("connection did not close after handler panic")
	}
}

// TestBaseConn_ResponseBeforeCloseWins: a delivered response must survive a
// subsequent close before the sender reaches its result select.
func TestBaseConn_ResponseBeforeCloseWins(t *testing.T) {
	for i := 0; i < 64; i++ {
		base := newFakeFrameTransport()
		base.writeStarted = make(chan struct{})
		base.releaseWrite = make(chan struct{})
		conn := newPingConn(&interruptibleFakeFrameTransport{fakeFrameTransport: base})
		conn.Start()

		result := make(chan rpcResult, 1)
		go func() {
			resp, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
			result <- rpcResult{resp: resp, err: err}
		}()
		request := receive(t, base.writes, "ping request")
		receive(t, base.writeStarted, "blocked writer")

		// The reader is idle. Deliver the response while the sender is still
		// inside WriteFrame, then close and release that write. Both select
		// branches are now ready when the sender resumes.
		conn.handleFrameSafely(&wire.ClusterFrame{
			Opcode:  byte(wire.ClusterOpPong),
			RPCID:   request.RPCID,
			Payload: validPongPayload(t),
		})
		conn.Close()

		res := receive(t, result, "completed RPC")
		if res.err != nil {
			t.Fatalf("iteration %d: delivered response replaced by %v", i, res.err)
		}
		if _, ok := res.resp.(*wire.PongResponse); !ok {
			t.Fatalf("unexpected response type %T", res.resp)
		}
		if conn.pendingLen() != 0 {
			t.Fatal("pending RPC remains after close")
		}
	}
}

// TestBaseConn_ResponseCancelRaceCleansPending: response and cancel racing for
// the same rpc id must complete the RPC exactly once and leave no pending
// entry (race detector coverage).
func TestBaseConn_ResponseCancelRaceCleansPending(t *testing.T) {
	const iterations = 100
	pongPayload := validPongPayload(t)
	for i := 0; i < iterations; i++ {
		tr := newFakeFrameTransport()
		conn := newPingConn(tr)
		conn.Start()

		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
			result <- err
		}()

		var request *wire.ClusterFrame
		select {
		case request = <-tr.writes:
		case <-time.After(time.Second):
			t.Fatal("fake transport did not receive request")
		}
		start := make(chan struct{})
		go func() {
			<-start
			cancel()
		}()
		go func() {
			<-start
			tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: request.RPCID, Payload: pongPayload}
		}()
		close(start)

		select {
		case <-result:
		case <-time.After(time.Second):
			t.Fatal("SendRPC did not complete")
		}
		if pending := conn.pendingLen(); pending != 0 {
			t.Fatalf("iteration %d: pending RPCs remain: %d", i, pending)
		}
		conn.Close()
	}
}

// TestBaseConn_ResponseCancelCloseRaceCompletesOnce: response, cancel, and
// close racing for the same rpc id must complete the RPC exactly once with
// no pending leak and no deadlock (the three-way race not covered
// by TestBaseConn_ResponseCancelRaceCleansPending).
func TestBaseConn_ResponseCancelCloseRaceCompletesOnce(t *testing.T) {
	const iterations = 200
	pongPayload := validPongPayload(t)
	for i := 0; i < iterations; i++ {
		tr := newFakeFrameTransport()
		conn := newPingConn(tr)
		conn.Start()

		ctx, cancel := context.WithCancel(context.Background())
		result := make(chan error, 1)
		go func() {
			_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
			result <- err
		}()

		var request *wire.ClusterFrame
		select {
		case request = <-tr.writes:
		case <-time.After(time.Second):
			t.Fatal("fake transport did not receive request")
		}

		start := make(chan struct{})
		go func() { <-start; cancel() }()
		go func() {
			<-start
			tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: request.RPCID, Payload: pongPayload}
		}()
		go func() { <-start; conn.Close() }()
		close(start)

		select {
		case err := <-result:
			// Any of the three outcomes is valid; completion is what matters.
			if err != nil && err != ErrConnectionClosed &&
				!errors.Is(err, context.Canceled) {
				t.Fatalf("iteration %d: unexpected error: %v", i, err)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("iteration %d: SendRPC never completed (deadlock)", i)
		}
		if pending := conn.pendingLen(); pending != 0 {
			t.Fatalf("iteration %d: pending RPCs remain: %d", i, pending)
		}
		cancel()
		conn.Close()
	}
}

func TestBaseConn_ReadFailureWakesPendingRPC(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := newConn(tr)
	conn.Start()

	result := make(chan error, 1)
	go func() {
		_, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
		result <- err
	}()
	select {
	case <-tr.writes:
	case <-time.After(time.Second):
		t.Fatal("fake transport did not receive request")
	}
	if err := tr.Close(); err != nil {
		t.Fatalf("close fake transport: %v", err)
	}
	select {
	case err := <-result:
		if err != ErrConnectionClosed {
			t.Fatalf("expected ErrConnectionClosed, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("read failure did not wake pending RPC")
	}
	<-conn.WaitUntilClosed()
	if pending := conn.pendingLen(); pending != 0 {
		t.Fatalf("pending RPCs remain after read failure: %d", pending)
	}
}

// TestBaseConn_CloseWakesEveryPendingRPC: N concurrently parked callers are
// all woken exactly once with ErrConnectionClosed; pending drains to zero.
func TestBaseConn_CloseWakesEveryPendingRPC(t *testing.T) {
	before := runtime.NumGoroutine()
	tr := newFakeFrameTransport()
	tr.writes = make(chan *wire.ClusterFrame, 64)
	conn := newPingConn(tr)
	conn.Start()

	const n = 16
	results := make(chan error, n)
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
			results <- err
		}()
	}
	for i := 0; i < n; i++ {
		receive(t, tr.writes, "pending request") // every caller reached its wait point
	}

	conn.Close()
	for i := 0; i < n; i++ {
		if err := receive(t, results, "woken rpc"); err != ErrConnectionClosed {
			t.Fatalf("woken rpc returned %v, want ErrConnectionClosed", err)
		}
	}
	wg.Wait()
	if n := conn.pendingLen(); n != 0 {
		t.Fatalf("pending = %d after shutdown, want 0", n)
	}
	waitForGoroutines(t, before)
}

// TestBaseConn_TruncatedFrameClosesConnection: EOF mid-frame shuts the
// connection down because the frame is incomplete.
func TestBaseConn_TruncatedFrameClosesConnection(t *testing.T) {
	// payload_len = 10 but the stream ends right after the length header.
	tr := newStaticReaderTransport(bytes.NewReader([]byte{0x00, 0x00, 0x00, 0x0a}))
	conn := newConn(tr)
	conn.Start()

	<-conn.WaitUntilClosed()
	if !conn.IsClosed() {
		t.Fatal("connection should be closed after truncated frame")
	}
}

// TestSendRPC_ConcurrentSendsPreserveRPCIDOrder: concurrent SendRPC calls must
// produce strictly increasing rpc ids — the peer's monotonic check depends on
// it.
func TestSendRPC_ConcurrentSendsPreserveRPCIDOrder(t *testing.T) {
	tr := newFakeFrameTransport()
	tr.writes = make(chan *wire.ClusterFrame, 64)
	conn := newConn(tr)
	conn.Start()

	const senders = 32
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(senders)
	for i := 0; i < senders; i++ {
		go func() {
			defer wg.Done()
			<-start
			_, _ = conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
		}()
	}

	close(start)
	frames := make([]*wire.ClusterFrame, 0, senders)
	for len(frames) < senders {
		select {
		case frame := <-tr.writes:
			frames = append(frames, frame)
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for concurrent RPC writes")
		}
	}
	for i := 1; i < len(frames); i++ {
		if frames[i].RPCID <= frames[i-1].RPCID {
			t.Fatalf("rpc ids not strictly increasing: %d then %d", frames[i-1].RPCID, frames[i].RPCID)
		}
	}
	if conn.IsClosed() {
		t.Fatal("connection closed during concurrent sends")
	}

	conn.Close()
	wg.Wait()
}

// TestBaseConn_LateHandlerCompletedAfterClose: a handler finishing after Close
// drops its response without panicking or leaking.
func TestBaseConn_LateHandlerCompletedAfterClose(t *testing.T) {
	base := newFakeFrameTransport()
	base.releaseWrite = make(chan struct{})
	tr := &interruptibleFakeFrameTransport{fakeFrameTransport: base}

	const op = byte(wire.ClusterOpPing)
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	conn := NewBaseConn(Config{
		Transport:   tr,
		Serializers: pingSerializers,
		Handlers: map[byte]TypedHandler{
			op: func(req wire.SSZMessage) (wire.SSZMessage, error) {
				close(handlerStarted)
				<-releaseHandler
				return &wire.PongResponse{}, nil
			},
		},
		Logger: log.New(),
	})

	before := runtime.NumGoroutine()
	conn.Start()
	<-conn.WaitUntilActive()

	tr.frames <- &wire.ClusterFrame{Opcode: op, RPCID: 1, Payload: validPingPayload(t)}
	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("handler did not start")
	}

	closeDone := make(chan struct{})
	go func() {
		conn.Close()
		close(closeDone)
	}()
	select {
	case <-closeDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Close blocked with in-flight handler")
	}

	close(releaseHandler)
	waitForGoroutines(t, before)
}

// TestBaseConn_StartCloseLifecycle: Start on closed is a no-op, Close without
// Start must not block, double Close closes the transport once.
func TestBaseConn_StartCloseLifecycle(t *testing.T) {
	t.Run("start on closed is no-op", func(t *testing.T) {
		conn := newConn(newFakeFrameTransport())
		conn.Close()
		conn.Start()
		if !conn.IsClosed() {
			t.Fatal("expected closed state after Start on closed connection")
		}
	})

	t.Run("close without start does not block", func(t *testing.T) {
		conn := newConn(newFakeFrameTransport())
		closeDone := make(chan struct{})
		go func() {
			conn.Close()
			close(closeDone)
		}()
		select {
		case <-closeDone:
		case <-time.After(2 * time.Second):
			t.Fatal("Close() blocked/deadlocked for a connection that was never started")
		}
	})

	t.Run("concurrent start and close", func(t *testing.T) {
		conn := newConn(newFakeFrameTransport())
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			conn.Start()
		}()
		go func() {
			defer wg.Done()
			conn.Close()
		}()
		wg.Wait()
		if !conn.IsClosed() {
			t.Fatal("connection should be closed after concurrent Start/Close")
		}
	})

	t.Run("double close closes transport once", func(t *testing.T) {
		tr := newFakeFrameTransport()
		conn := newConn(tr)
		conn.Close()
		conn.Close()
		if got := tr.closes(); got != 1 {
			t.Fatalf("transport closed %d times, want 1", got)
		}
	})
}

// TestBaseConn_WaitUntilActiveLifecycle pins the channel contract: it fires
// once startup completes, stays silent before Start, and still unblocks when
// Close wins the race against Start (shutdown closes activeChan while
// connecting).
func TestBaseConn_WaitUntilActiveLifecycle(t *testing.T) {
	t.Run("Start transitions Connecting to Active", func(t *testing.T) {
		conn := newConn(newFakeFrameTransport())
		if conn.State() != ConnectionStateConnecting {
			t.Fatalf("initial state = %v, want Connecting", conn.State())
		}
		conn.Start()
		select {
		case <-conn.WaitUntilActive():
		case <-time.After(time.Second):
			t.Fatal("WaitUntilActive did not fire after Start")
		}
		if !conn.IsActive() || conn.IsClosed() {
			t.Fatalf("state after Start: active=%v closed=%v", conn.IsActive(), conn.IsClosed())
		}
		conn.Close()
	})

	t.Run("does not fire before Start", func(t *testing.T) {
		conn := newConn(newFakeFrameTransport())
		select {
		case <-conn.WaitUntilActive():
			t.Fatal("active fired before Start")
		default:
		}
	})

	t.Run("unblocks when closed while connecting", func(t *testing.T) {
		conn := newConn(newFakeFrameTransport())
		conn.Close()
		select {
		case <-conn.WaitUntilActive():
		case <-time.After(2 * time.Second):
			t.Fatal("WaitUntilActive did not unblock after early Close")
		}
		if !conn.IsClosed() || conn.IsActive() {
			t.Fatal("expected Closed state after early close")
		}
	})
}

// TestBaseConn_StartIsIdempotent: repeat Start must be a no-op — state stays
// Active, dispatch keeps working exactly once per frame, and shutdown still
// closes the transport exactly once.
func TestBaseConn_StartIsIdempotent(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := newPingServerConn(tr)

	before := runtime.NumGoroutine()
	conn.Start()
	conn.Start()

	feedFrame(t, tr, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: validPingPayload(t)})
	reply := receive(t, tr.writes, "pong reply")
	if reply.RPCID != 1 || reply.Opcode != byte(wire.ClusterOpPong) {
		t.Fatalf("unexpected reply opcode=0x%x rpc_id=%d", reply.Opcode, reply.RPCID)
	}
	if conn.State() != ConnectionStateActive {
		t.Fatalf("state after second Start = %v, want Active", conn.State())
	}

	conn.Close()
	if got := tr.closes(); got != 1 {
		t.Fatalf("transport closed %d times, want 1", got)
	}
	waitForGoroutines(t, before)
}

func TestBaseConn_SendRPCRejectsNonRequestOpcodes(t *testing.T) {
	tr := newFakeFrameTransport()
	const commandOp = byte(0x42)
	conn := NewBaseConn(Config{
		Transport: tr,
		Serializers: map[byte]*OpSerializer{
			byte(wire.ClusterOpPing): pingSer,
			commandOp:                NonRPCSerializerFor[wire.PingRequest](),
		},
		NonRPCOps: map[byte]struct{}{commandOp: {}},
	})
	conn.Start()
	defer conn.Close()

	// Invalid opcodes must be rejected even while another sender holds locks.
	conn.writeMu.Lock()
	defer conn.writeMu.Unlock()
	conn.mu.Lock()
	defer conn.mu.Unlock()
	result := make(chan error, 1)
	for _, opcode := range []byte{0xff, byte(wire.ClusterOpPong), commandOp} {
		go func(op byte) {
			_, err := conn.SendRPC(context.Background(), op, nil)
			result <- err
		}(opcode)
		if err := receive(t, result, "invalid RPC opcode rejection"); err == nil {
			t.Fatalf("SendRPC accepted opcode 0x%x", opcode)
		}
	}
	select {
	case frame := <-tr.writes:
		t.Fatalf("unexpected frame: %+v", frame)
	default:
	}
}

func TestBaseConn_SendRPCZeroPeerID(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := newPingConn(tr)
	conn.Start()
	defer conn.Close()

	req := &wire.PingRequest{ChainID: [wire.UInt256Length]byte{1}}
	result := make(chan rpcResult, 1)
	go func() {
		resp, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), req)
		result <- rpcResult{resp: resp, err: err}
	}()

	request := receive(t, tr.writes, "RPC request")
	if request.Opcode != byte(wire.ClusterOpPing) || request.RPCID != 1 || request.ClusterPeerID != 0 {
		t.Fatalf("unexpected RPC request frame: %+v", request)
	}
	if !bytes.Equal(request.Payload, validPingPayload(t)) {
		t.Fatalf("unexpected RPC request payload: %x", request.Payload)
	}
	feedFrame(t, tr, &wire.ClusterFrame{
		Opcode:  byte(wire.ClusterOpPong),
		RPCID:   request.RPCID,
		Payload: validPongPayload(t),
	})
	res := receive(t, result, "RPC response")
	if res.err != nil {
		t.Fatalf("SendRPC failed: %v", res.err)
	}
	if _, ok := res.resp.(*wire.PongResponse); !ok {
		t.Fatalf("expected *wire.PongResponse, got %T", res.resp)
	}
}

// -- SendCommand tests --------------------------------------------------------

// TestBaseConn_SendCommand checks fire-and-forget commands use zero IDs, run
// the handler without a response, and fail on a closed connection.
func TestBaseConn_SendCommand(t *testing.T) {
	for _, tt := range []struct {
		name       string
		serializer *OpSerializer
	}{
		{name: "zero response opcode", serializer: NonRPCSerializerFor[wire.HelloCommand]()},
		{name: "self-referencing response", serializer: OpSerializerFor[wire.HelloCommand, wire.HelloCommand](byte(wire.CommandOpHello))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			tr := newFakeFrameTransport()
			handled := make(chan any, 1)
			conn := NewBaseConn(Config{
				Transport: tr,
				Serializers: map[byte]*OpSerializer{
					byte(wire.CommandOpHello): tt.serializer,
				},
				Handlers: map[byte]TypedHandler{
					byte(wire.CommandOpHello): func(req wire.SSZMessage) (wire.SSZMessage, error) {
						handled <- req
						return nil, nil
					},
				},
				NonRPCOps: map[byte]struct{}{byte(wire.CommandOpHello): {}},
			})
			conn.Start()
			defer conn.Close()

			err := conn.SendCommand(byte(wire.CommandOpHello), &wire.HelloCommand{})
			if err != nil {
				t.Fatalf("SendCommand failed: %v", err)
			}

			select {
			case f := <-tr.writes:
				if f.RPCID != 0 {
					t.Fatalf("expected rpc_id=0, got %d", f.RPCID)
				}
				if f.Opcode != byte(wire.CommandOpHello) {
					t.Fatalf("expected opcode 0x%x, got 0x%x", byte(wire.CommandOpHello), f.Opcode)
				}
				if f.ClusterPeerID != 0 {
					t.Fatalf("expected cluster peer ID 0, got %d", f.ClusterPeerID)
				}
				if !bytes.Equal(f.Payload, validHelloPayload(t)) {
					t.Fatalf("unexpected command payload: %x", f.Payload)
				}
				feedFrame(t, tr, f)
			case <-time.After(time.Second):
				t.Fatal("SendCommand did not write a frame")
			}

			if _, ok := receive(t, handled, "Hello handler").(*wire.HelloCommand); !ok {
				t.Fatal("Hello handler received wrong request type")
			}

			select {
			case f := <-tr.writes:
				t.Fatalf("expected no response for fire-and-forget command, got opcode 0x%x rpc_id=%d", f.Opcode, f.RPCID)
			case <-time.After(50 * time.Millisecond):
			}
			if !conn.IsActive() {
				t.Fatal("connection closed after valid non-RPC command")
			}
		})
	}

	t.Run("returns error on closed connection", func(t *testing.T) {
		conn := NewBaseConn(Config{
			Transport: newFakeFrameTransport(),
			Serializers: map[byte]*OpSerializer{
				byte(wire.CommandOpHello): NonRPCSerializerFor[wire.HelloCommand](),
			},
			NonRPCOps: map[byte]struct{}{byte(wire.CommandOpHello): {}},
		})
		conn.Close()

		err := conn.SendCommand(byte(wire.CommandOpHello), &wire.HelloCommand{})
		if err != ErrConnectionClosed {
			t.Fatalf("expected ErrConnectionClosed, got %v", err)
		}
	})
}

func TestBaseConn_SendCommandInvalidMessageKeepsConnection(t *testing.T) {
	marshalErr := errors.New("marshal failed")
	for _, tc := range []struct {
		name    string
		msg     wire.SSZMessage
		wantErr error
	}{
		{name: "nil message"},
		{name: "serialization error", msg: &stubSSZRequest{err: marshalErr}, wantErr: marshalErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := newFakeFrameTransport()
			conn := newConnWithHello(tr)
			conn.Start()
			defer conn.Close()
			err := conn.SendCommand(byte(wire.CommandOpHello), tc.msg)
			if err == nil || (tc.wantErr != nil && !errors.Is(err, tc.wantErr)) {
				t.Fatalf("invalid command error = %v", err)
			}
			if !conn.IsActive() {
				t.Fatal("invalid command closed the connection")
			}
			select {
			case frame := <-tr.writes:
				t.Fatalf("invalid command wrote a frame: %+v", frame)
			default:
			}
		})
	}
}

// -- Inbound dispatch tests ----------------------------------------------------

// TestDispatch_InvalidFramesCloseConnection: malformed or unrecognized inbound
// frames close the connection.
func TestDispatch_InvalidFramesCloseConnection(t *testing.T) {
	pingPayload := validPingPayload(t)

	tests := []struct {
		name   string
		server bool // true: server conn (handles PING); false: client conn (receives PONG)
		frames []*wire.ClusterFrame
	}{
		{
			name:   "unsupported request opcode",
			server: true,
			frames: []*wire.ClusterFrame{{Opcode: byte(wire.ClusterOpConnectToSlavesRequest), RPCID: 1, Payload: pingPayload}},
		},
		{
			name:   "truncated request payload",
			server: true,
			frames: []*wire.ClusterFrame{{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: pingPayload[:1]}},
		},
		{
			name:   "zero rpc_id on rpc request",
			server: true,
			frames: []*wire.ClusterFrame{{Opcode: byte(wire.ClusterOpPing), RPCID: 0, Payload: pingPayload}},
		},
		{
			name:   "duplicate rpc_id",
			server: true,
			frames: []*wire.ClusterFrame{
				{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: pingPayload},
				{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: pingPayload},
			},
		},
		{
			name:   "decreasing rpc_id",
			server: true,
			frames: []*wire.ClusterFrame{
				{Opcode: byte(wire.ClusterOpPing), RPCID: 2, Payload: pingPayload},
				{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: pingPayload},
			},
		},
		{
			name:   "unknown response opcode",
			server: false,
			frames: []*wire.ClusterFrame{{Opcode: 0xFF, RPCID: 1, Payload: []byte{0x00}}},
		},
		{
			name:   "zero response rpc_id",
			server: false,
			frames: []*wire.ClusterFrame{{Opcode: byte(wire.ClusterOpPong), RPCID: 0, Payload: validPongPayload(t)}},
		},
		{
			name:   "unknown response rpc_id",
			server: false,
			frames: []*wire.ClusterFrame{{Opcode: byte(wire.ClusterOpPong), RPCID: 999, Payload: validPongPayload(t)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newFakeFrameTransport()
			var conn *BaseConn
			if tt.server {
				conn = newPingServerConn(tr)
			} else {
				conn = newPingConn(tr)
			}
			conn.Start()
			defer conn.Close()

			for _, f := range tt.frames {
				select {
				case tr.frames <- f:
				case <-time.After(time.Second):
					t.Fatal("transport write blocked feeding frame")
				}
			}

			select {
			case <-conn.WaitUntilClosed():
				if !conn.IsClosed() {
					t.Fatal("expected connection to close")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("connection did not close after invalid frame")
			}
		})
	}
}

// -- Dispatch fatal paths ------------------------------------------------------
//
// One processing error must close the whole connection. The four reviewer-flag
// gaps are covered here: handler error, response serialization failure,
// response deserialization failure, and transport write failure
// (TestBaseConn_WriteFailureReleasesWriteMuAndShutsDown).

// TestBaseConn_MalformedSSZShutsDownConnection checks that malformed payloads
// are fatal in both directions.
func TestBaseConn_MalformedSSZShutsDownConnection(t *testing.T) {
	t.Run("truncated request payload", func(t *testing.T) {
		tr := newFakeFrameTransport()
		conn := newPingServerConn(tr)
		conn.Start()

		good := validPingPayload(t)
		bad := good[:1]
		feedFrame(t, tr, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: bad})
		awaitClosed(t, conn, "truncated request")
	})

	t.Run("response payload trailing bytes", func(t *testing.T) {
		tr := newFakeFrameTransport()
		conn := newPingConn(tr)
		conn.Start()

		result := make(chan rpcResult, 1)
		go func() {
			resp, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
			result <- rpcResult{resp: resp, err: err}
		}()
		request := receive(t, tr.writes, "ping request")

		good := validPongPayload(t)
		bad := append(good[:len(good):len(good)], 0xFF)
		feedFrame(t, tr, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: request.RPCID, Payload: bad})

		awaitClosed(t, conn, "response with trailing bytes")
		res := receive(t, result, "aborted rpc result")
		if res.err != ErrConnectionClosed {
			t.Fatalf("pending rpc error = %v, want ErrConnectionClosed", res.err)
		}
		if n := conn.pendingLen(); n != 0 {
			t.Fatalf("pending = %d after malformed response, want 0", n)
		}
	})
}

// TestBaseConn_HandlerErrorShutsDownConnection: a handler returning an error
// is fatal; no partial response may be written.
func TestBaseConn_HandlerErrorShutsDownConnection(t *testing.T) {
	tr := newFakeFrameTransport()
	handlerErr := errors.New("handler boom")
	conn := NewBaseConn(Config{
		Transport:   tr,
		Serializers: pingSerializers,
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): func(wire.SSZMessage) (wire.SSZMessage, error) { return nil, handlerErr },
		},
		Logger: log.New(),
	})
	conn.Start()

	feedFrame(t, tr, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 5, Payload: validPingPayload(t)})
	awaitClosed(t, conn, "handler error")

	select {
	case f := <-tr.writes:
		t.Fatalf("unexpected write after handler error: opcode 0x%x rpc_id=%d", f.Opcode, f.RPCID)
	default:
	}
}

func TestBaseConn_BusinessRejectionKeepsConnectionActive(t *testing.T) {
	tr := newFakeFrameTransport()
	requestOp := byte(wire.ClusterOpAddRootBlockRequest)
	responseOp := byte(wire.ClusterOpAddRootBlockResponse)
	conn := NewBaseConn(Config{
		Transport:   tr,
		Serializers: map[byte]*OpSerializer{requestOp: OpSerializerFor[wire.AddRootBlockRequest, wire.AddRootBlockResponse](responseOp)},
		Handlers: map[byte]TypedHandler{
			requestOp: func(wire.SSZMessage) (wire.SSZMessage, error) { return &wire.AddRootBlockResponse{ErrorCode: 7}, nil },
		},
	})
	conn.Start()
	defer conn.Close()
	feedFrame(t, tr, &wire.ClusterFrame{Opcode: requestOp, RPCID: 1, Payload: mustSerialize(t, &wire.AddRootBlockRequest{})})
	frame := receive(t, tr.writes, "business rejection response")
	response := new(wire.AddRootBlockResponse)
	if err := response.UnmarshalSSZ(frame.Payload); err != nil {
		t.Fatal(err)
	}
	if frame.Opcode != responseOp || frame.RPCID != 1 || response.ErrorCode != 7 || !conn.IsActive() {
		t.Fatalf("business rejection: frame=%+v response=%+v state=%v", frame, response, conn.State())
	}
}

type failingSSZResponse struct{ wire.PongResponse }

func (*failingSSZResponse) MarshalSSZ() ([]byte, error) {
	return nil, errors.New("serialize boom")
}

// TestBaseConn_ResponseSerializationFailureShutsDown checks a serialization
// failure closes the connection without writing a response.
func TestBaseConn_ResponseSerializationFailureShutsDown(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := NewBaseConn(Config{
		Transport: tr,
		Serializers: map[byte]*OpSerializer{
			byte(wire.ClusterOpPing): pingSer,
		},
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): func(wire.SSZMessage) (wire.SSZMessage, error) { return &failingSSZResponse{}, nil },
		},
		Logger: log.New(),
	})
	conn.Start()

	feedFrame(t, tr, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 7, Payload: validPingPayload(t)})
	awaitClosed(t, conn, "response serialization failure")

	select {
	case f := <-tr.writes:
		t.Fatalf("unexpected write after serialization failure: opcode 0x%x rpc_id=%d", f.Opcode, f.RPCID)
	default:
	}
}

// TestBaseConn_MalformedResponseClosesPendingRPC: an inbound response payload
// that cannot be deserialized shuts the connection down and ends the waiting
// caller with ErrConnectionClosed.
func TestBaseConn_MalformedResponseClosesPendingRPC(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := newPingConn(tr)
	conn.Start()

	result := make(chan rpcResult, 1)
	go func() {
		resp, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
		result <- rpcResult{resp: resp, err: err}
	}()
	request := receive(t, tr.writes, "ping request")

	// Structurally broken payload for the pong codec.
	feedFrame(t, tr, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: request.RPCID, Payload: []byte{0x01, 0x02, 0x03}})

	awaitClosed(t, conn, "malformed response payload")
	res := receive(t, result, "aborted rpc result")
	if res.err != ErrConnectionClosed {
		t.Fatalf("pending rpc error = %v, want ErrConnectionClosed", res.err)
	}
	if n := conn.pendingLen(); n != 0 {
		t.Fatalf("pending = %d after malformed response, want 0", n)
	}
}

// TestPeerRPCID_Uint64Monotonic: inbound rpc ids are compared as unsigned.
// Zero is reserved for fire-and-forget commands. High-half ids must remain
// monotonic; an int64 comparison would wrap them negative.
func TestPeerRPCID_Uint64Monotonic(t *testing.T) {
	conn := newConn(newFakeFrameTransport())

	if conn.defaultValidateRPCID(0) {
		t.Fatal("rpc_id=0 accepted; 0 is reserved for non-RPC commands")
	}
	if !conn.defaultValidateRPCID(1) {
		t.Fatal("first rpc_id=1 rejected")
	}
	if conn.defaultValidateRPCID(1) {
		t.Fatal("duplicate rpc_id=1 accepted")
	}
	for _, id := range []uint64{math.MaxInt64, 1 << 63, math.MaxUint64} {
		if !conn.defaultValidateRPCID(id) {
			t.Fatalf("rpc_id=%d rejected", id)
		}
	}
	if conn.defaultValidateRPCID(math.MaxUint64) {
		t.Fatal("duplicate MaxUint64 accepted")
	}
}

// -- Forwarder tests -----------------------------------------------------------

// TestForwarder_RoutesFrame: returning true (consumed) skips dispatch,
// returning false (pass) dispatches normally.
func TestForwarder_RoutesFrame(t *testing.T) {
	tests := []struct {
		name      string
		consumed  bool
		wantReply bool
	}{
		{"consumed", true, false},
		{"pass", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := newFakeFrameTransport()
			forwarded := make(chan *wire.ClusterFrame, 1)
			conn := NewBaseConn(Config{
				Transport:   tr,
				Serializers: pingSerializers,
				Handlers: map[byte]TypedHandler{
					byte(wire.ClusterOpPing): pongHandler(),
				},
				Forwarder: func(f *wire.ClusterFrame) bool {
					forwarded <- f
					return tt.consumed
				},
				Logger: log.New(),
			})
			conn.Start()
			defer conn.Close()

			tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: validPingPayload(t)}

			select {
			case f := <-forwarded:
				if f.RPCID != 1 {
					t.Fatalf("expected rpc_id=1, got %d", f.RPCID)
				}
			case <-time.After(time.Second):
				t.Fatal("forwarder did not receive frame")
			}

			if tt.wantReply {
				select {
				case f := <-tr.writes:
					if f.RPCID != 1 || f.Opcode != byte(wire.ClusterOpPong) {
						t.Fatalf("unexpected reply: opcode 0x%x rpc_id=%d", f.Opcode, f.RPCID)
					}
				case <-time.After(time.Second):
					t.Fatal("handler did not write response")
				}
			} else {
				select {
				case f := <-tr.writes:
					t.Fatalf("expected no response when forwarder consumed frame, got opcode 0x%x", f.Opcode)
				case <-time.After(50 * time.Millisecond):
				}
			}
		})
	}
}

// TestForwarder_CloseRequest: a forwarder calling c.Close() directly on the
// reader goroutine shuts the connection down without dispatching the frame and
// does not deadlock (regression: a forwarder calling Close() must be safe now
// that Close is non-blocking).
func TestForwarder_CloseRequest(t *testing.T) {
	tr := newFakeFrameTransport()
	var conn *BaseConn
	conn = NewBaseConn(Config{
		Transport:   tr,
		Serializers: pingSerializers,
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): pongHandler(),
		},
		Forwarder: func(f *wire.ClusterFrame) bool {
			// Close is non-blocking and safe to call from the reader
			// goroutine; return true so the closing frame is not dispatched.
			conn.Close()
			return true
		},
		Logger: log.New(),
	})
	conn.Start()

	tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: validPingPayload(t)}

	select {
	case <-conn.WaitUntilClosed():
	case <-time.After(time.Second):
		t.Fatal("forwarder Close() did not shut down the connection")
	}
	if !conn.IsClosed() {
		t.Fatal("connection should be closed after forwarder Close()")
	}

	// The frame must not have dispatched: no pong response.
	select {
	case f := <-tr.writes:
		t.Fatalf("expected no dispatch after forwarder Close(), got opcode 0x%x", f.Opcode)
	default:
	}

	closeDone := make(chan struct{})
	go func() {
		conn.Close()
		close(closeDone)
	}()
	select {
	case <-closeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Close() deadlocked waiting for readerLoop exit after forwarder Close()")
	}
	if got := tr.closes(); got != 1 {
		t.Fatalf("transport closed %d times, want 1", got)
	}
}

// TestForwarder_PanicIsolatesConnection: a panic inside the forwarder must
// not crash the process; it shuts the connection down with a descriptive
// error instead (reader-path panic isolation).
func TestForwarder_PanicIsolatesConnection(t *testing.T) {
	tr := newFakeFrameTransport()
	conn := NewBaseConn(Config{
		Transport:   tr,
		Serializers: pingSerializers,
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): pongHandler(),
		},
		Forwarder: func(f *wire.ClusterFrame) bool {
			panic("forwarder boom")
		},
		Logger: log.New(),
	})
	conn.Start()
	defer conn.Close()

	tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPing), RPCID: 1, Payload: validPingPayload(t)}

	select {
	case <-conn.WaitUntilClosed():
	case <-time.After(time.Second):
		t.Fatal("WaitUntilClosed did not fire after forwarder panic")
	}
	if !conn.IsClosed() {
		t.Fatal("connection should be closed after forwarder panic")
	}
}

// -- Non-RPC tests --------------------------------------------------------------

// TestNonRPC: a non-RPC opcode dispatches to the handler without a response;
// a non-RPC opcode with non-zero rpc_id is a protocol violation.
func TestNonRPC(t *testing.T) {
	t.Run("dispatches handler without response", func(t *testing.T) {
		tr := newFakeFrameTransport()
		done := make(chan struct{})
		conn := NewBaseConn(Config{
			Transport:   tr,
			Serializers: map[byte]*OpSerializer{byte(wire.CommandOpHello): NonRPCSerializerFor[wire.HelloCommand]()},
			Handlers: map[byte]TypedHandler{
				byte(wire.CommandOpHello): func(req wire.SSZMessage) (wire.SSZMessage, error) {
					close(done)
					return nil, nil
				},
			},
			NonRPCOps: map[byte]struct{}{
				byte(wire.CommandOpHello): {},
			},
			Logger: log.New(),
		})
		conn.Start()
		defer conn.Close()

		tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.CommandOpHello), RPCID: 0, Payload: validHelloPayload(t)}

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("handler was not invoked for non-RPC opcode")
		}

		select {
		case f := <-tr.writes:
			t.Fatalf("expected no response for non-RPC, got opcode 0x%x rpc_id=%d", f.Opcode, f.RPCID)
		case <-time.After(50 * time.Millisecond):
		}
		if !conn.IsActive() {
			t.Fatal("connection closed after valid non-RPC command")
		}
	})

	t.Run("non-zero rpc_id shuts down connection", func(t *testing.T) {
		tr := newFakeFrameTransport()
		conn := NewBaseConn(Config{
			Transport:   tr,
			Serializers: map[byte]*OpSerializer{byte(wire.CommandOpHello): NonRPCSerializerFor[wire.HelloCommand]()},
			Handlers: map[byte]TypedHandler{
				byte(wire.CommandOpHello): func(wire.SSZMessage) (wire.SSZMessage, error) { return nil, nil },
			},
			NonRPCOps: map[byte]struct{}{
				byte(wire.CommandOpHello): {},
			},
			Logger: log.New(),
		})
		conn.Start()

		tr.frames <- &wire.ClusterFrame{Opcode: byte(wire.CommandOpHello), RPCID: 99, Payload: validHelloPayload(t)}

		select {
		case <-conn.WaitUntilClosed():
		case <-time.After(time.Second):
			t.Fatal("connection did not close after non-RPC with non-zero rpc_id")
		}
	})
}

// A canceled RPC ID only permits late responses, never unsupported requests.
func TestBaseConn_OpcodeWithoutHandlerClosesConnection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		opcode  byte
		payload []byte
	}{
		{"RPC request", byte(wire.ClusterOpPing), validPingPayload(t)},
		{"non-RPC command", byte(wire.CommandOpHello), validHelloPayload(t)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tr := newFakeFrameTransport()
			conn := newConnWithHello(tr)
			conn.Start()
			defer conn.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() {
				_, err := conn.SendRPC(ctx, byte(wire.ClusterOpPing), &wire.PingRequest{})
				result <- err
			}()
			request := receive(t, tr.writes, "RPC request")
			cancel()
			if err := receive(t, result, "canceled RPC"); err != context.Canceled {
				t.Fatalf("RPC error = %v, want context canceled", err)
			}
			feedFrame(t, tr, &wire.ClusterFrame{Opcode: tc.opcode, RPCID: request.RPCID, Payload: tc.payload})
			awaitClosed(t, conn, "request opcode without a handler")
		})
	}
}

// TestBaseConn_NonRPCFrameDoesNotMatchPendingResponse: a registered non-RPC
// command is routed to its handler (never through the response matcher), and
// an outstanding outbound RPC stays matchable afterwards.
func TestBaseConn_NonRPCFrameDoesNotMatchPendingResponse(t *testing.T) {
	const cmdOp = byte(wire.CommandOpHello)
	tr := newFakeFrameTransport()
	handlerCalled := make(chan struct{})
	conn := NewBaseConn(Config{
		Transport: tr,
		Serializers: map[byte]*OpSerializer{
			byte(wire.ClusterOpPing): pingSer,
			cmdOp:                    NonRPCSerializerFor[wire.HelloCommand](),
		},
		Handlers: map[byte]TypedHandler{
			cmdOp: func(req wire.SSZMessage) (wire.SSZMessage, error) {
				close(handlerCalled)
				return nil, nil
			},
		},
		NonRPCOps: map[byte]struct{}{cmdOp: {}},
		Logger:    log.New(),
	})
	conn.Start()
	defer conn.Close()

	pending := make(chan rpcResult, 1)
	go func() {
		resp, err := conn.SendRPC(context.Background(), byte(wire.ClusterOpPing), &wire.PingRequest{})
		pending <- rpcResult{resp: resp, err: err}
	}()
	request := receive(t, tr.writes, "pending request")

	feedFrame(t, tr, &wire.ClusterFrame{Opcode: cmdOp, RPCID: 0, Payload: validHelloPayload(t)})
	select {
	case <-handlerCalled:
	case <-time.After(time.Second):
		t.Fatal("non-RPC handler was not invoked")
	}

	// The original RPC completes normally afterwards: the non-RPC dispatch
	// never touched the pending map or the response path.
	feedFrame(t, tr, &wire.ClusterFrame{Opcode: byte(wire.ClusterOpPong), RPCID: request.RPCID, Payload: validPongPayload(t)})
	res := receive(t, pending, "pending rpc result")
	if res.err != nil {
		t.Fatalf("pending rpc failed after non-RPC dispatch: %v", res.err)
	}

	// Fire-and-forget commands get no response frame.
	select {
	case f := <-tr.writes:
		t.Fatalf("unexpected write for non-RPC cmd: opcode 0x%x rpc_id=%d", f.Opcode, f.RPCID)
	case <-time.After(50 * time.Millisecond):
	}
	if conn.IsClosed() {
		t.Fatal("non-RPC dispatch must keep the connection open")
	}
}

// -- Full lifecycle simulation (real TCP + golden bytes) ----------------------
//
// These fixtures verify the current ClusterFrame layout, including its
// ClusterPeerID routing field, plus payload serialization and RPC handling.
const (
	goldenPingFrame = "000000250000000000000000810000000000000001" +
		"0100000000000000000000000000000000000000000000000000000000000000" + "0025000000"
	goldenPongFrame = "000000200000000000000000820000000000000001" +
		"0100000000000000000000000000000000000000000000000000000000000000"
	goldenPingFrameWithPeerID = "000000250000000000000007810000000000000005" +
		"0100000000000000000000000000000000000000000000000000000000000000" + "0025000000"
	goldenPongFrameWithPeerID = "000000200000000000000007820000000000000005" +
		"0100000000000000000000000000000000000000000000000000000000000000"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex fixture: %v", err)
	}
	return b
}

// newPeerIDConnPair creates a BaseConn over a real TCP socket using the
// ClusterFrame codec that carries a ClusterPeerID.
func newPeerIDConnPair(t *testing.T, cfg Config) (*BaseConn, net.Conn) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	raw, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	peer, err := ln.Accept()
	if err != nil {
		t.Fatalf("accept: %v", err)
	}

	cfg.Transport = NewTCPTransport(peer,
		func(r io.Reader) (*wire.ClusterFrame, error) {
			return wire.ReadClusterFrameWithPeerID(r, wire.MaxFramePayloadLen)
		},
		wire.WriteClusterFrameWithPeerID)
	cfg.Logger = log.New()
	return NewBaseConn(cfg), raw
}

// readFull reads exactly len(buf) bytes with a deadline.
func readFull(t *testing.T, c net.Conn, buf []byte) {
	t.Helper()
	if err := c.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := io.ReadFull(c, buf); err != nil {
		t.Fatalf("read %d bytes: %v", len(buf), err)
	}
}

// TestSimulation_ClusterFrameGoldenBytes: a raw peer writes ClusterFrame
// fixtures; the Go connection must deserialize them, dispatch the handler,
// and reply with the expected PONG frame bytes.
func TestSimulation_ClusterFrameGoldenBytes(t *testing.T) {
	var gotPing []*wire.PingRequest
	server, raw := newPeerIDConnPair(t, Config{
		Serializers: pingSerializers,
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): func(req wire.SSZMessage) (wire.SSZMessage, error) {
				ping := req.(*wire.PingRequest)
				gotPing = append(gotPing, ping)
				return &wire.PongResponse{ChainID: [wire.UInt256Length]byte{1}}, nil
			},
		},
	})
	server.Start()
	defer server.Close()

	// ClusterPeerID=0, rpc_id=1.
	if _, err := raw.Write(mustHex(t, goldenPingFrame)); err != nil {
		t.Fatalf("write golden ping: %v", err)
	}
	want := mustHex(t, goldenPongFrame)
	got := make([]byte, len(want))
	readFull(t, raw, got)
	if !bytes.Equal(got, want) {
		t.Fatalf("pong bytes mismatch\n got: %x\nwant: %x", got, want)
	}

	// ClusterPeerID=7, rpc_id=5: the response must preserve the route ID.
	if _, err := raw.Write(mustHex(t, goldenPingFrameWithPeerID)); err != nil {
		t.Fatalf("write golden ping (peer ID): %v", err)
	}
	want = mustHex(t, goldenPongFrameWithPeerID)
	got = make([]byte, len(want))
	readFull(t, raw, got)
	if !bytes.Equal(got, want) {
		t.Fatalf("pong bytes (peer ID) mismatch\n got: %x\nwant: %x", got, want)
	}

	if len(gotPing) != 2 {
		t.Fatalf("handler invoked %d times, want 2", len(gotPing))
	}
	if gotPing[0].ChainID != ([wire.UInt256Length]byte{1}) {
		t.Fatalf("deserialized ping mismatch: %+v", gotPing[0])
	}
	if gotPing[0].HasRootTip {
		t.Fatalf("expected no root tip, got %v", gotPing[0].RootTip)
	}
	if server.IsClosed() {
		t.Fatal("connection must stay open after golden exchange")
	}
}

// TestSimulation_FullLifecycle walks startup → bidirectional concurrent RPC →
// timeout with late response → clean close over one internal TCP connection.
func TestSimulation_FullLifecycle(t *testing.T) {
	var slow atomic.Bool // gates the server ping handler to force timeouts

	server, raw := newPeerIDConnPair(t, Config{
		Serializers: pingSerializers,
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): func(req wire.SSZMessage) (wire.SSZMessage, error) {
				if slow.Load() {
					time.Sleep(300 * time.Millisecond)
				}
				return &wire.PongResponse{ChainID: [wire.UInt256Length]byte{1}}, nil
			},
		},
	})
	client := NewBaseConn(Config{
		Transport: NewTCPTransport(raw,
			func(r io.Reader) (*wire.ClusterFrame, error) {
				return wire.ReadClusterFrameWithPeerID(r, wire.MaxFramePayloadLen)
			},
			wire.WriteClusterFrameWithPeerID),
		Serializers: map[byte]*OpSerializer{
			byte(wire.ClusterOpPing): pingSer,
		},
		Handlers: map[byte]TypedHandler{
			byte(wire.ClusterOpPing): pongHandler(),
		},
		Logger: log.New(),
	})

	baseline := runtime.NumGoroutine()
	server.Start()
	client.Start()

	// -- Startup: both sides become active.
	select {
	case <-server.WaitUntilActive():
	case <-time.After(time.Second):
		t.Fatal("server never became active")
	}
	select {
	case <-client.WaitUntilActive():
	case <-time.After(time.Second):
		t.Fatal("client never became active")
	}

	pingRequest := &wire.PingRequest{ChainID: [wire.UInt256Length]byte{1}}

	// -- Concurrent RPCs in both directions over the same connection.
	const callers = 8
	results := make(chan error, callers)
	for i := 0; i < callers; i++ {
		go func() {
			_, err := client.SendRPC(context.Background(), byte(wire.ClusterOpPing), pingRequest)
			results <- err
		}()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, err := server.SendRPC(ctx, byte(wire.ClusterOpPing), pingRequest)
			results <- err
		}()
	}
	for i := 0; i < 2*callers; i++ {
		if err := <-results; err != nil {
			t.Fatalf("bidirectional rpc failed: %v", err)
		}
	}

	// -- Timeout with a late response: the connection must survive both.
	slow.Store(true)
	slowCtx, slowCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer slowCancel()
	_, err := client.SendRPC(slowCtx, byte(wire.ClusterOpPing), pingRequest)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
	if client.IsClosed() || server.IsClosed() {
		t.Fatal("connection closed by rpc timeout")
	}
	// The late PONG for the timed-out call is dropped; the next RPC still works.
	freshCtx, freshCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer freshCancel()
	resp, err := client.SendRPC(freshCtx, byte(wire.ClusterOpPing), pingRequest)
	if err != nil {
		t.Fatalf("rpc after timeout failed: %v", err)
	}
	if _, ok := resp.(*wire.PongResponse); !ok {
		t.Fatalf("expected *PongResponse, got %T", resp)
	}
	if client.IsClosed() {
		t.Fatal("connection closed after late response")
	}

	// -- Close: pending RPCs are aborted; the peer observes clean EOF.
	inflight := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// The slow handler keeps this RPC in flight at Close time.
		_, err := client.SendRPC(ctx, byte(wire.ClusterOpPing), pingRequest)
		inflight <- err
	}()
	time.Sleep(50 * time.Millisecond) // let the request reach the server
	client.Close()

	select {
	case err := <-inflight:
		if err != ErrConnectionClosed {
			t.Fatalf("pending rpc at close: expected ErrConnectionClosed, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pending rpc not aborted by Close")
	}
	select {
	case <-server.WaitUntilClosed():
	case <-time.After(2 * time.Second):
		t.Fatal("server did not observe client close")
	}
	if server.pendingLen() != 0 {
		t.Fatalf("server pending rpcs remain: %d", server.pendingLen())
	}
	waitForGoroutines(t, baseline)
}

// mustSerialize is a test helper for payload serialization.
func mustSerialize(t *testing.T, v wire.SSZMessage) []byte {
	t.Helper()
	b, err := v.MarshalSSZ()
	if err != nil {
		t.Fatalf("serialize %T: %v", v, err)
	}
	return b
}

// -- TCP transport write deadline ----------------------------------------------
//
// geth bounded-write model: every WriteFrame arms a per-frame write deadline
// so a peer that stops reading cannot block the writer (and writeMu) forever.

// newPipeTransport returns a TCP transport over net.Pipe, which honors
// deadlines and is unbuffered: a write with no reader blocks until deadline.
func newPipeTransport(t *testing.T, c net.Conn) FrameTransport {
	t.Helper()
	return NewTCPTransport(c,
		func(r io.Reader) (*wire.ClusterFrame, error) {
			return wire.ReadClusterFrame(r, wire.MaxFramePayloadLen)
		},
		wire.WriteClusterFrame)
}

// TestTCPTransport_WriteFrameStalledPeer: with nobody reading the peer end,
// WriteFrame must return a deadline error within a bounded time instead of
// blocking indefinitely.
func TestTCPTransport_WriteFrameStalledPeer(t *testing.T) {
	orig := frameWriteTimeout
	frameWriteTimeout = 50 * time.Millisecond
	defer func() { frameWriteTimeout = orig }()

	local, peer := net.Pipe()
	defer local.Close()
	defer peer.Close()
	tr := newPipeTransport(t, local)

	start := time.Now()
	err := tr.WriteFrame(&wire.ClusterFrame{Opcode: 1, RPCID: 1, Payload: []byte("ping")})
	if err == nil {
		t.Fatal("expected deadline error, got nil")
	}
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		var ne net.Error
		if !errors.As(err, &ne) || !ne.Timeout() {
			t.Fatalf("expected deadline-exceeded error, got %v", err)
		}
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("write unblocked too late: %v", elapsed)
	}
}

// TestTCPTransport_WriteFrameClearsDeadline: a successful WriteFrame clears
// the write deadline, so it must not stay armed on the conn and break later
// raw writes once it expires.
func TestTCPTransport_WriteFrameClearsDeadline(t *testing.T) {
	orig := frameWriteTimeout
	frameWriteTimeout = 50 * time.Millisecond
	defer func() { frameWriteTimeout = orig }()

	local, peer := net.Pipe()
	defer local.Close()
	defer peer.Close()
	go io.Copy(io.Discard, peer) // drain so writes complete
	tr := newPipeTransport(t, local)

	if err := tr.WriteFrame(&wire.ClusterFrame{Opcode: 1, RPCID: 1, Payload: []byte("ping")}); err != nil {
		t.Fatalf("write frame: %v", err)
	}
	time.Sleep(100 * time.Millisecond) // the armed deadline would be past due now

	// If the deadline were still armed, this raw write would fail with a
	// timeout; with it cleared it must succeed.
	if _, err := local.Write([]byte("raw")); err != nil {
		t.Fatalf("raw write after deadline expiry (deadline not cleared?): %v", err)
	}
}
