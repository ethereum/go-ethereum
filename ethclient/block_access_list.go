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

package ethclient

import (
	"context"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

// BlockAccessListEntry describes the state changes of a single account within
// a block access list, keyed by the transaction that caused them.
type BlockAccessListEntry struct {
	Address        common.Address        `json:"address"`
	BalanceChanges []BalanceChangeEntry  `json:"balanceChanges"`
	CodeChanges    []CodeChangeEntry     `json:"codeChanges"`
	NonceChanges   []NonceChangeEntry    `json:"nonceChanges"`
	StorageChanges []StorageChangesEntry `json:"storageChanges"`
	StorageReads   []common.Hash         `json:"storageReads"`
}

// BalanceChangeEntry records the post-state balance of an account at a block
// access index.
type BalanceChangeEntry struct {
	Index hexutil.Uint64 `json:"index"`
	Value *hexutil.Big   `json:"value"`
}

// CodeChangeEntry records the code deployed to an account at a block access
// index.
type CodeChangeEntry struct {
	Index hexutil.Uint64 `json:"index"`
	Code  hexutil.Bytes  `json:"code"`
}

// NonceChangeEntry records the post-state nonce of an account at a block
// access index.
type NonceChangeEntry struct {
	Index hexutil.Uint64 `json:"index"`
	Value hexutil.Uint64 `json:"value"`
}

// StorageChangeEntry records the post-state value of a storage slot at a
// block access index.
type StorageChangeEntry struct {
	Index hexutil.Uint64 `json:"index"`
	Value common.Hash    `json:"value"`
}

// StorageChangesEntry pairs a storage key with the writes performed on it.
type StorageChangesEntry struct {
	Key     common.Hash          `json:"key"`
	Changes []StorageChangeEntry `json:"changes"`
}

// GetBlockAccessList returns the block access list of the given block.
func (ec *Client) GetBlockAccessList(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]BlockAccessListEntry, error) {
	var r []BlockAccessListEntry
	err := ec.c.CallContext(ctx, &r, "eth_getBlockAccessList", blockNrOrHash)
	if err == nil && r == nil {
		return nil, ethereum.NotFound
	}
	return r, err
}
