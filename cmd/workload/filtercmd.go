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
	"os"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
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
		Name:      "filter",
		Usage:     "Runs range log filter workload test against an RPC endpoint",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    filterTestCmd,
		Flags:     []cli.Flag{},
	}
)

func filterTestCmd(ctx *cli.Context) error {
	f := filterTest{ec: makeEthClient(ctx)}
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
		if query.err != nil {
			f.failed = append(f.failed, query)
			continue
		}
		if len(query.results) > 0 && len(query.results) <= maxFilterResultSize {
			for {
				extQuery := f.extendQuery(query)
				if extQuery == nil {
					break
				}
				f.query(extQuery)
				if extQuery.err == nil && len(extQuery.results) < len(query.results) {
					extQuery.err = fmt.Errorf("invalid result length; old range %d %d; old length %d; new range %d %d; new length %d; addresses %v; topics %v",
						query.begin, query.end, len(query.results),
						extQuery.begin, extQuery.end, len(extQuery.results),
						extQuery.addresses, extQuery.topics,
					)
				}
				if extQuery.err != nil {
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
				f.writeQueries("filter_queries")
				f.writeFailed("filter_errors")
				lastWrite = time.Now()
			}
		}
	}
	return nil
}

type filterTest struct {
	ec        *ethclient.Client
	finalized int64
	stored    [filterBuckets][]*filterQuery
	failed    []*filterQuery
}

func (f *filterTest) storeQuery(query *filterQuery) {
	query.resultHash = query.calculateHash()
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

func (f *filterTest) extendQuery(q *filterQuery) *filterQuery {
	rangeLen := q.end + 1 - q.begin
	extLen := rand.Int63n(rangeLen) + 1
	if rangeLen+extLen > maxFilterRange {
		return nil
	}
	extBefore := rand.Int63n(extLen + 1)
	if extBefore > q.begin {
		extBefore = q.begin
	}
	extAfter := extLen - extBefore
	if q.end+extAfter > f.finalized {
		d := f.finalized - q.end - extAfter
		extAfter -= d
		if extBefore+d <= q.begin {
			extBefore += d
		} else {
			extBefore = q.begin
		}
	}
	return &filterQuery{
		begin:     q.begin - extBefore,
		end:       q.end + extAfter,
		addresses: q.addresses,
		topics:    q.topics,
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
		begin: block,
		end:   block,
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
		block = q1.begin + rand.Int63n(q1.end+1-q1.begin)
		topicCount = len(q1.topics)
	} else {
		block = q2.begin + rand.Int63n(q2.end+1-q2.begin)
		topicCount = len(q2.topics)
	}
	m := &filterQuery{
		begin:  block,
		end:    block,
		topics: make([][]common.Hash, topicCount),
	}
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
	return m
}

func (f *filterTest) newNarrowedQuery() *filterQuery {
	q := f.randomQuery()
	if q == nil {
		return nil
	}
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
	if emptyCount == 0 {
		return nil
	}
	query := &filterQuery{
		begin:     q.begin,
		end:       q.end,
		addresses: q.addresses,
		topics:    q.topics,
	}
	pick := rand.Intn(emptyCount)
	if len(q.addresses) == 0 {
		if pick == 0 {
			q.addresses = []common.Address{log.Address}
			return query
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
	begin, end int64
	addresses  []common.Address
	topics     [][]common.Hash
	resultHash common.Hash
	results    []types.Log
	err        error
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
	fmt.Println("filter query range", query.end+1-query.begin, "results", len(logs))
}

func (f *filterTest) writeQueries(fn string) {
	w, err := os.Create(fn)
	if err != nil {
		exit(fmt.Errorf("Error creating filter pattern file", "name", fn, "error", err))
		return
	}
	defer w.Close()

	w.WriteString("\t{\n")
	for _, list := range f.stored {
		w.WriteString("\t\t{\n")
		for _, filter := range list {
			w.WriteString(fmt.Sprintf("\t\t\t{%d, %d, []common.Address{\n", filter.begin, filter.end))
			for _, addr := range filter.addresses {
				w.WriteString(fmt.Sprintf("\t\t\t\t\tcommon.HexToAddress(\"0x%040x\"),\n", addr))
			}
			w.WriteString(fmt.Sprintf("\t\t\t\t}, [][]common.Hash{\n"))
			for i, topics := range filter.topics {
				if i == 0 {
					w.WriteString(fmt.Sprintf("\t\t\t\t\t{\n"))
				}
				for _, topic := range topics {
					w.WriteString(fmt.Sprintf("\t\t\t\t\t\tcommon.HexToHash(\"0x%064x\"),\n", topic))
				}
				if i == len(filter.topics)-1 {
					w.WriteString(fmt.Sprintf("\t\t\t\t\t},\n"))
				} else {
					w.WriteString(fmt.Sprintf("\t\t\t\t\t}, {\n"))
				}
			}
			w.WriteString(fmt.Sprintf("\t\t\t\t}, common.HexToHash(\"0x%064x\"),\n", filter.resultHash))
			w.WriteString(fmt.Sprintf("\t\t\t},\n"))
		}
		w.WriteString("\t\t},\n")
	}
	w.WriteString("\t},\n")
}

func (f *filterTest) writeFailed(fn string) {
	w, err := os.Create(fn)
	if err != nil {
		exit(fmt.Errorf("Error creating filter error file", "name", fn, "error", err))
		return
	}
	defer w.Close()

	w.WriteString("\t{\n")
	for _, filter := range f.failed {
		w.WriteString(fmt.Sprintf("\t\t{%d, %d, []common.Address{\n", filter.begin, filter.end))
		for _, addr := range filter.addresses {
			w.WriteString(fmt.Sprintf("\t\t\t\tcommon.HexToAddress(\"0x%040x\"),\n", addr))
		}
		w.WriteString(fmt.Sprintf("\t\t\t}, [][]common.Hash{\n"))
		for i, topics := range filter.topics {
			if i == 0 {
				w.WriteString(fmt.Sprintf("\t\t\t\t{\n"))
			}
			for _, topic := range topics {
				w.WriteString(fmt.Sprintf("\t\t\t\t\tcommon.HexToHash(\"0x%064x\"),\n", topic))
			}
			if i == len(filter.topics)-1 {
				w.WriteString(fmt.Sprintf("\t\t\t\t},\n"))
			} else {
				w.WriteString(fmt.Sprintf("\t\t\t\t}, {\n"))
			}
		}
		w.WriteString(fmt.Sprintf("\t\t\t}, \"%v\"),\n", filter.err))
		w.WriteString(fmt.Sprintf("\t\t},\n"))
	}
	w.WriteString("\t},\n")
}
