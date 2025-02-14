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
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

const (
	maxFilterRange      = 100000
	maxFilterResultSize = 100
	filterBuckets       = 10
	maxFilterBucketSize = 100
	filterSeedChance    = 10
	filterMergeChance   = 45
)

var (
	filterCommand = &cli.Command{
		Name:      "filter",
		Usage:     "Runs range log filter workload test against an RPC endpoint",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    filterTestCmd,
		Flags:     []cli.Flag{},
	}
)

func filterTestCmd(ctx *cli.Context) error {
	f := filterTest{ec: makeEthClient(ctx)}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		f.getFinalizedBlock()
		query := f.newQuery()
		f.query(&query)
		if len(query.results) > 0 && len(query.results) <= maxFilterResultSize {
			for {
				extQuery := f.extendQuery(query)
				f.query(&extQuery)
				if len(query.results) <= maxFilterResultSize {
					break
				}
				query = extQuery
			}
			f.storeQuery(query)
		}
	}
	return nil
}

func (f *filterTest) storeQuery(query filterQuery) {
	logRatio := math.Log(float64(len(query.results))*maxFilterRange/float64(query.end+1-query.begin)) / math.Log(maxFilterRange*maxFilterResultSize)
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

func (f *filterTest) extendQuery(q filterQuery) filterQuery {
	rangeLen := q.end + 1 - q.begin
	extLen := rand.Int63n(rangeLen) + 1
	extBefore := rand.Int63n(extLen + 1)
	if extBefore > q.begin {
		extBefore = q.begin
	}
	return filterQuery{
		begin:     q.begin - extBefore,
		end:       q.end + extLen - extBefore,
		addresses: q.addresses,
		topics:    q.topics,
	}
}

func (f *filterTest) newQuery() filterQuery {
	for {
		t := rand.Intn(100)
		if t < filterSeedChance {
			return f.newSeedQuery()
		}
		if t < filterSeedChance+filterMergeChance {
			if query, ok := f.newMergedQuery(); ok {
				return query
			}
			continue
		}
		if query, ok := f.newNarrowedQuery(); ok {
			return query
		}
	}
}

func (f *filterTest) newSeedQuery() filterQuery {
	block := rand.Int63n(f.finalized + 1)
	return filterQuery{
		begin: block,
		end:   block,
	}
}

func (f *filterTest) newMergedQuery() (filterQuery, bool) {
	count := f.queryCount()
	if count < 2 {
		return filterQuery{}, false
	}
	pick1 := rand.Intn(count)
	pick2 := pick1
	for pick2 == pick1 {
		pick2 = rand.Intn(count)
	}
	q1 := f.pickQuery(pick1)
	q2 := f.pickQuery(pick2)
	var (
		m          filterQuery
		block      int64
		topicCount int
	)
	if rand.Intn(2) == 0 {
		block = q1.begin + rand.Int63n(q1.end+1-q1.begin)
		topicCount = len(q1.topics)
	} else {
		block = q2.begin + rand.Int63n(q2.end+1-q2.begin)
		topicCount = len(q2.topics)
	}
	m.begin = block
	m.end = block
	m.topics = make([][]common.Hash, topicCount)
	for _, addr := range q1.addresses {
		if rand.Intn(2) == 0 {
			m.addresses = append(m.addresses, addr)
		}
	}
	for _, addr := range q2.addresses {
		if rand.Intn(2) == 0 {
			m.addresses = append(m.addresses, addr)
		}
	}
	for i := range m.topics {
		if len(q1.topics) > i {
			for _, topic := range q1.topics[i] {
				if rand.Intn(2) == 0 {
					m.topics[i] = append(m.topics[i], topic)
				}
			}
		}
		if len(q2.topics) > i {
			for _, topic := range q2.topics[i] {
				if rand.Intn(2) == 0 {
					m.topics[i] = append(m.topics[i], topic)
				}
			}
		}
	}
	return m, true
}

func (f *filterTest) newNarrowedQuery() (filterQuery, bool) {
	count := f.queryCount()
	if count < 1 {
		return filterQuery{}, false
	}
	q := f.pickQuery(rand.Intn(count))
	log := q.results[rand.Intn(len(q.results))]
	var emptyCount int
	if len(q.addresses) == 0 {
		emptyCount++
	}
	for i := range log.Topics {
		if len(q.topics) <= i || len(q.topics[i]) == 0 {
			emptyCount++
		}
	}
	var query filterQuery
	if emptyCount == 0 {
		return query, false
	}
	query.addresses, query.topics = q.addresses, q.topics
	pick := rand.Intn(emptyCount)
	if len(q.addresses) == 0 {
		if pick == 0 {
			q.addresses = []common.Address{log.Address}
			return query, true
		}
		pick--
	}
	for i := range log.Topics {
		if len(q.topics) <= i || len(q.topics[i]) == 0 {
			if pick == 0 {
				if len(q.topics) <= i {
					q.topics = append(q.topics, make([][]common.Hash, i+1-len(q.topics))...)
				}
				q.topics[i] = []common.Hash{log.Topics[i]}
				return query, true
			}
			pick--
		}
	}
	panic(nil)
}

func (f *filterTest) queryCount() int {
	var count int
	for _, list := range f.stored {
		count += len(list)
	}
	return count
}

func (f *filterTest) pickQuery(pick int) filterQuery {
	for _, list := range f.stored {
		if pick < len(list) {
			return list[pick]
		}
		pick -= len(list)
	}
	panic(nil)
}

type filterTest struct {
	ec        *ethclient.Client
	finalized int64
	stored    [filterBuckets][]filterQuery
}

type filterQuery struct {
	begin, end int64
	addresses  []common.Address
	topics     [][]common.Hash
	results    []types.Log
	err        error
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	logs, err := f.ec.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(query.begin),
		ToBlock:   big.NewInt(query.end),
		Addresses: query.addresses,
		Topics:    query.topics,
	})
	if err != nil {
		query.err = err
		fmt.Println("filter query error", err)
		return
	}
	query.results = logs
	fmt.Println("filter query results", len(logs))
}
