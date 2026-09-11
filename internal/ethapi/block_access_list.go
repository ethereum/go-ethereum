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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// The result types below mirror the execution-apis block access list schema:
// each change entry pairs a 32-bit block-access index with the post-state
// value, storage slots pair a 32-byte key with per-index writes, balance
// values are 256-bit quantities and storage/read values are 32-byte ones.

type storageChangeResult struct {
	Index hexutil.Uint64 `json:"index"`
	Value common.Hash    `json:"value"`
}

type slotChangesResult struct {
	Key     common.Hash           `json:"key"`
	Changes []storageChangeResult `json:"changes"`
}

type balanceChangeResult struct {
	Index hexutil.Uint64 `json:"index"`
	Value *hexutil.Big   `json:"value"`
}

type nonceChangeResult struct {
	Index hexutil.Uint64 `json:"index"`
	Value hexutil.Uint64 `json:"value"`
}

type codeChangeResult struct {
	Index hexutil.Uint64 `json:"index"`
	Code  hexutil.Bytes  `json:"code"`
}

type accountAccessResult struct {
	Address        common.Address        `json:"address"`
	BalanceChanges []balanceChangeResult `json:"balanceChanges"`
	CodeChanges    []codeChangeResult    `json:"codeChanges"`
	NonceChanges   []nonceChangeResult   `json:"nonceChanges"`
	StorageChanges []slotChangesResult   `json:"storageChanges"`
	StorageReads   []common.Hash         `json:"storageReads"`
}

// GetBlockAccessList returns the block access list for the given block.
//
// The list is a lexicographically sorted array of account accesses, each
// describing the per-transaction post-state changes (balance, nonce, code,
// storage writes) and the storage reads performed on the account during
// block execution.
//
// Blocks predating the fork carrying the block access list (or otherwise
// lacking one) return an empty list. A null result means the block does not
// exist.
func (api *BlockChainAPI) GetBlockAccessList(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]accountAccessResult, error) {
	block, err := api.b.BlockByNumberOrHash(ctx, blockNrOrHash)
	if block == nil || err != nil {
		return nil, err
	}
	al := block.AccessList()
	if al == nil {
		return []accountAccessResult{}, nil
	}
	out := make([]accountAccessResult, 0, len(*al))
	for _, acc := range *al {
		// The spec requires all change arrays to be present (possibly empty).
		res := accountAccessResult{
			Address:        acc.Address,
			BalanceChanges: make([]balanceChangeResult, 0, len(acc.BalanceChanges)),
			NonceChanges:   make([]nonceChangeResult, 0, len(acc.NonceChanges)),
			CodeChanges:    make([]codeChangeResult, 0, len(acc.CodeChanges)),
			StorageChanges: make([]slotChangesResult, 0, len(acc.StorageChanges)),
			StorageReads:   make([]common.Hash, 0, len(acc.StorageReads)),
		}

		for _, change := range acc.BalanceChanges {
			res.BalanceChanges = append(res.BalanceChanges, balanceChangeResult{
				Index: hexutil.Uint64(change.BlockAccessIndex),
				Value: (*hexutil.Big)(change.PostBalance.ToBig()),
			})
		}
		for _, change := range acc.NonceChanges {
			res.NonceChanges = append(res.NonceChanges, nonceChangeResult{
				Index: hexutil.Uint64(change.BlockAccessIndex),
				Value: hexutil.Uint64(change.PostNonce),
			})
		}
		for _, change := range acc.CodeChanges {
			res.CodeChanges = append(res.CodeChanges, codeChangeResult{
				Index: hexutil.Uint64(change.BlockAccessIndex),
				Code:  hexutil.Bytes(change.NewCode),
			})
		}
		for _, slot := range acc.StorageChanges {
			changes := make([]storageChangeResult, 0, len(slot.SlotChanges))
			for _, write := range slot.SlotChanges {
				var value [32]byte
				write.PostValue.WriteToSlice(value[:])
				changes = append(changes, storageChangeResult{
					Index: hexutil.Uint64(write.BlockAccessIndex),
					Value: common.Hash(value),
				})
			}
			var key [32]byte
			slot.Slot.WriteToSlice(key[:])
			res.StorageChanges = append(res.StorageChanges, slotChangesResult{
				Key:     common.Hash(key),
				Changes: changes,
			})
		}
		for _, read := range acc.StorageReads {
			var key [32]byte
			read.WriteToSlice(key[:])
			res.StorageReads = append(res.StorageReads, common.Hash(key))
		}
		out = append(out, res)
	}
	return out, nil
}
