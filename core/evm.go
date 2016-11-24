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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

// BlockFetcher retrieves headers by their hash
type BlockFetcher interface {
	// GetHeader returns the hash corresponding to their hash
	GetHeader(common.Hash, uint64) *types.Header
}

// ToEVMContext creates a new context for use in the EVM.
func ToEVMContext(msg Message, header *types.Header, chain BlockFetcher) vm.Context {
	return vm.Context{
		CallContext: EVMCallContext{
			CanTransfer: CanTransfer,
			Transfer:    Transfer,
			GetHashFn:   GetHashFn(header, chain),
		},
		Origin:      msg.From(),
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
func GetHashFn(ref *types.Header, chain BlockFetcher) func(n uint64) common.Hash {
	return func(n uint64) common.Hash {
		for header := chain.GetHeader(ref.ParentHash, ref.Number.Uint64()-1); header != nil; header = chain.GetHeader(header.ParentHash, header.Number.Uint64()-1) {
			if header.Number.Uint64() == n {
				return header.Hash()
			}
		}

		return common.Hash{}
	}
}

// CanTransfer checks wether there are enough funds in the address' account to make a transfer.
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
