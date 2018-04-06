/*
 * Copyright 2017 Dgraph Labs, Inc. and Contributors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package badger

import (
	"github.com/dgraph-io/badger/options"
)

// NOTE: Keep the comments in the following to 75 chars width, so they
// format nicely in godoc.

// Options are params for creating DB object.
//
// This package provides DefaultOptions which contains options that should
// work for most applications. Consider using that as a starting point before
// customizing it for your own needs.
type Options struct {
	// 1. Mandatory flags
	// -------------------
	// Directory to store the data in. Should exist and be writable.
	Dir string
	// Directory to store the value log in. Can be the same as Dir. Should
	// exist and be writable.
	ValueDir string

	// 2. Frequently modified flags
	// -----------------------------
	// Sync all writes to disk. Setting this to true would slow down data
	// loading significantly.
	SyncWrites bool

	// How should LSM tree be accessed.
	TableLoadingMode options.FileLoadingMode

	// How should value log be accessed
	ValueLogLoadingMode options.FileLoadingMode

	// 3. Flags that user might want to review
	// ----------------------------------------
	// The following affect all levels of LSM tree.
	MaxTableSize        int64 // Each table (or file) is at most this size.
	LevelSizeMultiplier int   // Equals SizeOf(Li+1)/SizeOf(Li).
	MaxLevels           int   // Maximum number of levels of compaction.
	// If value size >= this threshold, only store value offsets in tree.
	ValueThreshold int
	// Maximum number of tables to keep in memory, before stalling.
	NumMemtables int
	// The following affect how we handle LSM tree L0.
	// Maximum number of Level 0 tables before we start compacting.
	NumLevelZeroTables int

	// If we hit this number of Level 0 tables, we will stall until L0 is
	// compacted away.
	NumLevelZeroTablesStall int

	// Maximum total size for L1.
	LevelOneSize int64

	// Size of single value log file.
	ValueLogFileSize int64

	// Number of compaction workers to run concurrently.
	NumCompactors int

	// Transaction start and commit timestamps are manaVgedTxns by end-user. This
	// is a private option used by ManagedDB.
	managedTxns bool

	// 4. Flags for testing purposes
	// ------------------------------
	DoNotCompact bool // Stops LSM tree from compactions.

	maxBatchCount int64 // max entries in batch
	maxBatchSize  int64 // max batch size in bytes
}

// DefaultOptions sets a list of recommended options for good performance.
// Feel free to modify these to suit your needs.
var DefaultOptions = Options{
	DoNotCompact:        false,
	LevelOneSize:        256 << 20,
	LevelSizeMultiplier: 10,
	TableLoadingMode:    options.LoadToRAM,
	ValueLogLoadingMode: options.MemoryMap,
	// table.MemoryMap to mmap() the tables.
	// table.Nothing to not preload the tables.
	MaxLevels:               7,
	MaxTableSize:            64 << 20,
	NumCompactors:           3,
	NumLevelZeroTables:      5,
	NumLevelZeroTablesStall: 10,
	NumMemtables:            5,
	SyncWrites:              true,
	// Nothing to read/write value log using standard File I/O
	// MemoryMap to mmap() the value log files
	ValueLogFileSize: 1 << 30,
	ValueThreshold:   20,
}
