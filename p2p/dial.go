// Copyright 2015 The go-ethereum Authors
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
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	mrand "math/rand"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/netutil"
)

const (
	// This is the amount of time spent waiting in between redialing a certain node. The
	// limit is a bit higher than inboundThrottleTime to prevent failing dials in small
	// private networks.
	dialHistoryExpiration = inboundThrottleTime + 5*time.Second

	// Config for the "Looking for peers" message.
	dialStatsLogInterval = 10 * time.Second // printed at most this often
	dialStatsPeerLimit   = 3                // but not if more than this many dialed peers

	// Endpoint resolution is throttled with bounded backoff.
	initialResolveDelay = 60 * time.Second
	maxResolveDelay     = time.Hour
)

// NodeDialer is used to connect to nodes in the network, typically by using
// an underlying net.Dialer but also using net.Pipe in tests.
type NodeDialer interface {
	Dial(context.Context, *enode.Node) (net.Conn, error)
}

type nodeResolver interface {
	Resolve(*enode.Node) *enode.Node
}

// tcpDialer implements NodeDialer using real TCP connections.
type tcpDialer struct {
	d *net.Dialer
}

func (t tcpDialer) Dial(ctx context.Context, dest *enode.Node) (net.Conn, error) {
	addr, _ := dest.TCPEndpoint()
	return t.d.DialContext(ctx, "tcp", addr.String())
}

// checkDial errors:
var (
	errSelf             = errors.New("is self")
	errAlreadyDialing   = errors.New("already dialing")
	errAlreadyConnected = errors.New("already connected")
	errRecentlyDialed   = errors.New("recently dialed")
	errNetRestrict      = errors.New("not contained in netrestrict list")
	errNoPort           = errors.New("node does not provide TCP port")
	errNoResolvedIP     = errors.New("node does not provide a resolved IP")
)

// dialer creates outbound connections and submits them into Server.
// Two types of peer connections can be created:
//
//   - static dials are pre-configured connections. The dialer attempts
//     keep these nodes connected at all times.
//
//   - dynamic dials are created from node discovery results. The dialer
//     continuously reads candidate nodes from its input iterator and attempts
//     to create peer connections to nodes arriving through the iterator.
type dialScheduler struct {
	dialConfig
	setupFunc     dialSetupFunc
	dnsLookupFunc func(ctx context.Context, network string, name string) ([]netip.Addr, error)
	wg            sync.WaitGroup
	cancel        context.CancelFunc
	ctx           context.Context
	nodesIn       chan *enode.Node
	doneCh        chan *dialTask
	addStaticCh   chan *enode.Node
	remStaticCh   chan *enode.Node
	addPeerCh     chan *conn
	remPeerCh     chan *conn

	// Everything below here belongs to loop and
	// should only be accessed by code on the loop goroutine.
	dialing   map[enode.ID]*dialTask // active tasks
	peers     map[enode.ID]struct{}  // all connected peers
	dialPeers int                    // current number of dialed peers

	// The static map tracks all static dial tasks. The subset of usable static dial tasks
	// (i.e. those passing checkDial) is kept in staticPool. The scheduler prefers
	// launching random static tasks from the pool over launching dynamic dials from the
	// iterator.
	static     map[enode.ID]*dialTask
	staticPool []*dialTask

	// The dial history keeps recently dialed nodes. Members of history are not dialed.
	history      expHeap
	historyTimer *mclock.Alarm

	// for logStats
	lastStatsLog     mclock.AbsTime
	doneSinceLastLog int
}

type dialSetupFunc func(net.Conn, connFlag, *enode.Node) error

type dialConfig struct {
	self           enode.ID         // our own ID
	maxDialPeers   int              // maximum number of dialed peers
	maxActiveDials int              // maximum number of active dials
	netRestrict    *netutil.Netlist // IP netrestrict list, disabled if nil
	resolver       nodeResolver
	dialer         NodeDialer
	log            log.Logger
	clock          mclock.Clock
	rand           *mrand.Rand
}

func (cfg dialConfig) withDefaults() dialConfig {
	if cfg.maxActiveDials == 0 {
		cfg.maxActiveDials = defaultMaxPendingPeers
	}
	if cfg.log == nil {
		cfg.log = log.Root()
	}
	if cfg.clock == nil {
		cfg.clock = mclock.System{}
	}
	if cfg.rand == nil {
		seedb := make([]byte, 8)
		crand.Read(seedb)
		seed := int64(binary.BigEndian.Uint64(seedb))
		cfg.rand = mrand.New(mrand.NewSource(seed))
	}
	return cfg
}

func newDialScheduler(config dialConfig, it enode.Iterator, setupFunc dialSetupFunc) *dialScheduler {
	cfg := config.withDefaults()
	d := &dialScheduler{
		dialConfig:    cfg,
		historyTimer:  mclock.NewAlarm(cfg.clock),
		setupFunc:     setupFunc,
		dnsLookupFunc: net.DefaultResolver.LookupNetIP,
		dialing:       make(map[enode.ID]*dialTask),
		static:        make(map[enode.ID]*dialTask),
		peers:         make(map[enode.ID]struct{}),
		doneCh:        make(chan *dialTask),
		nodesIn:       make(chan *enode.Node),
		addStaticCh:   make(chan *enode.Node),
		remStaticCh:   make(chan *enode.Node),
		addPeerCh:     make(chan *conn),
		remPeerCh:     make(chan *conn),
	}
	d.lastStatsLog = d.clock.Now()
	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.wg.Add(2)
	go d.readNodes(it)
	go d.loop(it)
	return d
}

// stop shuts down the dialer, canceling all current dial tasks.
func (d *dialScheduler) stop() {
	d.cancel()
	d.wg.Wait()
}

// addStatic adds a static dial candidate.
func (d *dialScheduler) addStatic(n *enode.Node) {
	select {
	case d.addStaticCh <- n:
	case <-d.ctx.Done():
	}
}

// removeStatic removes a static dial candidate.
func (d *dialScheduler) removeStatic(n *enode.Node) {
	select {
	case d.remStaticCh <- n:
	case <-d.ctx.Done():
	}
}

// peerAdded updates the peer set.
func (d *dialScheduler) peerAdded(c *conn) {
	select {
	case d.addPeerCh <- c:
	case <-d.ctx.Done():
	}
}

// peerRemoved updates the peer set.
func (d *dialScheduler) peerRemoved(c *conn) {
	select {
	case d.remPeerCh <- c:
	case <-d.ctx.Done():
	}
}

// loop is the main loop of the dialer.
func (d *dialScheduler) loop(it enode.Iterator) {
	var (
		nodesCh chan *enode.Node
	)

loop:
	for {
		// Launch new dials if slots are available.
		slots := d.freeDialSlots()
		slots -= d.startStaticDials(slots)
		if slots > 0 {
			nodesCh = d.nodesIn
		} else {
			nodesCh = nil
		}
		d.rearmHistoryTimer()
		d.logStats()

		select {
		case node := <-nodesCh:
			if err := d.checkDial(node); err != nil {
				d.log.Trace("Discarding dial candidate", "id", node.ID(), "ip", node.IPAddr(), "reason", err)
			} else {
				d.startDial(newDialTask(node, dynDialedConn))
			}

		case task := <-d.doneCh:
			id := task.dest().ID()
			delete(d.dialing, id)
			d.updateStaticPool(id)
			d.doneSinceLastLog++

		case c := <-d.addPeerCh:
			if c.is(dynDialedConn) || c.is(staticDialedConn) {
				d.dialPeers++
			}
			id := c.node.ID()
			d.peers[id] = struct{}{}
			// Remove from static pool because the node is now connected.
			task := d.static[id]
			if task != nil && task.staticPoolIndex >= 0 {
				d.removeFromStaticPool(task.staticPoolIndex)
			}
			// TODO: cancel dials to connected peers

		case c := <-d.remPeerCh:
			if c.is(dynDialedConn) || c.is(staticDialedConn) {
				d.dialPeers--
			}
			delete(d.peers, c.node.ID())
			d.updateStaticPool(c.node.ID())

		case node := <-d.addStaticCh:
			id := node.ID()
			_, exists := d.static[id]
			d.log.Trace("Adding static node", "id", id, "endpoint", nodeEndpointForLog(node), "added", !exists)
			if exists {
				continue loop
			}
			task := newDialTask(node, staticDialedConn)
			d.static[id] = task
			if d.checkDial(node) == nil {
				d.addToStaticPool(task)
			}

		case node := <-d.remStaticCh:
			id := node.ID()
			task := d.static[id]
			d.log.Trace("Removing static node", "id", id, "ok", task != nil)
			if task != nil {
				delete(d.static, id)
				if task.staticPoolIndex >= 0 {
					d.removeFromStaticPool(task.staticPoolIndex)
				}
			}

		case <-d.historyTimer.C():
			d.expireHistory()

		case <-d.ctx.Done():
			it.Close()
			break loop
		}
	}

	d.historyTimer.Stop()
	for range d.dialing {
		<-d.doneCh
	}
	d.wg.Done()
}

// readNodes runs in its own goroutine and delivers nodes from
// the input iterator to the nodesIn channel.
func (d *dialScheduler) readNodes(it enode.Iterator) {
	defer d.wg.Done()

	for it.Next() {
		select {
		case d.nodesIn <- it.Node():
		case <-d.ctx.Done():
		}
	}
}

// logStats prints dialer statistics to the log. The message is suppressed when enough
// peers are connected because users should only see it while their client is starting up
// or comes back online.
func (d *dialScheduler) logStats() {
	now := d.clock.Now()
	if d.lastStatsLog.Add(dialStatsLogInterval) > now {
		return
	}
	if d.dialPeers < dialStatsPeerLimit && d.dialPeers < d.maxDialPeers {
		d.log.Info("Looking for peers", "peercount", len(d.peers), "tried", d.doneSinceLastLog, "static", len(d.static))
	}
	d.doneSinceLastLog = 0
	d.lastStatsLog = now
}

// rearmHistoryTimer configures d.historyTimer to fire when the
// next item in d.history expires.
func (d *dialScheduler) rearmHistoryTimer() {
	if len(d.history) == 0 {
		return
	}
	d.historyTimer.Schedule(d.history.nextExpiry())
}

// expireHistory removes expired items from d.history.
func (d *dialScheduler) expireHistory() {
	d.history.expire(d.clock.Now(), func(hkey string) {
		var id enode.ID
		copy(id[:], hkey)
		d.updateStaticPool(id)
	})
}

// freeDialSlots returns the number of free dial slots. The result can be negative
// when peers are connected while their task is still running.
func (d *dialScheduler) freeDialSlots() int {
	slots := (d.maxDialPeers - d.dialPeers) * 2
	if slots > d.maxActiveDials {
		slots = d.maxActiveDials
	}
	free := slots - len(d.dialing)
	return free
}

// checkDial returns an error if node n should not be dialed.
func (d *dialScheduler) checkDial(n *enode.Node) error {
	if n.ID() == d.self {
		return errSelf
	}
	if n.IPAddr().IsValid() && n.TCP() == 0 {
		// This check can trigger if a non-TCP node is found
		// by discovery. If there is no IP, the node is a static
		// node and the actual endpoint will be resolved later in dialTask.
		return errNoPort
	}
	if _, ok := d.dialing[n.ID()]; ok {
		return errAlreadyDialing
	}
	if _, ok := d.peers[n.ID()]; ok {
		return errAlreadyConnected
	}
	if d.netRestrict != nil && !d.netRestrict.ContainsAddr(n.IPAddr()) {
		return errNetRestrict
	}
	if d.history.contains(string(n.ID().Bytes())) {
		return errRecentlyDialed
	}
	return nil
}

// startStaticDials starts n static dial tasks.
func (d *dialScheduler) startStaticDials(n int) (started int) {
	for started = 0; started < n && len(d.staticPool) > 0; started++ {
		idx := d.rand.Intn(len(d.staticPool))
		task := d.staticPool[idx]
		d.startDial(task)
		d.removeFromStaticPool(idx)
	}
	return started
}

// updateStaticPool attempts to move the given static dial back into staticPool.
func (d *dialScheduler) updateStaticPool(id enode.ID) {
	task, ok := d.static[id]
	if ok && task.staticPoolIndex < 0 && d.checkDial(task.dest()) == nil {
		d.addToStaticPool(task)
	}
}

func (d *dialScheduler) addToStaticPool(task *dialTask) {
	if task.staticPoolIndex >= 0 {
		panic("attempt to add task to staticPool twice")
	}
	d.staticPool = append(d.staticPool, task)
	task.staticPoolIndex = len(d.staticPool) - 1
}

// removeFromStaticPool removes the task at idx from staticPool. It does that by moving the
// current last element of the pool to idx and then shortening the pool by one.
func (d *dialScheduler) removeFromStaticPool(idx int) {
	task := d.staticPool[idx]
	end := len(d.staticPool) - 1
	d.staticPool[idx] = d.staticPool[end]
	d.staticPool[idx].staticPoolIndex = idx
	d.staticPool[end] = nil
	d.staticPool = d.staticPool[:end]
	task.staticPoolIndex = -1
}

// dnsResolveHostname updates the given node from its DNS hostname.
// This is used to resolve static dial targets.
func (d *dialScheduler) dnsResolveHostname(n *enode.Node) (*enode.Node, error) {
	if n.Hostname() == "" {
		return n, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	foundIPs, err := d.dnsLookupFunc(ctx, "ip", n.Hostname())
	if err != nil {
		return n, err
	}

	// Check for IP updates.
	var (
		nodeIP4, nodeIP6   netip.Addr
		foundIP4, foundIP6 netip.Addr
	)
	n.Load((*enr.IPv4Addr)(&nodeIP4))
	n.Load((*enr.IPv6Addr)(&nodeIP6))
	for _, ip := range foundIPs {
		if ip.Is4() && !foundIP4.IsValid() {
			foundIP4 = ip
		}
		if ip.Is6() && !foundIP6.IsValid() {
			foundIP6 = ip
		}
	}

	if !foundIP4.IsValid() && !foundIP6.IsValid() {
		// Lookup failed.
		return n, errNoResolvedIP
	}
	if foundIP4 == nodeIP4 && foundIP6 == nodeIP6 {
		// No updates necessary.
		d.log.Trace("Node DNS lookup had no update", "id", n.ID(), "name", n.Hostname(), "ip", foundIP4, "ip6", foundIP6)
		return n, nil
	}

	// Update the node. Note this invalidates the ENR signature, because we use SignNull
	// to create a modified copy. But this should be OK, since we just use the node as a
	// dial target. And nodes will usually only have a DNS hostname if they came from a
	// enode:// URL, which has no signature anyway. If it ever becomes a problem, the
	// resolved IP could also be stored into dialTask instead of the node.
	rec := n.Record()
	if foundIP4.IsValid() {
		rec.Set(enr.IPv4Addr(foundIP4))
	}
	if foundIP6.IsValid() {
		rec.Set(enr.IPv6Addr(foundIP6))
	}
	rec.SetSeq(n.Seq()) // ensure seq not bumped by update
	newNode := enode.SignNull(rec, n.ID()).WithHostname(n.Hostname())
	d.log.Debug("Node updated from DNS lookup", "id", n.ID(), "name", n.Hostname(), "ip", newNode.IP())
	return newNode, nil
}

// startDial runs the given dial task in a separate goroutine.
func (d *dialScheduler) startDial(task *dialTask) {
	node := task.dest()
	d.log.Trace("Starting p2p dial", "id", node.ID(), "endpoint", nodeEndpointForLog(node), "flag", task.flags)
	hkey := string(node.ID().Bytes())
	d.history.add(hkey, d.clock.Now().Add(dialHistoryExpiration))
	d.dialing[node.ID()] = task
	go func() {
		task.run(d)
		d.doneCh <- task
	}()
}

// A dialTask generated for each node that is dialed.
type dialTask struct {
	staticPoolIndex int
	flags           connFlag

	// These fields are private to the task and should not be
	// accessed by dialScheduler while the task is running.
	destPtr      atomic.Pointer[enode.Node]
	lastResolved mclock.AbsTime
	resolveDelay time.Duration
}

func newDialTask(dest *enode.Node, flags connFlag) *dialTask {
	t := &dialTask{flags: flags, staticPoolIndex: -1}
	t.destPtr.Store(dest)
	return t
}

type dialError struct {
	error
}

func (t *dialTask) dest() *enode.Node {
	return t.destPtr.Load()
}

func (t *dialTask) run(d *dialScheduler) {
	if t.isStatic() {
		// Resolve DNS.
		if n := t.dest(); n.Hostname() != "" {
			resolved, err := d.dnsResolveHostname(n)
			if err != nil {
				d.log.Warn("DNS lookup of static node failed", "id", n.ID(), "name", n.Hostname(), "err", err)
			} else {
				t.destPtr.Store(resolved)
			}
		}
		// Try resolving node ID through the DHT if there is no IP address.
		if !t.dest().IPAddr().IsValid() {
			if !t.resolve(d) {
				return // DHT resolve failed, skip dial.
			}
		}
	}

	err := t.dial(d, t.dest())
	if err != nil {
		// For static nodes, resolve one more time if dialing fails.
		var dialErr *dialError
		if errors.As(err, &dialErr) && t.isStatic() {
			if t.resolve(d) {
				t.dial(d, t.dest())
			}
		}
	}
}

func (t *dialTask) isStatic() bool {
	return t.flags&staticDialedConn != 0
}

// resolve attempts to find the current endpoint for the destination
// using discovery.
//
// Resolve operations are throttled with backoff to avoid flooding the
// discovery network with useless queries for nodes that don't exist.
// The backoff delay resets when the node is found.
func (t *dialTask) resolve(d *dialScheduler) bool {
	if d.resolver == nil {
		return false
	}
	if t.resolveDelay == 0 {
		t.resolveDelay = initialResolveDelay
	}
	if t.lastResolved > 0 && time.Duration(d.clock.Now()-t.lastResolved) < t.resolveDelay {
		return false
	}

	node := t.dest()
	resolved := d.resolver.Resolve(node)
	t.lastResolved = d.clock.Now()
	if resolved == nil {
		t.resolveDelay *= 2
		if t.resolveDelay > maxResolveDelay {
			t.resolveDelay = maxResolveDelay
		}
		d.log.Debug("Resolving node failed", "id", node.ID(), "newdelay", t.resolveDelay)
		return false
	}
	// The node was found.
	t.resolveDelay = initialResolveDelay
	t.destPtr.Store(resolved)
	resAddr, _ := resolved.TCPEndpoint()
	d.log.Debug("Resolved node", "id", resolved.ID(), "addr", resAddr)
	return true
}

// dial performs the actual connection attempt.
func (t *dialTask) dial(d *dialScheduler, dest *enode.Node) error {
	dialMeter.Mark(1)
	fd, err := d.dialer.Dial(d.ctx, dest)
	if err != nil {
		addr, _ := dest.TCPEndpoint()
		d.log.Trace("Dial error", "id", dest.ID(), "addr", addr, "conn", t.flags, "err", cleanupDialErr(err))
		dialConnectionError.Mark(1)
		return &dialError{err}
	}
	return d.setupFunc(newMeteredConn(fd), t.flags, dest)
}

func (t *dialTask) String() string {
	node := t.dest()
	id := node.ID()
	return fmt.Sprintf("%v %x %v:%d", t.flags, id[:8], node.IPAddr(), node.TCP())
}

func cleanupDialErr(err error) error {
	if netErr, ok := err.(*net.OpError); ok && netErr.Op == "dial" {
		return netErr.Err
	}
	return err
}

func nodeEndpointForLog(n *enode.Node) string {
	if n.Hostname() != "" {
		return n.Hostname()
	}
	return n.IPAddr().String()
}
