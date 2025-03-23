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
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
)

type filterTestSuite struct {
	cfg     testConfig
	queries [][]*filterQuery
}

func newFilterTestSuite(cfg testConfig) *filterTestSuite {
	s := &filterTestSuite{cfg: cfg}
	if err := s.loadQueries(); err != nil {
		exit(err)
	}
	return s
}

func (s *filterTestSuite) allTests() []utesting.Test {
	return []utesting.Test{
		{Name: "Filter/ShortRange", Fn: s.filterShortRange},
		{Name: "Filter/LongRange", Fn: s.filterLongRange, Slow: true},
		{Name: "Filter/FullRange", Fn: s.filterFullRange, Slow: true},
	}
}

func (s *filterTestSuite) filterRange(t *utesting.T, test func(query *filterQuery) bool, do func(t *utesting.T, query *filterQuery)) {
	var count, total int
	for _, bucket := range s.queries {
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
	for _, bucket := range s.queries {
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

// filterShortRange runs all short-range filter tests.
func (s *filterTestSuite) filterShortRange(t *utesting.T) {
	s.filterRange(t, func(query *filterQuery) bool {
		return query.ToBlock+1-query.FromBlock <= filterRangeThreshold
	}, s.queryAndCheck)
}

// filterShortRange runs all long-range filter tests.
func (s *filterTestSuite) filterLongRange(t *utesting.T) {
	s.filterRange(t, func(query *filterQuery) bool {
		return query.ToBlock+1-query.FromBlock > filterRangeThreshold
	}, s.queryAndCheck)
}

// filterFullRange runs all filter tests, extending their range from genesis up
// to the latest block. Note that results are only partially verified in this mode.
func (s *filterTestSuite) filterFullRange(t *utesting.T) {
	finalized := mustGetFinalizedBlock(s.cfg.client)
	s.filterRange(t, func(query *filterQuery) bool {
		return query.ToBlock+1-query.FromBlock > finalized/2
	}, s.fullRangeQueryAndCheck)
}

func (s *filterTestSuite) queryAndCheck(t *utesting.T, query *filterQuery) {
	query.run(s.cfg.client, s.cfg.historyPruneBlock)
	if query.Err == errPrunedHistory {
		return
	}
	if query.Err != nil {
		t.Errorf("Filter query failed (fromBlock: %d toBlock: %d addresses: %v topics: %v error: %v)", query.FromBlock, query.ToBlock, query.Address, query.Topics, query.Err)
		return
	}
	if *query.ResultHash != query.calculateHash() {
		t.Fatalf("Filter query result mismatch (fromBlock: %d toBlock: %d addresses: %v topics: %v)", query.FromBlock, query.ToBlock, query.Address, query.Topics)
	}
}

func (s *filterTestSuite) fullRangeQueryAndCheck(t *utesting.T, query *filterQuery) {
	frQuery := &filterQuery{ // create full range query
		FromBlock: 0,
		ToBlock:   int64(rpc.LatestBlockNumber),
		Address:   query.Address,
		Topics:    query.Topics,
	}
	frQuery.run(s.cfg.client, s.cfg.historyPruneBlock)
	if frQuery.Err == errPrunedHistory {
		return
	}
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

func (s *filterTestSuite) loadQueries() error {
	file, err := s.cfg.fsys.Open(s.cfg.filterQueryFile)
	if err != nil {
		return fmt.Errorf("can't open filterQueryFile: %v", err)
	}
	defer file.Close()

	var queries [][]*filterQuery
	if err := json.NewDecoder(file).Decode(&queries); err != nil {
		return fmt.Errorf("invalid JSON in %s: %v", s.cfg.filterQueryFile, err)
	}
	var count int
	for _, bucket := range queries {
		count += len(bucket)
	}
	if count == 0 {
		return fmt.Errorf("filterQueryFile %s is empty", s.cfg.filterQueryFile)
	}
	s.queries = queries
	return nil
}

// filterQuery is a single query for testing.
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

func (fq *filterQuery) run(client *client, historyPruneBlock *uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()
	logs, err := client.Eth.FilterLogs(ctx, ethereum.FilterQuery{
		FromBlock: big.NewInt(fq.FromBlock),
		ToBlock:   big.NewInt(fq.ToBlock),
		Addresses: fq.Address,
		Topics:    fq.Topics,
	})
	fq.results = logs
	fq.Err = validateHistoryPruneErr(err, uint64(fq.FromBlock), historyPruneBlock)
}

func (fq *filterQuery) printError() {
	fmt.Printf("Filter query failed: fromBlock: %d toBlock: %d addresses: %v topics: %v error: %v\n",
		fq.FromBlock, fq.ToBlock, fq.Address, fq.Topics, fq.Err)
}
