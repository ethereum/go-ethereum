// Copyright 2016 The go-ethereum Authors
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

// Contains wrappers for the p2p package.

package geth

import (
	"errors"

	"github.com/ethereum/go-ethereum/p2p"
)

// NodeInfo represents pi short summary of the information known about the host.
type NodeInfo struct {
	info *p2p.NodeInfo
}

func (ni *NodeInfo) GetID() string              { return ni.info.ID }
func (ni *NodeInfo) GetName() string            { return ni.info.Name }
func (ni *NodeInfo) GetEnode() string           { return ni.info.Enode }
func (ni *NodeInfo) GetIP() string              { return ni.info.IP }
func (ni *NodeInfo) GetDiscoveryPort() int      { return ni.info.Ports.Discovery }
func (ni *NodeInfo) GetListenerPort() int       { return ni.info.Ports.Listener }
func (ni *NodeInfo) GetListenerAddress() string { return ni.info.ListenAddr }
func (ni *NodeInfo) GetProtocols() *Strings {
	protos := []string{}
	for proto := range ni.info.Protocols {
		protos = append(protos, proto)
	}
	return &Strings{protos}
}

// PeerInfo represents pi short summary of the information known about pi connected peer.
type PeerInfo struct {
	info *p2p.PeerInfo
}

func (pi *PeerInfo) GetID() string            { return pi.info.ID }
func (pi *PeerInfo) GetName() string          { return pi.info.Name }
func (pi *PeerInfo) GetCaps() *Strings        { return &Strings{pi.info.Caps} }
func (pi *PeerInfo) GetLocalAddress() string  { return pi.info.Network.LocalAddress }
func (pi *PeerInfo) GetRemoteAddress() string { return pi.info.Network.RemoteAddress }

// PeerInfos represents a slice of infos about remote peers.
type PeerInfos struct {
	infos []*p2p.PeerInfo
}

// Size returns the number of peer info entries in the slice.
func (pi *PeerInfos) Size() int {
	return len(pi.infos)
}

// Get returns the peer info at the given index from the slice.
func (pi *PeerInfos) Get(index int) (info *PeerInfo, _ error) {
	if index < 0 || index >= len(pi.infos) {
		return nil, errors.New("index out of bounds")
	}
	return &PeerInfo{pi.infos[index]}, nil
}
