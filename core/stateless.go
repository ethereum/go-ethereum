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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/stateless"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
)

// ExecuteStateless runs a stateless execution based on a witness, fully
// validating the block including header, body, state root and receipt root.
//
// This method is a bit of a sore thumb here, but:
//   - It cannot be placed in core/stateless, because state.New prodces a circular dep
//   - It cannot be placed outside of core, because it needs to construct a dud headerchain
//
// TODO(karalabe): Would be nice to resolve both issues above somehow and move it.
func ExecuteStateless(ctx context.Context, config *params.ChainConfig, vmconfig vm.Config, block *types.Block, witness *stateless.Witness) error {
	// Create and populate the state database to serve as the stateless backend
	memdb := witness.MakeHashDB()
	db, err := state.New(witness.Root(), state.NewDatabase(triedb.NewDatabase(memdb, triedb.HashDefaults), nil))
	if err != nil {
		return err
	}
	// Create a blockchain that is idle, but can be used to access headers through
	engine := beacon.New(ethash.NewFaker())
	chain := &HeaderChain{
		config:      config,
		chainDb:     memdb,
		headerCache: lru.NewCache[common.Hash, *types.Header](256),
		engine:      engine,
	}
	// Verify the block header against the parent header from the witness
	if err := engine.VerifyHeader(chain, block.Header()); err != nil {
		return err
	}
	processor := NewStateProcessor(chain)
	validator := NewBlockValidator(config)

	// Verify the block body (transactions, withdrawals, blob gas, BAL) against the header
	if err := validator.ValidateBody(block); err != nil {
		return err
	}

	if config.IsAmsterdam(block.Number(), block.Time()) {
		db = db.WithReader(state.NewReaderWithTracker(db.Reader()))
	}

	// Run the stateless blocks processing and self-validate all fields
	res, err := processor.Process(ctx, block, db, vmconfig)
	if err != nil {
		return err
	}
	if err = validator.ValidateState(block, db, res); err != nil {
		return err
	}
	return nil
}
