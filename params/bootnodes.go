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
// Ubiq test network.
var TestnetBootnodes = []string{
	"enode://81b11410a96e0ea6ecc927f7714a2c256c13e200bc73d087ad120e5a3fc3e1e098760c6fae3dcd7a3c393e49c205e636bacfc10adf6581672f6d3a66e2442248@45.77.7.41:30388",
	"enode://0595ec507bb779873703f516072b37d07f3305271da3d9585ada3b1734535635eac50cffd8c9a413b87a77ede5f49af391a08ca9348d027fda74c38f1ea5ec91@108.61.188.12:30388",
	"enode://8cb060312b4667ed6a0f61dd6cc0cd5d39e70c17429cd5e8ca480fcd7caf72f1b9c92884ce1f8e06e84a7ed1580ba302df0e95ec2ce99f727297bd2787ed8149@45.76.90.144:30388",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{}
