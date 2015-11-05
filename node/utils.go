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

package node

import (
	"path/filepath"

	"github.com/ethereum/go-ethereum/ethdb"
)

// openDatabase opens an existing database with the given name from within the
// specified data directory, creating one if none exists. If the data directory
// is empty, an ephemeral memory database is returned.
func openDatabase(dataDir string, name string, cache int) (ethdb.Database, error) {
	if dataDir == "" {
		return ethdb.NewMemDatabase()
	}
	return ethdb.NewLDBDatabase(filepath.Join(dataDir, name), cache)
}
