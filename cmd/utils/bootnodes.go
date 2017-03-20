// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import "github.com/expanse-org/go-expanse/p2p/discover"

// FrontierBootNodes are the enode URLs of the P2P bootstrap nodes running on
// the Frontier network.
var FrontierBootNodes = []*discover.Node{
	// EXP/DEV Go Bootnodes
		discover.MustParseNode("enode://7f335a047654f3e70d6f91312a7cf89c39704011f1a584e2698250db3d63817e74b88e26b7854111e16b2c9d0c7173c05419aeee2d0321850227b126d8b1be3f@46.101.156.249:42786"),
		discover.MustParseNode("enode://df872f81e25f72356152b44cab662caf1f2e57c3a156ecd20e9ac9246272af68a2031b4239a0bc831f2c6ab34733a041464d46b3ea36dce88d6c11714446e06b@178.62.208.109:42786"),
		discover.MustParseNode("enode://96d3919b903e7f5ad59ac2f73c43be9172d9d27e2771355db03fd194732b795829a31fe2ea6de109d0804786c39a807e155f065b4b94c6fce167becd0ac02383@45.55.22.34:42786"),
		discover.MustParseNode("enode://5f6c625bf287e3c08aad568de42d868781e961cbda805c8397cfb7be97e229419bef9a5a25a75f97632787106bba8a7caf9060fab3887ad2cfbeb182ab0f433f@46.101.182.53:42786"),
		discover.MustParseNode("enode://d33a8d4c2c38a08971ed975b750f21d54c927c0bf7415931e214465a8d01651ecffe4401e1db913f398383381413c78105656d665d83f385244ab302d6138414@128.199.183.48:42786"),
		discover.MustParseNode("enode://df872f81e25f72356152b44cab662caf1f2e57c3a156ecd20e9ac9246272af68a2031b4239a0bc831f2c6ab34733a041464d46b3ea36dce88d6c11714446e06b@178.62.208.109:42786"),
		discover.MustParseNode("enode://f6f0d6b9b7d02ec9e8e4a16e38675f3621ea5e69860c739a65c1597ca28aefb3cec7a6d84e471ac927d42a1b64c1cbdefad75e7ce8872d57548ddcece20afdd1@159.203.64.95:42786"),
}

// TestNetBootNodes are the enode URLs of the P2P bootstrap nodes running on the
// Morden test network.
var TestNetBootNodes = []*discover.Node{
	// ETH/DEV Go Bootnodes
	discover.MustParseNode("enode://e4533109cc9bd7604e4ff6c095f7a1d807e15b38e9bfeb05d3b7c423ba86af0a9e89abbf40bd9dde4250fef114cd09270fa4e224cbeef8b7bf05a51e8260d6b8@94.242.229.4:40404"),
	discover.MustParseNode("enode://8c336ee6f03e99613ad21274f269479bf4413fb294d697ef15ab897598afb931f56beb8e97af530aee20ce2bcba5776f4a312bc168545de4d43736992c814592@94.242.229.203:30303"),

	// ETH/DEV Cpp Bootnodes
}
