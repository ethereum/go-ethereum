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

package rawdb

import "fmt"

// The list of table names of chain freezer.
const (
	// chainFreezerHeaderTable indicates the name of the freezer header table.
	chainFreezerHeaderTable = "headers"

	// chainFreezerHashTable indicates the name of the freezer canonical hash table.
	chainFreezerHashTable = "hashes"

	// chainFreezerBodiesTable indicates the name of the freezer block body table.
	chainFreezerBodiesTable = "bodies"

	// chainFreezerReceiptTable indicates the name of the freezer receipts table.
	chainFreezerReceiptTable = "receipts"

	// chainFreezerDifficultyTable indicates the name of the freezer total difficulty table.
	chainFreezerDifficultyTable = "diffs"
)

// chainFreezerNoSnappy configures whether compression is disabled for the ancient-tables.
// Hashes and difficulties don't compress well.
var chainFreezerNoSnappy = map[string]bool{
	chainFreezerHeaderTable:     false,
	chainFreezerHashTable:       true,
	chainFreezerBodiesTable:     false,
	chainFreezerReceiptTable:    false,
	chainFreezerDifficultyTable: true,
}

// The list of identifiers of ancient stores.
var (
	chainFreezerName = "chain" // the folder name of chain segment ancient store.
)

// freezers the collections of all builtin freezers.
var freezers = []string{chainFreezerName}

// InspectFreezerTable dumps out the index of a specific freezer table. The passed
// ancient indicates the path of root ancient directory where the chain freezer can
// be opened. Start and end specify the range for dumping out indexes.
// Note this function can only be used for debugging purposes.
func InspectFreezerTable(ancient string, freezerName string, tableName string, start, end int64) error {
	var (
		path   string
		tables map[string]bool
	)
	switch freezerName {
	case chainFreezerName:
		path, tables = resolveChainFreezerDir(ancient), chainFreezerNoSnappy
	default:
		return fmt.Errorf("unknown freezer, supported ones: %v", freezers)
	}
	noSnappy, exist := tables[tableName]
	if !exist {
		var names []string
		for name := range tables {
			names = append(names, name)
		}
		return fmt.Errorf("unknown table, supported ones: %v", names)
	}
	table, err := newFreezerTable(path, tableName, noSnappy, true)
	if err != nil {
		return err
	}
	table.dumpIndexStdout(start, end)
	return nil
}
