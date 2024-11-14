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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/ethdb"
	"github.com/ava-labs/libevm/triedb/database"
)

func TestDBOverride(t *testing.T) {
	config := &Config{
		DBOverride: func(d ethdb.Database, c *Config) DBOverride {
			return override{}
		},
	}

	db := NewDatabase(nil, config)
	got, err := db.Reader(common.Hash{})
	require.NoError(t, err)
	if _, ok := got.(reader); !ok {
		t.Errorf("with non-nil %T.DBOverride, %T.Reader() got concrete type %T; want %T", config, db, got, reader{})
	}
}

type override struct {
	PathDB
}

type reader struct {
	database.Reader
}

func (override) Reader(common.Hash) (database.Reader, error) {
	return reader{}, nil
}
