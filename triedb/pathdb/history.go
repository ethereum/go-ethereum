// Copyright 2025 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

var (
	errHeadTruncationOutOfRange = errors.New("history head truncation out of range")
	errTailTruncationOutOfRange = errors.New("history tail truncation out of range")
)

// truncateFromHead removes excess elements from the head of the freezer based
// on the given parameters. It returns the number of items that were removed.
func truncateFromHead(store ethdb.AncientStore, nhead uint64) (int, error) {
	ohead, err := store.Ancients()
	if err != nil {
		return 0, err
	}
	otail, err := store.Tail()
	if err != nil {
		return 0, err
	}
	log.Info("Truncating from head", "ohead", ohead, "tail", otail, "nhead", nhead)

	// Ensure that the truncation target falls within the valid range.
	if ohead < nhead || nhead < otail {
		return 0, fmt.Errorf("%w, tail: %d, head: %d, target: %d", errHeadTruncationOutOfRange, otail, ohead, nhead)
	}
	// Short circuit if nothing to truncate.
	if ohead == nhead {
		return 0, nil
	}
	ohead, err = store.TruncateHead(nhead)
	if err != nil {
		return 0, err
	}
	// Associated root->id mappings are left in the database and wait
	// for overwriting.
	return int(ohead - nhead), nil
}

// truncateFromTail removes excess elements from the end of the freezer based
// on the given parameters. It returns the number of items that were removed.
func truncateFromTail(store ethdb.AncientStore, ntail uint64) (int, error) {
	ohead, err := store.Ancients()
	if err != nil {
		return 0, err
	}
	otail, err := store.Tail()
	if err != nil {
		return 0, err
	}
	// Ensure that the truncation target falls within the valid range.
	if otail > ntail || ntail > ohead {
		return 0, fmt.Errorf("%w, tail: %d, head: %d, target: %d", errTailTruncationOutOfRange, otail, ohead, ntail)
	}
	// Short circuit if nothing to truncate.
	if otail == ntail {
		return 0, nil
	}
	otail, err = store.TruncateTail(ntail)
	if err != nil {
		return 0, err
	}
	// Associated root->id mappings are left in the database.
	return int(ntail - otail), nil
}
