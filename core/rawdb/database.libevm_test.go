// Copyright 2025 the libevm authors.
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

package rawdb_test

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/ava-labs/libevm/common"
	// To ensure that all methods are available to importing packages, this test
	// is defined in package `rawdb_test` instead of `rawdb`.
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/ethdb"
)

// ExampleDatabaseStat demonstrates the method signatures of DatabaseStat, which
// exposes an otherwise unexported type that won't have its methods documented.
func ExampleDatabaseStat() {
	var stat rawdb.DatabaseStat

	stat.Add(common.StorageSize(1)) // only to demonstrate param type
	stat.Add(2)

	fmt.Println("Sum:", stat.Size())    // sum of all values passed to Add()
	fmt.Println("Count:", stat.Count()) // number of calls to Add()

	// Output:
	// Sum: 3.00 B
	// Count: 2
}

func ExampleInspectDatabase() {
	db := &stubDatabase{
		iterator: &stubIterator{
			kvs: []keyValue{
				// Bloom bits total = 5 + 1 = 6
				{key: []byte("iBxxx"), value: []byte("m")},
				// Optional stat record total = 5 + 7 = 12
				{key: []byte("mykey"), value: []byte("myvalue")},
				// metadata total = (13 + 7) + (14 + 7) = 41
				{key: []byte("mymetadatakey"), value: []byte("myvalue")},
				{key: []byte("mymetadatakey2"), value: []byte("myvalue")},
			},
		},
	}

	keyPrefix := []byte(nil)
	keyStart := []byte(nil)

	var (
		myStat rawdb.DatabaseStat
	)
	options := []rawdb.InspectDatabaseOption{
		rawdb.WithDatabaseStatRecorder(func(key []byte, size common.StorageSize) bool {
			if bytes.Equal(key, []byte("mykey")) {
				myStat.Add(size)
				return true
			}
			return false
		}),
		rawdb.WithDatabaseMetadataKeys(func(key []byte) bool {
			return bytes.Equal(key, []byte("mymetadatakey"))
		}),
		rawdb.WithDatabaseMetadataKeys(func(key []byte) bool {
			return bytes.Equal(key, []byte("mymetadatakey2"))
		}),
		rawdb.WithDatabaseStatsTransformer(func(rows [][]string) [][]string {
			sort.Slice(rows, func(i, j int) bool {
				ri, rj := rows[i], rows[j]
				if ri[0] != rj[0] {
					return ri[0] < rj[0]
				}
				return ri[1] < rj[1]
			})

			return append(
				rows,
				[]string{"My database", "My category", myStat.Size(), myStat.Count()},
			)
		}),
	}

	err := rawdb.InspectDatabase(db, keyPrefix, keyStart, options...)
	if err != nil {
		fmt.Println(err)
	}
	// Output:
	// +-----------------------+-------------------------+---------+-------+
	// |       DATABASE        |        CATEGORY         |  SIZE   | ITEMS |
	// +-----------------------+-------------------------+---------+-------+
	// | Ancient store (Chain) | Bodies                  | 0.00 B  |     0 |
	// | Ancient store (Chain) | Diffs                   | 0.00 B  |     0 |
	// | Ancient store (Chain) | Hashes                  | 0.00 B  |     0 |
	// | Ancient store (Chain) | Headers                 | 0.00 B  |     0 |
	// | Ancient store (Chain) | Receipts                | 0.00 B  |     0 |
	// | Key-Value store       | Account snapshot        | 0.00 B  |     0 |
	// | Key-Value store       | Beacon sync headers     | 0.00 B  |     0 |
	// | Key-Value store       | Block hash->number      | 0.00 B  |     0 |
	// | Key-Value store       | Block number->hash      | 0.00 B  |     0 |
	// | Key-Value store       | Bloombit index          | 6.00 B  |     1 |
	// | Key-Value store       | Bodies                  | 0.00 B  |     0 |
	// | Key-Value store       | Clique snapshots        | 0.00 B  |     0 |
	// | Key-Value store       | Contract codes          | 0.00 B  |     0 |
	// | Key-Value store       | Difficulties            | 0.00 B  |     0 |
	// | Key-Value store       | Hash trie nodes         | 0.00 B  |     0 |
	// | Key-Value store       | Headers                 | 0.00 B  |     0 |
	// | Key-Value store       | Path trie account nodes | 0.00 B  |     0 |
	// | Key-Value store       | Path trie state lookups | 0.00 B  |     0 |
	// | Key-Value store       | Path trie storage nodes | 0.00 B  |     0 |
	// | Key-Value store       | Receipt lists           | 0.00 B  |     0 |
	// | Key-Value store       | Singleton metadata      | 41.00 B |     2 |
	// | Key-Value store       | Storage snapshot        | 0.00 B  |     0 |
	// | Key-Value store       | Transaction index       | 0.00 B  |     0 |
	// | Key-Value store       | Trie preimages          | 0.00 B  |     0 |
	// | Light client          | Bloom trie nodes        | 0.00 B  |     0 |
	// | Light client          | CHT trie nodes          | 0.00 B  |     0 |
	// | My database           | My category             | 12.00 B |     1 |
	// +-----------------------+-------------------------+---------+-------+
	// |                                  TOTAL          | 59.00 B |       |
	// +-----------------------+-------------------------+---------+-------+
}

type stubDatabase struct {
	ethdb.Database
	iterator ethdb.Iterator
}

func (s *stubDatabase) NewIterator(keyPrefix, keyStart []byte) ethdb.Iterator {
	return s.iterator
}

// AncientSize is used in [InspectDatabase] to determine the ancient sizes.
func (s *stubDatabase) AncientSize(kind string) (uint64, error) {
	return 0, nil
}

func (s *stubDatabase) Ancients() (uint64, error) {
	return 0, nil
}

func (s *stubDatabase) Tail() (uint64, error) {
	return 0, nil
}

func (s *stubDatabase) Get(key []byte) ([]byte, error) {
	return nil, nil
}

func (s *stubDatabase) ReadAncients(fn func(ethdb.AncientReaderOp) error) error {
	return nil
}

type stubIterator struct {
	ethdb.Iterator
	i   int // see [stubIterator.pos]
	kvs []keyValue
}

type keyValue struct {
	key   []byte
	value []byte
}

// pos returns the true iterator position, which is otherwise off by one because
// Next() is called _before_ usage.
func (s *stubIterator) pos() int {
	return s.i - 1
}

func (s *stubIterator) Next() bool {
	s.i++
	return s.pos() < len(s.kvs)
}

func (s *stubIterator) Release() {}

func (s *stubIterator) Key() []byte {
	return s.kvs[s.pos()].key
}

func (s *stubIterator) Value() []byte {
	return s.kvs[s.pos()].value
}
