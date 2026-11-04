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

package slaveconn

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/wire"
)

// XshardPool manages the local slave's cross-shard connections to other slaves,
// indexed by each remote slave's chain ID. Connections and chain IDs remain
// indexed until the pool closes. Concurrent dials may establish multiple
// connections to the same slave.
//
// TODO: Define xshard TCP disconnect handling, including pool cleanup and whether
// to stop the cluster or reconnect. Closed connections currently remain indexed.
type XshardPool struct {
	// mu guards all mutable pool state below. Registration
	// (registerConnection) and Close are multi-map transactions over these
	// fields plus closed, so they must stay in one critical section; do not
	// split locking per field.
	mu          sync.RWMutex
	conns       map[uint32][]*XshardConn
	connections map[*XshardConn]struct{}

	chainIDs     map[uint32]struct{}
	localChainID uint32
	// Only connections to slaves with configured chain IDs enter the routing index.
	configuredChainIDs map[uint32]struct{}
	maxPayloadSize     uint32        // 0 uses the maximum frame payload length.
	handler            XshardHandler // Serves cross-shard requests from other slaves.
	closed             bool
	log                log.Logger
}

// Public API

// NewXshardPool creates the local slave's pool of cross-shard connections.
// localChainID identifies the chain hosted by this slave; configuredChainIDs
// lists the chain IDs hosted by the cluster's slaves. A zero maxPayloadSize
// uses the protocol's maximum payload size. handler serves cross-shard requests
// from other slaves and must not be nil.
func NewXshardPool(localChainID uint32, configuredChainIDs []uint32, maxPayloadSize uint32, handler XshardHandler, logger log.Logger) (*XshardPool, error) {
	if handler == nil {
		return nil, errors.New("xshard handler must not be nil")
	}
	if len(configuredChainIDs) == 0 {
		return nil, errors.New("configured chain IDs must not be empty")
	}
	if maxPayloadSize > wire.MaxFramePayloadLen {
		return nil, fmt.Errorf("max payload size %d exceeds protocol limit %d", maxPayloadSize, wire.MaxFramePayloadLen)
	}
	if logger == nil {
		logger = log.Root()
	}
	clusterSet := make(map[uint32]struct{}, len(configuredChainIDs))
	for _, id := range configuredChainIDs {
		clusterSet[id] = struct{}{}
	}
	return &XshardPool{
		conns:              make(map[uint32][]*XshardConn),
		connections:        make(map[*XshardConn]struct{}),
		chainIDs:           make(map[uint32]struct{}),
		localChainID:       localChainID,
		handler:            handler,
		configuredChainIDs: clusterSet,
		maxPayloadSize:     maxPayloadSize,
		log:                logger,
	}, nil
}

// DialToSlave establishes an outbound cross-shard connection to the slave
// described by slaveInfo. Its ChainID identifies the chain hosted by that slave.
func (p *XshardPool) DialToSlave(ctx context.Context, slaveInfo wire.SlaveInfo) error {
	if _, ok := p.configuredChainIDs[slaveInfo.ChainID]; !ok {
		return fmt.Errorf("slave chain ID %d is not configured", slaveInfo.ChainID)
	}
	if p.knownSlave(slaveInfo.ChainID) {
		p.log.Info("outbound xshard connection skipped: slave already known", "remote_chain_id", slaveInfo.ChainID)
		return nil
	}

	addr := net.JoinHostPort(string(slaveInfo.Host), strconv.Itoa(int(slaveInfo.Port)))
	// Dialer.Timeout keeps bounding the dial when ctx is never cancelled.
	nc, err := (&net.Dialer{Timeout: defaultDialTimeout}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial xshard slave %s: %w", addr, err)
	}

	conn, err := newXshardConn(nc, p.maxPayloadSize, p.localChainID, slaveInfo.ChainID, true, p.handler, p.log)
	if err != nil {
		nc.Close()
		return fmt.Errorf("create xshard connection to slave %s: %w", addr, err)
	}

	if !p.trackConnection(conn) {
		conn.base.Close()
		return fmt.Errorf("xshard pool closed")
	}
	conn.base.Start()

	// The remote slave must confirm its advertised chain ID.
	// Bound the handshake independently of the caller's context so a peer
	// that keeps the TCP connection open but never responds to PING cannot
	// block DialToSlave indefinitely.
	pingCtx, cancel := context.WithTimeout(ctx, xshardHandshakeTimeout)
	defer cancel()

	id, err := conn.sendPing(pingCtx)
	if err != nil {
		p.rejectConnection(conn)
		return fmt.Errorf("ping failed for %s: %w", conn.base.RemoteAddr(), err)
	}
	if id != slaveInfo.ChainID {
		p.rejectConnection(conn)
		return fmt.Errorf("slave chain ID mismatch for %s: expected %d, got %d", conn.base.RemoteAddr(), slaveInfo.ChainID, id)
	}

	// Do not deduplicate again after the handshake: simultaneous dials could
	// otherwise cause both nodes to close their outbound connections.
	if err := p.registerConnection(conn); err != nil {
		return err
	}
	p.log.Info("indexed xshard connection to slave", "remote_chain_id", id)
	return nil
}

// HandleInbound takes ownership of an accepted cross-shard connection from
// another slave: it waits for the handshake and indexes the conn. Conns whose
// handshake fails or times out are discarded and never enter the routing index.
// Registration happens-before the PONG reply for inbound connections, so a successful
// handshake observed by the dialing peer implies that the inbound connection
// has already been indexed by this pool.
func (p *XshardPool) HandleInbound(nc net.Conn) {
	// Inbound identity arrives with the first PING.
	conn, err := newXshardConn(nc, p.maxPayloadSize, p.localChainID, 0, false, p.handler, p.log)
	if err != nil {
		nc.Close()
		p.log.Error("inbound xshard connection from slave rejected", "err", err)
		return
	}

	if !p.trackConnection(conn) {
		conn.base.Close()
		p.log.Warn("xshard pool closed, closing inbound connection from slave", "remote", conn.base.RemoteAddr())
		return
	}

	// Arm the registration barrier before Start, so the reader goroutine
	// observes the channel (goroutine creation publishes the write).
	conn.inboundRegistered = make(chan struct{})
	conn.base.Start()

	if !conn.waitUntilPingReceived() {
		p.log.Warn("inbound xshard connection from slave closed before ping", "remote", conn.base.RemoteAddr())
		p.rejectConnection(conn)
		return
	}

	// A remote slave may have multiple inbound connections.
	if err := p.registerConnection(conn); err != nil {
		conn.base.Close() // defensive
		p.log.Warn("inbound xshard connection from slave rejected", "remote", conn.base.RemoteAddr(), "err", err)
		return
	}
	// Release the barrier: the first PING's PONG is written only after this
	// point, so the dialing peer cannot trigger a reverse dial that this pool
	// would not know to skip. registerConnection's failure paths close the
	// conn, which unblocks the barrier with an error instead of a PONG.
	close(conn.inboundRegistered)

	p.log.Info("indexed inbound xshard connection from slave", "remote_chain_id", conn.RemoteChainID())
}

// Lookup returns cross-shard connections to the slave identified by chainID.
// They may be closed concurrently afterwards, so callers must tolerate failed
// sends.
// The returned slice is a copy holding the original pointers.
func (p *XshardPool) Lookup(chainID uint32) []*XshardConn {
	p.mu.RLock()
	conns := p.conns[chainID]
	result := make([]*XshardConn, len(conns))
	copy(result, conns)
	p.mu.RUnlock()
	return result
}

// Close closes all pool connections, including ones still handshaking
// (py close_all leaks those).
func (p *XshardPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true

	allConns := make([]*XshardConn, 0, len(p.connections))
	for conn := range p.connections {
		allConns = append(allConns, conn)
	}

	p.conns = nil
	p.connections = nil
	p.chainIDs = nil
	p.mu.Unlock()

	for _, conn := range allConns {
		conn.base.Close()
	}
	p.log.Info("xshard pool closed", "connections", len(allConns))
}

// Internal implementation

// rejectConnection closes a tracked but not-yet-registered connection and
// evicts it from the tracking set. Every failure path between
// trackConnection and registerConnection must go through here, never
// through closing the connection alone: a conn left only in the tracking set would
// linger until pool Close.
func (p *XshardPool) rejectConnection(conn *XshardConn) {
	conn.base.Close()

	p.mu.Lock()
	delete(p.connections, conn)
	p.mu.Unlock()
}

// registerConnection commits a verified connection into the routing index.
// It reports an error if the pool closed while the connection was handshaking,
// evicting any leftover tracking entry (normally none — Close empties the
// registries) and closing the conn.
func (p *XshardPool) registerConnection(conn *XshardConn) error {
	p.mu.Lock()
	if p.closed {
		delete(p.connections, conn)
		p.mu.Unlock()
		conn.base.Close()
		return fmt.Errorf("xshard pool closed")
	}
	if _, ok := p.configuredChainIDs[conn.RemoteChainID()]; !ok {
		delete(p.connections, conn)
		p.mu.Unlock()
		conn.base.Close()
		return fmt.Errorf("slave chain ID %d is not configured", conn.RemoteChainID())
	}

	p.addSlaveConnectionLocked(conn)
	p.mu.Unlock()
	return nil
}

// addSlaveConnectionLocked indexes the connection by the remote slave's chain ID.
func (p *XshardPool) addSlaveConnectionLocked(conn *XshardConn) {
	id := conn.RemoteChainID()
	p.chainIDs[id] = struct{}{}
	p.conns[id] = append(p.conns[id], conn)
}

// trackConnection registers a newly created conn. It reports false if the
// pool is already closed.
func (p *XshardPool) trackConnection(conn *XshardConn) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return false
	}
	p.connections[conn] = struct{}{}
	return true
}

// knownSlave reports whether chainID identifies the local slave or a slave
// already in the routing index.
func (p *XshardPool) knownSlave(chainID uint32) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.localChainID == chainID {
		return true
	}
	_, known := p.chainIDs[chainID]
	return known
}
