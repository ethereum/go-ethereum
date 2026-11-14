// Copyright 2026 The go-ethereum Authors
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

package ethapi

import (
	"context"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/rpc"
)

// GetBlockAccessList returns the block access list for the given block.
//
// The list is a lexicographically sorted array of account accesses, each
// describing the per-transaction post-state changes (balance, nonce, code,
// storage writes) and the storage reads performed on the account during
// block execution.
//
// A null result means the block does not exist or the pending tag was
// requested. Blocks predating the fork carrying the block access list (or
// otherwise lacking one) return an empty list. Note the access list remains
// retrievable even when the block's body and receipts have been pruned.
func (api *BlockChainAPI) GetBlockAccessList(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) (*bal.BlockAccessList, error) {
	if number, ok := blockNrOrHash.Number(); ok && number == rpc.PendingBlockNumber {
		return nil, nil // the pending block's access list is not durable
	}
	header, err := api.b.HeaderByNumberOrHash(ctx, blockNrOrHash)
	if err != nil {
		// The header lookup reports unknown hashes as an error; those surface
		// as null per the RPC spec. All other failures pass through.
		if _, byHash := blockNrOrHash.Hash(); byHash && !blockNrOrHash.RequireCanonical {
			return nil, nil
		}
		return nil, err
	}
	if header == nil {
		return nil, nil
	}
	accessList := rawdb.ReadAccessList(api.b.ChainDb(), header.Hash(), header.Number.Uint64())
	if accessList == nil || len(*accessList) == 0 {
		return &bal.BlockAccessList{}, nil
	}
	accessList.NormalizeJSON()
	return accessList, nil
}
