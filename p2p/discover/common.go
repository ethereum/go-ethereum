// Copyright 2019 The go-ethereum Authors
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

package discover

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"sync"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/netutil"
)

// UDPConn is a network connection on which discovery can operate.
type UDPConn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	LocalAddr() net.Addr
}

// Config holds settings for the discovery listener.
type Config struct {
	// These settings are required and configure the UDP listener:
	PrivateKey *ecdsa.PrivateKey

	// These settings are optional:
	NetRestrict *netutil.Netlist  // network whitelist
	Bootnodes   []*enode.Node     // list of bootstrap nodes
	Unhandled   chan<- ReadPacket // unhandled packets are sent on this channel
	Log         log.Logger        // if set, log messages go here
}

// ListenUDP starts listening for discovery packets on the given UDP socket.
func ListenUDP(c UDPConn, ln *enode.LocalNode, cfg Config) (*UDPv4, error) {
	return ListenV4(c, ln, cfg)
}

// ReadPacket is a packet that couldn't be handled. Those packets are sent to the unhandled
// channel if configured. This is exported for internal use, do not use this type.
type ReadPacket struct {
	Data []byte
	Addr *net.UDPAddr
}

type lookupFunc func(cancel <-chan struct{}, seenNode func(*enode.Node))

// lookupWalker performs recursive lookups, walking the DHT.
// It manages a set iterators which receive lookup results as they are found.
type lookupWalker struct {
	lookup  lookupFunc
	closeCh chan struct{}

	mu    sync.Mutex
	cond  *sync.Cond
	wg    sync.WaitGroup
	iters map[*lookupIterator]struct{}
}

func newLookupWalker(fn lookupFunc) *lookupWalker {
	w := &lookupWalker{
		lookup:  fn,
		closeCh: make(chan struct{}),
		iters:   make(map[*lookupIterator]struct{}),
	}
	w.cond = sync.NewCond(&w.mu)
	w.wg.Add(1)
	go w.loop()
	return w
}

func (w *lookupWalker) close() {
	close(w.closeCh)
	w.wg.Wait()
}

// loop schedules lookups. It ensures a lookup is running while
// any live iterator needs more nodes.
func (w *lookupWalker) loop() {
	var (
		done    = make(chan struct{})
		cancel  = make(chan struct{})
		running bool
	)
	for {
		if !running {
			go w.runLookup(cancel, done)
		}
		select {
		case <-done:
		case <-w.closeCh:
			if running {
				close(cancel)
				<-done
			}
			goto shutdown
		}
	}

shutdown:
	w.mu.Lock()
	defer w.mu.Unlock()
	for it := range w.iters {
		it.close()
	}
	w.wg.Done()
}

func (w *lookupWalker) runLookup(cancel, done chan struct{}) {
	w.lookup(cancel, w.foundNode)
	done <- struct{}{}
}

func (w *lookupWalker) foundNode(n *enode.Node) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for it := range w.iters {
		it.deliver(n)
		fmt.Println("delivered", len(it.buf), it.needsNodes())
	}
	for !anyIterNeedsNodes(w.iters) {
		w.cond.Wait()
	}
}

func (w *lookupWalker) newIterator(filter filterFunc) *lookupIterator {
	it := newLookupIterator(w, filter)
	it.walker.add(it)
	return it
}

func (w *lookupWalker) add(it *lookupIterator) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.iters[it] = struct{}{}
	w.unblockLookup()
}

func (w *lookupWalker) remove(it *lookupIterator) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.iters, it)
	w.unblockLookup()
}

func (w *lookupWalker) unblockLookup() {
	w.cond.Signal()
}

func anyIterNeedsNodes(iters map[*lookupIterator]struct{}) bool {
	for it := range iters {
		if it.needsNodes() {
			return true
		}
	}
	return false
}

// lookupIterator is a sequence of discovered nodes.
type lookupIterator struct {
	cur    *enode.Node
	walker *lookupWalker
	filter filterFunc
	mu     sync.Mutex
	cond   *sync.Cond
	buf    []*enode.Node
}

const lookupIteratorBuffer = 100

type filterFunc func(*enode.Node) bool

func newLookupIterator(w *lookupWalker, filter filterFunc) *lookupIterator {
	if filter == nil {
		filter = func(*enode.Node) bool { return true }
	}
	it := &lookupIterator{
		walker: w,
		filter: filter,
		buf:    make([]*enode.Node, 0, lookupIteratorBuffer),
	}
	it.cond = sync.NewCond(&it.mu)
	return it
}

func (it *lookupIterator) Next() bool {
	it.cur = nil

	// Wait for the buffer to be filled.
	it.mu.Lock()
	defer it.mu.Unlock()
	for it.buf != nil && len(it.buf) == 0 {
		it.cond.Wait()
	}
	if it.buf == nil {
		return false // closed
	}
	it.cur = it.buf[0]
	copy(it.buf, it.buf[1:])
	it.buf = it.buf[:len(it.buf)-1]
	fmt.Println("read node", len(it.buf))
	it.walker.unblockLookup()
	return true
}

func (it *lookupIterator) Node() *enode.Node {
	return it.cur
}

func (it *lookupIterator) Close() {
	it.walker.remove(it)
	it.close()
}

func (it *lookupIterator) close() {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.buf != nil {
		it.buf = nil
		it.cond.Signal()
	}
}

// deliver places a node into the iterator buffer.
func (it *lookupIterator) deliver(n *enode.Node) bool {
	it.mu.Lock()
	defer it.mu.Unlock()

	if it.buf == nil || !it.filter(n) {
		return true
	}
	if len(it.buf) == cap(it.buf) {
		return false
	}
	it.buf = append(it.buf, n)
	it.cond.Signal()
	return true
}

// needsNodes reports whether the iterator is low on nodes.
func (it *lookupIterator) needsNodes() bool {
	it.mu.Lock()
	defer it.mu.Unlock()

	return len(it.buf) < lookupIteratorBuffer/3
}
