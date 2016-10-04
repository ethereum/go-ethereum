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
}

type journal []journalEntry

type (
	// Changes to the account trie.
	createAccountChange struct {
		account *common.Address
	}
	deleteAccountChange struct {
		object *StateObject
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
		prev *big.Int
	}
	addLogChange struct {
		txhash common.Hash
	}
)

func (ch deleteAccountChange) undo(s *StateDB) {
	ch.object.remove = false
	s.SetStateObject(ch.object)
}

func (ch createAccountChange) undo(s *StateDB) {
	s.delete(*ch.account)
}

func (ch balanceChange) undo(s *StateDB) {
	s.GetOrNewStateObject(*ch.account).setBalance(ch.prev)
}

func (ch nonceChange) undo(s *StateDB) {
	s.GetOrNewStateObject(*ch.account).setNonce(ch.prev)
}

func (ch codeChange) undo(s *StateDB) {
	s.GetOrNewStateObject(*ch.account).setCode(common.BytesToHash(ch.prevhash), ch.prevcode)
}

func (ch storageChange) undo(s *StateDB) {
	s.GetOrNewStateObject(*ch.account).setState(ch.key, ch.prevalue)
}

func (ch refundChange) undo(s *StateDB) {
	s.refund = ch.prev
}

func (ch addLogChange) undo(s *StateDB) {
	logs := s.logs[ch.txhash]
	s.logs[ch.txhash] = logs[:len(logs)-1]
}
