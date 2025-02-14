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
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

const (
	maxFilterRange      = 10000000
	maxFilterResultSize = 300
	filterBuckets       = 10
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
			filterPerfCommand,
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
	filterPerfCommand = &cli.Command{
		Name:      "performance",
		Usage:     "Runs log filter performance test against an RPC endpoint",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    filterPerfCmd,
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

type filterTest struct {
	filterQueryFile, filterErrorFile string
	filterQueries                    [filterBuckets][]*filterQuery
	filterQueriesLoaded              bool
	filterErrors                     []*filterQuery
}

func (f *filterTest) initFilterTest(ctx *cli.Context) {
	f.filterQueryFile = ctx.String(filterQueryFileFlag.Name)
	f.filterErrorFile = ctx.String(filterErrorFileFlag.Name)
}

func (s *testSuite) filterRange(t *utesting.T, test func(query *filterQuery) bool, do func(t *utesting.T, query *filterQuery)) {
	if !s.filterQueriesLoaded {
		s.loadQueries()
	}
	var count, total int
	for _, bucket := range s.filterQueries {
		for _, query := range bucket {
			if test(query) {
				total++
			}
		}
	}
	if total == 0 {
		t.Fatalf("No suitable queries available")
	}
	start := time.Now()
	last := start
	for _, bucket := range s.filterQueries {
		for _, query := range bucket {
			if test(query) {
				do(t, query)
				count++
				if time.Since(last) > time.Second*5 {
					t.Logf("Making filter query %d/%d (elapsed: %v)", count, total, time.Since(start))
					last = time.Now()
				}
			}
		}
	}
	t.Logf("Made %d filter queries (elapsed: %v)", count, time.Since(start))
}

const filterRangeThreshold = 10000

func (s *testSuite) filterShortRange(t *utesting.T) {
	s.filterRange(t, func(query *filterQuery) bool {
		return query.ToBlock+1-query.FromBlock <= filterRangeThreshold
	}, s.queryAndCheck)
}

func (s *testSuite) filterLongRange(t *utesting.T) {
	s.filterRange(t, func(query *filterQuery) bool {
		return query.ToBlock+1-query.FromBlock > filterRangeThreshold
	}, s.queryAndCheck)
}

func (s *testSuite) filterFullRange(t *utesting.T) {
	s.filterRange(t, func(query *filterQuery) bool {
		return query.ToBlock+1-query.FromBlock > s.finalizedBlock/2
	}, s.fullRangeQueryAndCheck)
}

func (s *testSuite) queryAndCheck(t *utesting.T, query *filterQuery) {
	s.query(query)
	if query.Err != nil {
		t.Errorf("Filter query failed (fromBlock: %d toBlock: %d addresses: %v topics: %v error: %v)", query.FromBlock, query.ToBlock, query.Address, query.Topics, query.Err)
		return
	}
	if *query.ResultHash != query.calculateHash() {
		t.Fatalf("Filter query result mismatch (fromBlock: %d toBlock: %d addresses: %v topics: %v)", query.FromBlock, query.ToBlock, query.Address, query.Topics)
	}
}

func (s *testSuite) fullRangeQueryAndCheck(t *utesting.T, query *filterQuery) {
	frQuery := &filterQuery{ // create full range query
		FromBlock: 0,
		ToBlock:   int64(rpc.LatestBlockNumber),
		Address:   query.Address,
		Topics:    query.Topics,
	}
	s.query(frQuery)
	if frQuery.Err != nil {
		t.Errorf("Full range filter query failed (addresses: %v topics: %v error: %v)", frQuery.Address, frQuery.Topics, frQuery.Err)
		return
	}
	// filter out results outside the original query range
	j := 0
	for _, log := range frQuery.results {
		if int64(log.BlockNumber) >= query.FromBlock && int64(log.BlockNumber) <= query.ToBlock {
			frQuery.results[j] = log
			j++
		}
	}
	frQuery.results = frQuery.results[:j]
	if *query.ResultHash != frQuery.calculateHash() {
		t.Fatalf("Full range filter query result mismatch (fromBlock: %d toBlock: %d addresses: %v topics: %v)", query.FromBlock, query.ToBlock, query.Address, query.Topics)
	}
}

const passCount = 1

func filterPerfCmd(ctx *cli.Context) error {
	f := newTestSuite(ctx)
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

	for i, bucket := range f.filterQueries[:] {
		for j, query := range bucket {
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
				failed++
				continue
			}
			if rhash := qt.query.calculateHash(); *qt.query.ResultHash != rhash {
				fmt.Printf("Filter query result mismatch: fromBlock: %d toBlock: %d addresses: %v topics: %v expected hash: %064x calculated hash: %064x\n", qt.query.FromBlock, qt.query.ToBlock, qt.query.Address, qt.query.Topics, *qt.query.ResultHash, rhash)
				continue
			}
			processed = append(processed, qt)
			if len(processed)%50 == 0 {
				fmt.Println(" processed:", len(processed), "remaining", len(queries), "failed:", failed, "result mismatch:", mismatch)
			}
		}
		queries, processed = processed, nil
	}
	fmt.Println("Performance test finished; processed:", len(queries), "failed:", failed, "result mismatch:", mismatch)

	type bucketStats struct {
		blocks      int64
		count, logs int
		runtime     time.Duration
	}
	stats := make([]bucketStats, len(f.filterQueries))
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
		fmt.Printf("%-20s queries: %4d  average block length: %12.2f  average log count: %7.2f  average runtime: %13v\n",
			name, stats.count, float64(stats.blocks)/float64(stats.count), float64(stats.logs)/float64(stats.count), stats.runtime/time.Duration(stats.count))
	}

	fmt.Println()
	for i := range stats {
		printStats(fmt.Sprintf("bucket #%d", i+1), &stats[i])
	}
	printStats("wild card queries", &wildcardStats)
	fmt.Println()
	sort.Slice(queries, func(i, j int) bool {
		return queries[i].medianTime > queries[j].medianTime
	})
	for i := 0; i < 10; i++ {
		q := queries[i]
		fmt.Printf("Most expensive query #%-2d   median runtime: %13v  max runtime: %13v  result count: %4d  fromBlock: %9d  toBlock: %9d  addresses: %v  topics: %v\n",
			i+1, q.medianTime, q.runtime[len(q.runtime)-1], len(q.query.results), q.query.FromBlock, q.query.ToBlock, q.query.Address, q.query.Topics)
	}
	return nil
}

func filterGenCmd(ctx *cli.Context) error {
	f := newTestSuite(ctx)
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
			f.filterErrors = append(f.filterErrors, query)
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
					f.filterErrors = append(f.filterErrors, extQuery)
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
}

func (s *testSuite) storeQuery(query *filterQuery) {
	query.ResultHash = new(common.Hash)
	*query.ResultHash = query.calculateHash()
	logRatio := math.Log(float64(len(query.results))*float64(s.finalizedBlock)/float64(query.ToBlock+1-query.FromBlock)) / math.Log(float64(s.finalizedBlock)*maxFilterResultSize)
	bucket := int(math.Floor(logRatio * filterBuckets))
	if bucket >= filterBuckets {
		bucket = filterBuckets - 1
	}
	if len(s.filterQueries[bucket]) < maxFilterBucketSize {
		s.filterQueries[bucket] = append(s.filterQueries[bucket], query)
	} else {
		s.filterQueries[bucket][rand.Intn(len(s.filterQueries[bucket]))] = query
	}
	fmt.Print("Generated queries per bucket:")
	for _, list := range s.filterQueries {
		fmt.Print(" ", len(list))
	}
	fmt.Println()
}

func (s *testSuite) extendRange(q *filterQuery) *filterQuery {
	rangeLen := q.ToBlock + 1 - q.FromBlock
	extLen := rand.Int63n(rangeLen) + 1
	if rangeLen+extLen > s.finalizedBlock {
		return nil
	}
	extBefore := rand.Int63n(extLen + 1)
	if extBefore > q.FromBlock {
		extBefore = q.FromBlock
	}
	extAfter := extLen - extBefore
	if q.ToBlock+extAfter > s.finalizedBlock {
		d := q.ToBlock + extAfter - s.finalizedBlock
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

func (s *testSuite) newQuery() *filterQuery {
	for {
		t := rand.Intn(100)
		if t < filterSeedChance {
			return s.newSeedQuery()
		}
		if t < filterSeedChance+filterMergeChance {
			if query := s.newMergedQuery(); query != nil {
				return query
			}
			continue
		}
		if query := s.newNarrowedQuery(); query != nil {
			return query
		}
	}
}

func (s *testSuite) newSeedQuery() *filterQuery {
	block := rand.Int63n(s.finalizedBlock + 1)
	return &filterQuery{
		FromBlock: block,
		ToBlock:   block,
	}
}

func (s *testSuite) newMergedQuery() *filterQuery {
	q1 := s.randomQuery()
	q2 := s.randomQuery()
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

func (s *testSuite) newNarrowedQuery() *filterQuery {
	q := s.randomQuery()
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

func (s *testSuite) randomQuery() *filterQuery {
	var bucket, bucketCount int
	for _, list := range s.filterQueries {
		if len(list) > 0 {
			bucketCount++
		}
	}
	if bucketCount == 0 {
		return nil
	}
	pick := rand.Intn(bucketCount)
	for i, list := range s.filterQueries {
		if len(list) > 0 {
			if pick == 0 {
				bucket = i
				break
			}
			pick--
		}
	}
	return s.filterQueries[bucket][rand.Intn(len(s.filterQueries[bucket]))]
}

type filterQuery struct {
	FromBlock  int64            `json:"fromBlock"`
	ToBlock    int64            `json:"toBlock"`
	Address    []common.Address `json:"address"`
	Topics     [][]common.Hash  `json:"topics"`
	ResultHash *common.Hash     `json:"resultHash,omitempty"`
	results    []types.Log
	Err        error `json:"error,omitempty"`
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
		exit(fmt.Errorf("Error encoding logs: %v", err))
	}
	return crypto.Keccak256Hash(enc)
}

func (s *testSuite) query(query *filterQuery) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	logs, err := s.ec.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(query.FromBlock),
		ToBlock:   big.NewInt(query.ToBlock),
		Addresses: query.Address,
		Topics:    query.Topics,
	})
	if err != nil {
		query.Err = err
		fmt.Printf("Filter query failed: fromBlock: %d toBlock: %d addresses: %v topics: %v error: %v\n", query.FromBlock, query.ToBlock, query.Address, query.Topics, err)
		return
	}
	query.results = logs
}

func (s *testSuite) loadQueries() int {
	file, err := os.Open(s.filterQueryFile)
	if err != nil {
		fmt.Println("Error opening filter test query file:", err)
		return 0
	}
	json.NewDecoder(file).Decode(&s.filterQueries)
	file.Close()
	var count int
	for _, bucket := range s.filterQueries {
		count += len(bucket)
	}
	fmt.Println("Loaded", count, "filter test queries")
	s.filterQueriesLoaded = true
	return count
}

func (s *testSuite) writeQueries() {
	file, err := os.Create(s.filterQueryFile)
	if err != nil {
		exit(fmt.Errorf("Error creating filter test query file %s: %v", s.filterQueryFile, err))
		return
	}
	json.NewEncoder(file).Encode(&s.filterQueries)
	file.Close()
}

func (f *filterTest) writeErrors() {
	file, err := os.Create(f.filterErrorFile)
	if err != nil {
		exit(fmt.Errorf("Error creating filter error file %s: %v", f.filterErrorFile, err))
		return
	}
	json.NewEncoder(file).Encode(f.filterErrors)
	file.Close()
}
