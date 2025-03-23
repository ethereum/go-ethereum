// Copyright 2025 The go-ethereum Authors
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
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/urfave/cli/v2"
)

var (
	filterPerfCommand = &cli.Command{
		Name:      "filterperf",
		Usage:     "Runs log filter performance test against an RPC endpoint",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    filterPerfCmd,
		Flags: []cli.Flag{
			testSepoliaFlag,
			testMainnetFlag,
			filterQueryFileFlag,
			filterErrorFileFlag,
		},
	}
)

const passCount = 3

func filterPerfCmd(ctx *cli.Context) error {
	cfg := testConfigFromCLI(ctx)
	f := newFilterTestSuite(cfg)

	type queryTest struct {
		query         *filterQuery
		bucket, index int
		runtime       []time.Duration
		medianTime    time.Duration
	}
	var queries, processed []queryTest
	for i, bucket := range f.queries[:] {
		for j, query := range bucket {
			queries = append(queries, queryTest{query: query, bucket: i, index: j})
		}
	}

	// Run test queries.
	var (
		failed, pruned, mismatch int
		errors                   []*filterQuery
	)
	for i := 1; i <= passCount; i++ {
		fmt.Println("Performance test pass", i, "/", passCount)
		for len(queries) > 0 {
			pick := rand.Intn(len(queries))
			qt := queries[pick]
			queries[pick] = queries[len(queries)-1]
			queries = queries[:len(queries)-1]
			start := time.Now()
			qt.query.run(cfg.client, cfg.historyPruneBlock)
			if qt.query.Err == errPrunedHistory {
				pruned++
				continue
			}
			qt.runtime = append(qt.runtime, time.Since(start))
			slices.Sort(qt.runtime)
			qt.medianTime = qt.runtime[len(qt.runtime)/2]
			if qt.query.Err != nil {
				qt.query.printError()
				errors = append(errors, qt.query)
				failed++
				continue
			}
			if rhash := qt.query.calculateHash(); *qt.query.ResultHash != rhash {
				fmt.Printf("Filter query result mismatch: fromBlock: %d toBlock: %d addresses: %v topics: %v expected hash: %064x calculated hash: %064x\n", qt.query.FromBlock, qt.query.ToBlock, qt.query.Address, qt.query.Topics, *qt.query.ResultHash, rhash)
				errors = append(errors, qt.query)
				mismatch++
				continue
			}
			processed = append(processed, qt)
			if len(processed)%50 == 0 {
				fmt.Println(" processed:", len(processed), "remaining", len(queries), "failed:", failed, "pruned:", pruned, "result mismatch:", mismatch)
			}
		}
		queries, processed = processed, nil
	}

	// Show results and stats.
	fmt.Println("Performance test finished; processed:", len(queries), "failed:", failed, "pruned:", pruned, "result mismatch:", mismatch)
	stats := make([]bucketStats, len(f.queries))
	var wildcardStats bucketStats
	for _, qt := range queries {
		bs := &stats[qt.bucket]
		if qt.query.isWildcard() {
			bs = &wildcardStats
		}
		bs.blocks += qt.query.ToBlock + 1 - qt.query.FromBlock
		bs.count++
		bs.logs += len(qt.query.results)
		bs.runtime += qt.medianTime
	}

	fmt.Println()
	for i := range stats {
		stats[i].print(fmt.Sprintf("bucket #%d", i+1))
	}
	wildcardStats.print("wild card queries")
	fmt.Println()
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].medianTime > queries[j].medianTime
	})
	for i, q := range queries {
		if i >= 10 {
			break
		}
		fmt.Printf("Most expensive query #%-2d   median runtime: %13v  max runtime: %13v  result count: %4d  fromBlock: %9d  toBlock: %9d  addresses: %v  topics: %v\n",
			i+1, q.medianTime, q.runtime[len(q.runtime)-1], len(q.query.results), q.query.FromBlock, q.query.ToBlock, q.query.Address, q.query.Topics)
	}
	writeErrors(ctx.String(filterErrorFileFlag.Name), errors)
	return nil
}

type bucketStats struct {
	blocks      int64
	count, logs int
	runtime     time.Duration
}

func (st *bucketStats) print(name string) {
	if st.count == 0 {
		return
	}
	fmt.Printf("%-20s queries: %4d  average block length: %12.2f  average log count: %7.2f  average runtime: %13v\n",
		name, st.count, float64(st.blocks)/float64(st.count), float64(st.logs)/float64(st.count), st.runtime/time.Duration(st.count))
}

// writeQueries serializes the generated errors to the error file.
func writeErrors(errorFile string, errors []*filterQuery) {
	file, err := os.Create(errorFile)
	if err != nil {
		exit(fmt.Errorf("Error creating filter error file %s: %v", errorFile, err))
		return
	}
	defer file.Close()
	json.NewEncoder(file).Encode(errors)
}
