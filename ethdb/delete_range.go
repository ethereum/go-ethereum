// Copyright 2024 The go-ethereum Authors
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

// Package ethdb defines the interfaces for an Ethereum data store.
package ethdb

import (
	"bytes"
	"errors"
)

var ErrTooManyKeys = errors.New("too many keys in deleted range")

// DeleteRangeWithIterator is a fallback method for deleting a key range from a
// database that does not natively support range deletion.
// Note that the number of deleted keys is limited in order to avoid blocking for
// a very long time. ErrTooManyKeys is returned if the range has only been
// partially deleted. In this case the caller can repeat the call until it
// finally succeeds.
func DeleteRangeWithIterator(db KeyValueStore, start, end []byte) error {
	batch := db.NewBatch()
	it := db.NewIterator(nil, start)
	defer it.Release()

	var count int
	for it.Next() && bytes.Compare(end, it.Key()) > 0 {
		count++
		if count > 10000 { // should not block for more than a second
			if err := batch.Write(); err != nil {
				return err
			}
			return ErrTooManyKeys
		}
		if err := batch.Delete(it.Key()); err != nil {
			return err
		}
	}
	return batch.Write()
}
