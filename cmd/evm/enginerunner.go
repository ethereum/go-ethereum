// Copyright 2026 The go-ethereum Authors
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
	"maps"
	"os"
	"regexp"
	"slices"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/txpool/blobpool"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/fetcher"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/urfave/cli/v2"
)

var engineTestCommand = &cli.Command{
	Action:    engineTestCmd,
	Name:      "enginetest",
	Usage:     "Executes the given engine API tests. Filenames can be fed via standard input (batch mode) or as arguments.",
	ArgsUsage: "<path>...",
	Flags: slices.Concat([]cli.Flag{
		DumpFlag,
		HumanReadableFlag,
		RunFlag,
		FuzzFlag,
		WorkersFlag,
		BALSequentialFlag,
		CacheNoPrecompileFlag,
		BALReportFlag,
	}, traceFlags),
}

func engineTestCmd(ctx *cli.Context) error {
	if ctx.Bool(BALReportFlag.Name) {
		reportExecution(log.Root().Handler(), false)
	}

	// If paths are provided, run the tests at those paths.
	if ctx.Args().Present() {
		files, err := collectFixtures(ctx.Args().Slice()...)
		if err != nil {
			return cli.Exit(err, exitIncomplete)
		}
		results, err := runFiles(ctx, files, runEngineTest)
		report(ctx, results)
		if err != nil {
			return cli.Exit(err, exitIncomplete)
		}
		return failures(results)
	}
	// Otherwise, read filenames from stdin and execute back-to-back.
	return runStdin(ctx, runEngineTest)
}

func runEngineTest(ctx *cli.Context, fname string) ([]testResult, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var testsByName map[string]*tests.EngineTest
	if err = json.Unmarshal(src, &testsByName); err != nil {
		return nil, fmt.Errorf("unable to read test file %s: %w", fname, err)
	}
	re, err := regexp.Compile(ctx.String(RunFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid regex -%s: %v", RunFlag.Name, err)
	}
	tracer := tracerFromFlags(ctx)

	if ctx.Bool(FuzzFlag.Name) {
		discardLogs(ctx)
	}

	keys := slices.Sorted(maps.Keys(testsByName))

	var results []testResult
	for _, name := range keys {
		if !re.MatchString(name) {
			continue
		}
		test := testsByName[name]
		result := &testResult{Name: name, Pass: true}
		if err := test.Run(rawdb.PathScheme, tracer, runConfig(ctx), attachEngineAPI, func(res error, chain *core.BlockChain) {
			if ctx.Bool(DumpFlag.Name) {
				if s, _ := chain.State(); s != nil {
					result.State = dump(s)
				}
			}
		}); err != nil {
			result.Pass, result.Error = false, err.Error()
		}

		result.Fork = test.Network()
		result.Rejections = test.Rejections

		if ctx.Bool(FuzzFlag.Name) {
			report(ctx, []testResult{*result})
		}
		results = append(results, *result)
	}
	return results, nil
}

// engineBackend is the part of the eth service that the engine API needs to
// import payloads and update the forkchoice: the chain, its database and an
// idle full-sync downloader. Building payloads, serving blobs and updating
// custody columns are not supported, so the pool, miner and blob methods
// return nil.
type engineBackend struct {
	chain      *core.BlockChain
	db         ethdb.Database
	downloader *downloader.Downloader
	synced     atomic.Bool
}

func (b *engineBackend) BlockChain() *core.BlockChain       { return b.chain }
func (b *engineBackend) ChainDb() ethdb.Database            { return b.db }
func (b *engineBackend) Downloader() *downloader.Downloader { return b.downloader }
func (b *engineBackend) Synced() bool                       { return b.synced.Load() }
func (b *engineBackend) SetSynced()                         { b.synced.Store(true) }
func (b *engineBackend) EngineMaxReorgDepth() uint64        { return ethconfig.Defaults.EngineMaxReorgDepth }
func (b *engineBackend) TxPool() *txpool.TxPool             { return nil }
func (b *engineBackend) Miner() *miner.Miner                { return nil }
func (b *engineBackend) BlobCache() *blobpool.Cache         { return nil }
func (b *engineBackend) BlobFetcher() *fetcher.BlobFetcher  { return nil }

// attachEngineAPI puts geth's engine API, eth/catalyst.ConsensusAPI, on a
// test's chain and returns an in-process RPC client for it. No node,
// networking, RPC transport or other services are started.
func attachEngineAPI(chain *core.BlockChain, db ethdb.Database) (*rpc.Client, func(), error) {
	backend := &engineBackend{
		chain:      chain,
		db:         db,
		downloader: downloader.New(db, ethconfig.FullSync, chain, func(string) {}, func() {}, false),
	}
	server := rpc.NewServer()
	if err := server.RegisterName("engine", catalyst.NewConsensusAPIWithoutHeartbeat(backend)); err != nil {
		backend.downloader.Terminate()
		return nil, nil, err
	}
	client := rpc.DialInProc(server)
	release := func() {
		client.Close()
		server.Stop()
		backend.downloader.Terminate()
	}
	return client, release, nil
}
