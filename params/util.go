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

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

var (
	TestNetGenesisHash = common.HexToHash("0x0cd786a2425d16f152c658316c423e6ce1181e15c3295826d7c9904cba9ce303") // Testnet genesis hash to enforce below configs on
	MainNetGenesisHash = common.HexToHash("0xd4e56740f876aef8c010b86a40d5f56745a118d0906a34e69aec8c0db1cb8fa3") // Mainnet genesis hash to enforce below configs on

	TestNetHomesteadBlock = big.NewInt(494000)  // Testnet homestead block
	MainNetHomesteadBlock = big.NewInt(1150000) // Mainnet homestead block

	TestNetHomesteadGasRepriceBlock = big.NewInt(1783000) // Testnet gas reprice block
	MainNetHomesteadGasRepriceBlock = big.NewInt(2463000) // Mainnet gas reprice block

	TestNetHomesteadGasRepriceHash = common.HexToHash("0xf376243aeff1f256d970714c3de9fd78fa4e63cf63e32a51fe1169e375d98145") // Testnet gas reprice block hash (used by fast sync)
	MainNetHomesteadGasRepriceHash = common.HexToHash("0x2086799aeebeae135c246c65021c82b4e15a2c451340993aacfd2751886514f0") // Mainnet gas reprice block hash (used by fast sync)
)
