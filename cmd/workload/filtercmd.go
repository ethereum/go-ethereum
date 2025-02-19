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
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"os"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/flags"

	//"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

const (
	maxFilterRange      = 10000000
	maxFilterResultSize = 300
	filterBuckets       = 16
	maxFilterBucketSize = 100
	filterSeedChance    = 10
	filterMergeChance   = 45
)

var (
	filterCommand = &cli.Command{
		Name:  "filter",
		Usage: "Log filter workload test commands",
		Subcommands: []*cli.Command{
			filterGenCommand,
			filterTestCommand,
		},
	}
	filterGenCommand = &cli.Command{
		Name:      "generate",
		Usage:     "Generates query set for log filter workload test",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    filterGenCmd,
		Flags: []cli.Flag{
			filterQueryFileFlag,
			filterErrorFileFlag,
		},
	}
	filterTestCommand = &cli.Command{
		Name:      "test",
		Usage:     "Runs log filter workload test against an RPC endpoint",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    filterTestCmd,
		Flags: []cli.Flag{
			filterQueryFileFlag,
			filterErrorFileFlag,
		},
	}
	filterQueryFileFlag = &cli.StringFlag{
		Name:     "queries",
		Usage:    "JSON file containing filter test queries",
		Category: flags.TestingCategory,
		Value:    "filter_queries.json",
	}
	filterErrorFileFlag = &cli.StringFlag{
		Name:     "errors",
		Usage:    "JSON file containing failed filter queries",
		Category: flags.TestingCategory,
		Value:    "filter_errors.json",
	}
)

const passCount = 5

func filterTestCmd(ctx *cli.Context) error {
	f := newFilterTest(ctx)
	if f.loadQueries() == 0 {
		exit("No test requests loaded")
	}
	f.getFinalizedBlock()

	type queryTest struct {
		query         *filterQuery
		bucket, index int
		runtime       []time.Duration
		medianTime    time.Duration
	}
	var queries, processed []queryTest

	for i, bucket := range f.stored[:] {
		for j, query := range bucket {
			if query.ToBlock > f.finalized {
				fmt.Println("invalid range")
				continue
			}
			queries = append(queries, queryTest{query: query, bucket: i, index: j})
		}
	}

	var failed, mismatch int
	for i := 1; i <= passCount; i++ {
		fmt.Println("Performance test pass", i, "/", passCount)
		for len(queries) > 0 {
			pick := rand.Intn(len(queries))
			qt := queries[pick]
			queries[pick] = queries[len(queries)-1]
			queries = queries[:len(queries)-1]
			start := time.Now()
			f.query(qt.query)
			qt.runtime = append(qt.runtime, time.Since(start))
			sort.Slice(qt.runtime, func(i, j int) bool { return qt.runtime[i] < qt.runtime[j] })
			qt.medianTime = qt.runtime[len(qt.runtime)/2]
			if qt.query.Err != nil {
				fmt.Println(qt.bucket, qt.index, "err", qt.query.Err)
				failed++
				continue
			}
			if *qt.query.ResultHash != qt.query.calculateHash() {
				fmt.Println(qt.bucket, qt.index, "mismatch")
				mismatch++
				continue
			}
			processed = append(processed, qt)
			if len(processed)%50 == 0 {
				fmt.Println("processed:", len(processed), "remaining", len(queries), "failed:", failed, "result mismatch:", mismatch)
			}
		}
		queries, processed = processed, nil
	}
	fmt.Println("Done; processed:", len(queries), "failed:", failed, "result mismatch:", mismatch)

	type bucketStats struct {
		blocks      int64
		count, logs int
		runtime     time.Duration
	}
	stats := make([]bucketStats, len(f.stored))
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

	printStats := func(name string, stats *bucketStats) {
		if stats.count == 0 {
			return
		}
		fmt.Println(name, "query count", stats.count, "avg block count", float64(stats.blocks)/float64(stats.count), "avg log count", float64(stats.logs)/float64(stats.count), "avg runtime", stats.runtime/time.Duration(stats.count))
	}

	fmt.Println()
	for i := range stats {
		printStats(fmt.Sprintf("bucket #%d", i), &stats[i])
	}
	printStats("wild card queries", &wildcardStats)
	fmt.Println()
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].medianTime > queries[j].medianTime
	})
	for i := 0; i < 100; i++ {
		q := queries[i]
		fmt.Println("Most expensive query #", i+1, "median time", q.medianTime, "max time", q.runtime[len(q.runtime)-1], "results", len(q.query.results), "fromBlock", q.query.FromBlock, "toBlock", q.query.ToBlock, "addresses", q.query.Address, "topics", q.query.Topics)
	}
	return nil
}

func filterGenCmd(ctx *cli.Context) error {
	f := newFilterTest(ctx)
	//f.loadQueries()  //TODO
	lastWrite := time.Now()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		f.getFinalizedBlock()
		query := f.newQuery()
		f.query(query)
		if query.Err != nil {
			f.failed = append(f.failed, query)
			continue
		}
		if len(query.results) > 0 && len(query.results) <= maxFilterResultSize {
			for {
				extQuery := f.extendRange(query)
				if extQuery == nil {
					break
				}
				f.query(extQuery)
				if extQuery.Err == nil && len(extQuery.results) < len(query.results) {
					extQuery.Err = fmt.Errorf("invalid result length; old range %d %d; old length %d; new range %d %d; new length %d; address %v; Topics %v",
						query.FromBlock, query.ToBlock, len(query.results),
						extQuery.FromBlock, extQuery.ToBlock, len(extQuery.results),
						extQuery.Address, extQuery.Topics,
					)
				}
				if extQuery.Err != nil {
					f.failed = append(f.failed, extQuery)
					break
				}
				if len(extQuery.results) > maxFilterResultSize {
					break
				}
				query = extQuery
			}
			f.storeQuery(query)
			if time.Since(lastWrite) > time.Second*10 {
				f.writeQueries()
				f.writeErrors()
				lastWrite = time.Now()
			}
		}
	}
	return nil
}

type filterTest struct {
	ec                   *ethclient.Client
	finalized            int64
	queryFile, errorFile string
	stored               [filterBuckets][]*filterQuery
	failed               []*filterQuery
}

func newFilterTest(ctx *cli.Context) *filterTest {
	return &filterTest{
		ec:        makeEthClient(ctx),
		queryFile: ctx.String(filterQueryFileFlag.Name),
		errorFile: ctx.String(filterErrorFileFlag.Name),
	}
}

func (f *filterTest) storeQuery(query *filterQuery) {
	query.ResultHash = new(common.Hash)
	*query.ResultHash = query.calculateHash()
	logRatio := math.Log(float64(len(query.results))*maxFilterRange/float64(query.ToBlock+1-query.FromBlock)) / math.Log(maxFilterRange*maxFilterResultSize)
	bucket := int(math.Floor(logRatio * filterBuckets))
	if bucket >= filterBuckets {
		bucket = filterBuckets - 1
	}
	if len(f.stored[bucket]) < maxFilterBucketSize {
		f.stored[bucket] = append(f.stored[bucket], query)
	} else {
		f.stored[bucket][rand.Intn(len(f.stored[bucket]))] = query
	}
	fmt.Print("stored")
	for _, list := range f.stored {
		fmt.Print(" ", len(list))
	}
	fmt.Println()
}

func (f *filterTest) extendRange(q *filterQuery) *filterQuery {
	rangeLen := q.ToBlock + 1 - q.FromBlock
	extLen := rand.Int63n(rangeLen) + 1
	if rangeLen+extLen > maxFilterRange {
		return nil
	}
	extBefore := rand.Int63n(extLen + 1)
	if extBefore > q.FromBlock {
		extBefore = q.FromBlock
	}
	extAfter := extLen - extBefore
	if q.ToBlock+extAfter > f.finalized {
		d := q.ToBlock + extAfter - f.finalized
		extAfter -= d
		if extBefore+d <= q.FromBlock {
			extBefore += d
		} else {
			extBefore = q.FromBlock
		}
	}
	return &filterQuery{
		FromBlock: q.FromBlock - extBefore,
		ToBlock:   q.ToBlock + extAfter,
		Address:   q.Address,
		Topics:    q.Topics,
	}
}

func (f *filterTest) newQuery() *filterQuery {
	for {
		t := rand.Intn(100)
		if t < filterSeedChance {
			fmt.Println("* seed")
			return f.newSeedQuery()
		}
		if t < filterSeedChance+filterMergeChance {
			if query := f.newMergedQuery(); query != nil {
				fmt.Println("* merged")
				return query
			}
			fmt.Println("* merged x")
			continue
		}
		if query := f.newNarrowedQuery(); query != nil {
			fmt.Println("* narrowed")
			return query
		}
		fmt.Println("* narrowed x")
	}
}

func (f *filterTest) newSeedQuery() *filterQuery {
	block := rand.Int63n(f.finalized + 1)
	return &filterQuery{
		FromBlock: block,
		ToBlock:   block,
	}
}

func (f *filterTest) newMergedQuery() *filterQuery {
	q1 := f.randomQuery()
	q2 := f.randomQuery()
	if q1 == nil || q2 == nil || q1 == q2 {
		return nil
	}
	var (
		block      int64
		topicCount int
	)
	if rand.Intn(2) == 0 {
		block = q1.FromBlock + rand.Int63n(q1.ToBlock+1-q1.FromBlock)
		topicCount = len(q1.Topics)
	} else {
		block = q2.FromBlock + rand.Int63n(q2.ToBlock+1-q2.FromBlock)
		topicCount = len(q2.Topics)
	}
	m := &filterQuery{
		FromBlock: block,
		ToBlock:   block,
		Topics:    make([][]common.Hash, topicCount),
	}
	for _, addr := range q1.Address {
		if rand.Intn(2) == 0 {
			m.Address = append(m.Address, addr)
		}
	}
	for _, addr := range q2.Address {
		if rand.Intn(2) == 0 {
			m.Address = append(m.Address, addr)
		}
	}
	for i := range m.Topics {
		if len(q1.Topics) > i {
			for _, topic := range q1.Topics[i] {
				if rand.Intn(2) == 0 {
					m.Topics[i] = append(m.Topics[i], topic)
				}
			}
		}
		if len(q2.Topics) > i {
			for _, topic := range q2.Topics[i] {
				if rand.Intn(2) == 0 {
					m.Topics[i] = append(m.Topics[i], topic)
				}
			}
		}
	}
	return m
}

func (f *filterTest) newNarrowedQuery() *filterQuery {
	q := f.randomQuery()
	if q == nil {
		return nil
	}
	log := q.results[rand.Intn(len(q.results))]
	var emptyCount int
	if len(q.Address) == 0 {
		emptyCount++
	}
	for i := range log.Topics {
		if len(q.Topics) <= i || len(q.Topics[i]) == 0 {
			emptyCount++
		}
	}
	if emptyCount == 0 {
		return nil
	}
	query := &filterQuery{
		FromBlock: q.FromBlock,
		ToBlock:   q.ToBlock,
		Address:   make([]common.Address, len(q.Address)),
		Topics:    make([][]common.Hash, len(q.Topics)),
	}
	copy(query.Address, q.Address)
	for i, topics := range q.Topics {
		if len(topics) > 0 {
			query.Topics[i] = make([]common.Hash, len(topics))
			copy(query.Topics[i], topics)
		}
	}
	pick := rand.Intn(emptyCount)
	if len(query.Address) == 0 {
		if pick == 0 {
			query.Address = []common.Address{log.Address}
			return query
		}
		pick--
	}
	for i := range log.Topics {
		if len(query.Topics) <= i || len(query.Topics[i]) == 0 {
			if pick == 0 {
				if len(query.Topics) <= i {
					query.Topics = append(query.Topics, make([][]common.Hash, i+1-len(query.Topics))...)
				}
				query.Topics[i] = []common.Hash{log.Topics[i]}
				return query
			}
			pick--
		}
	}
	panic(nil)
}

func (f *filterTest) randomQuery() *filterQuery {
	var bucket, bucketCount int
	for _, list := range f.stored {
		if len(list) > 0 {
			bucketCount++
		}
	}
	if bucketCount == 0 {
		return nil
	}
	pick := rand.Intn(bucketCount)
	for i, list := range f.stored {
		if len(list) > 0 {
			if pick == 0 {
				bucket = i
				break
			}
			pick--
		}
	}
	return f.stored[bucket][rand.Intn(len(f.stored[bucket]))]
}

type filterQuery struct {
	FromBlock  int64            `json: fromBlock`
	ToBlock    int64            `json: toBlock`
	Address    []common.Address `json: address`
	Topics     [][]common.Hash  `json: topics`
	ResultHash *common.Hash     `json: resultHash, omitEmpty`
	results    []types.Log
	Err        error `json: error, omitEmpty`
}

func (fq *filterQuery) isWildcard() bool {
	if len(fq.Address) != 0 {
		return false
	}
	for _, topics := range fq.Topics {
		if len(topics) != 0 {
			return false
		}
	}
	return true
}

func (fq *filterQuery) calculateHash() common.Hash {
	enc, err := rlp.EncodeToBytes(&fq.results)
	if err != nil {
		exit(fmt.Errorf("Error encoding logs", "error", err))
	}
	return crypto.Keccak256Hash(enc)
}

func (f *filterTest) getFinalizedBlock() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	header, err := f.ec.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		fmt.Println("finalized header error", err)
		return
	}
	f.finalized = header.Number.Int64()
	fmt.Println("finalized header updated", f.finalized)
}

func (f *filterTest) query(query *filterQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	logs, err := f.ec.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(query.FromBlock),
		ToBlock:   big.NewInt(query.ToBlock),
		Addresses: query.Address,
		Topics:    query.Topics,
	})
	if err != nil {
		query.Err = err
		fmt.Println("filter query error", err)
		return
	}
	query.results = logs
	//fmt.Println("filter query range", query.ToBlock+1-query.FromBlock, "results", len(logs))
}

func (f *filterTest) loadQueries() int {
	file, err := os.Open(f.queryFile)
	if err != nil {
		fmt.Println("Error opening", f.queryFile, ":", err)
		return 0
	}
	json.NewDecoder(file).Decode(&f.stored)
	file.Close()
	var count int
	for _, bucket := range f.stored {
		count += len(bucket)
	}
	fmt.Println("Loaded", count, "filter test queries")
	return count
}

func (f *filterTest) writeQueries() {
	file, err := os.Create(f.queryFile)
	if err != nil {
		exit(fmt.Errorf("Error creating filter test query file", "name", f.queryFile, "error", err))
		return
	}
	json.NewEncoder(file).Encode(&f.stored)
	file.Close()
}

func (f *filterTest) writeErrors() {
	file, err := os.Create(f.errorFile)
	if err != nil {
		exit(fmt.Errorf("Error creating filter error file", "name", f.errorFile, "error", err))
		return
	}
	json.NewEncoder(file).Encode(f.failed)
	file.Close()
}
