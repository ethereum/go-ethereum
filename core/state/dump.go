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
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

// DumpAccount represents an account in the state
type DumpAccount struct {
	Balance  string                 `json:"balance"`
	Nonce    uint64                 `json:"nonce"`
	Root     string                 `json:"root"`
	CodeHash string                 `json:"codeHash"`
	Code     string                 `json:"code"`
	Storage  map[common.Hash]string `json:"storage"`
	Address  *common.Address        `json:"address,omitempty"` // Address only present in iterative (line-by-line) mode
}

// Dump represents the full dump in a collected format, as one large map
type Dump struct {
	Root     string                         `json:"root"`
	Accounts map[common.Address]DumpAccount `json:"accounts"`
}

// iterativeDump is a 'collector'-implementation which dump output line-by-line iteratively
type iterativeDump json.Encoder

// Collector interface which the state trie calls during iteration
type collector interface {
	onRoot(common.Hash)
	onAccount(common.Address, DumpAccount)
}

func (self *Dump) onRoot(root common.Hash) {
	self.Root = fmt.Sprintf("%x", root)
}

func (self *Dump) onAccount(addr common.Address, account DumpAccount) {
	self.Accounts[addr] = account
}

func (self iterativeDump) onAccount(addr common.Address, account DumpAccount) {
	dumpAccount := &DumpAccount{
		Balance:  account.Balance,
		Nonce:    account.Nonce,
		Root:     account.Root,
		CodeHash: account.CodeHash,
		Code:     account.Code,
		Storage:  account.Storage,
		Address:  nil,
	}
	if addr != (common.Address{}) {
		dumpAccount.Address = &addr
	}
	(*json.Encoder)(&self).Encode(dumpAccount)
}
func (self iterativeDump) onRoot(root common.Hash) {
	(*json.Encoder)(&self).Encode(struct {
		Root common.Hash `json:"root"`
	}{root})
}

func (self *StateDB) dump(c collector, excludeCode, excludeStorage bool) {
	emptyAddress := (common.Address{})
	missingPreimages := 0
	c.onRoot(self.trie.Hash())
	it := trie.NewIterator(self.trie.NodeIterator(nil))
	for it.Next() {
		addr := common.BytesToAddress(self.trie.GetKey(it.Key))
		if emptyAddress == addr {
			// We don't have the preimage. All accounts missing preimages
			// will be 'mapped' and overwrite the same entry, which is quite useless.
			// Make note and continue
			missingPreimages++
			continue
		}
		var data Account
		if err := rlp.DecodeBytes(it.Value, &data); err != nil {
			panic(err)
		}
		obj := newObject(nil, addr, data)
		account := DumpAccount{
			Balance:  data.Balance.String(),
			Nonce:    data.Nonce,
			Root:     common.Bytes2Hex(data.Root[:]),
			CodeHash: common.Bytes2Hex(data.CodeHash),
		}
		if !excludeCode {
			account.Code = common.Bytes2Hex(obj.Code(self.db))
		}
		if !excludeStorage {
			account.Storage = make(map[common.Hash]string)
			storageIt := trie.NewIterator(obj.getTrie(self.db).NodeIterator(nil))
			for storageIt.Next() {
				account.Storage[common.BytesToHash(self.trie.GetKey(storageIt.Key))] = common.Bytes2Hex(storageIt.Value)
			}
		}
		c.onAccount(addr, account)
	}
	if missingPreimages > 0 {
		log.Warn("Dump incomplete due to missing preimages", "missing", missingPreimages)
	}
}

// RawDump returns the entire state an a single large object
func (self *StateDB) RawDump(excludeCode, excludeStorage bool) Dump {
	dump := &Dump{
		Accounts: make(map[common.Address]DumpAccount),
	}
	self.dump(dump, excludeCode, excludeStorage)
	return *dump
}

// Dump returns a JSON string representing the entire state as a single json-object
func (self *StateDB) Dump(excludeCode, excludeStorage bool) []byte {
	dump := self.RawDump(excludeCode, excludeStorage)
	json, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		fmt.Println("dump err", err)
	}
	return json
}

// IterativeDump dumps out accounts as json-objects, delimited by linebreaks on stdout
func (self *StateDB) IterativeDump(excludeCode, excludeStorage bool, output *json.Encoder) {
	self.dump(iterativeDump(*output), excludeCode, excludeStorage)
}
