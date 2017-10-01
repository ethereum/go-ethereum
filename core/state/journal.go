// Copyright 2016 The go-ethereum Authors
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
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type journalEntry interface {
	undo(*StateDB)
	getAccount() *common.Address
}

type journal struct {
	entries        []journalEntry
	dirtyOverrides []common.Address
}

func (j *journal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
}

func (j *journal) flatten() map[common.Address]struct{} {

	dirtyObjects := make(map[common.Address]struct{})
	for _, journalEntry := range j.entries {
		if addr := journalEntry.getAccount(); addr != nil {
			dirtyObjects[*addr] = struct{}{}
		}
	}
	for _, addr := range j.dirtyOverrides {
		dirtyObjects[addr] = struct{}{}
	}
	return dirtyObjects
}

// Length returns the number of journal entries in the journal
func (j *journal) Length() int {
	return len(j.entries)
}

func (j *journal) dirtyOverride(address common.Address) {
	j.dirtyOverrides = append(j.dirtyOverrides, address)
}

type (
	// Changes to the account trie.
	createObjectChange struct {
		account *common.Address
	}
	resetObjectChange struct {
		prev *stateObject
	}
	suicideChange struct {
		account     *common.Address
		prev        bool // whether account had already suicided
		prevbalance *big.Int
	}

	// Changes to individual accounts.
	balanceChange struct {
		account *common.Address
		prev    *big.Int
	}
	nonceChange struct {
		account *common.Address
		prev    uint64
	}
	storageChange struct {
		account       *common.Address
		key, prevalue common.Hash
	}
	codeChange struct {
		account            *common.Address
		prevcode, prevhash []byte
	}

	// Changes to other state values.
	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash common.Hash
	}
	addPreimageChange struct {
		hash common.Hash
	}
	touchChange struct {
		account   *common.Address
		prev      bool
		prevDirty bool
	}
)

func (ch createObjectChange) undo(s *StateDB) {
	delete(s.stateObjects, *ch.account)
	delete(s.stateObjectsDirty, *ch.account)
}

func (ch createObjectChange) getAccount() *common.Address {
	return ch.account
}

func (ch resetObjectChange) undo(s *StateDB) {
	s.setStateObject(ch.prev)
}

func (ch resetObjectChange) getAccount() *common.Address {
	return nil
}

func (ch suicideChange) undo(s *StateDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}
func (ch suicideChange) getAccount() *common.Address {
	return ch.account
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) undo(s *StateDB) {
}
func (ch touchChange) getAccount() *common.Address {
	return ch.account
}

func (ch balanceChange) undo(s *StateDB) {
	s.getStateObject(*ch.account).setBalance(ch.prev)
}
func (ch balanceChange) getAccount() *common.Address {
	return ch.account
}

func (ch nonceChange) undo(s *StateDB) {
	s.getStateObject(*ch.account).setNonce(ch.prev)
}

func (ch nonceChange) getAccount() *common.Address {
	return ch.account
}
func (ch codeChange) undo(s *StateDB) {
	s.getStateObject(*ch.account).setCode(common.BytesToHash(ch.prevhash), ch.prevcode)
}
func (ch codeChange) getAccount() *common.Address {
	return ch.account
}

func (ch storageChange) undo(s *StateDB) {
	s.getStateObject(*ch.account).setState(ch.key, ch.prevalue)
}
func (ch storageChange) getAccount() *common.Address {
	return ch.account
}

func (ch refundChange) undo(s *StateDB) {
	s.refund = ch.prev
}
func (ch refundChange) getAccount() *common.Address {
	return nil
}

func (ch addLogChange) undo(s *StateDB) {
	logs := s.logs[ch.txhash]
	if len(logs) == 1 {
		delete(s.logs, ch.txhash)
	} else {
		s.logs[ch.txhash] = logs[:len(logs)-1]
	}
	s.logSize--
}
func (ch addLogChange) getAccount() *common.Address {
	return nil
}

func (ch addPreimageChange) undo(s *StateDB) {
	delete(s.preimages, ch.hash)
}

func (ch addPreimageChange) getAccount() *common.Address {
	return nil
}
