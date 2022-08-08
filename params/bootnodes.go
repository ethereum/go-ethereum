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

import "github.com/daefrom/go-dae/common"

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	// Ethereum Foundation Go Bootnodes
	"enode://ce3d7c55a2b799f42e15d3c15e4d51b0458be366751bd1a2b7110e9bea8413680396f319b4fdff097ce6cf9f6f064b9346dd94d30baecaf233d4f7ad8479a556@176.102.65.246:32323",  // bootnode - bohemia1
	"enode://7f3b7ec4a3ee20a097667d09c1717b2463c0d066b194a769279db65d56306d3c5e811d868d84726149d8602ffbb117afa81cdac5ff4b8fce3982687d7532d3c6@89.103.124.2:30303",    // bootnode - bohemia2
	"enode://3d6d105c172c7ae42ec7b028c2d4bca65d58582a6516e82e0ca0e0bc910481e5244e3843776dc08ea0222ad16ae690a87d2cb832f9711bbdc8bb4c655dcf89cf@167.235.26.120:32323",  // bootnode - hetzner2
	"enode://50d34136cb9bc2e35161329d629d49a2ce0770d6d3f17570958669b890e6b5430dde6b2c2d70b33e44ef033af0926302ee8c434f88cfedba79c4ca037e7e2d68@209.250.246.177:30303", // bootnode - amsterdam
}

// RopstenBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var RopstenBootnodes = []string{}

// SepoliaBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Sepolia test network.
var SepoliaBootnodes = []string{}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{}

// GoerliBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Görli test network.
var GoerliBootnodes = []string{}

var KilnBootnodes = []string{}

var V5Bootnodes = []string{}

const dnsPrefix = ""

// KnownDNSNetwork returns the address of a public DNS-based node list for the given
// genesis hash and protocol. See https://github.com/ethereum/discv4-dns-lists for more
// information.
func KnownDNSNetwork(genesis common.Hash, protocol string) string {
	var net string
	switch genesis {
	case MainnetGenesisHash:
		net = "mainnet"
	case RopstenGenesisHash:
		net = "ropsten"
	case RinkebyGenesisHash:
		net = "rinkeby"
	case GoerliGenesisHash:
		net = "goerli"
	case SepoliaGenesisHash:
		net = "sepolia"
	default:
		return ""
	}
	return dnsPrefix + protocol + "." + net + ".ethdisco.net"
}
