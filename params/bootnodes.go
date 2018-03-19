// Copyright 2015 The go-etherinc Authors
// This file is part of the go-etherinc library.
//
// The go-etherinc library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-etherinc library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-etherinc library. If not, see <http://www.gnu.org/licenses/>.

package params

// MainnetBootnodes are the enode URLs of the P2P bootstrap nodes running on
// the main Ethereum network.
var MainnetBootnodes = []string{
	// Ethereum inc, Go Bootnodes
	"enode://519db130d32d64a56379d8c93ade07bfe1355b5b1164d4b1e38bee20feedc1686c8f7c72fbb308f030f140f3a3e02ce805e7a04984a93fb8d0fa64e1099f9f2d@13.228.232.99:30103", 
	"enode://2a5b293371e6a1813351de15b5d7a210e3259e74b2db3a356e298b301fbe9dc20e0720689820e4ca96444fa9fa4a61a75d7280b70fceff2399853f543d58536c@13.229.171.102:30103",  
	"enode://27df34f774a5d4e74c4cafcef15a2fe4a07ee86b22a741bd260b6b23201c8a8b4d9d76b129c80f284b11948a90cc9673ae096c0880357871012992af81a7ebc1@13.250.151.92:30103",  
	"enode://23c049cfc57345656e1fc9924a121859723a6cc3adea62e6ddd5c15f4b04b8ed044a29cd188d7c26d798da93aa828b911d65e37914935c34f92c9d6f671b3e7b@13.229.1.39:30103",  
	"enode://93b386fa167f9b87d06e34546e5cb9cd3f153c47c432eb8161c81b0db01ff55be4f6d4fc072e5784c16106968be24b1ec25741b026917103e3db981bd8a13c35@54.153.196.155:30103",  
	"enode://0d6a4d6f9864c8baced942536204dec865464a91b5b9d4fe6642c7eb934b4419524a5f69360922ae8a7e029a351b0ee06db8dfec2ce7e2ff60a1092a19f9cadf@54.252.194.96:30103",  
	"enode://3e82df78848c0380023cae171a3c80337cfd0248b8301e536f4a4e746535fb2e6c79850017751376eabeaa9afe86965da5f9fa53ff13116e802d1ced4f105bd4@13.55.88.217:30103", 
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	"enode://425c5aea21effeaad0fae8296625cb0d38671618cb8f6f63990fd67c12114db89ad0ffde7a739bc27b4d085469bd7d236b6d505aab50d08cd034496ebf118fd2@13.250.220.4:30103", 
	"enode://ed921763675e39249426a1741804a778b195a974da420c49dbfe54113528bb81f2736b535ee7481fd0d1155aa401016577011c0c780f21693baaff8ecc55a9c1@54.153.222.128:30103", 
	"enode://75673d1da98c59d515b6ca8c6ef7ede3bb12601e7e01a738afef1ac0c02af9c4fa4ff6eeb6c6ce7c42dbd63b3877ba0dd66db67c1d41c7c5eace4c5f4ef41000@13.211.100.173:30103", 
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{}
