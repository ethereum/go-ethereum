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

// Package conn provides the generic RPC engine used by cluster connection
// implementations.
package conn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/wire"
)

// TypedHandler processes a deserialized request and returns a deserialized
// response. The framework handles payload serialization/deserialization.
// RPC handlers must return a non-nil response. Business rejections belong in
// the response (for example, ErrorCode), with a nil error. A non-nil error or
// panic is fatal to the connection. Non-RPC handlers may return a nil response.
// Handlers run concurrently, without an execution-order guarantee, and are
// neither canceled nor awaited by Close.
type TypedHandler func(req wire.SSZMessage) (resp wire.SSZMessage, err error)

// OpSerializer describes how to deserialize a request and serialize a response
// for a specific opcode.
type OpSerializer struct {
	NewRequest     func() wire.SSZMessage
	NewResponse    func() wire.SSZMessage
	ResponseOpCode byte
}

// sszPointer constrains a pointer to a message type to implement wire.SSZMessage.
type sszPointer[T any] interface {
	*T
	wire.SSZMessage
}

// OpSerializerFor creates an OpSerializer for request type R and response type
// S, with the response opcode set to respOp.
func OpSerializerFor[R, S any, RP sszPointer[R], SP sszPointer[S]](respOp byte) *OpSerializer {
	return &OpSerializer{
		NewRequest:     func() wire.SSZMessage { return RP(new(R)) },
		NewResponse:    func() wire.SSZMessage { return SP(new(S)) },
		ResponseOpCode: respOp,
	}
}

// pendingRPC represents an in-flight RPC call waiting for its response.
type pendingRPC struct {
	result     chan wire.SSZMessage // cap 1
	expectedOp byte
}

// ConnectionState is the connection lifecycle state.
type ConnectionState int32

const (
	ConnectionStateConnecting ConnectionState = iota
	ConnectionStateActive
	ConnectionStateClosed
)

// BaseConn is the shared RPC engine used by cluster connection
// implementations.
//
// Concurrency model:
//
//   - Reader: a single readerLoop owns transport.ReadFrame() and inbound
//     frame processing. No other goroutine reads the transport.
//
//   - Writer: there is no writer loop. All transport.WriteFrame() calls are
//     serialized by writeMu. SendRPC allocates rpc_id while holding writeMu,
//     preserving the ordering between rpc_id allocation and serialized writes.
//
//   - Protocol state (mu): mu protects mutable connection state shared across goroutines:
//     state, nextRPCID, and pending.
//
// Lock rules:
//
//   - writeMu may be acquired before mu.
//   - mu must never be held while acquiring writeMu.
//   - mu must never be held during network I/O.
//   - A failed write calls shutdown with writeMu held, so neither shutdown nor
//     FrameTransport.Close may acquire writeMu.
//
// Invariants:
//
//   - Never hold writeMu while waiting for an RPC response.
//   - Each RPC caller removes its pending entry before returning.
//   - Late and duplicate responses are ignored.
type BaseConn struct {
	// transport is the frame I/O backend. All writes go through
	// writeMu; all reads through readerLoop.
	transport FrameTransport

	// Configuration. Immutable after construction.
	requestSerializers  map[byte]*OpSerializer
	responseSerializers map[byte]*OpSerializer
	typedHandlers       map[byte]TypedHandler
	nonRPCOps           map[byte]struct{}
	forwarder           func(*wire.ClusterFrame) bool

	// -- Protocol state (mu) --
	mu        sync.RWMutex
	state     ConnectionState
	pending   map[uint64]*pendingRPC
	nextRPCID uint64

	// Owned by readerLoop. Zero value is the "no request seen yet" sentinel:
	// rpc_id=0 is reserved for non-RPC (fire-and-forget) commands (see
	// SendCommand), so every valid inbound RPC request has rpc_id >= 1 and
	// passes the monotonic check against the zero-value state.
	peerRPCID uint64

	// -- Frame send serialization (writeMu) --
	writeMu sync.Mutex

	// -- Synchronization primitives --
	shutdownOnce sync.Once
	// activeChan signals that startup has completed. The connection may be ACTIVE
	// or already closed; callers must check IsActive().
	activeChan chan struct{}
	closedChan chan struct{} // closed during shutdown

	log log.Logger
}

// NewBaseConn creates a BaseConn from the supplied configuration.
// The caller is responsible for calling Start().
func NewBaseConn(cfg Config) *BaseConn {
	cfg.validate()

	logger := cfg.Logger
	if logger == nil {
		logger = log.Root()
	}

	// Keep request and response opcodes separate. A request without a local
	// handler is outbound-only and must never be accepted as a response.
	requestSerializers := make(map[byte]*OpSerializer, len(cfg.Serializers))
	responseSerializers := make(map[byte]*OpSerializer, len(cfg.Serializers))
	for opcode, ser := range cfg.Serializers {
		requestSerializers[opcode] = ser

		if _, nonRPC := cfg.NonRPCOps[opcode]; !nonRPC {
			responseSerializers[ser.ResponseOpCode] = ser
		}
	}

	// Copy maps so the caller's maps are not shared.
	nonRPCOps := make(map[byte]struct{}, len(cfg.NonRPCOps))
	for op := range cfg.NonRPCOps {
		nonRPCOps[op] = struct{}{}
	}
	handlers := make(map[byte]TypedHandler, len(cfg.Handlers))
	for op, h := range cfg.Handlers {
		handlers[op] = h
	}

	rc := &BaseConn{
		transport:           cfg.Transport,
		typedHandlers:       handlers,
		requestSerializers:  requestSerializers,
		responseSerializers: responseSerializers,
		nonRPCOps:           nonRPCOps,
		forwarder:           cfg.Forwarder,
		activeChan:          make(chan struct{}),
		closedChan:          make(chan struct{}),
		pending:             make(map[uint64]*pendingRPC),
		state:               ConnectionStateConnecting,
		log:                 logger,
	}
	return rc
}

// -- Public API ---------------------------------------------------------------

// Start transitions the connection to ACTIVE and starts the reader loop.
// If the connection is already started or closed, Start is a no-op.
func (c *BaseConn) Start() {
	c.mu.Lock()
	if c.state != ConnectionStateConnecting {
		c.mu.Unlock()
		return
	}
	c.state = ConnectionStateActive
	close(c.activeChan)
	c.mu.Unlock()

	go c.readerLoop()
}

// Close closes the connection and wakes all pending RPCs.
// It does not cancel or wait for handler goroutines.
func (c *BaseConn) Close() {
	c.shutdown(nil)
}

// SendRPC serializes a request with ClusterPeerID zero and waits for its response.
// Serialization errors are returned without closing the connection.
// ctx controls response waiting; it does not interrupt lock waits or writes.
func (c *BaseConn) SendRPC(ctx context.Context, opcode byte, req wire.SSZMessage) (wire.SSZMessage, error) {
	ser, ok := c.requestSerializers[opcode]
	_, nonRPC := c.nonRPCOps[opcode]
	if !ok || nonRPC {
		return nil, fmt.Errorf("unknown rpc request opcode 0x%x", opcode)
	}
	if req == nil {
		return nil, fmt.Errorf("nil rpc request")
	}
	payload, err := req.MarshalSSZ()
	if err != nil {
		return nil, fmt.Errorf("serialize rpc request: %w", err)
	}
	call := &pendingRPC{
		result:     make(chan wire.SSZMessage, 1),
		expectedOp: ser.ResponseOpCode,
	}

	c.writeMu.Lock()

	if err := ctx.Err(); err != nil {
		c.writeMu.Unlock()
		return nil, err
	}
	c.mu.Lock()
	if err := c.checkActiveLocked(); err != nil {
		c.mu.Unlock()
		c.writeMu.Unlock()
		return nil, err
	}

	c.nextRPCID++
	rpcID := c.nextRPCID
	c.pending[rpcID] = call
	c.mu.Unlock()

	err = c.writeFrameLocked(&wire.ClusterFrame{
		Opcode:  opcode,
		RPCID:   rpcID,
		Payload: payload,
	})
	c.writeMu.Unlock()

	defer func() {
		c.mu.Lock()
		delete(c.pending, rpcID)
		c.mu.Unlock()
	}()

	if err != nil {
		return nil, err
	}

	select {
	case resp := <-call.result:
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-c.closedChan:
		// Preserve a response delivered before the connection closed.
		select {
		case resp := <-call.result:
			return resp, nil
		default:
			return nil, ErrConnectionClosed
		}
	}
}

// SendCommand serializes a fire-and-forget command with ClusterPeerID zero.
// The opcode must be registered in Config.NonRPCOps.
// Serialization errors are returned without closing the connection.
func (c *BaseConn) SendCommand(opcode byte, cmd wire.SSZMessage) error {
	if _, ok := c.nonRPCOps[opcode]; !ok {
		return fmt.Errorf("unknown command opcode 0x%x", opcode)
	}
	if cmd == nil {
		return fmt.Errorf("nil command")
	}
	payload, err := cmd.MarshalSSZ()
	if err != nil {
		return fmt.Errorf("serialize command: %w", err)
	}
	frame := &wire.ClusterFrame{
		Opcode:  opcode,
		RPCID:   0,
		Payload: payload,
	}
	return c.writeFrame(frame)
}

// WriteFrame writes a pre-built frame to the transport, serialized by writeMu
// together with every other outbound frame on this connection. It is the
// low-level "write a complete frame verbatim" entry: unlike SendRPC/SendCommand
// it does not allocate an rpc_id or construct a new frame.
//
// It is exposed so connections that route already-constructed frames from other
// connections can reuse this connection's writeMu for physical serialization
// (e.g. forwarding virtual peer traffic over a Master↔slave connection).
func (c *BaseConn) WriteFrame(f *wire.ClusterFrame) error {
	return c.writeFrame(f)
}

// -- Query methods -----------------------------------------------------------

// RemoteAddr returns the transport's remote address.
func (c *BaseConn) RemoteAddr() string { return c.transport.RemoteAddr() }

// WaitUntilActive returns a channel closed when startup completes.
// The connection may be ACTIVE or already closed; use IsActive() to check.
func (c *BaseConn) WaitUntilActive() <-chan struct{} { return c.activeChan }

// WaitUntilClosed returns a channel closed when shutdown begins.
func (c *BaseConn) WaitUntilClosed() <-chan struct{} { return c.closedChan }

// Logger returns the connection logger.
func (c *BaseConn) Logger() log.Logger { return c.log }

// State returns the current connection state.
func (c *BaseConn) State() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// IsActive reports whether the connection is active.
func (c *BaseConn) IsActive() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == ConnectionStateActive
}

// IsClosed reports whether the connection is closed.
func (c *BaseConn) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state == ConnectionStateClosed
}

// -- Internal helpers --------------------------------------------------------

// pendingLen returns the number of in-flight RPCs. Used by tests.
func (c *BaseConn) pendingLen() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.pending)
}

// checkActiveLocked maps the connection state to the error an in-flight write
// should return. It must be called with mu held: Active is nil, Closed is
// ErrConnectionClosed, and Connecting is ErrNotActive.
func (c *BaseConn) checkActiveLocked() error {
	switch c.state {
	case ConnectionStateActive:
		return nil
	case ConnectionStateClosed:
		return ErrConnectionClosed
	default:
		return ErrNotActive
	}
}

// -- Write path ---------------------------------------------------------------
//
// writeFrame writes a pre-built frame, serialized with all other writes.
func (c *BaseConn) writeFrame(f *wire.ClusterFrame) error {
	c.writeMu.Lock()
	c.mu.RLock()
	if err := c.checkActiveLocked(); err != nil {
		c.mu.RUnlock()
		c.writeMu.Unlock()
		return err
	}
	c.mu.RUnlock()

	err := c.writeFrameLocked(f)
	c.writeMu.Unlock()

	return err
}

// writeFrameLocked requires writeMu. Except for validation errors, a failed or
// panicked write shuts down the connection before releasing writeMu to prevent
// further writes.
func (c *BaseConn) writeFrameLocked(f *wire.ClusterFrame) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("transport write panic: %v", r)
		}
		if err != nil && !errors.Is(err, wire.ErrInvalidClusterFrame) {
			c.shutdown(fmt.Errorf("write frame: %w", err))
			err = fmt.Errorf("%w: %w", ErrConnectionClosed, err)
		}
	}()
	return c.transport.WriteFrame(f)
}

// -- readerLoop --------------------------------------------------------------

// readerLoop reads frames from the transport and dispatches them.
// Read errors trigger shutdown.
func (c *BaseConn) readerLoop() {
	for {
		frame, err := c.transport.ReadFrame()
		if err != nil {
			c.shutdown(normalizeReadErr(err))
			return
		}
		c.handleFrameSafely(frame)
	}
}

// -- handleFrame -------------------------------------------------------------

// handleFrameSafely runs handleFrame with panic isolation: any panic from
// frame processing (the forwarder, response deserialization, serializer
// callbacks, or future extension points) is converted into a connection
// shutdown instead of crashing the process.
func (c *BaseConn) handleFrameSafely(frame *wire.ClusterFrame) {
	defer func() {
		if recovered := recover(); recovered != nil {
			c.log.Error("frame processing panic",
				"opcode", frame.Opcode, "rpcid", frame.RPCID,
				"panic", recovered, "stack", string(debug.Stack()))
			c.shutdown(fmt.Errorf("frame processing panic (opcode=0x%x rpc_id %d): %v",
				frame.Opcode, frame.RPCID, recovered))
		}
	}()
	c.handleFrame(frame)
}

func (c *BaseConn) handleFrame(frame *wire.ClusterFrame) {
	// A closed transport can still have complete frames in its read buffer.
	if c.IsClosed() {
		return
	}

	// Run the forwarder first. A true return means it consumed the frame, so
	// normal dispatch is skipped. The forwarder may call c.Close() directly to
	// shut the connection down; Close does not wait for the reader goroutine.
	if fwd := c.forwarder; fwd != nil && fwd(frame) {
		return
	}

	if ser := c.responseSerializers[frame.Opcode]; ser != nil {
		c.handleResponse(frame, ser)
		return
	}
	if handler, ok := c.typedHandlers[frame.Opcode]; ok {
		_, isNonRPC := c.nonRPCOps[frame.Opcode]
		c.handleRequest(frame, handler, c.requestSerializers[frame.Opcode], isNonRPC)
		return
	}
	c.shutdown(fmt.Errorf("unsupported inbound opcode 0x%x", frame.Opcode))
}

// -- handleResponse (inbound response matching) -------------------------------

// handleResponse matches an inbound response to a pending RPC.
// Late and duplicate responses are ignored.
func (c *BaseConn) handleResponse(frame *wire.ClusterFrame, ser *OpSerializer) {
	c.mu.Lock()
	// A missing call with an allocated ID is a late or duplicate response.
	allocated := frame.RPCID > 0 && frame.RPCID <= c.nextRPCID
	call := c.pending[frame.RPCID]
	if call != nil && frame.Opcode != call.expectedOp {
		c.mu.Unlock()
		c.shutdown(fmt.Errorf("unexpected response opcode 0x%x for rpc %d (want 0x%x)",
			frame.Opcode, frame.RPCID, call.expectedOp))
		return
	}
	delete(c.pending, frame.RPCID)
	c.mu.Unlock()

	if call == nil {
		// An allocated ID with no pending call is a late or duplicate response;
		// ignore it. An ID that was never allocated is a protocol error.
		if !allocated {
			c.shutdown(fmt.Errorf("unexpected rpc response %d", frame.RPCID))
		}
		return
	}

	resp := ser.NewResponse()
	if err := resp.UnmarshalSSZ(frame.Payload); err != nil {
		c.log.Warn("malformed response payload", "opcode", frame.Opcode, "err", err)
		c.shutdown(fmt.Errorf("malformed response payload for opcode 0x%x: %w", frame.Opcode, err))
		return
	}

	call.result <- resp
}

// -- handleRequest (inbound request dispatch) --------------------------------

func (c *BaseConn) handleRequest(frame *wire.ClusterFrame, handler TypedHandler, ser *OpSerializer, isNonRPC bool) {
	if isNonRPC && frame.RPCID != 0 {
		c.log.Warn("non-rpc command with non-zero rpc_id", "opcode", frame.Opcode, "rpcid", frame.RPCID)
		c.shutdown(fmt.Errorf("non-rpc command with rpc id %d", frame.RPCID))
		return
	}

	if !isNonRPC {
		// defaultValidateRPCID is owned exclusively by readerLoop, so no lock needed.
		ok := c.defaultValidateRPCID(frame.RPCID)
		if !ok {
			c.log.Warn("incorrect rpc request id sequence", "rpcid", frame.RPCID)
			c.shutdown(fmt.Errorf("incorrect rpc request id sequence"))
			return
		}
	}

	// TODO: Bound concurrent handler dispatches to prevent unbounded goroutines.
	go c.dispatch(frame, handler, ser)
}

// -- dispatch (handler execution + response write) ----------------------------

func (c *BaseConn) dispatch(frame *wire.ClusterFrame, handler TypedHandler, ser *OpSerializer) {
	defer func() {
		if recovered := recover(); recovered != nil {
			c.shutdown(fmt.Errorf("handler panic (opcode=0x%x): %v", frame.Opcode, recovered))
		}
	}()

	req := ser.NewRequest()
	if err := req.UnmarshalSSZ(frame.Payload); err != nil {
		c.shutdown(fmt.Errorf("deserialize failed: %w", err))
		return
	}
	resp, err := handler(req)
	if err != nil {
		c.log.Error("request handler failed", "opcode", frame.Opcode, "err", err)
		c.shutdown(err)
		return
	}

	// Fire-and-forget commands (rpc_id=0) get no response frame.
	if frame.RPCID == 0 {
		return
	}
	respPayload, err := resp.MarshalSSZ()
	if err != nil {
		c.shutdown(fmt.Errorf("serialize response failed: %w", err))
		return
	}

	respFrame := &wire.ClusterFrame{
		ClusterPeerID: frame.ClusterPeerID,
		Opcode:        ser.ResponseOpCode,
		RPCID:         frame.RPCID,
		Payload:       respPayload,
	}

	if err := c.writeFrame(respFrame); err != nil {
		if errors.Is(err, wire.ErrInvalidClusterFrame) {
			// writeFrame leaves the connection open on validation errors so callers
			// can correct invalid outbound frames. This response is constructed
			// internally, and we cannot complete the peer's RPC, so close the
			// connection rather than leave it waiting for a response.
			c.shutdown(fmt.Errorf("invalid response frame rpc=%d: %w", frame.RPCID, err))
		}
		c.log.Debug("response write failed, connection shutting down", "rpcid", frame.RPCID, "err", err)
	}
}

// -- Shutdown -----------------------------------------------------------------

// shutdown does not wait for the reader or handler goroutines, but it calls
// transport.Close synchronously. sync.Once guarantees one execution.
func (c *BaseConn) shutdown(cause error) {
	var didClose bool
	var closeErr error
	c.shutdownOnce.Do(func() {
		didClose = true
		c.mu.Lock()
		wasConnecting := c.state == ConnectionStateConnecting
		if wasConnecting {
			close(c.activeChan)
		}

		c.state = ConnectionStateClosed
		close(c.closedChan)

		clear(c.pending)
		c.mu.Unlock()

		// Close transport to interrupt blocked I/O.
		// Writes already accepted by the transport may still complete.
		closeErr = c.transport.Close()
	})
	// Finish shutdown before logging so a slow logger cannot delay RPCs or
	// transport closure. Logging outside mu and sync.Once also lets a logger
	// safely call State or Close.
	if didClose {
		if cause != nil {
			c.log.Error("connection closed with error", "err", cause)
		}
		if closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			c.log.Warn("transport close failed", "err", closeErr)
		}
	}
}

// -- RPC ID validation (default) ---------------------------------------------

// defaultValidateRPCID preserves the monotonic request-ID rule inherited from
// pyquarkchain's AbstractConnection.validate_and_update_peer_rpc_id in quarkchain/protocol.py.
// It validates arrival order only; dispatch goroutines may execute out of order.
func (c *BaseConn) defaultValidateRPCID(rpcID uint64) bool {
	if rpcID <= c.peerRPCID {
		return false
	}
	c.peerRPCID = rpcID
	return true
}

// -- Helpers ------------------------------------------------------------------

func normalizeReadErr(err error) error {
	if errors.Is(err, io.EOF) {
		return nil // clean close — not an error
	}
	return err
}
