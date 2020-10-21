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

package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type DerivableList interface {
	Len() int
	GetRlp(i int) []byte
}

// Hasher is the tool used to calculate the hash of derivable list.
type Hasher interface {
	Reset()
	Update([]byte, []byte)
	Hash() common.Hash
}

func DeriveSha(list DerivableList, hasher Hasher) common.Hash {
	hasher.Reset()

	// StackTrie requires values to be inserted in increasing
	// hash order, which is not the order that `list` provides
	// hashes in. This insertion sequence ensures that the
	// order is correct.

	buf := make([]byte, 9) // 9 bytes is the max an rlp-encoded int will ever use

	for i := 1; i < list.Len() && i <= 0x7f; i++ {
		off := rlp.PutInt(buf, i)
		hasher.Update(buf[:off], list.GetRlp(i))

	}
	if list.Len() > 0 {
		off := rlp.PutInt(buf, 0)
		hasher.Update(buf[:off], list.GetRlp(0))

	}
	for i := 0x80; i < list.Len(); i++ {
		off := rlp.PutInt(buf, i)
		hasher.Update(buf[:off], list.GetRlp(i))

	}
	return hasher.Hash()
}
