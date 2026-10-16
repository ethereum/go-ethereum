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
	"bufio"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/urfave/cli/v2"
)

var engineTestCommand = &cli.Command{
	Action:    engineTestCmd,
	Name:      "enginetest",
	Usage:     "Executes the given engine API tests. Filenames can be fed via standard input (batch mode) or as an argument (one-off execution).",
	ArgsUsage: "<path>",
	Flags: slices.Concat([]cli.Flag{
		DumpFlag,
		HumanReadableFlag,
		RunFlag,
		FuzzFlag,
		WorkersFlag,
	}, traceFlags),
}

func engineTestCmd(ctx *cli.Context) error {
	path := ctx.Args().First()

	// If path is provided, run the tests at that path.
	if len(path) != 0 {
		results, err := runFiles(ctx, collectFiles(path), runEngineTest)
		if err != nil {
			return err
		}
		report(ctx, results)
		return nil
	}
	// Otherwise, read filenames from stdin and execute back-to-back.
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fname := scanner.Text()
		if len(fname) == 0 {
			return nil
		}
		results, err := runEngineTest(ctx, fname)
		if err != nil {
			return err
		}
		if !ctx.Bool(FuzzFlag.Name) {
			report(ctx, results)
		}
	}
	return nil
}

func runEngineTest(ctx *cli.Context, fname string) ([]testResult, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var testsByName map[string]*tests.EngineTest
	if err = json.Unmarshal(src, &testsByName); err != nil {
		return nil, err
	}
	re, err := regexp.Compile(ctx.String(RunFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid regex -%s: %v", RunFlag.Name, err)
	}
	tracer := tracerFromFlags(ctx)

	if ctx.Bool(FuzzFlag.Name) {
		log.SetDefault(log.NewLogger(log.DiscardHandler()))
	}

	keys := slices.Sorted(maps.Keys(testsByName))

	var results []testResult
	for _, name := range keys {
		if !re.MatchString(name) {
			continue
		}
		test := testsByName[name]
		result := &testResult{Name: name, Pass: true}
		if err := test.Run(rawdb.PathScheme, tracer, attachEngineAPI, func(res error, chain *core.BlockChain) {
			if ctx.Bool(DumpFlag.Name) {
				if s, _ := chain.State(); s != nil {
					result.State = dump(s)
				}
			}
		}); err != nil {
			result.Pass, result.Error = false, err.Error()
		}

		result.Fork = test.Network()

		if ctx.Bool(FuzzFlag.Name) {
			report(ctx, []testResult{*result})
		}
		results = append(results, *result)
	}
	return results, nil
}

// attachEngineAPI puts geth's engine API, eth/catalyst.ConsensusAPI, on a
// test's chain and returns an in-process RPC client for it. The API is backed
// by the chain alone (see eth.NewEngineBackend): no node, networking, RPC
// transport or other services are started.
func attachEngineAPI(chain *core.BlockChain, db ethdb.Database) (*rpc.Client, func(), error) {
	backend := eth.NewEngineBackend(chain, db)
	server := rpc.NewServer()
	if err := server.RegisterName("engine", catalyst.NewConsensusAPIWithoutHeartbeat(backend)); err != nil {
		backend.Downloader().Terminate()
		return nil, nil, err
	}
	client := rpc.DialInProc(server)
	release := func() {
		client.Close()
		server.Stop()
		backend.Downloader().Terminate()
	}
	return client, release, nil
}
