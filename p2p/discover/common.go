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

type UDPConn interface {
	ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error)
	Close() error
	LocalAddr() net.Addr
}

// Config holds Table-related settings.
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
// channel if configured.
type ReadPacket struct {
	Data []byte
	Addr *net.UDPAddr
}

type lookupFunc func(func(*enode.Node))

// lookupWalker performs recursive lookups, walking the DHT.
// It manages a set iterators which receive lookup results in real time.
type lookupWalker struct {
	lookup lookupFunc

	newIterCh chan *Iterator
	delIterCh chan *Iterator
	triggerCh chan struct{}
	closeCh   chan struct{}
	wg        sync.WaitGroup
}

func newLookupWalker(fn lookupFunc) *lookupWalker {
	w := &lookupWalker{
		lookup:    fn,
		newIterCh: make(chan *Iterator),
		delIterCh: make(chan *Iterator),
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
		iters      = make(map[*Iterator]struct{})
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
				close(it.buf)
			}
			w.wg.Done()
			return
		}
	}
}

func (w *lookupWalker) runLookup(nodes chan<- *enode.Node, done chan struct{}) {
	w.lookup(func(n *enode.Node) {
		select {
		case nodes <- n:
		case <-w.closeCh:
		}
	})
	done <- struct{}{}
}

// Iterator is a sequence of discovered nodes.
type Iterator struct {
	w      *lookupWalker
	buf    chan *enode.Node
	closed bool
}

const lookupIteratorBuffer = 100

func (w *lookupWalker) newIterator() *Iterator {
	it := &Iterator{w, make(chan *enode.Node, lookupIteratorBuffer), false}
	select {
	case w.newIterCh <- it:
	case <-w.closeCh:
		it.closed = true
		close(it.buf)
	}
	return it
}

// NextNode retrieves the next node if one could be discovered before the passed context
// was canceled. This triggers a lookup operation if none is running. The isLive return
// value says whether the iterator is still open. NextNode returns (nil, false) after Close
// has been called.
//
// NextNode is not safe for concurrent use.
func (it *Iterator) NextNode(ctx context.Context) (n *enode.Node, isLive bool) {
	for {
		select {
		case it.w.triggerCh <- struct{}{}:
			// lookup triggered
		case <-it.w.closeCh:
			it.closed = true
			return nil, false
		case n, ok := <-it.buf:
			if !ok {
				it.closed = true
			}
			return n, it.closed
		case <-ctx.Done():
			return nil, it.closed // TODO: should be permanently closed if channel is closed once.
		}
	}
}

// Close ends the iterator. This can be called concurrently with NextNode.
func (it *Iterator) Close() {
	select {
	case it.w.delIterCh <- it:
		close(it.buf)
	case <-it.w.closeCh:
	}
}

// deliver sends n to the iterator buffer.
func (it *Iterator) deliver(n *enode.Node) {
	// We don't want deliver to block and replacing stale results is OK if they're not
	// being read fast enough. Check whether the buffer is full and enable the select case
	// which removes an element if so. This doesn't race because deliver is only called by
	// a single goroutine at a time.
	remove := it.buf
	if len(it.buf) < cap(it.buf) {
		remove = nil
	}
	select {
	case it.buf <- n:
		return
	case <-remove:
	}
}
