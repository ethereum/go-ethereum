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

package indexer

import (
	"testing"

	"github.com/ethereum/go-ethereum/statediff/indexer/postgres"
)

// TearDownDB is used to tear down the watcher dbs after tests
func TearDownDB(t *testing.T, db *postgres.DB) {
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}

	_, err = tx.Exec(`DELETE FROM eth.header_cids`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(`DELETE FROM eth.transaction_cids`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(`DELETE FROM eth.receipt_cids`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(`DELETE FROM eth.state_cids`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(`DELETE FROM eth.storage_cids`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = tx.Exec(`DELETE FROM blocks`)
	if err != nil {
		t.Fatal(err)
	}
	err = tx.Commit()
	if err != nil {
		t.Fatal(err)
	}
}
