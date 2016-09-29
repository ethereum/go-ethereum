// Copyright 2014 The go-ethereum Authors
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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// ToEVMContext creates a new context for use in the EVM.
func ToEVMContext(config *params.ChainConfig, msg Message, header *types.Header) vm.Context {
	var from common.Address
	if config.IsHomestead(header.Number) {
		from, _ = msg.From()
	} else {
		from, _ = msg.FromFrontier()
	}

	return vm.Context{
		CallContext: EVMCallContext{CanTransfer, Transfer},
		Origin:      from,
		Coinbase:    header.Coinbase,
		BlockNumber: new(big.Int).Set(header.Number),
		Time:        new(big.Int).Set(header.Time),
		Difficulty:  new(big.Int).Set(header.Difficulty),
		GasLimit:    new(big.Int).Set(header.GasLimit),
		GasPrice:    new(big.Int).Set(msg.GasPrice()),
	}
}

// GetHashFn returns a function for which the VM env can query block hashes through
// up to the limit defined by the Yellow Paper and uses the given block chain
// to query for information.
func GetHashFn(ref common.Hash, chain *BlockChain) func(n uint64) common.Hash {
	return func(n uint64) common.Hash {
		for block := chain.GetBlockByHash(ref); block != nil; block = chain.GetBlock(block.ParentHash(), block.NumberU64()-1) {
			if block.NumberU64() == n {
				return block.Hash()
			}
		}

		return common.Hash{}
	}
}

// CanTransfer checks wether there are enough funds in the address' account to make a transfer.
func CanTransfer(db vm.Database, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.Database, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// EVMBackend implements vm.Backend and provides an interface to keep track of the current
// state.
type EVMBackend struct {
	GetHashFn func(uint64) common.Hash // getHashFn callback is used to retrieve block hashes
	State     *state.StateDB
}

// MakeSnapshot returns a copy of the state.
func (b *EVMBackend) SnapshotDatabase() int {
	return b.State.Snapshot()
}

// Get returns the state
func (b *EVMBackend) Get() vm.Database { return b.State }

// Set sets the current state
func (b *EVMBackend) RevertToSnapshot(revision int) { b.State.RevertToSnapshot(revision) }

// GetHash returns the canonical hash referenced by the depth
func (b *EVMBackend) GetHash(n uint64) common.Hash {
	return b.GetHashFn(n)
}
