// Copyright 2022 The go-ethereum Authors
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

package pathdb

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

// These methods are implemented by hashdb but not supported by pathdb. They are
// included for interface parity to avoid triedb having to check concrete types.

var errUnsupported = errors.New("method not supported by pathdb")

// Cap isn't supported and always returns an error.
func (*Database) Cap(limit common.StorageSize) error { return errUnsupported }

// Reference isn't supported and always returns an error.
func (*Database) Reference(root, parent common.Hash) error { return errUnsupported }

// Dereference isn't supported and always returns an error.
func (*Database) Dereference(root common.Hash) error { return errUnsupported }
