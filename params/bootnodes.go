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
// the main Ethereum network.
var MainnetBootnodes = []string{
<<<<<<< HEAD
	// EXP/DEV Go Bootnodes
	"enode://7f335a047654f3e70d6f91312a7cf89c39704011f1a584e2698250db3d63817e74b88e26b7854111e16b2c9d0c7173c05419aeee2d0321850227b126d8b1be3f@46.101.156.249:42786",
	"enode://df872f81e25f72356152b44cab662caf1f2e57c3a156ecd20e9ac9246272af68a2031b4239a0bc831f2c6ab34733a041464d46b3ea36dce88d6c11714446e06b@178.62.208.109:42786",
	"enode://96d3919b903e7f5ad59ac2f73c43be9172d9d27e2771355db03fd194732b795829a31fe2ea6de109d0804786c39a807e155f065b4b94c6fce167becd0ac02383@45.55.22.34:42786",
	"enode://5f6c625bf287e3c08aad568de42d868781e961cbda805c8397cfb7be97e229419bef9a5a25a75f97632787106bba8a7caf9060fab3887ad2cfbeb182ab0f433f@46.101.182.53:42786",
	"enode://d33a8d4c2c38a08971ed975b750f21d54c927c0bf7415931e214465a8d01651ecffe4401e1db913f398383381413c78105656d665d83f385244ab302d6138414@128.199.183.48:42786",
	"enode://df872f81e25f72356152b44cab662caf1f2e57c3a156ecd20e9ac9246272af68a2031b4239a0bc831f2c6ab34733a041464d46b3ea36dce88d6c11714446e06b@178.62.208.109:42786",
	"enode://f6f0d6b9b7d02ec9e8e4a16e38675f3621ea5e69860c739a65c1597ca28aefb3cec7a6d84e471ac927d42a1b64c1cbdefad75e7ce8872d57548ddcece20afdd1@159.203.64.95:42786",
=======

	// Ethereum Foundation Go Bootnodes
	"enode://a979fb575495b8d6db44f750317d0f4622bf4c2aa3365d6af7c284339968eef29b69ad0dce72a4d8db5ebb4968de0e3bec910127f134779fbcb0cb6d3331163c@52.16.188.185:30303", // IE
	"enode://3f1d12044546b76342d59d4a05532c14b85aa669704bfe1f864fe079415aa2c02d743e03218e57a33fb94523adb54032871a6c51b2cc5514cb7c7e35b3ed0a99@13.93.211.84:30303",  // US-WEST
	"enode://78de8a0916848093c73790ead81d1928bec737d565119932b98c6b100d944b7a95e94f847f689fc723399d2e31129d182f7ef3863f2b4c820abbf3ab2722344d@191.235.84.50:30303", // BR
	"enode://158f8aab45f6d19c6cbf4a089c2670541a8da11978a2f90dbf6a502a4a3bab80d288afdbeb7ec0ef6d92de563767f3b1ea9e8e334ca711e9f8e2df5a0385e8e6@13.75.154.138:30303", // AU
	"enode://1118980bf48b0a3640bdba04e0fe78b1add18e1cd99bf22d53daac1fd9972ad650df52176e7c7d89d1114cfef2bc23a2959aa54998a46afcf7d91809f0855082@52.74.57.123:30303",  // SG

	// Ethereum Foundation Cpp Bootnodes
	"enode://979b7fa28feeb35a4741660a16076f1943202cb72b6af70d327f053e248bab9ba81760f39d0701ef1d8f89cc1fbd2cacba0710a12cd5314d5e0c9021aa3637f9@5.1.83.226:30303", // DE

>>>>>>> refs/remotes/ethereum/master
}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Morden test network.
var TestnetBootnodes = []string{
	"enode://e4533109cc9bd7604e4ff6c095f7a1d807e15b38e9bfeb05d3b7c423ba86af0a9e89abbf40bd9dde4250fef114cd09270fa4e224cbeef8b7bf05a51e8260d6b8@94.242.229.4:40404",
	"enode://8c336ee6f03e99613ad21274f269479bf4413fb294d697ef15ab897598afb931f56beb8e97af530aee20ce2bcba5776f4a312bc168545de4d43736992c814592@94.242.229.203:42786",
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{

}
