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

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ubiq network.
var MainnetBootnodes = []string{
	// Ubiq Go Bootnodes
	"enode://e51f1a9e4d92e71d4e4104c12447bbc0433351607e73e860bbf4340d464c798dd1b1054cd733fc7cf5279a486a38275a27f64bedb11ff5eec6d6191384f64aa5@107.191.104.192:30388",
	"enode://b3a58d00c799f5181ebed01df04cb0bd714cde5b87a2c00d953227c85cf96ef98235ca856646d5419d01c2b8ff688dbce8fd4e6078ac9619502f0eabe93f404c@45.63.95.155:30388",
	"enode://6c94a1caeef18228bdaecbef0802332566a73398d9f371f486e4ae2e7aa9a88f4e98c549dd60697d3f4ccb1ec65c0ed8e1554b274fee89da99e7358c580cc408@45.63.65.79:30388",
	"enode://21c70554811047ed6fe2314cfd5500808d07e1ebd34cde073e2c18e66ea90112c9f7e212b2b145ada89625264f2f44bc762b61b6dfc9efacf0a3ed67b59c496f@159.203.0.101:30388",
	"enode://f293b3a51bc42d48c8c9cb57954b0d4db1cc1b3e1582d1dfb8865bbd386bc874122e04ce47868f1ec386839a2661f3158071b8d603164cb6e7b9fc9901aed4a3@104.168.87.91:30388",
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Morden test network.
var TestnetBootnodes = []string{}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{}
