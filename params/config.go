// Copyright 2016 The go-ethereum Authors
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

// ChainConfig is the core config which determines the blockchain settings.
//
// ChainConfig is stored in the database on a per block basis. This means
// that any network, identified by its genesis block, can have its own
// set of configuration options.
type ChainConfig struct {
	HomesteadBlock *big.Int `json:"homesteadBlock"` // Homestead switch block (nil = no fork, 0 = already homestead)
	DAOForkBlock   *big.Int `json:"daoForkBlock"`   // TheDAO hard-fork switch block (nil = no fork)
	DAOForkSupport bool     `json:"daoForkSupport"` // Whether the nodes supports or opposes the DAO hard-fork

	// EIP150 implements the Gas price changes (https://github.com/ethereum/EIPs/issues/150)
	EIP150Block *big.Int    `json:"EIP150Block"` // EIP150 HF block (nil = no fork)
	EIP150Hash  common.Hash `json:"EIP150Hash"`  // EIP150 HF hash (fast sync aid)

	EIP158Block *big.Int `json:"EIP158Block"` // EIP158 HF block
}

var TestChainConfig = &ChainConfig{new(big.Int), new(big.Int), true, new(big.Int), common.Hash{}, new(big.Int)}

// IsHomestead returns whether num is either equal to the homestead block or greater.
func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	if c.HomesteadBlock == nil || num == nil {
		return false
	}
	return num.Cmp(c.HomesteadBlock) >= 0
}

// GasTable returns the gas table corresponding to the current phase (homestead or homestead reprice).
//
// The returned GasTable's fields shouldn't, under any circumstances, be changed.
func (c *ChainConfig) GasTable(num *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}

	switch {
	case c.EIP158Block != nil && num.Cmp(c.EIP158Block) >= 0:
		return GasTableEIP158
	case c.EIP150Block != nil && num.Cmp(c.EIP150Block) >= 0:
		return GasTableHomesteadGasRepriceFork
	default:
		return GasTableHomestead
	}
}

func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	if c.EIP150Block == nil || num == nil {
		return false
	}
	return num.Cmp(c.EIP150Block) >= 0

}

func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	if c.EIP158Block == nil || num == nil {
		return false
	}
	return num.Cmp(c.EIP158Block) >= 0

}
