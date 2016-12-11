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

package params

import (
	"github.com/ubiq/go-ubiq/p2p/discover"
	"github.com/ubiq/go-ubiq/p2p/discv5"
)

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ubiq network.
var MainnetBootnodes = []*discover.Node{
	// Ubiq Go Bootnodes
	discover.MustParseNode("enode://e51f1a9e4d92e71d4e4104c12447bbc0433351607e73e860bbf4340d464c798dd1b1054cd733fc7cf5279a486a38275a27f64bedb11ff5eec6d6191384f64aa5@107.191.104.192:30388"),
	discover.MustParseNode("enode://b3a58d00c799f5181ebed01df04cb0bd714cde5b87a2c00d953227c85cf96ef98235ca856646d5419d01c2b8ff688dbce8fd4e6078ac9619502f0eabe93f404c@45.63.95.155:30388"),
	discover.MustParseNode("enode://6c94a1caeef18228bdaecbef0802332566a73398d9f371f486e4ae2e7aa9a88f4e98c549dd60697d3f4ccb1ec65c0ed8e1554b274fee89da99e7358c580cc408@45.63.65.79:30388"),
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Morden test network.
var TestnetBootnodes = []*discover.Node{
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []*discv5.Node{
}
