// Copyright 2021 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"strings"
	"time"

	pebblev2 "github.com/cockroachdb/pebble/v2"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
)

type trieBenchmarkMetrics struct {
	db        *pebblev2.Metrics
	stalls    int64
	stallTime time.Duration
}

func readTrieBenchmarkMetrics(disk *pebble.Database) trieBenchmarkMetrics {
	stalls, stallTime := disk.WriteStallStats()
	return trieBenchmarkMetrics{
		db:        disk.Metrics(),
		stalls:    stalls,
		stallTime: stallTime,
	}
}

// reportTrieBenchmarkMetrics excludes work completed before the timed run.
// End-of-run gauges show background work still outstanding at the cutoff.
func reportTrieBenchmarkMetrics(before, after trieBenchmarkMetrics) {
	var (
		placement []string
		ingested  uint64
		flushed   uint64
		read      uint64
		compacted uint64
	)
	for i, level := range after.db.Levels {
		start := before.db.Levels[i]
		ingestBytes := level.TableBytesIngested - start.TableBytesIngested
		flushBytes := level.TableBytesFlushed - start.TableBytesFlushed
		if tables := level.TablesIngested - start.TablesIngested; tables > 0 {
			placement = append(placement, fmt.Sprintf("L%d %d ingested SSTs (%d B)", i, tables, ingestBytes))
		}
		if tables := level.TablesFlushed - start.TablesFlushed; tables > 0 {
			placement = append(placement, fmt.Sprintf("L%d %d flushed SSTs (%d B)", i, tables, flushBytes))
		}
		ingested += ingestBytes
		flushed += flushBytes
		read += level.TableBytesRead - start.TableBytesRead
		compacted += level.TableBytesCompacted - start.TableBytesCompacted
	}
	if len(placement) == 0 {
		placement = append(placement, "no SSTs installed by generation end")
	}
	rewriteRatio := "n/a"
	if initial := ingested + flushed; initial > 0 {
		rewriteRatio = fmt.Sprintf("%.4f%%", 100*float64(compacted)/float64(initial))
	}
	startCompact, endCompact := before.db.Compact, after.db.Compact

	fmt.Println("\nPebble during trie generation")
	fmt.Printf("  initial SST placement:    %s\n", strings.Join(placement, ", "))
	fmt.Printf("  compaction SST read:      %d B\n", read)
	fmt.Printf("  compaction SST written:   %d B\n", compacted)
	fmt.Printf("  SST rewrite ratio:        %s (compacted / (ingested + flushed))\n", rewriteRatio)
	fmt.Printf("  compactions:              %d (%d moves), cumulative time %s\n",
		endCompact.Count-startCompact.Count, endCompact.MoveCount-startCompact.MoveCount, endCompact.Duration-startCompact.Duration)
	fmt.Printf("  write stalls:             %d started, completed stall time %s\n",
		after.stalls-before.stalls, after.stallTime-before.stallTime)
	fmt.Printf("  background at start:      %d compactions, %d flushes, %d B estimated debt\n",
		startCompact.NumInProgress, before.db.Flush.NumInProgress, startCompact.EstimatedDebt)
	fmt.Printf("  background at end:        %d compactions, %d flushes, %d B estimated debt\n",
		endCompact.NumInProgress, after.db.Flush.NumInProgress, endCompact.EstimatedDebt)
}
