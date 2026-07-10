// Copyright 2026-2027, QuarkChain.

package slave

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/cluster/wire"
	"github.com/ethereum/go-ethereum/qkc/serialize"
)

// serializeBytes serializes a wire message using the qkc/serialize package.
func serializeBytes(v any) ([]byte, error) {
	return serialize.SerializeToBytes(v)
}

// deserializeBytes deserializes a wire message from payload bytes.
func deserializeBytes(p []byte, v any) error {
	return serialize.Deserialize(serialize.NewByteBuffer(p), v)
}

// TypedHandler processes a deserialized request and returns a deserialized
// response. The framework handles payload serialization/deserialization.
type TypedHandler func(req any) (resp any, err error)

// OpSerializer describes how to deserialize a request and serialize a response
// for a specific opcode. It mirrors Python's op_ser_map entries.
type OpSerializer struct {
	NewRequest     func() any
	Deserialize    func([]byte, any) error
	Serialize      func(any) ([]byte, error)
	ResponseOpCode byte // optional: if non-zero, used as response opcode
}

// OpSerializerFor creates an OpSerializer for wire types R (request) and S (response).
func OpSerializerFor[R, S any]() *OpSerializer {
	return &OpSerializer{
		NewRequest: func() any { return new(R) },
		Deserialize: func(p []byte, v any) error {
			return deserializeBytes(p, v)
		},
		Serialize: func(v any) ([]byte, error) {
			return serializeBytes(v)
		},
	}
}

// ConnectionState mirrors Python's protocol.ConnectionState.
type ConnectionState int32

const (
	ConnectionStateConnecting ConnectionState = iota
	ConnectionStateActive
	ConnectionStateClosed
)

// ── transport: pure I/O layer ─────────────────────────────────────────────────

// transport wraps a net.Conn with metadata-aware frame read/write.
// writeMu serializes writes because bufio.Writer is not goroutine-safe and
// both SendRPC (any goroutine) and readLoop handler goroutines write frames.
type transport struct {
	conn net.Conn
	r    *bufio.Reader
	w    *bufio.Writer

	writeMu sync.Mutex

	readFrameFn  func(io.Reader) (*wire.Frame, error)
	writeFrameFn func(io.Writer, *wire.Frame) error

	remoteAddr string
}

func newTransport(
	conn net.Conn,
	readFrame func(io.Reader) (*wire.Frame, error),
	writeFrame func(io.Writer, *wire.Frame) error,
) *transport {
	return &transport{
		conn:         conn,
		r:            bufio.NewReader(conn),
		w:            bufio.NewWriter(conn),
		readFrameFn:  readFrame,
		writeFrameFn: writeFrame,
		remoteAddr:   conn.RemoteAddr().String(),
	}
}

func (t *transport) readFrame() (*wire.Frame, error) {
	return t.readFrameFn(t.r)
}

func (t *transport) writeFrame(f *wire.Frame) error {
	t.writeMu.Lock()
	defer t.writeMu.Unlock()

	if err := t.writeFrameFn(t.w, f); err != nil {
		return fmt.Errorf("write frame: %w", err)
	}
	if err := t.w.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	return nil
}

func (t *transport) close() error {
	return t.conn.Close()
}

func (t *transport) RemoteAddr() string {
	return t.remoteAddr
}

// ── rpcConn: RPC protocol engine ─────────────────────────────────────────────

// rpcResult is the value delivered over a pending RPC response channel.
type rpcResult struct {
	frame *wire.Frame
	err   error
}

// rpcConn is the shared RPC engine used by XshardConn (and later MasterConn).
// It handles lifecycle, handler/serializer registration, readLoop dispatch,
// RPC request/response matching, and monotonic RPC ID validation.
//
// The forwarder hook is an extension point for MasterConn to route peer traffic
// to PeerShardConn. For XshardConn it remains nil.
//
// Lock ordering (must be maintained to avoid deadlocks):
//
//	closeMu → pendingMu     (SendRPCMeta, Close)
//	closeMu → stateMu       (Close)
//
// pendingMu and stateMu are never held together; readLoop only holds pendingMu.
type rpcConn struct {
	*transport

	stateMu    sync.Mutex
	state      ConnectionState
	activeChan chan struct{}
	closedChan chan struct{}

	errChan   chan error
	startOnce sync.Once

	handlersMu    sync.RWMutex
	typedHandlers map[byte]TypedHandler

	serializersMu sync.RWMutex
	serializers   map[byte]*OpSerializer

	nonRPCOps map[byte]struct{}

	pendingMu sync.Mutex
	pending   map[uint64]chan rpcResult

	nextRPCID uint64

	// peerRPCID tracks the most recent inbound RPC ID for monotonic validation.
	// Initialized to -1 (like Python) so the first valid rpc_id must be >= 1.
	peerRPCID   int64
	peerRPCIDMu sync.Mutex

	// validateRPCID is called by readLoop for every RPC request frame.
	// Default: simple global monotonic validation.
	// MasterConn replaces with per-peer tracking.
	validateRPCID func(clusterPeerID uint64, rpcID uint64) bool

	forwarder   func(*wire.Frame) bool
	forwarderMu sync.RWMutex

	closeMu sync.Mutex
	closed  bool

	log log.Logger
}

func newRPCConn(
	conn net.Conn,
	readFrame func(io.Reader) (*wire.Frame, error),
	writeFrame func(io.Writer, *wire.Frame) error,
	logger log.Logger,
) *rpcConn {
	if logger == nil {
		logger = log.Root()
	}
	rc := &rpcConn{
		transport:     newTransport(conn, readFrame, writeFrame),
		typedHandlers: make(map[byte]TypedHandler),
		serializers:   make(map[byte]*OpSerializer),
		pending:       make(map[uint64]chan rpcResult),
		peerRPCID:     -1,
		nonRPCOps:     make(map[byte]struct{}),
		state:         ConnectionStateConnecting,
		activeChan:    make(chan struct{}),
		closedChan:    make(chan struct{}),
		errChan:       make(chan error, 1),
		log:           logger,
	}
	rc.validateRPCID = rc.defaultValidateRPCID
	return rc
}

// Start transitions the connection to ACTIVE and launches the read loop.
func (c *rpcConn) Start() {
	c.startOnce.Do(func() {
		c.stateMu.Lock()
		c.state = ConnectionStateActive
		close(c.activeChan)
		c.stateMu.Unlock()
		go c.readLoop()
	})
}

// Close closes the connection and wakes all pending RPCs.
func (c *rpcConn) Close() error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil
	}
	c.closed = true
	c.closeMu.Unlock()

	c.stateMu.Lock()
	if c.state != ConnectionStateClosed {
		c.state = ConnectionStateClosed
		close(c.closedChan)
		// Wake up any goroutines waiting on WaitUntilActive().
		// Matches Python's finally block in active_and_loop_forever that sets active_event.
		select {
		case <-c.activeChan:
			// Already closed (Start was called)
		default:
			close(c.activeChan)
		}
	}
	c.stateMu.Unlock()

	c.pendingMu.Lock()
	for rpcID, ch := range c.pending {
		select {
		case ch <- rpcResult{err: ErrConnectionClosed}:
		default:
		}
		delete(c.pending, rpcID)
	}
	c.pendingMu.Unlock()

	return c.transport.close()
}

func (c *rpcConn) defaultValidateRPCID(clusterPeerID uint64, rpcID uint64) bool {
	c.peerRPCIDMu.Lock()
	defer c.peerRPCIDMu.Unlock()
	if int64(rpcID) <= c.peerRPCID {
		return false
	}
	c.peerRPCID = int64(rpcID)
	return true
}

// RegisterTypedHandlers registers opcode handlers. Nil handlers panic.
func (c *rpcConn) RegisterTypedHandlers(handlers map[byte]TypedHandler) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	for opcode, handler := range handlers {
		if handler == nil {
			panic("handler must not be nil")
		}
		c.typedHandlers[opcode] = handler
	}
}

// RegisterOpSerializers registers opcode serializers.
func (c *rpcConn) RegisterOpSerializers(serializers map[byte]*OpSerializer) {
	c.serializersMu.Lock()
	defer c.serializersMu.Unlock()
	for opcode, ser := range serializers {
		if ser == nil {
			panic("serializer must not be nil")
		}
		c.serializers[opcode] = ser
	}
}

// RegisterNonRPCOps marks opcodes as non-RPC (fire-and-forget), meaning they
// must have rpc_id == 0.
func (c *rpcConn) RegisterNonRPCOps(ops []byte) {
	c.handlersMu.Lock()
	defer c.handlersMu.Unlock()
	for _, op := range ops {
		c.nonRPCOps[op] = struct{}{}
	}
}

// SetForwarder installs a raw-frame forwarder hook. If it returns true the
// frame is consumed and readLoop continues without dispatching it.
func (c *rpcConn) SetForwarder(f func(*wire.Frame) bool) {
	c.forwarderMu.Lock()
	defer c.forwarderMu.Unlock()
	c.forwarder = f
}

// SendRPC sends a request with zero metadata and waits for the response.
// For connections that need metadata (e.g. MasterConn with 12-byte
// ClusterMetadata), use SendRPCMeta directly.
func (c *rpcConn) SendRPC(ctx context.Context, opcode byte, payload []byte) (*wire.Frame, error) {
	return c.SendRPCMeta(ctx, opcode, payload, wire.ClusterMetadata{})
}

// SendRPCMeta sends a request with the given metadata and waits for the response.
// XshardConn uses zero metadata (0-byte wire format).
// MasterConn uses ClusterMetadata{Branch, ClusterPeerID} (12-byte wire format).
func (c *rpcConn) SendRPCMeta(ctx context.Context, opcode byte, payload []byte, meta wire.ClusterMetadata) (*wire.Frame, error) {
	c.stateMu.Lock()
	state := c.state
	c.stateMu.Unlock()

	switch state {
	case ConnectionStateClosed:
		return nil, ErrConnectionClosed
	case ConnectionStateConnecting:
		return nil, ErrNotActive
	}

	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil, ErrConnectionClosed
	}

	rpcID := atomic.AddUint64(&c.nextRPCID, 1)
	respChan := make(chan rpcResult, 1)
	c.pendingMu.Lock()
	c.pending[rpcID] = respChan
	c.pendingMu.Unlock()
	c.closeMu.Unlock()

	defer func() {
		c.pendingMu.Lock()
		delete(c.pending, rpcID)
		c.pendingMu.Unlock()
	}()

	frame := &wire.Frame{
		Meta:    meta,
		Opcode:  opcode,
		RPCID:   rpcID,
		Payload: payload,
	}
	if err := c.transport.writeFrame(frame); err != nil {
		return nil, err
	}

	select {
	case res := <-respChan:
		if res.err != nil {
			return nil, res.err
		}
		if res.frame == nil {
			return nil, ErrConnectionClosed
		}
		return res.frame, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("rpc timeout: %w", ctx.Err())
	}
}

// ── Read loop ─────────────────────────────────────────────────────────────────

// readLoop reads frames until a fatal error, then closes the connection.
// Follows Python's protocol validation rules strictly.
func (c *rpcConn) readLoop() {
	defer c.Close()

	for {
		frame, err := c.transport.readFrame()
		if err != nil {
			select {
			case c.errChan <- err:
			default:
			}
			return
		}

		// Forwarder hook (extension point for MasterConn).
		c.forwarderMu.RLock()
		fwd := c.forwarder
		c.forwarderMu.RUnlock()
		if fwd != nil && fwd(frame) {
			continue
		}

		c.handlersMu.RLock()
		handler, isRequest := c.typedHandlers[frame.Opcode]
		_, isNonRPC := c.nonRPCOps[frame.Opcode]
		c.handlersMu.RUnlock()

		c.serializersMu.RLock()
		ser := c.serializers[frame.Opcode]
		c.serializersMu.RUnlock()

		// No handler: could be a pending RPC response or unsupported opcode.
		if !isRequest {
			if frame.RPCID != 0 {
				c.pendingMu.Lock()
				if ch, ok := c.pending[frame.RPCID]; ok {
					delete(c.pending, frame.RPCID)
					c.pendingMu.Unlock()
					select {
					case ch <- rpcResult{frame: frame}:
					default:
						c.log.Warn("response channel full", "rpcid", frame.RPCID)
					}
					continue
				}
				c.pendingMu.Unlock()
				// INTENTIONAL DEVIATION FROM PYTHON: Python closes connection on
				// unexpected RPC response (rpc_id not in rpc_future_map). Go keeps
				// connection open and logs error. This is more robust for distributed
				// systems where late/duplicate responses are normal after timeout.
				// If strict Python compatibility is needed, change to: return
				c.log.Error("unexpected rpc response (rpc_id not in pending map)",
					"rpcid", frame.RPCID, "opcode", frame.Opcode)
				continue
			}
			c.log.Warn("unsupported opcode", "opcode", frame.Opcode)
			return
		}

		if ser == nil {
			c.log.Warn("handler without serializer", "opcode", frame.Opcode)
			return
		}

		if isNonRPC && frame.RPCID != 0 {
			c.log.Warn("non-rpc command with non-zero rpc_id", "opcode", frame.Opcode, "rpcid", frame.RPCID)
			return
		}

		if !isNonRPC {
			if !c.validateRPCID(frame.Meta.ClusterPeerID, frame.RPCID) {
				c.log.Warn("incorrect rpc request id sequence", "rpcid", frame.RPCID)
				return
			}
		}

		go c.dispatch(frame, handler, ser)
	}
}

func (c *rpcConn) dispatch(frame *wire.Frame, handler TypedHandler, ser *OpSerializer) {
	defer func() {
		if r := recover(); r != nil {
			c.log.Error("handler panic", "opcode", frame.Opcode, "panic", r)
			c.Close()
		}
	}()

	req := ser.NewRequest()
	if err := ser.Deserialize(frame.Payload, req); err != nil {
		c.log.Error("deserialize failed", "opcode", frame.Opcode, "err", err)
		c.Close()
		return
	}

	resp, err := handler(req)
	if err != nil {
		// NOTE: All handler errors close the connection. This matches Python's
		// close_with_error pattern and is intentional for protocol safety.
		// The QuarkChain cluster protocol treats handler errors as fatal because
		// there's no error response mechanism — the only way to signal failure
		// is to close the connection. If recoverable errors are needed in the
		// future, the protocol would need to be extended with error responses.
		c.log.Error("handler error", "opcode", frame.Opcode, "err", err)
		c.Close()
		return
	}

	if frame.RPCID == 0 {
		return // non-RPC: no response
	}

	respPayload, err := ser.Serialize(resp)
	if err != nil {
		c.log.Error("serialize response failed", "opcode", frame.Opcode, "err", err)
		c.Close()
		return
	}
	respOp := frame.Opcode + 1
	if ser.ResponseOpCode != 0 {
		respOp = ser.ResponseOpCode
	}
	respFrame := &wire.Frame{
		Meta:    frame.Meta,
		Opcode:  respOp,
		RPCID:   frame.RPCID,
		Payload: respPayload,
	}
	if err := c.transport.writeFrame(respFrame); err != nil {
		c.log.Error("write response failed", "opcode", respFrame.Opcode, "err", err)
		c.Close()
	}
}

// ── Query helpers ─────────────────────────────────────────────────────────────

func (c *rpcConn) Error() <-chan error              { return c.errChan }
func (c *rpcConn) RemoteAddr() string               { return c.transport.RemoteAddr() }
func (c *rpcConn) WaitUntilActive() <-chan struct{} { return c.activeChan }
func (c *rpcConn) WaitUntilClosed() <-chan struct{} { return c.closedChan }

func (c *rpcConn) State() ConnectionState {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.state
}

func (c *rpcConn) IsActive() bool {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.state == ConnectionStateActive
}

func (c *rpcConn) IsClosed() bool {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	return c.state == ConnectionStateClosed
}

func (c *rpcConn) Closed() bool {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	return c.closed
}
