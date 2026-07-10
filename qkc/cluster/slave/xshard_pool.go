// Copyright 2026-2027, QuarkChain.

package slave

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/qkc/cluster/wire"
)

// XshardPool manages direct slave-to-slave xshard connections, indexed by full
// shard ID. It corresponds to Python's SlaveConnectionManager.
type XshardPool struct {
	mu       sync.RWMutex
	conns    map[uint32][]*XshardConn
	inbound  []*XshardConn
	slaveIDs map[string]bool // Tracks slave IDs to prevent duplicate connections
	closed   bool
	log      log.Logger
}

// NewXshardPool creates a new, empty connection pool.
func NewXshardPool(logger log.Logger) *XshardPool {
	return &XshardPool{
		conns:    make(map[uint32][]*XshardConn),
		slaveIDs: make(map[string]bool),
		log:      logger,
	}
}

// Add adds a connection to the pool for the given full shard ID.
// If the pool is already closed, the connection is closed immediately.
// If the slave ID is already tracked, the connection is closed and a warning is logged.
func (p *XshardPool) Add(fullShardID uint32, conn *XshardConn) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		conn.Close()
		p.log.Warn("xshard pool closed, closing outbound conn immediately", "remote", conn.RemoteAddr())
		return
	}

	// Check for duplicate slave ID (matches Python's slave_ids deduplication)
	remoteID := string(conn.RemoteID())
	if remoteID != "" && p.slaveIDs[remoteID] {
		p.mu.Unlock()
		conn.Close()
		p.log.Warn("duplicate slave connection rejected", "slave_id", remoteID, "full_shard_id", fullShardID)
		return
	}

	// Track the slave ID
	if remoteID != "" {
		p.slaveIDs[remoteID] = true
	}

	p.conns[fullShardID] = append(p.conns[fullShardID], conn)
	p.mu.Unlock()
	p.log.Info("added xshard connection", "full_shard_id", fullShardID, "remote", conn.RemoteAddr())
}

// VerifyAndAdd performs PING-based identity verification on an outbound
// connection before adding it to the pool. It matches Python's
// SlaveConnectionManager.connect_to_slave().
//
// The connection must already have been started (Start() called).
// On verification failure the connection is closed.
func (p *XshardPool) VerifyAndAdd(ctx context.Context, fullShardID uint32, conn *XshardConn, expectedID []byte, expectedShardList []uint32) error {
	id, shardList, err := conn.SendPing(ctx)
	if err != nil {
		conn.Close()
		return fmt.Errorf("ping failed for %s: %w", conn.RemoteAddr(), err)
	}
	if !bytes.Equal(id, expectedID) {
		conn.Close()
		return fmt.Errorf("slave id mismatch for %s: expected %x, got %x", conn.RemoteAddr(), expectedID, id)
	}
	if len(shardList) != len(expectedShardList) {
		conn.Close()
		return fmt.Errorf("shard list length mismatch for %s: expected %d, got %d", conn.RemoteAddr(), len(expectedShardList), len(shardList))
	}
	for i := range shardList {
		if shardList[i] != expectedShardList[i] {
			conn.Close()
			return fmt.Errorf("shard list mismatch for %s: expected %v, got %v", conn.RemoteAddr(), expectedShardList, shardList)
		}
	}
	p.Add(fullShardID, conn)
	return nil
}

// Get returns a snapshot of connections for the given full shard ID.
func (p *XshardPool) Get(fullShardID uint32) []*XshardConn {
	p.mu.RLock()
	conns := p.conns[fullShardID]
	result := make([]*XshardConn, len(conns))
	copy(result, conns)
	p.mu.RUnlock()
	return result
}

// Remove removes a specific connection from the pool. It also cleans up the
// slave ID tracking so the same slave can reconnect later.
func (p *XshardPool) Remove(fullShardID uint32, conn *XshardConn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	conns := p.conns[fullShardID]
	for i, c := range conns {
		if c == conn {
			copy(conns[i:], conns[i+1:])
			conns[len(conns)-1] = nil
			p.conns[fullShardID] = conns[:len(conns)-1]
			if len(p.conns[fullShardID]) == 0 {
				delete(p.conns, fullShardID)
			}
			if remoteID := string(conn.RemoteID()); remoteID != "" {
				delete(p.slaveIDs, remoteID)
			}
			p.log.Info("removed xshard connection", "full_shard_id", fullShardID, "remote", conn.RemoteAddr())
			return
		}
	}
}

// RemoveTarget removes and closes all connections for a full shard ID.
func (p *XshardPool) RemoveTarget(fullShardID uint32) {
	p.mu.Lock()
	conns := p.conns[fullShardID]
	delete(p.conns, fullShardID)
	for _, conn := range conns {
		if remoteID := string(conn.RemoteID()); remoteID != "" {
			delete(p.slaveIDs, remoteID)
		}
	}
	p.mu.Unlock()

	for _, conn := range conns {
		conn.Close()
	}
	p.log.Info("removed all xshard connections to shard", "full_shard_id", fullShardID)
}

// SendXshardTx broadcasts xshard transactions to all active connections for the
// target shard via RPC. Returns the first successful response or an error if no
// connection exists or all connections fail.
//
// This matches Python's broadcast_xshard_tx_list behavior: sends to ALL connections
// concurrently and checks that all responses have error_code == 0.
func (p *XshardPool) SendXshardTx(ctx context.Context, fullShardID uint32, payload []byte) (*wire.Frame, error) {
	conns := p.Get(fullShardID)
	if len(conns) == 0 {
		return nil, fmt.Errorf("no xshard connection to full shard %d", fullShardID)
	}

	// Filter active connections
	var activeConns []*XshardConn
	for _, conn := range conns {
		if conn.IsActive() && !conn.Closed() {
			activeConns = append(activeConns, conn)
		}
	}

	if len(activeConns) == 0 {
		return nil, fmt.Errorf("no live xshard connection to full shard %d", fullShardID)
	}

	// Broadcast to all active connections concurrently (matches Python's asyncio.gather)
	type result struct {
		resp *wire.Frame
		err  error
	}
	results := make([]result, len(activeConns))
	var wg sync.WaitGroup

	for i, conn := range activeConns {
		wg.Add(1)
		go func(idx int, c *XshardConn) {
			defer wg.Done()
			resp, err := c.SendXshardTxList(ctx, payload)
			results[idx] = result{resp: resp, err: err}
		}(i, conn)
	}
	wg.Wait()

	// Check all responses (matches Python's check(all([response.error_code == 0 ...])))
	var firstErr error
	var firstResp *wire.Frame
	for _, r := range results {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		if firstResp == nil {
			firstResp = r.resp
		}
	}

	if firstErr != nil {
		return nil, firstErr
	}
	return firstResp, nil
}

// TrackInbound registers an already-started inbound connection for lifecycle
// management. The pool will close it when Close is called.
//
// TrackInbound only handles lifecycle (close-on-shutdown). Use WatchAndIndex
// to additionally wait for identity exchange and index by shard for routing.
func (p *XshardPool) TrackInbound(conn *XshardConn) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		conn.Close()
		p.log.Warn("xshard pool closed, closing inbound conn immediately", "remote", conn.RemoteAddr())
		return
	}
	p.inbound = append(p.inbound, conn)
	p.mu.Unlock()
	p.log.Info("tracked inbound xshard connection", "remote", conn.RemoteAddr())
}

// WatchAndIndex waits for the inbound connection to complete PING-based identity
// exchange, then indexes it by all remote shard IDs for routing purposes.
// It also registers the slave ID for deduplication.
//
// Returns false if the connection closes before identity exchange completes.
// The connection should already be tracked via TrackInbound before calling this.
func (p *XshardPool) WatchAndIndex(conn *XshardConn) bool {
	if !conn.WaitUntilPingReceived() {
		p.log.Warn("inbound xshard connection closed before ping", "remote", conn.RemoteAddr())
		return false
	}

	remoteID := conn.RemoteID()
	shardList := conn.RemoteFullShardIDList()

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		conn.Close()
		return false
	}

	// Register slave ID for deduplication
	if len(remoteID) > 0 {
		p.slaveIDs[string(remoteID)] = true
	}

	// Index by remote shard IDs for routing
	for _, shardID := range shardList {
		p.conns[shardID] = append(p.conns[shardID], conn)
	}
	p.mu.Unlock()

	p.log.Info("indexed inbound xshard connection", "remote_id", string(remoteID), "shards", shardList)
	return true
}

// Close closes all connections in the pool and prevents new additions.
func (p *XshardPool) Close() {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return
	}
	p.closed = true

	var allConns []*XshardConn
	for _, conns := range p.conns {
		allConns = append(allConns, conns...)
	}
	allConns = append(allConns, p.inbound...)

	p.conns = nil
	p.inbound = nil
	p.slaveIDs = nil
	p.mu.Unlock()

	for _, conn := range allConns {
		conn.Close()
	}
	p.log.Info("xshard pool closed", "connections", len(allConns))
}

// OutboundSize returns the number of outbound connections (indexed by shard ID).
func (p *XshardPool) OutboundSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total := 0
	for _, conns := range p.conns {
		total += len(conns)
	}
	return total
}

// InboundSize returns the number of tracked inbound connections.
func (p *XshardPool) InboundSize() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.inbound)
}

// Targets returns all full shard IDs that have outbound connections.
func (p *XshardPool) Targets() []uint32 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	targets := make([]uint32, 0, len(p.conns))
	for id := range p.conns {
		targets = append(targets, id)
	}
	return targets
}
