// VulcanizeDB
// Copyright © 2019 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package shared

import (
	"reflect"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"

	"github.com/ethereum/go-ethereum/statediff/indexer/node"
	"github.com/ethereum/go-ethereum/statediff/indexer/postgres"
)

func ExpectEqual(t *testing.T, got interface{}, want interface{}) {
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Expected: %v\nActual: %v", want, got)
	}
}

// SetupDB is use to setup a db for watcher tests
func SetupDB() (*postgres.DB, error) {
	uri := postgres.DbConnectionString(postgres.ConnectionParams{
		User:     "postgres",
		Password: "",
		Hostname: "localhost",
		Name:     "vulcanize_testing",
		Port:     5432,
	})
	return postgres.NewDB(uri, postgres.ConnectionConfig{}, node.Info{})
}

// ListContainsString used to check if a list of strings contains a particular string
func ListContainsString(sss []string, s string) bool {
	for _, str := range sss {
		if s == str {
			return true
		}
	}
	return false
}

// TestCID creates a basic CID for testing purposes
func TestCID(b []byte) cid.Cid {
	pref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   multihash.KECCAK_256,
		MhLength: -1,
	}
	c, _ := pref.Sum(b)
	return c
}
