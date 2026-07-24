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
	"bufio"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"regexp"
	"runtime"
	"slices"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/urfave/cli/v2"
)

var (
	WorkersFlag = &cli.IntFlag{
		Name:  "workers",
		Usage: "Number of parallel workers for processing fixture files",
		Value: 1,
	}
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
		collected := collectFiles(path)
		workers := ctx.Int(WorkersFlag.Name)
		if workers <= 0 {
			workers = runtime.NumCPU()
		}
		results, err := runEngineTestsParallel(ctx, collected, workers)
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
		if !ctx.IsSet(FuzzFlag.Name) {
			report(ctx, results)
		}
	}
	return nil
}

// fileResult holds the results from processing a single fixture file.
type fileResult struct {
	index   int
	results []testResult
	err     error
}

// runEngineTestsParallel processes fixture files using a worker pool.
func runEngineTestsParallel(ctx *cli.Context, files []string, workers int) ([]testResult, error) {
	if workers == 1 {
		// Fast path: no goroutine overhead for single worker
		var results []testResult
		for _, fname := range files {
			r, err := runEngineTest(ctx, fname)
			if err != nil {
				return nil, err
			}
			results = append(results, r...)
		}
		return results, nil
	}
	// Parallel execution
	var (
		wg     sync.WaitGroup
		fileCh = make(chan struct {
			index int
			fname string
		}, len(files))
		resultCh = make(chan fileResult, len(files))
	)
	// Feed files into the channel
	for i, fname := range files {
		fileCh <- struct {
			index int
			fname string
		}{i, fname}
	}
	close(fileCh)

	// Start workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range fileCh {
				r, err := runEngineTest(ctx, item.fname)
				resultCh <- fileResult{index: item.index, results: r, err: err}
			}
		}()
	}
	// Close result channel when all workers are done
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results in order
	ordered := make([]fileResult, len(files))
	for fr := range resultCh {
		if fr.err != nil {
			return nil, fr.err
		}
		ordered[fr.index] = fr
	}
	var results []testResult
	for _, fr := range ordered {
		results = append(results, fr.results...)
	}
	return results, nil
}

func runEngineTest(ctx *cli.Context, fname string) ([]testResult, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var testsByName map[string]*tests.EngineTest
	if err = json.Unmarshal(src, &testsByName); err != nil {
		// Skip non-fixture JSON files (e.g. .meta/index.json)
		return nil, nil
	}
	re, err := regexp.Compile(ctx.String(RunFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid regex -%s: %v", RunFlag.Name, err)
	}
	tracer := tracerFromFlags(ctx)

	if ctx.IsSet(FuzzFlag.Name) {
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
		var finalHash *common.Hash
		if err := test.Run(rawdb.PathScheme, tracer, func(res error, chain *core.BlockChain) {
			if ctx.Bool(DumpFlag.Name) {
				if s, _ := chain.State(); s != nil {
					result.State = dump(s)
				}
			}
			if chain != nil {
				hash := chain.CurrentBlock().Hash()
				finalHash = &hash
			}
		}); err != nil {
			result.Pass, result.Error = false, err.Error()
		}

		result.Fork = test.Network()
		if finalHash != nil {
			result.BlockHash = finalHash
		}
		result.PayloadStatus = test.LastPayloadStatus
		if result.Pass && test.LastValidationError != "" {
			result.Error = test.LastValidationError
		}

		if ctx.IsSet(FuzzFlag.Name) {
			report(ctx, []testResult{*result})
		}
		results = append(results, *result)
	}
	return results, nil
}
