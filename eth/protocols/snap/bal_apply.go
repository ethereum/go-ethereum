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

package snap

import (
	"bytes"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// verifyAccessList checks that the given block access list matches the hash
// committed in the block header.
func verifyAccessList(b *bal.BlockAccessList, header *types.Header) error {
	if header.BlockAccessListHash == nil {
		return fmt.Errorf("header %d has no access list hash", header.Number)
	}
	have := b.Hash()
	if have != *header.BlockAccessListHash {
		return fmt.Errorf("access list hash mismatch for block %d: have %v, want %v", header.Number, have, *header.BlockAccessListHash)
	}
	return nil
}

// applyAccessList applies a single block's access list diffs to the flat state
// in the database. For each account, it applies the post-block values (highest
// TxIdx entry) for balance, nonce, code, and storage. The storageRoot field is
// intentionally left stale. It will be recomputed during the trie rebuild.
func (s *Syncer) applyAccessList(b *bal.BlockAccessList) error {
	batch := s.db.NewBatch()

	for _, access := range b.Accesses {
		addr := common.Address(access.Address)
		accountHash := crypto.Keccak256Hash(addr[:])

		// Read the existing account from flat state (may not exist yet)
		var (
			account types.StateAccount
			isNew   bool
		)
		if data := rawdb.ReadAccountSnapshot(s.db, accountHash); len(data) > 0 {
			existing, err := types.FullAccount(data)
			if err != nil {
				return fmt.Errorf("failed to decode account %v: %w", addr, err)
			}
			account = *existing
		} else {
			// New account — initialize with defaults
			isNew = true
			account.Balance = new(uint256.Int)
			account.Root = types.EmptyRootHash
			account.CodeHash = types.EmptyCodeHash[:]
		}

		// Apply balance change (last entry = post-block state)
		if n := len(access.BalanceChanges); n > 0 {
			raw := access.BalanceChanges[n-1].Balance
			account.Balance = new(uint256.Int).SetBytes(raw[:])
		}

		// Apply nonce change (last entry = post-block state)
		if n := len(access.NonceChanges); n > 0 {
			account.Nonce = access.NonceChanges[n-1].Nonce
		}

		// Apply code change (last entry = post-block state)
		if n := len(access.CodeChanges); n > 0 {
			code := access.CodeChanges[n-1].Code
			if len(code) > 0 {
				codeHash := crypto.Keccak256(code)
				rawdb.WriteCode(batch, common.BytesToHash(codeHash), code)
				account.CodeHash = codeHash
			} else {
				account.CodeHash = types.EmptyCodeHash[:]
			}
		}

		// Apply storage writes (last entry per slot = post-block state)
		for _, slotWrites := range access.StorageWrites {
			if n := len(slotWrites.Accesses); n > 0 {
				value := slotWrites.Accesses[n-1].ValueAfter
				storageHash := crypto.Keccak256Hash(slotWrites.Slot[:])
				rawdb.WriteStorageSnapshot(batch, accountHash, storageHash, value[:])
			}
		}

		// Don't create empty accounts in flat state (EIP-161).
		// This handles the case where an account is created and
		// self-destructed in the same transaction. The BAL will
		// include it with a balance change to zero, but the account
		// should not exist in state.
		if isNew && account.Balance.IsZero() && account.Nonce == 0 &&
			bytes.Equal(account.CodeHash, types.EmptyCodeHash[:]) {
			continue
		}

		// Write the updated account (storageRoot intentionally left stale)
		rawdb.WriteAccountSnapshot(batch, accountHash, types.SlimAccountRLP(account))
	}
	return batch.Write()
}
