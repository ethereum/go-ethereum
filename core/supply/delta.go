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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// Delta calculates the ether delta across two state tries. That is, the
// issuance minus the ether destroyed.
func Delta(src, dst *types.Header, db *trie.Database) (*big.Int, error) {
	// Open src and dst tries.
	srcTrie, err := trie.New(trie.StateTrieID(src.Root), db)
	if err != nil {
		return nil, fmt.Errorf("failed to open source trie: %v", err)
	}
	dstTrie, err := trie.New(trie.StateTrieID(dst.Root), db)
	if err != nil {
		return nil, fmt.Errorf("failed to open destination trie: %v", err)
	}
	delta := new(big.Int)

	// Gather all the changes across from source to destination.
	fwdDiffIt, _ := trie.NewDifferenceIterator(srcTrie.MustNodeIterator(nil), dstTrie.MustNodeIterator(nil))
	fwdIt := trie.NewIterator(fwdDiffIt)

	for fwdIt.Next() {
		acc := new(types.StateAccount)
		if err := rlp.DecodeBytes(fwdIt.Value, acc); err != nil {
			panic(err)
		}
		delta.Add(delta, acc.Balance)
	}
	// Gather all the changes across from destination to source.
	revDiffIt, _ := trie.NewDifferenceIterator(dstTrie.MustNodeIterator(nil), srcTrie.MustNodeIterator(nil))
	revIt := trie.NewIterator(revDiffIt)

	for revIt.Next() {
		acc := new(types.StateAccount)
		if err := rlp.DecodeBytes(revIt.Value, acc); err != nil {
			panic(err)
		}
		delta.Sub(delta, acc.Balance)
	}

	return delta, nil
}

// Issuance calculates the amount of ether issued by the protocol. There are
// currently two ways for ether to be creates, the first is from block rewards
// and the second is via withdrawals.
func Issuance(block *types.Block, config *params.ChainConfig) (*big.Int, *big.Int) {
	var (
		rewards     = new(big.Int)
		withdrawals = new(big.Int)
	)
	// If block is ethash, calculate the coinbase and uncle rewards.
	if config.Ethash != nil && block.Difficulty().BitLen() != 0 {
		acc := func(h *types.Header, amt *big.Int) {
			rewards.Add(rewards, amt)
		}
		ethash.AccumulateRewards(config, block.Header(), block.Uncles(), acc, acc)
	}
	// Sum up withdrawals.
	for _, w := range block.Withdrawals() {
		withdrawals.Add(withdrawals, newGwei(w.Amount))
	}
	return rewards, withdrawals
}

// Burn calculates the amount of ether burned due to EIP-1559 base fee.
func Burn(header *types.Header) *big.Int {
	burn := new(big.Int)
	if header.BaseFee != nil {
		burn = new(big.Int).Mul(new(big.Int).SetUint64(header.GasUsed), header.BaseFee)
	}
	return burn
}

func newGwei(n uint64) *big.Int {
	return new(big.Int).Mul(big.NewInt(int64(n)), big.NewInt(params.GWei))
}
