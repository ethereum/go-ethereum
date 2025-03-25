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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

var (
	filterGenerateCommand = &cli.Command{
		Name:      "filtergen",
		Usage:     "Generates query set for log filter workload test",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    filterGenCmd,
		Flags: []cli.Flag{
			filterQueryFileFlag,
		},
	}
	filterQueryFileFlag = &cli.StringFlag{
		Name:     "queries",
		Usage:    "JSON file containing filter test queries",
		Value:    "filter_queries.json",
		Category: flags.TestingCategory,
	}
	filterErrorFileFlag = &cli.StringFlag{
		Name:     "errors",
		Usage:    "JSON file containing failed filter queries",
		Value:    "filter_errors.json",
		Category: flags.TestingCategory,
	}
)

// filterGenCmd is the main function of the filter tests generator.
func filterGenCmd(ctx *cli.Context) error {
	f := newFilterTestGen(ctx)
	lastWrite := time.Now()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		f.updateFinalizedBlock()
		query := f.newQuery()
		query.run(f.client, nil)
		if query.Err != nil {
			query.printError()
			exit("filter query failed")
		}
		if len(query.results) > 0 && len(query.results) <= maxFilterResultSize {
			for {
				extQuery := f.extendRange(query)
				if extQuery == nil {
					break
				}
				extQuery.run(f.client, nil)
				if extQuery.Err == nil && len(extQuery.results) < len(query.results) {
					extQuery.Err = fmt.Errorf("invalid result length; old range %d %d; old length %d; new range %d %d; new length %d; address %v; Topics %v",
						query.FromBlock, query.ToBlock, len(query.results),
						extQuery.FromBlock, extQuery.ToBlock, len(extQuery.results),
						extQuery.Address, extQuery.Topics,
					)
				}
				if extQuery.Err != nil {
					extQuery.printError()
					exit("filter query failed")
				}
				if len(extQuery.results) > maxFilterResultSize {
					break
				}
				query = extQuery
			}
			f.storeQuery(query)
			if time.Since(lastWrite) > time.Second*10 {
				f.writeQueries()
				lastWrite = time.Now()
			}
		}
	}
}

// filterTestGen is the filter query test generator.
type filterTestGen struct {
	client    *client
	queryFile string

	finalizedBlock int64
	queries        [filterBuckets][]*filterQuery
}

func newFilterTestGen(ctx *cli.Context) *filterTestGen {
	return &filterTestGen{
		client:    makeClient(ctx),
		queryFile: ctx.String(filterQueryFileFlag.Name),
	}
}

func (s *filterTestGen) updateFinalizedBlock() {
	s.finalizedBlock = mustGetFinalizedBlock(s.client)
}

const (
	// Parameter of the random filter query generator.
	maxFilterRange      = 10000000
	maxFilterResultSize = 300
	filterBuckets       = 10
	maxFilterBucketSize = 100
	filterSeedChance    = 10
	filterMergeChance   = 45
)

// storeQuery adds a filter query to the output file.
func (s *filterTestGen) storeQuery(query *filterQuery) {
	query.ResultHash = new(common.Hash)
	*query.ResultHash = query.calculateHash()
	logRatio := math.Log(float64(len(query.results))*float64(s.finalizedBlock)/float64(query.ToBlock+1-query.FromBlock)) / math.Log(float64(s.finalizedBlock)*maxFilterResultSize)
	bucket := int(math.Floor(logRatio * filterBuckets))
	if bucket >= filterBuckets {
		bucket = filterBuckets - 1
	}
	if len(s.queries[bucket]) < maxFilterBucketSize {
		s.queries[bucket] = append(s.queries[bucket], query)
	} else {
		s.queries[bucket][rand.Intn(len(s.queries[bucket]))] = query
	}
	fmt.Print("Generated queries per bucket:")
	for _, list := range s.queries {
		fmt.Print(" ", len(list))
	}
	fmt.Println()
}

func (s *filterTestGen) extendRange(q *filterQuery) *filterQuery {
	rangeLen := q.ToBlock + 1 - q.FromBlock
	extLen := rand.Int63n(rangeLen) + 1
	if rangeLen+extLen > s.finalizedBlock {
		return nil
	}
	extBefore := min(rand.Int63n(extLen+1), q.FromBlock)
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

// newQuery generates a new filter query.
func (s *filterTestGen) newQuery() *filterQuery {
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

// newSeedQuery creates a query that gets all logs in a random non-finalized block.
func (s *filterTestGen) newSeedQuery() *filterQuery {
	block := rand.Int63n(s.finalizedBlock + 1)
	return &filterQuery{
		FromBlock: block,
		ToBlock:   block,
	}
}

// newMergedQuery creates a new query by combining (with OR) the filter criteria
// of two existing queries (chosen at random).
func (s *filterTestGen) newMergedQuery() *filterQuery {
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

// newNarrowedQuery creates a new query by 'narrowing' an existing (randomly chosen)
// query. The new query is made more specific by analyzing the filter criteria and adding
// topics/addresses from the known result set.
func (s *filterTestGen) newNarrowedQuery() *filterQuery {
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
	panic("unreachable")
}

// randomQuery returns a random query from the ones that were already generated.
func (s *filterTestGen) randomQuery() *filterQuery {
	var bucket, bucketCount int
	for _, list := range s.queries {
		if len(list) > 0 {
			bucketCount++
		}
	}
	if bucketCount == 0 {
		return nil
	}
	pick := rand.Intn(bucketCount)
	for i, list := range s.queries {
		if len(list) > 0 {
			if pick == 0 {
				bucket = i
				break
			}
			pick--
		}
	}
	return s.queries[bucket][rand.Intn(len(s.queries[bucket]))]
}

// writeQueries serializes the generated queries to the output file.
func (s *filterTestGen) writeQueries() {
	file, err := os.Create(s.queryFile)
	if err != nil {
		exit(fmt.Errorf("Error creating filter test query file %s: %v", s.queryFile, err))
		return
	}
	json.NewEncoder(file).Encode(&s.queries)
	file.Close()
}

func mustGetFinalizedBlock(client *client) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	header, err := client.Eth.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		exit(fmt.Errorf("could not fetch finalized header (error: %v)", err))
	}
	return header.Number.Int64()
}
