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

package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
)

// binaryHasher is a Hasher implementation backed by a unified single-layer
// binary trie. Accounts, storage slots, and contract code all reside in one
// trie, keyed according to the EIP-7864 address space layout.
type binaryHasher struct {
	db   *triedb.Database
	root common.Hash
	trie *bintrie.BinaryTrie
}

func newBinaryHasher(root common.Hash, db *triedb.Database) (*binaryHasher, error) {
	tr, err := bintrie.NewBinaryTrie(root, db)
	if err != nil {
		return nil, err
	}
	return &binaryHasher{
		db:   db,
		root: root,
		trie: tr,
	}, nil
}

func (h *binaryHasher) UpdateAccount(addresses []common.Address, accounts []AccountMut) error {
	for i, addr := range addresses {
		acct := accounts[i]

		// Deletion: zero out account basic data and code hash so that
		// GetAccount returns nil for this address.
		if acct.Account == nil {
			if err := h.trie.DeleteAccount(addr); err != nil {
				return err
			}
			continue
		}
		// Determine code size: use the new code length if provided,
		// otherwise fall back to the cached or trie-stored value.
		//
		// TODO(rjl493456442) the contract code length is not assigned
		// if it's not modified, fix it.
		codeLen := 0
		if acct.Code != nil {
			codeLen = len(acct.Code.Code)
		}
		sa := &types.StateAccount{
			Nonce:    acct.Account.Nonce,
			Balance:  acct.Account.Balance,
			CodeHash: acct.Account.CodeHash,
		}
		if err := h.trie.UpdateAccount(addr, sa, codeLen); err != nil {
			return err
		}
		// Write chunked code into the trie when dirty.
		if acct.Code != nil && len(acct.Code.Code) > 0 {
			codeHash := common.BytesToHash(acct.Account.CodeHash)
			if err := h.trie.UpdateContractCode(addr, codeHash, acct.Code.Code); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *binaryHasher) UpdateStorage(address common.Address, keys []common.Hash, values []common.Hash) error {
	for i, key := range keys {
		if values[i] == (common.Hash{}) {
			if err := h.trie.DeleteStorage(address, key[:]); err != nil {
				return err
			}
		} else {
			if err := h.trie.UpdateStorage(address, key[:], values[i][:]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *binaryHasher) Hash() common.Hash {
	return h.trie.Hash()
}

func (h *binaryHasher) Commit() (common.Hash, *trienode.MergedNodeSet, map[common.Address]Hashes, error) {
	root, set := h.trie.Commit(false)
	nodes := trienode.NewMergedNodeSet()
	if set != nil {
		if err := nodes.Merge(set); err != nil {
			return common.Hash{}, nil, nil, err
		}
	}
	// The binary trie is a single unified structure with no per-account
	// storage sub-tries, so there are no secondary hashes to report.
	return root, nodes, nil, nil
}

func (h *binaryHasher) Close() {}

func (h *binaryHasher) Copy() Hasher {
	return &binaryHasher{
		db:   h.db,
		root: h.root,
		trie: h.trie.Copy(),
	}
}
