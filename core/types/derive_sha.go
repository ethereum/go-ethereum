// Copyright 2014 The go-expanse Authors
// This file is part of the go-expanse library.
//
// The go-expanse library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-expanse library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-expanse library. If not, see <http://www.gnu.org/licenses/>.

package types

import (
	"github.com/expanse-project/go-expanse/common"
	"github.com/expanse-project/go-expanse/ethdb"
	"github.com/expanse-project/go-expanse/rlp"
	"github.com/expanse-project/go-expanse/trie"
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

func DeriveSha(list DerivableList) common.Hash {
	db, _ := ethdb.NewMemDatabase()
	trie := trie.New(nil, db)
	for i := 0; i < list.Len(); i++ {
		key, _ := rlp.EncodeToBytes(uint(i))
		trie.Update(key, list.GetRlp(i))
	}

	return common.BytesToHash(trie.Root())
}
