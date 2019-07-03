// Copyright 2019 The go-ethereum Authors
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

package eth

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/forkid"
	"github.com/ethereum/go-ethereum/p2p/enr"
)

// ENR is the "eth" Ethereum Node Record, holding the fork id as specified by
// EIP-2124 (https://eips.ethereum.org/EIPS/eip-2124).
type ENR forkid.ID

// NewENR calculates the Ethereum network ENR from the fork ID.
func NewENR(chain *core.BlockChain) ENR {
	return ENR(forkid.NewID(chain))
}

// ENRKey implements enr.Entry, returning the key for the chain config.
func (e ENR) ENRKey() string { return "eth" }

// NewENRFilter creates an ENR filter that returns if a record should be rejected
// or not (may be rejected by another filter).
func NewENRFilter(chain *core.BlockChain) func(r *enr.Record) error {
	filter := forkid.NewFilter(chain)

	return func(r *enr.Record) error {
		// Retrieve the remote chain ENR entry, accept record if not found
		var entry ENR
		if err := r.Load(&entry); err != nil {
			return nil
		}
		// If found, run it across the fork ID validator
		return filter(forkid.ID(entry))
	}
}
