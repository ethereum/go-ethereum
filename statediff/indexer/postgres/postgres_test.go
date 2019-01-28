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

package postgres_test

import (
	"fmt"
	"strings"
	"testing"

	"math/big"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/ethereum/go-ethereum/statediff/indexer/node"
	"github.com/ethereum/go-ethereum/statediff/indexer/postgres"
	"github.com/ethereum/go-ethereum/statediff/indexer/shared"
)

var DBParams postgres.ConnectionParams

func expectContainsSubstring(t *testing.T, full string, sub string) {
	if !strings.Contains(full, sub) {
		t.Fatalf("Expected \"%v\" to contain substring \"%v\"\n", full, sub)
	}
}

func TestPostgresDB(t *testing.T) {
	var sqlxdb *sqlx.DB

	t.Run("connects to the database", func(t *testing.T) {
		var err error
		pgConfig := postgres.DbConnectionString(DBParams)

		sqlxdb, err = sqlx.Connect("postgres", pgConfig)

		if err != nil {
			t.Fatal(err)
		}
		if sqlxdb == nil {
			t.Fatal("DB is nil")
		}
	})

	t.Run("serializes big.Int to db", func(t *testing.T) {
		// postgres driver doesn't support go big.Int type
		// various casts in golang uint64, int64, overflow for
		// transaction value (in wei) even though
		// postgres numeric can handle an arbitrary
		// sized int, so use string representation of big.Int
		// and cast on insert

		pgConnectString := postgres.DbConnectionString(DBParams)
		db, err := sqlx.Connect("postgres", pgConnectString)
		if err != nil {
			t.Fatal(err)
		}
		if err != nil {
			t.Fatal(err)
		}

		bi := new(big.Int)
		bi.SetString("34940183920000000000", 10)
		shared.ExpectEqual(t, bi.String(), "34940183920000000000")

		defer db.Exec(`DROP TABLE IF EXISTS example`)
		_, err = db.Exec("CREATE TABLE example ( id INTEGER, data NUMERIC )")
		if err != nil {
			t.Fatal(err)
		}

		sqlStatement := `  
			INSERT INTO example (id, data)
			VALUES (1, cast($1 AS NUMERIC))`
		_, err = db.Exec(sqlStatement, bi.String())
		if err != nil {
			t.Fatal(err)
		}

		var data string
		err = db.QueryRow(`SELECT data FROM example WHERE id = 1`).Scan(&data)
		if err != nil {
			t.Fatal(err)
		}

		shared.ExpectEqual(t, bi.String(), data)
		actual := new(big.Int)
		actual.SetString(data, 10)
		shared.ExpectEqual(t, actual, bi)
	})

	t.Run("throws error when can't connect to the database", func(t *testing.T) {
		invalidDatabase := postgres.ConnectionParams{}
		node := node.Info{GenesisBlock: "GENESIS", NetworkID: "1", ID: "x123", ClientName: "geth"}

		_, err := postgres.NewDB(postgres.DbConnectionString(invalidDatabase),
			postgres.ConnectionConfig{}, node)

		if err == nil {
			t.Fatal("Expected an error")
		}

		expectContainsSubstring(t, err.Error(), postgres.DbConnectionFailedMsg)
	})

	t.Run("throws error when can't create node", func(t *testing.T) {
		badHash := fmt.Sprintf("x %s", strings.Repeat("1", 100))
		node := node.Info{GenesisBlock: badHash, NetworkID: "1", ID: "x123", ClientName: "geth"}

		_, err := postgres.NewDB(postgres.DbConnectionString(DBParams), postgres.ConnectionConfig{}, node)

		if err == nil {
			t.Fatal("Expected an error")
		}
		expectContainsSubstring(t, err.Error(), postgres.SettingNodeFailedMsg)
	})
}
