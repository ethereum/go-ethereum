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
	// Egem Go Bootnodes
	"enode://fba5bbff0f302105be69ca689c0724d95591e314bd87295237085dd1972f7ebe2f13a231c5a1817a7b1fca6500c1ab56c6ead2de34db3d849a4035b0e7e4a07f@154.20.195.166:30303", // NW CANADA
	"enode://f90107f8efc23e2e38553b6c194cfbe2f0e0af29e79c99b828b38168eeb02a06b37c3b205e085b48d182d4c644a7ec42d951286d7a416f70a16966335fdc4f7a@[45.77.210.216]:30303", // N0 Seattle
	"enode://735dcc50b9ab58d0ffcd3d7ba44c9ab0187ca296f10f90b9667893ffe989846648d82edfb7fe1cfa9a26b450747c142699996d8696fe4f0b8dd3d53f14bfd6a1@[144.202.88.155]:30303", //N1 Seattle
	"enode://0c2460ed19224fef474dcd18d02e06a8eef52d847ac1f3de9ed22a32582460a7836a9c29b687d01e9a53474e4ec96bd9a307f16c0475121152ab4644ea96f8dd@[144.202.87.223]:30303", //N2 Seattle
	"enode://9e83b1c845805d012c96310eaac4debc8b0590753ed3cc51c328aecaca73e7931427756163f47d46f174445c8d73260f3a5e74f008e340d551661cb3756c5655@[209.222.30.122]:30303", //N3 London
	"enode://fbb4b509a419b5db20405e2ce8b36eecd4d2fb9ebae7c87a6ef9ee68a074a58f6fa87dc08376635dfb8d23c72b2afaf078cb465c825c40be1455d15490cd4966@144.202.101.110:30303", // MakeMoneyOz Sil valley
	"enode://8fb089d66eb948048cc91c8588c316d29e9a35f9b2cae42d3b15938bf2a0978fe2aa41185b259c7816e10b77fcf0931e2057514cb176828555bd82248c319bc4@198.13.36.85:30303", // MakeMoneyOz Japan
	"enode://b532392fcfd0244c572f8a269e7fbd6066d30b35f82ec47edb7547fe9aa5399b1a24a586feeba0357b41c164d4085907a4db938d9eac1939bd6a48e8d3820ee1@173.199.119.64:30303", // MakeMoneyOz NewYork-NJ

}

// TestnetBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Ropsten test network.
var TestnetBootnodes = []string{
	"enode://30b7ab30a01c124a6cceca36863ece12c4f5fa68e3ba9b0b51407ccc002eeed3b3102d20a88f1c1d3c3154e2449317b8ef95090e77b312d5cc39354f86d5d606@52.176.7.10:30303",    // US-Azure geth
	"enode://865a63255b3bb68023b6bffd5095118fcc13e79dcf014fe4e47e065c350c7cc72af2e53eff895f11ba1bbb6a2b33271c1116ee870f266618eadfc2e78aa7349c@52.176.100.77:30303",  // US-Azure parity
	"enode://6332792c4a00e3e4ee0926ed89e0d27ef985424d97b6a45bf0f23e51f0dcb5e66b875777506458aea7af6f9e4ffb69f43f3778ee73c81ed9d34c51c4b16b0b0f@52.232.243.152:30303", // Parity
	"enode://94c15d1b9e2fe7ce56e458b9a3b672ef11894ddedd0c6f247e0f1d3487f52b66208fb4aeb8179fce6e3a749ea93ed147c37976d67af557508d199d9594c35f09@192.81.208.223:30303", // @gpip
}

// RinkebyBootnodes are the enode URLs of the P2P bootstrap nodes running on the
// Rinkeby test network.
var RinkebyBootnodes = []string{
	"enode://a24ac7c5484ef4ed0c5eb2d36620ba4e4aa13b8c84684e1b4aab0cebea2ae45cb4d375b77eab56516d34bfbd3c1a833fc51296ff084b770b94fb9028c4d25ccf@52.169.42.101:30303", // IE
	"enode://343149e4feefa15d882d9fe4ac7d88f885bd05ebb735e547f12e12080a9fa07c8014ca6fd7f373123488102fe5e34111f8509cf0b7de3f5b44339c9f25e87cb8@52.3.158.184:30303",  // INFURA
	"enode://b6b28890b006743680c52e64e0d16db57f28124885595fa03a562be1d2bf0f3a1da297d56b13da25fb992888fd556d4c1a27b1f39d531bde7de1921c90061cc6@159.89.28.211:30303", // AKASHA
}

// DiscoveryV5Bootnodes are the enode URLs of the P2P bootstrap nodes for the
// experimental RLPx v5 topic-discovery network.
var DiscoveryV5Bootnodes = []string{
	"enode://fbb4b509a419b5db20405e2ce8b36eecd4d2fb9ebae7c87a6ef9ee68a074a58f6fa87dc08376635dfb8d23c72b2afaf078cb465c825c40be1455d15490cd4966@144.202.101.110:30303", // MakeMoneyOz Sil valley
}
