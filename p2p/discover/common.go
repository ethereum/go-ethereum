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
	"context"
	"crypto/ecdsa"
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
	lookup lookupFunc

	newIterCh chan *lookupIterator
	delIterCh chan *lookupIterator
	triggerCh chan struct{}
	closeCh   chan struct{}
	wg        sync.WaitGroup
}

func newLookupWalker(fn lookupFunc) *lookupWalker {
	w := &lookupWalker{
		lookup:    fn,
		newIterCh: make(chan *lookupIterator),
		delIterCh: make(chan *lookupIterator),
		triggerCh: make(chan struct{}),
		closeCh:   make(chan struct{}),
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
	var (
		iters      = make(map[*lookupIterator]struct{})
		foundNode  = make(chan *enode.Node)
		lookupDone = make(chan struct{}, 1)
		trigger    = w.triggerCh
	)
	for {
		select {
		case it := <-w.newIterCh:
			iters[it] = struct{}{}

		case it := <-w.delIterCh:
			delete(iters, it)

		case <-trigger:
			trigger = nil // stop listening to trigger until lookupDone
			go w.runLookup(foundNode, lookupDone)

		case <-lookupDone:
			trigger = w.triggerCh

		case n := <-foundNode:
			for it := range iters {
				it.deliver(n)
			}

		case <-w.closeCh:
			for it := range iters {
				it.drainAndClose()
			}
			if trigger == nil {
				<-lookupDone
			}
			w.wg.Done()
			return
		}
	}
}

func (w *lookupWalker) runLookup(nodes chan<- *enode.Node, done chan<- struct{}) {
	w.lookup(func(n *enode.Node) {
		select {
		case nodes <- n:
		case <-w.closeCh:
		}
	})
	done <- struct{}{}
}

// lookupIterator is a sequence of discovered nodes.
type lookupIterator struct {
	w         *lookupWalker
	buf       chan *enode.Node
	closed    bool
	closeOnce sync.Once
}

const lookupIteratorBuffer = 100

func (w *lookupWalker) newIterator() *lookupIterator {
	it := &lookupIterator{w: w, buf: make(chan *enode.Node, lookupIteratorBuffer)}
	select {
	case w.newIterCh <- it:
	case <-w.closeCh:
		it.closed = true
		close(it.buf)
	}
	return it
}

func (it *lookupIterator) NextNode(ctx context.Context) (n *enode.Node, isLive bool) {
	for {
		select {
		case it.w.triggerCh <- struct{}{}:
			// lookup triggered
		case n, ok := <-it.buf:
			if !ok {
				it.closed = true
			}
			return n, !it.closed
		case <-ctx.Done():
			return nil, !it.closed
		}
	}
}

func (it *lookupIterator) Close() {
	it.closeOnce.Do(func() {
		select {
		case it.w.delIterCh <- it:
		case <-it.w.closeCh:
		}
		it.drainAndClose()
	})
}

// deliver sends a node to the iterator buffer.
func (it *lookupIterator) deliver(n *enode.Node) {
	// We don't want deliver to block and replace stale results when they're not being
	// read. Check whether the buffer is full and allow one receive from the buffer if so.
	// This is OK because there is only one writer.
	var remove chan *enode.Node
	if len(it.buf) == cap(it.buf) {
		remove = it.buf
	}
	for {
		select {
		case it.buf <- n:
			return
		case <-remove:
			remove = nil
		}
	}
}

func (it *lookupIterator) drainAndClose() {
	for len(it.buf) > 0 {
		<-it.buf
	}
	close(it.buf)
}
