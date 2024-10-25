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
	"github.com/XinFinOrg/XDPoSChain/trie"
)

type liquidationTimeState struct {
	time        common.Hash
	lendingBook common.Hash
	data        itemList

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by TradingStateDB.Commit.
	dbErr error

	// Write caches.
	trie Trie // storage trie, which becomes non-nil on first access

	cachedStorage map[common.Hash]common.Hash
	dirtyStorage  map[common.Hash]common.Hash

	onDirty func(time common.Hash) // Callback method to mark a state object newly dirty
}

func (lt *liquidationTimeState) empty() bool {
	return lt.data.Volume == nil || lt.data.Volume.Sign() == 0
}

func newLiquidationTimeState(time common.Hash, lendingBook common.Hash, data itemList, onDirty func(time common.Hash)) *liquidationTimeState {
	return &liquidationTimeState{
		lendingBook:   lendingBook,
		time:          time,
		data:          data,
		cachedStorage: make(map[common.Hash]common.Hash),
		dirtyStorage:  make(map[common.Hash]common.Hash),
		onDirty:       onDirty,
	}
}

func (lt *liquidationTimeState) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, lt.data)
}

func (lt *liquidationTimeState) setError(err error) {
	if lt.dbErr == nil {
		lt.dbErr = err
	}
}

func (lt *liquidationTimeState) getTrie(db Database) Trie {
	if lt.trie == nil {
		var err error
		lt.trie, err = db.OpenStorageTrie(lt.lendingBook, lt.data.Root)
		if err != nil {
			lt.trie, _ = db.OpenStorageTrie(lt.time, EmptyHash)
			lt.setError(fmt.Errorf("can't create storage trie: %v", err))
		}
	}
	return lt.trie
}

func (lt *liquidationTimeState) Exist(db Database, tradeId common.Hash) bool {
	amount, exists := lt.cachedStorage[tradeId]
	if exists {
		return true
	}
	// Load from DB in case it is missing.
	enc, err := lt.getTrie(db).TryGet(tradeId[:])
	if err != nil {
		lt.setError(err)
		return false
	}
	if len(enc) > 0 {
		_, content, _, err := rlp.Split(enc)
		if err != nil {
			lt.setError(err)
		}
		amount.SetBytes(content)
	}
	if (amount != common.Hash{}) {
		lt.cachedStorage[tradeId] = amount
	}
	return true
}

func (lt *liquidationTimeState) getAllTradeIds(db Database) []common.Hash {
	tradeIds := []common.Hash{}
	lendingBookTrie := lt.getTrie(db)
	if lendingBookTrie == nil {
		return tradeIds
	}
	for id, value := range lt.cachedStorage {
		if !common.EmptyHash(value) {
			tradeIds = append(tradeIds, id)
		}
	}
	orderListIt := trie.NewIterator(lendingBookTrie.NodeIterator(nil))
	for orderListIt.Next() {
		id := common.BytesToHash(orderListIt.Key)
		if _, exist := lt.cachedStorage[id]; exist {
			continue
		}
		tradeIds = append(tradeIds, id)
	}
	return tradeIds
}

func (lt *liquidationTimeState) insertTradeId(db Database, tradeId common.Hash) {
	lt.setTradeId(tradeId, tradeId)
	lt.setError(lt.getTrie(db).TryUpdate(tradeId[:], tradeId[:]))
}

func (lt *liquidationTimeState) removeTradeId(db Database, tradeId common.Hash) {
	tr := lt.getTrie(db)
	lt.setError(tr.TryDelete(tradeId[:]))
	lt.setTradeId(tradeId, EmptyHash)
}

func (lt *liquidationTimeState) setTradeId(tradeId common.Hash, value common.Hash) {
	lt.cachedStorage[tradeId] = value
	lt.dirtyStorage[tradeId] = value

	if lt.onDirty != nil {
		lt.onDirty(lt.lendingBook)
		lt.onDirty = nil
	}
}

func (lt *liquidationTimeState) updateTrie(db Database) Trie {
	tr := lt.getTrie(db)
	for key, value := range lt.dirtyStorage {
		delete(lt.dirtyStorage, key)
		if value == EmptyHash {
			lt.setError(tr.TryDelete(key[:]))
			continue
		}
		v, _ := rlp.EncodeToBytes(bytes.TrimLeft(value[:], "\x00"))
		lt.setError(tr.TryUpdate(key[:], v))
	}
	return tr
}

func (lt *liquidationTimeState) updateRoot(db Database) error {
	lt.updateTrie(db)
	if lt.dbErr != nil {
		return lt.dbErr
	}
	root, err := lt.trie.Commit(nil)
	if err == nil {
		lt.data.Root = root
	}
	return err
}

func (lt *liquidationTimeState) deepCopy(db *LendingStateDB, onDirty func(time common.Hash)) *liquidationTimeState {
	stateLendingBook := newLiquidationTimeState(lt.lendingBook, lt.time, lt.data, onDirty)
	if lt.trie != nil {
		stateLendingBook.trie = db.db.CopyTrie(lt.trie)
	}
	for key, value := range lt.dirtyStorage {
		stateLendingBook.dirtyStorage[key] = value
	}
	for key, value := range lt.cachedStorage {
		stateLendingBook.cachedStorage[key] = value
	}
	return stateLendingBook
}

func (lt *liquidationTimeState) AddVolume(amount *big.Int) {
	lt.setVolume(new(big.Int).Add(lt.data.Volume, amount))
}

func (lt *liquidationTimeState) subVolume(amount *big.Int) {
	lt.setVolume(new(big.Int).Sub(lt.data.Volume, amount))
}

func (lt *liquidationTimeState) setVolume(volume *big.Int) {
	lt.data.Volume = volume
	if lt.onDirty != nil {
		lt.onDirty(lt.lendingBook)
		lt.onDirty = nil
	}
}

func (lt *liquidationTimeState) Volume() *big.Int {
	return lt.data.Volume
}
