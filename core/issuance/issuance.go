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

package issuance

import (
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// Issuance calculates the Ether issuance (or burn) across two state tries. In
// normal mode of operation, the expectation is to calculate the issuance between
// two consecutive blocks.
func Issuance(block *types.Block, parent *types.Header, db *trie.Database, config *params.ChainConfig) (*big.Int, error) {
	var (
		issuance = new(big.Int)
		start    = time.Now()
	)
	// Open the two tries
	if block.ParentHash() != parent.Hash() {
		return nil, fmt.Errorf("parent hash mismatch: have %s, want %s", block.ParentHash().Hex(), parent.Hash().Hex())
	}
	src, err := trie.New(parent.Root, db)
	if err != nil {
		return nil, fmt.Errorf("failed to open source trie: %v", err)
	}
	dst, err := trie.New(block.Root(), db)
	if err != nil {
		return nil, fmt.Errorf("failed to open destination trie: %v", err)
	}
	// Gather all the changes across from source to destination
	fwdDiffIt, _ := trie.NewDifferenceIterator(src.NodeIterator(nil), dst.NodeIterator(nil))
	fwdIt := trie.NewIterator(fwdDiffIt)

	for fwdIt.Next() {
		acc := new(types.StateAccount)
		if err := rlp.DecodeBytes(fwdIt.Value, acc); err != nil {
			panic(err)
		}
		issuance.Add(issuance, acc.Balance)
	}
	// Gather all the changes across from destination to source
	rewDiffIt, _ := trie.NewDifferenceIterator(dst.NodeIterator(nil), src.NodeIterator(nil))
	rewIt := trie.NewIterator(rewDiffIt)

	for rewIt.Next() {
		acc := new(types.StateAccount)
		if err := rlp.DecodeBytes(rewIt.Value, acc); err != nil {
			panic(err)
		}
		issuance.Sub(issuance, acc.Balance)
	}
	// Calculate the block subsidy based on chain rules and progression
	subsidy, uncles, burn := Subsidy(block, config)

	// Calculate the difference between the "calculated" and "crawled" issuance
	diff := new(big.Int).Set(issuance)
	diff.Sub(diff, subsidy)
	diff.Sub(diff, uncles)
	diff.Add(diff, burn)

	log.Info("Calculated issuance for block", "number", block.Number(), "hash", block.Hash(), "state", issuance, "subsidy", subsidy, "uncles", uncles, "burn", burn, "diff", diff, "elapsed", time.Since(start))
	return issuance, nil
}

// Subsidy calculates the block mining and uncle subsidy as well as the 1559 burn
// solely based on header fields. This method is a very accurate approximation of
// the true issuance, but cannot take into account Ether burns via selfdestructs,
// so it will always be ever so slightly off.
func Subsidy(block *types.Block, config *params.ChainConfig) (subsidy *big.Int, uncles *big.Int, burn *big.Int) {
	// Calculate the block subsidy based on chain rules and progression
	subsidy = new(big.Int)
	uncles = new(big.Int)

	// Select the correct block reward based on chain progression
	if config.Ethash != nil {
		if block.Difficulty().BitLen() != 0 {
			subsidy = ethash.FrontierBlockReward
			if config.IsByzantium(block.Number()) {
				subsidy = ethash.ByzantiumBlockReward
			}
			if config.IsConstantinople(block.Number()) {
				subsidy = ethash.ConstantinopleBlockReward
			}
		}
		// Accumulate the rewards for inclded uncles
		var (
			big8  = big.NewInt(8)
			big32 = big.NewInt(32)
			r     = new(big.Int)
		)
		for _, uncle := range block.Uncles() {
			// Add the reward for the side blocks
			r.Add(uncle.Number, big8)
			r.Sub(r, block.Number())
			r.Mul(r, subsidy)
			r.Div(r, big8)
			uncles.Add(uncles, r)

			// Add the reward for accumulating the side blocks
			r.Div(subsidy, big32)
			uncles.Add(uncles, r)
		}
	}
	// Calculate the burn based on chain rules and progression
	burn = new(big.Int)
	if block.BaseFee() != nil {
		burn = new(big.Int).Mul(new(big.Int).SetUint64(block.GasUsed()), block.BaseFee())
	}
	return subsidy, uncles, burn
}
