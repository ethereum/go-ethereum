// Copyright 2019 The Swarm Authors
// This file is part of the Swarm library.
//
// The Swarm library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Swarm library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Swarm library. If not, see <http://www.gnu.org/licenses/>.

package newstream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethersphere/swarm/chunk"
	"github.com/ethersphere/swarm/network"
	"github.com/ethersphere/swarm/network/bitvector"
	"github.com/ethersphere/swarm/state"
)

var ErrEmptyBatch = errors.New("empty batch")

const (
	HashSize  = 32
	BatchSize = 16
)

// Peer is the Peer extension for the streaming protocol
type Peer struct {
	*network.BzzPeer
	mtx            sync.Mutex
	providers      map[string]StreamProvider
	intervalsStore state.Store

	streamCursorsMu   sync.Mutex
	streamCursors     map[string]uint64 // key: Stream ID string representation, value: session cursor. Keeps cursors for all streams. when unset - we are not interested in that bin
	dirtyStreams      map[string]bool   // key: stream ID, value: whether cursors for a stream should be updated
	activeBoundedGets map[string]chan struct{}
	openWants         map[uint]*want // maintain open wants on the client side
	openOffers        map[uint]offer // maintain open offers on the server side
	quit              chan struct{}  // closed when peer is going offline
}

// NewPeer is the constructor for Peer
func NewPeer(peer *network.BzzPeer, i state.Store, providers map[string]StreamProvider) *Peer {
	p := &Peer{
		BzzPeer:        peer,
		providers:      providers,
		intervalsStore: i,
		streamCursors:  make(map[string]uint64),
		dirtyStreams:   make(map[string]bool),
		openWants:      make(map[uint]*want),
		openOffers:     make(map[uint]offer),
		quit:           make(chan struct{}),
	}
	return p
}
func (p *Peer) Left() {
	close(p.quit)
}

// HandleMsg is the message handler that delegates incoming messages
func (p *Peer) HandleMsg(ctx context.Context, msg interface{}) error {
	switch msg := msg.(type) {
	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
	return nil
}

type offer struct {
	ruid      uint
	stream    ID
	hashes    []byte
	requested time.Time
}

type want struct {
	ruid      uint
	from      uint64
	to        uint64
	stream    ID
	hashes    map[string]bool
	bv        *bitvector.BitVector
	requested time.Time
	remaining uint64
	chunks    chan chunk.Chunk
	done      chan error
}
