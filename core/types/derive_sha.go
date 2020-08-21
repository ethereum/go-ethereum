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
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// DerivableList is the interface which can derive the hash.
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
	keybuf := new(bytes.Buffer)
	for i := 0; i < list.Len(); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		hasher.Update(keybuf.Bytes(), list.GetRlp(i))
	}
	return hasher.Hash()
}
