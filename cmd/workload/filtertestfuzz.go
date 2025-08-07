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
	"math/big"
	"reflect"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

const maxFilterRangeForTestFuzz = 300

var (
	filterFuzzCommand = &cli.Command{
		Name:      "filterfuzz",
		Usage:     "Generates queries and compares results against matches derived from receipts",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    filterFuzzCmd,
		Flags:     []cli.Flag{},
	}
)

// filterFuzzCmd is the main function of the filter fuzzer.
func filterFuzzCmd(ctx *cli.Context) error {
	f := newFilterTestGen(ctx, maxFilterRangeForTestFuzz)
	var lastHead *types.Header
	headerCache := lru.NewCache[common.Hash, *types.Header](200)

	commonAncestor := func(oldPtr, newPtr *types.Header) *types.Header {
		if oldPtr == nil || newPtr == nil {
			return nil
		}
		if newPtr.Number.Uint64() > oldPtr.Number.Uint64()+100 || oldPtr.Number.Uint64() > newPtr.Number.Uint64()+100 {
			return nil
		}
		for oldPtr.Hash() != newPtr.Hash() {
			if newPtr.Number.Uint64() >= oldPtr.Number.Uint64() {
				if parent, _ := headerCache.Get(newPtr.ParentHash); parent != nil {
					newPtr = parent
				} else {
					newPtr, _ = getHeaderByHash(f.client, newPtr.ParentHash)
					if newPtr == nil {
						return nil
					}
					headerCache.Add(newPtr.Hash(), newPtr)
				}
			}
			if oldPtr.Number.Uint64() > newPtr.Number.Uint64() {
				oldPtr, _ = headerCache.Get(oldPtr.ParentHash)
				if oldPtr == nil {
					return nil
				}
			}
		}
		return newPtr
	}

	fetchHead := func() (*types.Header, bool) {
		currentHead, err := getLatestHeader(f.client)
		if err != nil {
			fmt.Println("Could not fetch head block", err)
			return nil, false
		}
		headerCache.Add(currentHead.Hash(), currentHead)
		if lastHead != nil && currentHead.Hash() == lastHead.Hash() {
			return currentHead, false
		}
		f.blockLimit = currentHead.Number.Int64()
		ca := commonAncestor(lastHead, currentHead)
		fmt.Print("*** New head ", f.blockLimit)
		if ca == nil {
			fmt.Println("  <no common ancestor>")
		} else {
			if reorged := lastHead.Number.Uint64() - ca.Number.Uint64(); reorged > 0 {
				fmt.Print("  reorged ", reorged)
			}
			if missed := currentHead.Number.Uint64() - ca.Number.Uint64() - 1; missed > 0 {
				fmt.Print("  missed ", missed)
			}
			fmt.Println()
		}
		lastHead = currentHead
		return currentHead, true
	}

	tryExtendQuery := func(query *filterQuery) *filterQuery {
		for {
			extQuery := f.extendRange(query)
			if extQuery == nil {
				return query
			}
			extQuery.checkLastBlockHash(f.client)
			extQuery.run(f.client, nil)
			if extQuery.Err == nil && len(extQuery.results) == 0 {
				// query is useless now due to major reorg; abandon and continue
				fmt.Println("Zero length results")
				return nil
			}
			if extQuery.Err != nil {
				extQuery.printError()
				return nil
			}
			if len(extQuery.results) > maxFilterResultSize {
				return query
			}
			query = extQuery
		}
	}

	var (
		mmQuery              *filterQuery
		mmRetry, mmNextRetry int
	)

mainLoop:
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		var query *filterQuery
		if mmQuery != nil {
			if mmRetry == 0 {
				query = mmQuery
				mmRetry = mmNextRetry
				mmNextRetry *= 2
				query.checkLastBlockHash(f.client)
				query.run(f.client, nil)
				if query.Err != nil {
					query.printError()
					continue
				}
				fmt.Println("Retrying query  from:", query.FromBlock, "to:", query.ToBlock, "results:", len(query.results))
			} else {
				mmRetry--
			}
		}
		if query == nil {
			currentHead, isNewHead := fetchHead()
			if currentHead == nil {
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(time.Second):
				}
				continue mainLoop
			}
			if isNewHead {
				query = f.newHeadSeedQuery(currentHead.Number.Int64())
			} else {
				query = f.newQuery()
			}
			query.checkLastBlockHash(f.client)
			query.run(f.client, nil)
			if query.Err != nil {
				query.printError()
				continue
			}
			fmt.Println("New query  from:", query.FromBlock, "to:", query.ToBlock, "results:", len(query.results))
			if len(query.results) == 0 || len(query.results) > maxFilterResultSize {
				continue mainLoop
			}
			if query = tryExtendQuery(query); query == nil {
				continue mainLoop
			}
		}
		if !query.checkLastBlockHash(f.client) {
			fmt.Println("Reorg during search")
			continue mainLoop
		}
		// now we have a new query; check results
		results, err := query.getResultsFromReceipts(f.client)
		if err != nil {
			fmt.Println("Could not fetch results from receipts", err)
			continue mainLoop
		}
		if !query.checkLastBlockHash(f.client) {
			fmt.Println("Reorg during search")
			continue mainLoop
		}
		if !reflect.DeepEqual(query.results, results) {
			fmt.Println("Results mismatch  from:", query.FromBlock, "to:", query.ToBlock, "addresses:", query.Address, "topics:", query.Topics)
			resShared, resGetLogs, resReceipts := compareResults(query.results, results)
			fmt.Println(" shared:", len(resShared))
			fmt.Println(" only from getLogs:", len(resGetLogs), resGetLogs)
			fmt.Println(" only from receipts:", len(resReceipts), resReceipts)
			if mmQuery != query {
				mmQuery = query
				mmRetry = 0
				mmNextRetry = 1
			}
			continue mainLoop
		}
		fmt.Println("Successful query  from:", query.FromBlock, "to:", query.ToBlock, "results:", len(query.results))
		f.storeQuery(query)
	}
}

func compareResults(a, b []types.Log) (shared, onlya, onlyb []types.Log) {
	for len(a) > 0 && len(b) > 0 {
		if reflect.DeepEqual(a[0], b[0]) {
			shared = append(shared, a[0])
			a = a[1:]
			b = b[1:]
		} else {
			for i := 1; ; i++ {
				if i >= len(a) { // b[0] not found in a
					onlyb = append(onlyb, b[0])
					b = b[1:]
					break
				}
				if i >= len(b) { // a[0] not found in b
					onlya = append(onlya, a[0])
					a = a[1:]
					break
				}
				if reflect.DeepEqual(b[0], a[i]) { // a[:i] not found in b
					onlya = append(onlya, a[:i]...)
					a = a[i:]
					break
				}
				if reflect.DeepEqual(a[0], b[i]) { // b[:i] not found in a
					onlyb = append(onlyb, b[:i]...)
					b = b[i:]
					break
				}
			}
		}
	}
	onlya = append(onlya, a...)
	onlyb = append(onlyb, b...)
	return
}

func getLatestHeader(client *client) (*types.Header, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	return client.Eth.HeaderByNumber(ctx, big.NewInt(int64(rpc.LatestBlockNumber)))
}

func getHeaderByHash(client *client, hash common.Hash) (*types.Header, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	return client.Eth.HeaderByHash(ctx, hash)
}

// newHeadSeedQuery creates a query that gets all logs from the latest head.
func (s *filterTestGen) newHeadSeedQuery(head int64) *filterQuery {
	return &filterQuery{
		FromBlock: head,
		ToBlock:   head,
	}
}

func (fq *filterQuery) checkLastBlockHash(client *client) bool {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	header, err := client.Eth.HeaderByNumber(ctx, big.NewInt(fq.ToBlock))
	if err != nil {
		fmt.Println("Cound not fetch last block hash of query  number:", fq.ToBlock, "error:", err)
		fq.lastBlockHash = common.Hash{}
		return false
	}
	hash := header.Hash()
	if fq.lastBlockHash == hash {
		return true
	}
	fq.lastBlockHash = hash
	return false
}

func (fq *filterQuery) filterLog(log *types.Log) bool {
	if len(fq.Address) > 0 && !slices.Contains(fq.Address, log.Address) {
		return false
	}
	// If the to filtered topics is greater than the amount of topics in logs, skip.
	if len(fq.Topics) > len(log.Topics) {
		return false
	}
	for i, sub := range fq.Topics {
		if len(sub) == 0 {
			continue // empty rule set == wildcard
		}
		if !slices.Contains(sub, log.Topics[i]) {
			return false
		}
	}
	return true
}

func (fq *filterQuery) getResultsFromReceipts(client *client) ([]types.Log, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	var results []types.Log
	for blockNumber := fq.FromBlock; blockNumber <= fq.ToBlock; blockNumber++ {
		receipts, err := client.Eth.BlockReceipts(ctx, rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(blockNumber)))
		if err != nil {
			return nil, err
		}
		for _, receipt := range receipts {
			for _, log := range receipt.Logs {
				if fq.filterLog(log) {
					results = append(results, *log)
				}
			}
		}
	}
	return results, nil
}
