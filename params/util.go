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
	TestNetHomesteadBlock = big.NewInt(494000)  // Testnet homestead block
	MainNetHomesteadBlock = big.NewInt(1150000) // Mainnet homestead block

	TestNetDAOForkBlock = big.NewInt(8888888)                            // Testnet dao hard-fork block
	MainNetDAOForkBlock = big.NewInt(9999999)                            // Mainnet dao hard-fork block
	DAOForkBlockExtra   = common.FromHex("0x64616f2d686172642d666f726b") // Block extradata to signel the fork with ("dao-hard-fork")
	DAOForkExtraRange   = big.NewInt(10)                                 // Number of blocks to override the extradata (prevent no-fork attacks)
)
