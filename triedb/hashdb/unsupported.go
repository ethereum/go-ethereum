// Copyright 2018 The go-ethereum Authors
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

package hashdb

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// These methods are implemented by pathdb but not supported by hashdb. They are
// included for interface parity to avoid triedb having to check concrete types.

var errUnsupported = errors.New("method not supported by hashdb")

// Recover isn't supported and always returns an error.
func (*Database) Recover(target common.Hash) error { return errUnsupported }

// Recoverable isn't supported and always returns an error.
func (*Database) Recoverable(root common.Hash) (bool, error) { return false, errUnsupported }

// Disable isn't supported and always returns an error.
func (*Database) Disable() error { return errUnsupported }

// Enable isn't supported and always returns an error.
func (*Database) Enable(root common.Hash) error { return errUnsupported }

// Journal isn't supported and always returns an error.
func (*Database) Journal(root common.Hash) error { return errUnsupported }

// SetBufferSize isn't supported and always returns an error.
func (*Database) SetBufferSize(int) error { return errUnsupported }

// AccountHistory isn't supported and always returns an error.
func (*Database) AccountHistory(_ common.Address, start, end uint64) (*pathdb.HistoryStats, error) {
	return nil, errUnsupported
}

// StorageHistory isn't supported and always returns an error.
func (*Database) StorageHistory(_ common.Address, slot common.Hash, start, end uint64) (*pathdb.HistoryStats, error) {
	return nil, errUnsupported
}

// HistoryRange isn't supported and always returns an error.
func (*Database) HistoryRange() (uint64, uint64, error) { return 0, 0, errUnsupported }
