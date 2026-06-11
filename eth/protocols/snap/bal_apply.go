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
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
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

// isFetched tells us if accountHash has been downloaded.
func (s *syncerV2) isFetched(accountHash common.Hash) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, task := range s.tasks {
		if bytes.Compare(accountHash[:], task.Last[:]) <= 0 {
			return bytes.Compare(accountHash[:], task.Next[:]) < 0
		}
	}
	return true
}

// isStorageFetched reports whether the specified storage slot has already
// been downloaded in a previous cycle.
func (s *syncerV2) isStorageFetched(accountHash, storageHash common.Hash) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, task := range s.tasks {
		if bytes.Compare(accountHash[:], task.Last[:]) > 0 {
			continue
		}
		// The account falls within a completed account range.
		if bytes.Compare(accountHash[:], task.Next[:]) < 0 {
			return true
		}
		// All storage for this account has been synchronized.
		if _, ok := task.stateCompleted[accountHash]; ok {
			return true
		}
		// No storage sync task exists for this account yet.
		subtasks, ok := task.SubTasks[accountHash]
		if !ok {
			return false
		}
		// Check whether the slot falls within a completed storage subrange.
		for _, sub := range subtasks {
			if bytes.Compare(storageHash[:], sub.Last[:]) <= 0 {
				return bytes.Compare(storageHash[:], sub.Next[:]) < 0
			}
		}
		return true // All storage subranges for this slot have been completed.
	}
	return true // The account belongs to a completed account range.
}

// applyAccessList applies a single block's access list diffs to the flat state
// in the database. For each account, it applies the post-block values (highest
// TxIdx entry) for balance, nonce, code, and storage. The storageRoot field is
// intentionally left stale. It will be recomputed during the trie generation.
func (s *syncerV2) applyAccessList(b *bal.BlockAccessList, batch ethdb.Batch) error {
	// Iterate over all accounts in the access list
	for _, access := range *b {
		addr := access.Address
		accountHash := crypto.Keccak256Hash(addr[:])

		for _, slotWrites := range access.StorageChanges {
			if n := len(slotWrites.SlotChanges); n > 0 {
				value := slotWrites.SlotChanges[n-1].PostValue
				slotKey := slotWrites.Slot.Bytes32()
				storageHash := crypto.Keccak256Hash(slotKey[:])

				if !s.isStorageFetched(accountHash, storageHash) {
					continue
				}
				if value.IsZero() {
					rawdb.DeleteStorageSnapshot(batch, accountHash, storageHash)
				} else {
					// Store the slot in the same encoding the snapshot and the
					// trie generation use: RLP of the minimal big-endian value
					// (leading zeros trimmed), matching core/state's snapshot
					// writes.
					blob, _ := rlp.EncodeToBytes(value.Bytes())
					rawdb.WriteStorageSnapshot(batch, accountHash, storageHash, blob)
				}
			}
		}
		if !s.isFetched(accountHash) {
			continue
		}
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
			account.Balance = new(uint256.Int).Set(access.BalanceChanges[n-1].PostBalance)
		}

		// Apply nonce change (last entry = post-block state)
		if n := len(access.NonceChanges); n > 0 {
			account.Nonce = access.NonceChanges[n-1].PostNonce
		}

		// Apply code change (last entry = post-block state)
		if n := len(access.CodeChanges); n > 0 {
			code := access.CodeChanges[n-1].NewCode
			if len(code) > 0 {
				codeHash := crypto.Keccak256(code)
				rawdb.WriteCode(batch, common.BytesToHash(codeHash), code)
				account.CodeHash = codeHash
			} else {
				account.CodeHash = types.EmptyCodeHash[:]
			}
		}

		// Don't create empty accounts in flat state (EIP-161).
		isEmpty := account.Balance.IsZero() && account.Nonce == 0 &&
			bytes.Equal(account.CodeHash, types.EmptyCodeHash[:])
		switch {
		case isEmpty && isNew:
			// This covers cases where an account is created and destroyed within the
			// same transaction, or where its net state change across the block is zero.
			// The empty -> empty transition should be excluded from account update.
		case isEmpty && !isNew:
			// Existing account got fully drained (e.g., pre-funded
			// address that gets deployed to with init code that
			// self-destructs). Delete the entry so the trie generation
			// doesn't pick it up as an empty leaf.
			rawdb.DeleteAccountSnapshot(batch, accountHash)
		default:
			// Write the updated account (storageRoot intentionally left stale)
			rawdb.WriteAccountSnapshot(batch, accountHash, types.SlimAccountRLP(account))
		}
	}
	return nil
}
