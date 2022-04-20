// Copyright 2022 The go-ethereum Authors
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

package supplydelta

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// SupplyDelta calculates the Ether delta across two state tries. That is, the
// issuance minus the ETH destroyed.
func SupplyDelta(block *types.Block, parent *types.Header, db *trie.Database, config *params.ChainConfig) (*big.Int, error) {
	var (
		supplyDelta = new(big.Int)
		start       = time.Now()
	)
	// Open the two tries.
	if block.ParentHash() != parent.Hash() {
		return nil, fmt.Errorf("parent hash mismatch: have %s, want %s", block.ParentHash().Hex(), parent.Hash().Hex())
	}
	src, err := trie.New(trie.StateTrieID(parent.Root), db)
	if err != nil {
		return nil, fmt.Errorf("failed to open source trie: %v", err)
	}
	dst, err := trie.New(trie.StateTrieID(block.Root()), db)
	if err != nil {
		return nil, fmt.Errorf("failed to open destination trie: %v", err)
	}
	// Gather all the changes across from source to destination.
	fwdDiffIt, _ := trie.NewDifferenceIterator(src.MustNodeIterator(nil), dst.MustNodeIterator(nil))
	fwdIt := trie.NewIterator(fwdDiffIt)

	for fwdIt.Next() {
		acc := new(types.StateAccount)
		if err := rlp.DecodeBytes(fwdIt.Value, acc); err != nil {
			panic(err)
		}
		supplyDelta.Add(supplyDelta, acc.Balance)
	}
	// Gather all the changes across from destination to source.
	rewDiffIt, _ := trie.NewDifferenceIterator(dst.MustNodeIterator(nil), src.MustNodeIterator(nil))
	rewIt := trie.NewIterator(rewDiffIt)

	for rewIt.Next() {
		acc := new(types.StateAccount)
		if err := rlp.DecodeBytes(rewIt.Value, acc); err != nil {
			panic(err)
		}
		supplyDelta.Sub(supplyDelta, acc.Balance)
	}
	// Calculate the block fixedReward based on chain rules and progression.
	fixedReward, unclesReward, burn, withdrawals := Subsidy(block, config)

	// Calculate the difference between the "calculated" and "crawled" supply
	// delta.
	diff := new(big.Int).Set(supplyDelta)
	diff.Sub(diff, fixedReward)
	diff.Sub(diff, unclesReward)
	diff.Add(diff, burn)

	log.Info("Calculated supply delta for block", "number", block.Number(), "hash", block.Hash(), "supplydelta", supplyDelta, "fixedreward", fixedReward, "unclesreward", unclesReward, "burn", burn, "withdrawals", withdrawals, "diff", diff, "elapsed", time.Since(start))
	return supplyDelta, nil
}

// Subsidy calculates the block mining (fixed) and uncle subsidy as well as the
// 1559 burn solely based on header fields. This method is a very accurate
// approximation of the true supply delta, but cannot take into account Ether
// burns via selfdestructs, so it will always be ever so slightly off.
func Subsidy(block *types.Block, config *params.ChainConfig) (fixedReward *big.Int, unclesReward *big.Int, burn *big.Int, withdrawals *big.Int) {
	// Calculate the block rewards based on chain rules and progression.
	fixedReward = new(big.Int)
	unclesReward = new(big.Int)
	withdrawals = new(big.Int)

	// Select the correct block reward based on chain progression.
	if config.Ethash != nil {
		if block.Difficulty().BitLen() != 0 {
			fixedReward = ethash.FrontierBlockReward
			if config.IsByzantium(block.Number()) {
				fixedReward = ethash.ByzantiumBlockReward
			}
			if config.IsConstantinople(block.Number()) {
				fixedReward = ethash.ConstantinopleBlockReward
			}
		}
		// Accumulate the rewards for included uncles.
		var (
			big8  = big.NewInt(8)
			big32 = big.NewInt(32)
			r     = new(big.Int)
		)
		for _, uncle := range block.Uncles() {
			// Add the reward for the side blocks.
			r.Add(uncle.Number, big8)
			r.Sub(r, block.Number())
			r.Mul(r, fixedReward)
			r.Div(r, big8)
			unclesReward.Add(unclesReward, r)

			// Add the reward for accumulating the side blocks.
			r.Div(fixedReward, big32)
			unclesReward.Add(unclesReward, r)
		}
	}
	// Calculate the burn based on chain rules and progression.
	burn = new(big.Int)
	if block.BaseFee() != nil {
		burn = new(big.Int).Mul(new(big.Int).SetUint64(block.GasUsed()), block.BaseFee())
	}

	for _, w := range block.Withdrawals() {
		withdrawals.Add(withdrawals, big.NewInt(int64(w.Amount)))
	}

	return fixedReward, unclesReward, burn, withdrawals
}

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
