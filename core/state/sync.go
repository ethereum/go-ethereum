// Copyright 2015 The go-ethereum Authors
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
	"bytes"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
)

type StateSync struct {
	db          ethdb.Database
	sync        *trie.TrieSync
	codeReqs    map[common.Hash]struct{} // requested but not yet written to database
	codeReqList []common.Hash            // requested since last Missing
}

var sha3_nil = common.BytesToHash(sha3.NewKeccak256().Sum(nil))

func NewStateSync(root common.Hash, db ethdb.Database) *StateSync {
	ss := &StateSync{
		db:       db,
		codeReqs: make(map[common.Hash]struct{}),
	}
	ss.codeReqs[sha3_nil] = struct{}{} // never request the nil hash
	ss.sync = trie.NewTrieSync(root, db, ss.leafFound)
	return ss
}

func (self *StateSync) leafFound(leaf []byte, parent common.Hash) error {
	var obj struct {
		Nonce    uint64
		Balance  *big.Int
		Root     common.Hash
		CodeHash []byte
	}
	if err := rlp.Decode(bytes.NewReader(leaf), &obj); err != nil {
		return err
	}
	self.sync.AddSubTrie(obj.Root, 64, parent, nil)

	codehash := common.BytesToHash(obj.CodeHash)
	if _, ok := self.codeReqs[codehash]; !ok {
		code, _ := self.db.Get(obj.CodeHash)
		if code == nil {
			self.codeReqs[codehash] = struct{}{}
			self.codeReqList = append(self.codeReqList, codehash)
		}
	}
	return nil
}

func (self *StateSync) Missing(max int) []common.Hash {
	cr := len(self.codeReqList)
	gh := 0
	if max != 0 {
		if cr > max {
			cr = max
		}
		gh = max - cr
	}
	list := append(self.sync.Missing(gh), self.codeReqList[:cr]...)
	self.codeReqList = self.codeReqList[cr:]
	return list
}

func (self *StateSync) Process(list []trie.SyncResult) error {
	for i := 0; i < len(list); i++ {
		if _, ok := self.codeReqs[list[i].Hash]; ok { // code data, not a node
			self.db.Put(list[i].Hash[:], list[i].Data)
			delete(self.codeReqs, list[i].Hash)
			list[i] = list[len(list)-1]
			list = list[:len(list)-1]
			i--
		}
	}
	_, err := self.sync.Process(list)
	return err
}
