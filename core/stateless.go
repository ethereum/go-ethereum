// Copyright 2024 The go-ethereum Authors
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

package core

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
)

// ExecuteStateless runs a stateless execution based on a witness, verifies
// everything it can locally and returns the state root and receipt root, that
// need the other side to explicitly check.
//
// This method is a bit of a sore thumb here, but:
//   - It cannot be placed in core/stateless, because state.New prodces a circular dep
//   - It cannot be placed outside of core, because it needs to construct a dud headerchain
//
// TODO(karalabe): Would be nice to resolve both issues above somehow and move it.
func ExecuteStateless(ctx context.Context, config *params.ChainConfig, vmconfig vm.Config, block *types.Block, witness *stateless.Witness, validateHeader bool) (common.Hash, common.Hash, error) {
	// Create and populate the state database to serve as the stateless backend
	memdb := witness.MakeHashDB()
	db, err := state.New(witness.Root(), state.NewDatabase(triedb.NewDatabase(memdb, triedb.HashDefaults), nil))
	if err != nil {
		return common.Hash{}, common.Hash{}, err
	}
	// Create a blockchain that is idle, but can be used to access headers through
	chain := &HeaderChain{
		config:      config,
		chainDb:     memdb,
		headerCache: lru.NewCache[common.Hash, *types.Header](256),
		engine:      beacon.New(ethash.NewFaker()),
	}
	processor := NewStateProcessor(chain)
	validator := NewBlockValidator(config, nil) // No chain, we only validate the state, not the block

	if validateHeader {
		if err := beacon.New(ethash.NewFaker()).VerifyHeader(chain, block.Header()); err != nil {
			return common.Hash{}, common.Hash{}, fmt.Errorf("error verifying header in stateless validation: %w", err)
		}

		validator := NewBlockValidator(config, nil)
		if err := validator.ValidateBody(block); err != nil {
			return common.Hash{}, common.Hash{}, fmt.Errorf("error validating body in stateless validation: %w", err)
		}
	}

	// Run the stateless blocks processing and self-validate certain fields
	res, err := processor.Process(ctx, block, db, vmconfig)
	if err != nil {
		return common.Hash{}, common.Hash{}, err
	}
	if err = validator.ValidateState(block, db, res, validateHeader); err != nil {
		return common.Hash{}, common.Hash{}, err
	}
	// Almost everything validated, but receipt and state root needs to be returned
	receiptRoot := types.DeriveSha(res.Receipts, trie.NewStackTrie(nil))
	stateRoot := db.IntermediateRoot(config.IsEIP158(block.Number()))
	return stateRoot, receiptRoot, nil
}
