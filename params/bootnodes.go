// Copyright 2015 The go-etherfact Authors
// This file is part of the go-etherfact library.
//
// The go-etherfact library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-etherfact library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-etherfact library. If not, see <http://www.gnu.org/licenses/>.

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	"enode://1f50ce38fd1b4111b2b34db984f1cb0f743da71f8dcc46e58bc04737d8315470d6eaf4d079ef11bf1dc99eb7fc86709963e2515d3a934779bf2a0342d1401341@199.168.139.155:30303",    // US-1
	"enode://02ccc8cd20a29e2076c29068ce422b287803d448a29715a0bf53f98bc8bb988a5612d5c72fa5f9b69eb98606758de94a85c4c894935daa4e65811a3bb9163e79@199.168.139.152:30303",   // US-2
	"enode://baef26dfb307d86333a15fc3023a6177d29b56b89e5f0dbe91aaa15af6419cb78500654dd1adbde035a2d90b00d0cd35ac414e5576d4c1e3a46cde7042a3e01f@199.168.139.158:30303",  // US-3
	"enode://b5048b516561b8ac68a6a235aeecfaf73c1835403999dc5d30dd360ebaadbc7999e41fbd2887abda5cb62f0951a2baea7fb3821b50cf33c308a7fc8702214e8b@104.245.98.115:30303",  // US-4

	// Ethereum Foundation C++ Bootnodes
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
}
