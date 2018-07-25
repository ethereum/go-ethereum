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
	"enode://e68e5e6e1a27c1191c09ca3b05fe4e391cfb9648e00c6d085ba4b48931345636bc76117282c2155838d98f63d03994bb88ea9e8b8ecc254da8077398af1c6710@104.156.230.85:30388",
	"enode://f0862b1210672c50f32ec7827159aedd16c8790f64083a5830662e853abb04771ff79d88b2165da8741908aff7ded653e4419f0959f52be607c15b76b318f562@45.76.112.217:30388",
	"enode://3c50be8974756f304ac0195a2a11f9b5ba826354c8617d4b58da21a36102928ddecd96395c7227e9dd1409110ec1414d25b1cfe7f9e4b40732c507d605a7b2b9@45.32.179.15:30388",
	"enode://b920d3f9d2a3333dba7d2df8c655974ade36e435bca315ca8b9b307b0f4b4b8aae1004759755e8fc63d4b4d0e5a5452137896de9c141fdb244a65800c26d9df5@104.207.131.9:30388",
	"enode://2b9e41dc5a86f398111cb6b51d81005dece8b0f67c2560adae4bc3e2c54c3dab0db259300a4224b6862fae4a6aa38ad56df7cd0363e396697fdbe3ea4e3ea0c5@45.32.253.23:30388",
	"enode://966f1895b085bf7fdad648afed684b79de9e030a7303c1ebd2acae436e69d754e8d5d35238a08112fd049066c0d310d71ca61e94c16ec0dda4336c065674604c@45.32.117.58:30388",
	"enode://7435d85612144c7777f3eaf14dd754c35cbb97caad364add8c4eeb9bba00bd9ce15fe256ec8c674b3975f87c23ee7a48472c79d19e1d0feb432f3aceb35ab0cf@107.191.104.192:30388",
	"enode://b902a1538d5bbd6c676676c139e9470fcd942e0d299f5db8bd8ea690af9035f696fe3d88118fece4f74949beb4cf2ba9c3437a002f9a9d08e2b4bfc58fac490f@107.191.104.97:30388",
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
