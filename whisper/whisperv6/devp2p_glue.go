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
	"github.com/ethereum/go-ethereum/p2p"
)

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
