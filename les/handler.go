// Copyright 2022 The go-ethereum Authors
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

package les

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p"
)

// handshakeModule defines methods to be called when a peer is performing handshake
type handshakeModule interface {
	sendHandshake(*peer, *keyValueList)
	receiveHandshake(*peer, keyValueMap) error
}

// handshakeModule defines methods to be called when a peer is connected or disconnected
type connectionModule interface {
	peerConnected(*peer) (func(), error)
}

// messageHandlerModule defines protocol message handlers for certain message codes and protocol versions
type messageHandlerModule interface {
	messageHandlers() messageHandlers
}

// auxModule defines a method for any auxiliary tasks to be performed at handler start/shutdown
type auxModule interface {
	start(wg *sync.WaitGroup, closeCh chan struct{})
}

// messageHandler is a handler function for a specific message type
type messageHandler func(*peer, p2p.Msg) error

// messageHandlerWithCodeAndVersion defines a handler together with the message code and protocol version range
type messageHandlerWithCodeAndVersion struct {
	code                      uint64
	firstVersion, lastVersion int
	handler                   messageHandler
}

// messageHandlers lists a set of message handlers defined by a handler module
type messageHandlers []messageHandlerWithCodeAndVersion

type codeAndVersion struct {
	code    uint64
	version int
}

// handler is a modular protocol handler framework
type handler struct {
	handshakeModules  []handshakeModule
	connectionModules []connectionModule
	messageHandlers   map[codeAndVersion]messageHandler
	auxModules        []auxModule
	networkId         uint64

	peers   *peerSet
	closeCh chan struct{}
	wg      sync.WaitGroup
}

// newHandler creates a new protocol handler (empty, without modules)
func newHandler(peers *peerSet, networkId uint64) *handler {
	return &handler{
		peers:           peers,
		networkId:       networkId,
		closeCh:         make(chan struct{}),
		messageHandlers: make(map[codeAndVersion]messageHandler),
	}
}

// start starts the protocol handler (modules should be registered before starting)
func (h *handler) start() {
	for _, m := range h.auxModules {
		m.start(&h.wg, h.closeCh)
	}
}

// stop stops the protocol handler.
func (h *handler) stop() {
	close(h.closeCh)
	h.peers.close() // h.wg.Add is called after successful h.peers.register so it cannot happen after this line
	h.wg.Wait()
}

// registerModule registers a new protocol handler module. This module can implement any
// subset of the handler module interfaces.
func (h *handler) registerModule(module interface{}) {
	if m, ok := module.(handshakeModule); ok {
		h.handshakeModules = append(h.handshakeModules, m)
	}
	if m, ok := module.(connectionModule); ok {
		h.connectionModules = append(h.connectionModules, m)
	}
	if m, ok := module.(auxModule); ok {
		h.auxModules = append(h.auxModules, m)
	}
	if m, ok := module.(messageHandlerModule); ok {
		for _, hh := range m.messageHandlers() {
			for version := hh.firstVersion; version <= hh.lastVersion; version++ {
				h.messageHandlers[codeAndVersion{code: hh.code, version: version}] = hh.handler
			}
		}
	}
}

// runPeer is the p2p protocol run function for the given version.
func (h *handler) runPeer(version uint, p *p2p.Peer, rw p2p.MsgReadWriter) error {
	peer := newPeer(int(version), h.networkId, p, newMeteredMsgWriter(rw, int(version)))
	defer peer.close()
	return h.handle(peer)
}

// handle handles a recently connected peer
func (h *handler) handle(p *peer) error {
	p.Log().Debug("Light Ethereum peer connected", "name", p.Name())

	p.connectedAt = mclock.Now()
	if err := p.handshake(h.handshakeModules); err != nil {
		p.Log().Debug("Light Ethereum handshake failed", "err", err)
		return err
	}

	var discFns []func()
	defer func() {
		for i := len(discFns) - 1; i >= 0; i-- {
			discFns[i]()
		}
	}()

	for _, m := range h.connectionModules {
		discFn, err := m.peerConnected(p)
		if discFn != nil {
			discFns = append(discFns, discFn)
		}
		if err != nil {
			return err
		}
	}

	// Register the peer locally
	if err := h.peers.register(p); err != nil {
		p.Log().Error("Light Ethereum peer registration failed", "err", err)
		return err
	}

	h.wg.Add(1)
	defer func() {
		p.wg.Wait()
		h.peers.unregister(p.ID())
		connectionTimer.Update(time.Duration(mclock.Now() - p.connectedAt))
		h.wg.Done()
	}()

	for {
		select {
		case err := <-p.errCh:
			p.Log().Debug("Protocol handler error", "err", err)
			return err
		default:
		}
		if err := h.handleMsg(p); err != nil {
			p.Log().Debug("Light Ethereum message handling failed", "err", err)
			return err
		}
	}
}

// handleMsg is invoked whenever an inbound message is received from a remote
// peer. The remote connection is torn down upon returning any error.
func (h *handler) handleMsg(p *peer) error {
	// Read the next message from the remote peer, and ensure it's fully consumed
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	defer msg.Discard()
	p.Log().Trace("Light Ethereum message arrived", "code", msg.Code, "bytes", msg.Size)
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}

	if handler, ok := h.messageHandlers[codeAndVersion{code: msg.Code, version: p.version}]; ok {
		return handler(p, msg)
	} else {
		p.Log().Trace("Received invalid message", "code", msg.Code, "protocolVersion", p.version)
		return errResp(ErrInvalidMsgCode, "code: %v  protocolVersion: %v", msg.Code, p.version)
	}
}
