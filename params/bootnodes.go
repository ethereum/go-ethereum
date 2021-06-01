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

import "github.com/ethereum/go-ethereum/common"

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the test network.
var TestnetBootnodes = []string{
	"enode://b2aa772d077311e827228e65cf0cd8fccf8e0a2d7d1933debde45737124140253469e47faa46fb2cc7014eaaa05828c599855cc4d7803fa76a336f44b6e67c05@54.168.253.129:30303", // kcc-testnet-node-boot-01
	"enode://223d6e031bc7ae5911dc377451641ad5807bfdcf46f0809c642f044751dfbf89de52b36ac1c379048f30a566de8bc4908d85c244b9f541599a419be97439bab1@3.216.120.238:30303",  // kcc-testnet-node-boot-02
	"enode://a12eb76fd1c7c2ae9a237b86e9357762a15a8e46c66c6f9b668acb349e937e27fd676275fe73a9e08ce1b667da59fb3ae6f8c29016d55bef17a415d65f767924@54.254.194.123:30303", // kcc-testnet-node-boot-03
	"enode://68ed7bf65b937eefc5bc9d8c08d83d2c8e7d3a5f628f0f5cc53b055d66793afcd12829854c8d9dd85754eb6badb8c05bbe2e25bdc1a004b8122571184e01540e@13.208.138.241:30303", // kcc-testnet-node-boot-04
	"enode://71bea53f03654e1c9bfdf9488c88087492a207c30129e93b5b5c89d79fcd4cbd432eee55ed12ab2cacb7b84aa24bd03c1b76ca9d95164f33b1088d64756e17a7@15.152.3.151:53370",   // kcc-testnet-node-sync-01
}

var V5Bootnodes []string

// KnownDNSNetwork returns the address of a public DNS-based node list for the given
// genesis hash and protocol. See https://github.com/ethereum/discv4-dns-lists for more
// information.
func KnownDNSNetwork(genesis common.Hash, protocol string) string {
	return ""
}
