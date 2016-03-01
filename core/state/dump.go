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

package state

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
)

type Account struct {
	Balance  string            `json:"balance"`
	Nonce    uint64            `json:"nonce"`
	Root     string            `json:"root"`
	CodeHash string            `json:"codeHash"`
	Code     string            `json:"code"`
	Storage  map[string]string `json:"storage"`
}

type World struct {
	Root     string             `json:"root"`
	Accounts map[string]Account `json:"accounts"`
}

func (self *StateDB) RawDump() World {
	world := World{
		Root:     hex.EncodeToString(self.trie.Root()),
		Accounts: make(map[string]Account),
	}

	it := self.trie.Iterator()
	for it.Next() {
		addr := self.trie.GetKey(it.Key)
		stateObject, _ := DecodeObject(common.BytesToAddress(addr), self.db, it.Value)

		account := Account{Balance: stateObject.balance.String(), Nonce: stateObject.nonce, Root: hex.EncodeToString(stateObject.Root()), CodeHash: hex.EncodeToString(stateObject.codeHash), Code: hex.EncodeToString(stateObject.Code())}
		account.Storage = make(map[string]string)

		storageIt := stateObject.trie.Iterator()
		for storageIt.Next() {
			account.Storage[hex.EncodeToString(self.trie.GetKey(storageIt.Key))] = hex.EncodeToString(storageIt.Value)
		}
		world.Accounts[hex.EncodeToString(addr)] = account
	}
	return world
}

func (self *StateDB) Dump() []byte {
	json, err := json.MarshalIndent(self.RawDump(), "", "    ")
	if err != nil {
		fmt.Println("dump err", err)
	}

	return json
}

// Debug stuff
func (self *StateObject) CreateOutputForDiff() {
	fmt.Printf("%x %x %x %x\n", self.Address(), self.Root(), self.balance.Bytes(), self.nonce)
	it := self.trie.Iterator()
	for it.Next() {
		fmt.Printf("%x %x\n", it.Key, it.Value)
	}
}
