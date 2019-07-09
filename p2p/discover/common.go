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
	"math/rand"
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

type lookupFunc func(func(*enode.Node))

// lookupWalker performs recursive lookups, walking the DHT.
// It manages a set iterators which receive lookup results as they are found.
type lookupWalker struct {
	newIterCh chan *lookupIterator
	delIterCh chan *lookupIterator
	triggerCh chan struct{}
	closeCh   chan struct{}
	wg        sync.WaitGroup

	lookup     lookupFunc
	lookupDone chan struct{}
	foundNode  chan *enode.Node
	iters      map[*lookupIterator]struct{}
}

func newLookupWalker(fn lookupFunc) *lookupWalker {
	w := &lookupWalker{
		lookup:    fn,
		iters:     make(map[*lookupIterator]struct{}),
		newIterCh: make(chan *lookupIterator),
		delIterCh: make(chan *lookupIterator),
		triggerCh: make(chan struct{}),
		closeCh:   make(chan struct{}),
		foundNode: make(chan *enode.Node),
	}
	w.wg.Add(1)
	go w.loop()
	return w
}

func (w *lookupWalker) close() {
	close(w.closeCh)
	w.wg.Wait()
}

func (w *lookupWalker) loop() {
	for {
		select {
		case it := <-w.newIterCh:
			w.iters[it] = struct{}{}
			w.startNewLookup()

		case it := <-w.delIterCh:
			delete(w.iters, it)

		case <-w.triggerCh:
			w.startNewLookup()

		case <-w.lookupDone:
			w.lookupDone = nil
			w.startNewLookup()

		case n := <-w.foundNode:
			for it := range w.iters {
				it.deliver(n)
			}

		case <-w.closeCh:
			for it := range w.iters {
				it.close()
			}
			if w.lookupDone != nil {
				<-w.lookupDone
			}
			w.wg.Done()
			return
		}
	}
}

func (w *lookupWalker) startNewLookup() {
	if w.lookupDone != nil {
		return // already running
	}
	for it := range w.iters {
		if it.needsNodes() {
			w.lookupDone = make(chan struct{})
			go w.runLookup()
			return
		}
	}
	// all iterators have full buffer
}

func (w *lookupWalker) runLookup() {
	w.lookup(func(n *enode.Node) {
		select {
		case w.foundNode <- n:
		case <-w.closeCh:
		}
	})
	w.lookupDone <- struct{}{}
}

// lookupIterator is a sequence of discovered nodes.
type lookupIterator struct {
	cur  *enode.Node
	w    *lookupWalker
	mu   sync.Mutex
	cond *sync.Cond
	buf  []*enode.Node
}

const lookupIteratorBuffer = 100

func (w *lookupWalker) newIterator() *lookupIterator {
	it := &lookupIterator{w: w, buf: make([]*enode.Node, 0, lookupIteratorBuffer)}
	it.cond = sync.NewCond(&it.mu)
	select {
	case w.newIterCh <- it:
	case <-w.closeCh:
		it.buf = nil
	}
	return it
}

func (it *lookupIterator) Next() bool {
	select {
	case it.w.triggerCh <- struct{}{}:
	case <-it.w.closeCh:
	}
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
	return true
}

func (it *lookupIterator) Node() *enode.Node {
	return it.cur
}

func (it *lookupIterator) Close() {
	select {
	case it.w.delIterCh <- it:
	case <-it.w.closeCh:
	}
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

// deliver sends a node to the iterator buffer.
func (it *lookupIterator) deliver(n *enode.Node) {
	it.mu.Lock()
	defer it.mu.Unlock()

	if len(it.buf) == lookupIteratorBuffer {
		it.buf[rand.Intn(len(it.buf))] = n
	} else {
		it.buf = append(it.buf, n)
	}
	it.cond.Signal()
}

func (it *lookupIterator) needsNodes() bool {
	it.mu.Lock()
	defer it.mu.Unlock()

	return len(it.buf) < lookupIteratorBuffer/3
}
