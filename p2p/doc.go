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

/*
Package p2p implements the devp2p protocol suite.

The devp2p suite is a framework for the definition of RLP-based
peer-to-peer protocols.

From Nodes to Peers and Protocols

Devp2p distinguishes nodes and peers. A node is any devp2p-capable
host participating in the network. Peers are nodes to which a
connection has been established. Nodes are identified by their Node
ID, a secp256k1 public key.

On any particular connection, one or more protocols are spoken. The
protocols understood by the local node are declared when creating the
local Server. When the connection is established, the sets of
available protocols are matched against each other and the Run
function of each protocol present on both sides is launched.

RLPx

Connections between peers use the RLPx wire protocol, which provides
an encrypted and authenticated communication channel over TCP. RLPx
supports concurrent transfer of protocol messages, ensuring that all
protocols get an equal share of the available bandwidth.

Connection Handling

Package p2p establishes peer connections automatically by selecting
randomly from the pool of all existing nodes in the network. The
connectivity graph approaches an unstructured network with low
diameter. If a protocol requires stronger connectivity properties
(e.g. when building a structured overlay), the protocol implementation
should provide information about preferred nodes on its Prefer channel.

In order to accomodate new nodes joining the network, non-preferred
peer connections may be terminated under certain conditions. Protocol
implementations are free to terminate connections at any time by
simply returning from the Run function.

Users can configure static connectivity targets through the AddPeer
method of Server. It will attempt to keep such nodes connected at all
times.
*/
package p2p
