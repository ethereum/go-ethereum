// Copyright 2025 The go-ethereum Authors
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

package partial

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
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/holiman/uint256"
)

// PartialState manages state for partial stateful nodes.
// It applies BAL diffs to update state without re-executing transactions.
type PartialState struct {
	db      ethdb.Database
	trieDB  *triedb.Database
	filter  ContractFilter
	history *BALHistory

	// Current state root
	stateRoot common.Hash
}

// NewPartialState creates a new partial state manager.
func NewPartialState(db ethdb.Database, trieDB *triedb.Database, filter ContractFilter, balRetention uint64) *PartialState {
	return &PartialState{
		db:      db,
		trieDB:  trieDB,
		filter:  filter,
		history: NewBALHistory(db, balRetention),
	}
}

// Filter returns the contract filter used by this partial state.
func (s *PartialState) Filter() ContractFilter {
	return s.filter
}

// SetRoot sets the current state root.
func (s *PartialState) SetRoot(root common.Hash) {
	s.stateRoot = root
}

// Root returns the current state root.
func (s *PartialState) Root() common.Hash {
	return s.stateRoot
}

// History returns the BAL history manager.
func (s *PartialState) History() *BALHistory {
	return s.history
}

// accountState tracks an account being processed with origin info for PathDB StateSet.
type accountState struct {
	account     *types.StateAccount
	origin      *types.StateAccount // Original state (for PathDB StateSet)
	addr        common.Address
	existed     bool        // true if account existed before this block
	modified    bool        // true if any field was changed
	storageRoot common.Hash // updated after storage trie commit
}

// ApplyBALAndComputeRoot applies BAL diffs and returns the new state root.
// This is the core method for partial state block processing.
//
// Commit ordering (critical for correct state root):
// Phase 1: For each account, apply storage changes and commit storage trie
// Phase 2: Update account Root fields with committed storage roots
// Phase 3: Commit account trie to get final state root
func (s *PartialState) ApplyBALAndComputeRoot(parentRoot common.Hash, accessList *bal.BlockAccessList) (common.Hash, error) {
	// Open state trie at parent root
	tr, err := trie.NewStateTrie(trie.StateTrieID(parentRoot), s.trieDB)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to open state trie: %w", err)
	}

	// Collect all account states with origin tracking
	accounts := make([]*accountState, 0, len(accessList.Accesses))

	// Collect all trie nodes for batched update
	allNodes := trienode.NewMergedNodeSet()

	// Phase 1: Process each account's changes from BAL
	for _, access := range accessList.Accesses {
		addr := common.BytesToAddress(access.Address[:])

		// Get current account state with origin tracking
		data, err := tr.GetAccount(addr)
		if err != nil {
			return common.Hash{}, fmt.Errorf("failed to get account %s: %w", addr.Hex(), err)
		}

		existed := data != nil
		var account *types.StateAccount
		if existed {
			account = data
		} else {
			// New account - create with defaults
			account = &types.StateAccount{
				Balance:  new(uint256.Int),
				Root:     types.EmptyRootHash,
				CodeHash: types.EmptyCodeHash.Bytes(),
			}
		}

		// Copy original state for PathDB StateSet
		var origin *types.StateAccount
		if existed {
			origin = &types.StateAccount{
				Nonce:    account.Nonce,
				Balance:  new(uint256.Int).Set(account.Balance),
				Root:     account.Root,
				CodeHash: common.CopyBytes(account.CodeHash),
			}
		}

		state := &accountState{
			account:     account,
			origin:      origin,
			addr:        addr,
			existed:     existed,
			modified:    false,
			storageRoot: account.Root,
		}

		// Apply balance changes (use final value from last tx)
		if len(access.BalanceChanges) > 0 {
			lastChange := access.BalanceChanges[len(access.BalanceChanges)-1]
			account.Balance = new(uint256.Int).SetBytes(lastChange.Balance[:])
			state.modified = true
		}

		// Apply nonce changes
		if len(access.NonceChanges) > 0 {
			lastNonce := access.NonceChanges[len(access.NonceChanges)-1]
			account.Nonce = lastNonce.Nonce
			state.modified = true
		}

		// Apply code changes
		if len(access.Code) > 0 {
			lastCode := access.Code[len(access.Code)-1]
			codeHash := crypto.Keccak256Hash(lastCode.Code)
			account.CodeHash = codeHash.Bytes()
			state.modified = true

			// Only store code bytes for tracked contracts
			if s.filter.IsTracked(addr) {
				rawdb.WriteCode(s.db, codeHash, lastCode.Code)
			}
		}

		// Apply storage changes (only for tracked contracts)
		// CRITICAL: Commit storage trie HERE, before account trie
		if len(access.StorageWrites) > 0 && s.filter.IsTracked(addr) {
			newStorageRoot, storageNodes, err := s.applyStorageChanges(
				addr, parentRoot, account.Root, &access)
			if err != nil {
				return common.Hash{}, fmt.Errorf("failed to apply storage for %s: %w",
					addr.Hex(), err)
			}
			state.storageRoot = newStorageRoot
			state.modified = true

			// Merge storage nodes
			if storageNodes != nil {
				if err := allNodes.Merge(storageNodes); err != nil {
					return common.Hash{}, err
				}
			}
		}

		accounts = append(accounts, state)
	}

	// Phase 2: Update account Root fields and write to account trie
	for _, state := range accounts {
		// Update storage root (may have changed in Phase 1)
		state.account.Root = state.storageRoot

		// Only consider deletion if modified AND now empty (EIP-161)
		if state.modified && s.isEmptyAccount(state.account) {
			// Only delete if it existed before (don't delete never-existed accounts)
			if state.existed {
				if err := tr.DeleteAccount(state.addr); err != nil {
					return common.Hash{}, fmt.Errorf("failed to delete account %s: %w",
						state.addr.Hex(), err)
				}
			}
			// Skip update for accounts that didn't exist and are still empty
			continue
		}

		if err := tr.UpdateAccount(state.addr, state.account, 0); err != nil {
			return common.Hash{}, fmt.Errorf("failed to update account %s: %w",
				state.addr.Hex(), err)
		}
	}

	// Phase 3: Commit account trie
	root, accountNodes := tr.Commit(false)

	// Merge account nodes
	if accountNodes != nil {
		if err := allNodes.Merge(accountNodes); err != nil {
			return common.Hash{}, err
		}
	}

	// Build StateSet for PathDB compatibility
	stateSet := s.buildStateSet(accounts, accessList)

	// Write all trie nodes and state to database
	if err := s.trieDB.Update(root, parentRoot, 0, allNodes, stateSet); err != nil {
		return common.Hash{}, fmt.Errorf("failed to update trie db: %w", err)
	}

	s.stateRoot = root
	return root, nil
}

// buildStateSet constructs StateSet for trieDB.Update() (required for PathDB).
// The StateSet tracks account and storage changes along with their original values,
// which PathDB uses for efficient state diff tracking.
func (s *PartialState) buildStateSet(accounts []*accountState, accessList *bal.BlockAccessList) *triedb.StateSet {
	stateSet := triedb.NewStateSet()

	for _, state := range accounts {
		addrHash := crypto.Keccak256Hash(state.addr.Bytes())

		// Add account data (slim RLP encoding)
		if s.isEmptyAccount(state.account) && state.existed {
			stateSet.Accounts[addrHash] = nil // nil = deletion
		} else if state.modified {
			stateSet.Accounts[addrHash] = types.SlimAccountRLP(*state.account)
		}

		// Add account origin (original state before this block)
		if state.origin != nil {
			stateSet.AccountsOrigin[state.addr] = types.SlimAccountRLP(*state.origin)
		}

		// Add storage changes for tracked contracts
		if s.filter.IsTracked(state.addr) {
			s.addStorageToStateSet(stateSet, state.addr, addrHash, accessList)
		}
	}
	return stateSet
}

// addStorageToStateSet finds storage writes for the given address and adds them to the StateSet.
func (s *PartialState) addStorageToStateSet(stateSet *triedb.StateSet, addr common.Address, addrHash common.Hash, accessList *bal.BlockAccessList) {
	// Find this account's storage writes in BAL
	for _, access := range accessList.Accesses {
		accessAddr := common.BytesToAddress(access.Address[:])
		if accessAddr != addr {
			continue
		}
		if len(access.StorageWrites) == 0 {
			break
		}

		storageMap := make(map[common.Hash][]byte)
		for _, slotWrite := range access.StorageWrites {
			slotHash := crypto.Keccak256Hash(slotWrite.Slot[:])
			if len(slotWrite.Accesses) > 0 {
				lastWrite := slotWrite.Accesses[len(slotWrite.Accesses)-1]
				value := common.BytesToHash(lastWrite.ValueAfter[:])
				if value == (common.Hash{}) {
					storageMap[slotHash] = nil // nil = deletion
				} else {
					// Prefix-zero-trimmed RLP encoding
					blob, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
					storageMap[slotHash] = blob
				}
			}
		}
		stateSet.Storages[addrHash] = storageMap
		break
	}
}

// isEmptyAccount checks if account is empty per EIP-161.
// An account is empty if it has zero nonce, zero balance, empty storage root,
// and empty code hash.
func (s *PartialState) isEmptyAccount(account *types.StateAccount) bool {
	return account.Balance.IsZero() &&
		account.Nonce == 0 &&
		account.Root == types.EmptyRootHash &&
		bytes.Equal(account.CodeHash, types.EmptyCodeHash.Bytes())
}

// applyStorageChanges applies storage writes and returns new root + nodes.
// Note: Does NOT write to trieDB - caller batches all writes.
func (s *PartialState) applyStorageChanges(
	addr common.Address,
	stateRoot common.Hash,
	currentStorageRoot common.Hash,
	access *bal.AccountAccess,
) (common.Hash, *trienode.NodeSet, error) {
	// Open storage trie (use parent state root for ID, not current)
	addrHash := crypto.Keccak256Hash(addr.Bytes())
	storageID := trie.StorageTrieID(stateRoot, addrHash, currentStorageRoot)
	storageTrie, err := trie.NewStateTrie(storageID, s.trieDB)
	if err != nil {
		return common.Hash{}, nil, err
	}

	// Apply each storage write (use final value)
	for _, slotWrite := range access.StorageWrites {
		slot := common.BytesToHash(slotWrite.Slot[:])

		// Get final value (last write wins)
		if len(slotWrite.Accesses) == 0 {
			continue
		}
		lastWrite := slotWrite.Accesses[len(slotWrite.Accesses)-1]
		value := common.BytesToHash(lastWrite.ValueAfter[:])

		if value == (common.Hash{}) {
			// Delete slot
			if err := storageTrie.DeleteStorage(addr, slot.Bytes()); err != nil {
				return common.Hash{}, nil, err
			}
		} else {
			// Update slot
			if err := storageTrie.UpdateStorage(addr, slot.Bytes(), value.Bytes()); err != nil {
				return common.Hash{}, nil, err
			}
		}
	}

	// Commit storage trie (collect nodes, don't write to DB yet)
	storageRoot, nodes := storageTrie.Commit(false)

	return storageRoot, nodes, nil
}
