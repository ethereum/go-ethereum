// Copyright 2019 The go-ethereum Authors
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

package rawdb

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// ReadCode retrieves the bytecode associated with a given hash.
func ReadCode(db DatabaseReader, hash common.Hash) []byte {
	code, _ := db.Get(codeKey(hash))
	return code
}

// WriteCode stores the bytecode associated with a given hash.
func WriteCode(db DatabaseWriter, hash common.Hash, code []byte) {
	if err := db.Put(codeKey(hash), code); err != nil {
		log.Crit("Failed to store bytecode", "err", err)
	}
}

// DeleteCode removes the bytecode associated with a given hash.
func DeleteCode(db DatabaseDeleter, hash common.Hash) {
	if err := db.Delete(codeKey(hash)); err != nil {
		log.Crit("Failed to delete bytecode", "err", err)
	}
}
