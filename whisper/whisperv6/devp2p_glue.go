// Copyright 2018 The go-ethereum Authors
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

package whisperv6

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	set "gopkg.in/fatih/set.v0"
	"time"
)

// DevP2PPeer is the DevP2P implementation of the Peer interface
type DevP2PPeer struct {
	*PeerBase

	peer *p2p.Peer
}

// newPeer creates a new whisper peer object, but does not run the handshake itself.
func newPeer(host *Whisper, remote *p2p.Peer, rw p2p.MsgReadWriter) Peer {
	return &DevP2PPeer{
		&PeerBase{
			host:           host,
			ws:             rw,
			trusted:        false,
			powRequirement: 0.0,
			known:          set.New(),
			quit:           make(chan struct{}),
			bloomFilter:    MakeFullNodeBloom(),
			fullNode:       true,
		},
		remote,
	}
}

// handshake sends the protocol initiation status message to the remote peer and
// verifies the remote status too.
func (peer *DevP2PPeer) handshake() error {
	err := peer.handshakeBase()
	if err != nil {
		return fmt.Errorf("peer [%x] %s", peer.ID(), err.Error())
	}
	return nil
}

// start initiates the peer updater, periodically broadcasting the whisper packets
// into the network.
func (peer *DevP2PPeer) start() {
	go peer.update()
	log.Trace("start", "peer", peer.ID())
}

// stop terminates the peer updater, stopping message forwarding to it.
func (peer *DevP2PPeer) stop() {
	close(peer.quit)
	log.Trace("stop", "peer", peer.ID())
}

// update executes periodic operations on the peer, including message transmission
// and expiration.
func (peer *DevP2PPeer) update() {
	// Start the tickers for the updates
	expire := time.NewTicker(expirationCycle)
	transmit := time.NewTicker(transmissionCycle)

	// Loop and transmit until termination is requested
	for {
		select {
		case <-expire.C:
			peer.expire()

		case <-transmit.C:
			if err := peer.broadcast(); err != nil {
				log.Trace("broadcast failed", "reason", err)
				return
			}

		case <-peer.quit:
			return
		}
	}
}

// DevP2PWhisperServer implements WhisperServer with a DevP2P backend
type DevP2PWhisperServer struct {
	Server *p2p.Server
}

// Start starts the server
func (server *DevP2PWhisperServer) Start() error {
	return server.Server.Start()
}

// Stop stops the server
func (server *DevP2PWhisperServer) Stop() {
	server.Server.Stop()
}

// PeerCount returns the peer count for the node
func (server *DevP2PWhisperServer) PeerCount() int {
	return server.Server.PeerCount()
}

// Enode returns the enode address of the node
func (server *DevP2PWhisperServer) Enode() string {
	return server.Server.NodeInfo().Enode
}
