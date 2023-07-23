// Copyright 2023 The go-ethereum Authors
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

package supply

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

// Supply crawls the state snapshot at a given header and gathers all the account
// balances to sum into the total Ether supply.
func Supply(header *types.Header, snaptree *snapshot.Tree) (*big.Int, error) {
	accIt, err := snaptree.AccountIterator(header.Root, common.Hash{})
	if err != nil {
		return nil, err
	}
	defer accIt.Release()

	log.Info("Ether supply counting started", "block", header.Number, "hash", header.Hash(), "root", header.Root)
	var (
		start    = time.Now()
		logged   = time.Now()
		accounts uint64
	)
	supply := big.NewInt(0)
	for accIt.Next() {
		account, err := types.FullAccount(accIt.Account())
		if err != nil {
			return nil, err
		}
		supply.Add(supply, account.Balance)
		accounts++
		if time.Since(logged) > 8*time.Second {
			log.Info("Ether supply counting in progress", "at", accIt.Hash(),
				"accounts", accounts, "supply", supply, "elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
	}
	log.Info("Ether supply counting complete", "block", header.Number, "hash", header.Hash(), "root", header.Root,
		"accounts", accounts, "supply", supply, "elapsed", common.PrettyDuration(time.Since(start)))

	return supply, nil
}
