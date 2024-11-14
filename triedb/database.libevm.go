// Copyright 2024 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package triedb

import (
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/ethdb"
	"github.com/ava-labs/libevm/log"
	"github.com/ava-labs/libevm/trie/triestate"
	"github.com/ava-labs/libevm/triedb/database"
	"github.com/ava-labs/libevm/triedb/hashdb"
	"github.com/ava-labs/libevm/triedb/pathdb"
)

// BackendDB defines the intersection of methods shared by [hashdb.Database] and
// [pathdb.Database]. It is defined to export an otherwise internal type used by
// the non-libevm geth implementation.
type BackendDB backend

// A ReaderProvider exposes its underlying Reader as an interface. Both
// [hashdb.Database] and [pathdb.Database] return concrete types so Go's lack of
// support for [covariant types] means that this method can't be defined on
// [BackendDB].
//
// [covariant types]: https://go.dev/doc/faq#covariant_types
type ReaderProvider interface {
	Reader(common.Hash) (database.Reader, error)
}

// A DBConstructor constructs alternative backend-database implementations.
type DBConstructor func(ethdb.Database, *Config) DBOverride

// A DBOverride is an arbitrary implementation of a [Database] backend. It MUST
// be either a [HashDB] or a [PathDB].
type DBOverride interface {
	BackendDB
	ReaderProvider
}

func (db *Database) overrideBackend(diskdb ethdb.Database, config *Config) bool {
	if config.DBOverride == nil {
		return false
	}
	if config.HashDB != nil || config.PathDB != nil {
		log.Crit("Database override provided when 'hash' or 'path' mode are configured")
	}

	db.backend = config.DBOverride(diskdb, config)
	switch db.backend.(type) {
	case HashDB:
	case PathDB:
	default:
		log.Crit("Database override is neither hash- nor path-based")
	}
	return true
}

var (
	// If either of these break then the respective interface SHOULD be updated.
	_ HashDB = (*hashdb.Database)(nil)
	_ PathDB = (*pathdb.Database)(nil)
)

// A HashDB mirrors the functionality of a [hashdb.Database].
type HashDB interface {
	BackendDB

	Cap(limit common.StorageSize) error
	Reference(root common.Hash, parent common.Hash)
	Dereference(root common.Hash)
}

// A PathDB mirrors the functionality of a [pathdb.Database].
type PathDB interface {
	BackendDB

	Recover(root common.Hash, loader triestate.TrieLoader) error
	Recoverable(root common.Hash) bool
	Disable() error
	Enable(root common.Hash) error
	Journal(root common.Hash) error
	SetBufferSize(size int) error
}
