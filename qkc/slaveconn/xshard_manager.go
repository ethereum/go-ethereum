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
	"math/big"
	"net"
	"strconv"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/wire"
)

// XshardConnManager manages the local slave's cross-shard connections to other slaves,
// indexed by each remote slave's chain ID. Connections remain indexed until
// the manager closes. Master sends ConnectToSlavesRequest to one slave at a
// time and waits for its response, so a successful outbound PING sees the
// inbound connection registered before the next slave is asked to dial.
// Concurrent dials within one request target distinct slaves and do not
// affect this. Duplicate connections are rejected if this ordering is violated.
type XshardConnManager struct {
	localChainID *big.Int
	// Only connections to slaves with configured chain IDs enter the routing index.
	configuredChainIDs map[string]struct{}
	maxPayloadSize     uint32               // 0 uses the maximum frame payload length.
	handler            XshardRequestHandler // Serves cross-shard requests from other slaves.
	log                log.Logger

	// mu guards the connection indexes and closed state together.
	mu          sync.RWMutex
	conns       map[string]*XshardConn
	connections map[*XshardConn]struct{}

	closed        bool
	shutdownOnce  sync.Once
	shutdownSlave func() // Stops the slave after an indexed connection drops.
}

// Public API

// NewXshardConnManager creates the local slave's manager of cross-shard connections.
// localChainID identifies the chain hosted by this slave; configuredChainIDs
// lists the chain IDs hosted by the cluster's slaves. A zero maxPayloadSize
// uses the protocol's maximum payload size. handler serves cross-shard requests
// from other slaves and must not be nil. shutdownSlave must stop the local
// slave when an established cross-shard connection closes unexpectedly.
func NewXshardConnManager(localChainID *big.Int, configuredChainIDs []*big.Int, maxPayloadSize uint32, handler XshardRequestHandler, shutdownSlave func(), logger log.Logger) (*XshardConnManager, error) {
	if handler == nil {
		return nil, errors.New("xshard handler must not be nil")
	}
	if shutdownSlave == nil {
		return nil, errors.New("xshard shutdown callback must not be nil")
	}
	if err := validateXshardChainID(localChainID); err != nil {
		return nil, err
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
	clusterSet := make(map[string]struct{}, len(configuredChainIDs))
	for _, id := range configuredChainIDs {
		if err := validateXshardChainID(id); err != nil {
			return nil, err
		}
		clusterSet[id.String()] = struct{}{}
	}
	if _, ok := clusterSet[localChainID.String()]; !ok {
		return nil, fmt.Errorf("local chain ID %s is not configured", localChainID)
	}
	return &XshardConnManager{
		conns:              make(map[string]*XshardConn),
		connections:        make(map[*XshardConn]struct{}),
		localChainID:       new(big.Int).Set(localChainID),
		handler:            handler,
		shutdownSlave:      shutdownSlave,
		configuredChainIDs: clusterSet,
		maxPayloadSize:     maxPayloadSize,
		log:                logger,
	}, nil
}

// DialToSlave establishes an outbound cross-shard connection to the slave
// described by slaveInfo. Its ChainID identifies the chain hosted by that slave.
func (p *XshardConnManager) DialToSlave(ctx context.Context, slaveInfo wire.SlaveInfo) error {
	remoteChainID := wire.ChainIDFromSSZ(slaveInfo.ChainID)
	if _, ok := p.configuredChainIDs[remoteChainID.String()]; !ok {
		return fmt.Errorf("slave chain ID %d is not configured", remoteChainID)
	}
	if p.knownSlave(remoteChainID) {
		p.log.Info("outbound xshard connection skipped: slave already known", "remote_chain_id", remoteChainID)
		return nil
	}

	addr := net.JoinHostPort(string(slaveInfo.Host), strconv.Itoa(int(slaveInfo.Port)))
	// Dialer.Timeout keeps bounding the dial when ctx is never cancelled.
	nc, err := (&net.Dialer{Timeout: defaultDialTimeout}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial xshard slave %s: %w", addr, err)
	}

	conn, err := newXshardConn(nc, p.maxPayloadSize, p.localChainID, remoteChainID, true, p.handler, p.log)
	if err != nil {
		nc.Close()
		return fmt.Errorf("create xshard connection to slave %s: %w", addr, err)
	}

	if !p.trackConnection(conn) {
		conn.Close()
		return fmt.Errorf("xshard manager closed")
	}
	conn.Start()

	// The remote slave must confirm its advertised chain ID.
	// Bound the handshake independently of the caller's context so a peer
	// that keeps the TCP connection open but never responds to PING cannot
	// block DialToSlave indefinitely.
	pingCtx, cancel := context.WithTimeout(ctx, xshardHandshakeTimeout)
	defer cancel()

	id, err := conn.sendPing(pingCtx)
	if err != nil {
		p.rejectConnection(conn)
		return fmt.Errorf("ping failed for %s: %w", conn.RemoteAddr(), err)
	}
	if id.Cmp(remoteChainID) != 0 {
		p.rejectConnection(conn)
		return fmt.Errorf("slave chain ID mismatch for %s: expected %d, got %d", conn.RemoteAddr(), remoteChainID, id)
	}

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
// has already been indexed by this manager.
//
// The caller must pass only slave-to-slave connections. As in pyquarkchain,
// a slave treats its first accepted connection as the Master connection and
// later connections as slave connections. This relies on Master connecting
// to every slave before sending ConnectToSlavesRequest, and on the Master
// connection never being re-established: the slave shuts down if it drops.
func (p *XshardConnManager) HandleInbound(nc net.Conn) {
	// Inbound identity arrives with the first PING.
	conn, err := newXshardConn(nc, p.maxPayloadSize, p.localChainID, nil, false, p.handler, p.log)
	if err != nil {
		nc.Close()
		p.log.Error("inbound xshard connection from slave rejected", "err", err)
		return
	}

	if !p.trackConnection(conn) {
		conn.Close()
		p.log.Warn("xshard manager closed, closing inbound connection from slave", "remote", conn.RemoteAddr())
		return
	}

	// Arm the registration barrier before Start, so the reader goroutine
	// observes the channel (goroutine creation publishes the write).
	conn.inboundRegistered = make(chan struct{})
	conn.Start()

	if !conn.waitUntilPingReceived() {
		p.log.Warn("inbound xshard connection from slave closed before ping", "remote", conn.RemoteAddr())
		p.rejectConnection(conn)
		return
	}

	if err := p.registerConnection(conn); err != nil {
		p.log.Warn("inbound xshard connection from slave rejected", "remote", conn.RemoteAddr(), "err", err)
		return
	}
	// Release the barrier: the first PING's PONG is written only after this
	// point, so the dialing peer cannot trigger a reverse dial that this manager
	// would not know to skip. registerConnection's failure paths close the
	// conn, which unblocks the barrier with an error instead of a PONG.
	close(conn.inboundRegistered)

	p.log.Info("indexed inbound xshard connection from slave", "remote_chain_id", conn.RemoteChainID())
}

// Lookup returns the connection to the slave identified by chainID. It may
// close concurrently afterwards, so callers must tolerate failed sends.
func (p *XshardConnManager) Lookup(chainID *big.Int) *XshardConn {
	if chainID == nil {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.conns[chainID.String()]
}

// Close closes all manager connections, including ones still handshaking
// (py close_all leaks those).
func (p *XshardConnManager) Close() {
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
	p.mu.Unlock()

	for _, conn := range allConns {
		conn.Close()
	}
	p.log.Info("xshard manager closed", "connections", len(allConns))
}

// Internal implementation

// rejectConnection closes a tracked but not-yet-registered connection and
// evicts it from the tracking set. Every failure path between
// trackConnection and registerConnection must go through here, never
// through closing the connection alone: a conn left only in the tracking set would
// linger until manager Close.
func (p *XshardConnManager) rejectConnection(conn *XshardConn) {
	conn.Close()

	p.mu.Lock()
	delete(p.connections, conn)
	p.mu.Unlock()
}

// registerConnection commits a verified connection into the routing index.
// It reports an error if the manager closed while the connection was handshaking,
// evicting any leftover tracking entry (normally none — Close empties the
// registries) and closing the conn.
func (p *XshardConnManager) registerConnection(conn *XshardConn) error {
	p.mu.Lock()
	if p.closed {
		delete(p.connections, conn)
		p.mu.Unlock()
		conn.Close()
		return fmt.Errorf("xshard manager closed")
	}
	if _, ok := p.configuredChainIDs[conn.RemoteChainID().String()]; !ok {
		delete(p.connections, conn)
		p.mu.Unlock()
		conn.Close()
		return fmt.Errorf("slave chain ID %d is not configured", conn.RemoteChainID())
	}
	if p.conns[conn.RemoteChainID().String()] != nil {
		delete(p.connections, conn)
		p.mu.Unlock()
		conn.Close()
		return fmt.Errorf("slave chain ID %d already connected", conn.RemoteChainID())
	}

	p.conns[conn.RemoteChainID().String()] = conn
	p.mu.Unlock()
	go p.watchConnection(conn)
	return nil
}

// watchConnection shuts down the slave if an indexed connection drops unexpectedly.
func (p *XshardConnManager) watchConnection(conn *XshardConn) {
	<-conn.WaitUntilClosed()
	p.mu.RLock()
	closed := p.closed
	p.mu.RUnlock()
	if !closed {
		// A normal manager shutdown closes its connections too; only an
		// unexpected disconnect should stop the slave.
		p.shutdownOnce.Do(func() {
			p.log.Error("indexed xshard connection closed; shutting down slave", "remote_chain_id", conn.RemoteChainID())
			p.shutdownSlave()
		})
	}
}

// trackConnection registers a newly created conn. It reports false if the
// manager is already closed.
func (p *XshardConnManager) trackConnection(conn *XshardConn) bool {
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
func (p *XshardConnManager) knownSlave(chainID *big.Int) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.localChainID.Cmp(chainID) == 0 {
		return true
	}
	_, known := p.conns[chainID.String()]
	return known
}
