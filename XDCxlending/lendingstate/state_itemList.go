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

package lendingstate

import (
	"bytes"
	"fmt"
	"io"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

type itemListState struct {
	lendingBook common.Hash
	key         common.Hash
	data        itemList

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by LendingStateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	cachedStorage map[common.Hash]common.Hash // Storage entry cache to avoid duplicate reads
	dirtyStorage  map[common.Hash]common.Hash // Storage entries that need to be flushed to disk

	onDirty func(price common.Hash) // Callback method to mark a state object newly dirty
}

func (il *itemListState) empty() bool {
	return il.data.Volume == nil || il.data.Volume.Sign() == 0
}

func newItemListState(lendingBook common.Hash, key common.Hash, data itemList, onDirty func(price common.Hash)) *itemListState {
	return &itemListState{
		lendingBook:   lendingBook,
		key:           key,
		data:          data,
		cachedStorage: make(map[common.Hash]common.Hash),
		dirtyStorage:  make(map[common.Hash]common.Hash),
		onDirty:       onDirty,
	}
}

// EncodeRLP implements rlp.Encoder.
func (il *itemListState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, il.data)
}

// setError remembers the first non-nil error it is called with.
func (il *itemListState) setError(err error) {
	if il.dbErr == nil {
		il.dbErr = err
	}
}

func (il *itemListState) getTrie(db Database) Trie {
	if il.trie == nil {
		var err error
		il.trie, err = db.OpenStorageTrie(il.key, il.data.Root)
		if err != nil {
			il.trie, _ = db.OpenStorageTrie(il.key, EmptyHash)
			il.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return il.trie
}

func (il *itemListState) GetOrderAmount(db Database, orderId common.Hash) common.Hash {
	amount, exists := il.cachedStorage[orderId]
	if exists {
		return amount
	}
	// Load from DB in case it is missing.
	enc, err := il.getTrie(db).TryGet(orderId[:])
	if err != nil {
		il.setError(err)
		return EmptyHash
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			il.setError(err)
		}
		amount.SetBytes(content)
	}
	if (amount != common.Hash{}) {
		il.cachedStorage[orderId] = amount
	}
	return amount
}

func (il *itemListState) insertLendingItem(db Database, orderId common.Hash, amount common.Hash) {
	il.setOrderItem(orderId, amount)
	il.setError(il.getTrie(db).TryUpdate(orderId[:], amount[:]))
}

func (il *itemListState) removeOrderItem(db Database, orderId common.Hash) {
	tr := il.getTrie(db)
	il.setError(tr.TryDelete(orderId[:]))
	il.setOrderItem(orderId, EmptyHash)
}

func (il *itemListState) setOrderItem(orderId common.Hash, amount common.Hash) {
	il.cachedStorage[orderId] = amount
	il.dirtyStorage[orderId] = amount

	if il.onDirty != nil {
		il.onDirty(il.key)
		il.onDirty = nil
	}
}

// updateAskTrie writes cached storage modifications into the object's storage trie.
func (il *itemListState) updateTrie(db Database) Trie {
	tr := il.getTrie(db)
	for orderId, amount := range il.dirtyStorage {
		delete(il.dirtyStorage, orderId)
		if amount == EmptyHash {
			il.setError(tr.TryDelete(orderId[:]))
			continue
		}
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(amount[:], "\x00"))
		il.setError(tr.TryUpdate(orderId[:], v))
	}
	return tr
}

// UpdateRoot sets the trie root to the current root tradeId of
func (il *itemListState) updateRoot(db Database) error {
	il.updateTrie(db)
	if il.dbErr != nil {
		return il.dbErr
	}
	root, err := il.trie.Commit(nil)
	if err == nil {
		il.data.Root = root
	}
	return err
}

func (il *itemListState) deepCopy(db *LendingStateDB, onDirty func(price common.Hash)) *itemListState {
	stateOrderList := newItemListState(il.lendingBook, il.key, il.data, onDirty)
	if il.trie != nil {
		stateOrderList.trie = db.db.CopyTrie(il.trie)
	}
	for orderId, amount := range il.dirtyStorage {
		stateOrderList.dirtyStorage[orderId] = amount
	}
	for orderId, amount := range il.cachedStorage {
		stateOrderList.cachedStorage[orderId] = amount
	}
	return stateOrderList
}

func (il *itemListState) AddVolume(amount *big.Int) {
	il.setVolume(new(big.Int).Add(il.data.Volume, amount))
}

func (il *itemListState) subVolume(amount *big.Int) {
	il.setVolume(new(big.Int).Sub(il.data.Volume, amount))
}

func (il *itemListState) setVolume(volume *big.Int) {
	il.data.Volume = volume
	if il.onDirty != nil {
		il.onDirty(il.key)
		il.onDirty = nil
	}
}

func (il *itemListState) Volume() *big.Int {
	return il.data.Volume
}
