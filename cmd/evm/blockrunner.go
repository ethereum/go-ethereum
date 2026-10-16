// Copyright 2023 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/urfave/cli/v2"
)

var blockTestCommand = &cli.Command{
	Action:    blockTestCmd,
	Name:      "blocktest",
	Usage:     "Executes the given blockchain tests. Filenames can be fed via standard input (batch mode) or as arguments.",
	ArgsUsage: "<path>...",
	Flags: slices.Concat([]cli.Flag{
		DumpFlag,
		HumanReadableFlag,
		RunFlag,
		WitnessCrossCheckFlag,
		BALSequentialFlag,
		BALWithholdFlag,
		BALReportFlag,
		FuzzFlag,
		WorkersFlag,
	}, traceFlags),
}

func blockTestCmd(ctx *cli.Context) error {
	if ctx.Bool(BALReportFlag.Name) {
		reportExecution(log.Root().Handler(), ctx.Bool(BALWithholdFlag.Name))
	}

	// If paths are provided, run the tests at those paths.
	if ctx.Args().Present() {
		files, err := collectFixtures(ctx.Args().Slice()...)
		if err != nil {
			return cli.Exit(err, exitIncomplete)
		}
		results, err := runFiles(ctx, files, runBlockTest)
		report(ctx, results)
		if err != nil {
			return cli.Exit(err, exitIncomplete)
		}
		return failures(results)
	}
	// Otherwise, read filenames from stdin and execute back-to-back.
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		fname := scanner.Text()
		if len(fname) == 0 {
			return nil
		}
		results, err := runBlockTest(ctx, fname)
		if err != nil {
			return err
		}
		// During fuzzing, we report the result after every block
		if !ctx.Bool(FuzzFlag.Name) {
			report(ctx, results)
		}
	}
	return nil
}

func runBlockTest(ctx *cli.Context, fname string) ([]testResult, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var tests map[string]*tests.BlockTest
	if err = json.Unmarshal(src, &tests); err != nil {
		return nil, fmt.Errorf("unable to read test file %s: %w", fname, err)
	}
	re, err := regexp.Compile(ctx.String(RunFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid regex -%s: %v", RunFlag.Name, err)
	}
	tracer := tracerFromFlags(ctx)

	// Suppress INFO logs during fuzzing
	if ctx.Bool(FuzzFlag.Name) {
		discardLogs(ctx)
	}

	// Pull out keys to sort and ensure tests are run in order.
	keys := slices.Sorted(maps.Keys(tests))

	// Run all the tests.
	var results []testResult
	for _, name := range keys {
		if !re.MatchString(name) {
			continue
		}
		test := tests[name]
		result := &testResult{Name: name, Pass: true}
		var finalRoot *common.Hash
		if err := test.Run(false, rawdb.PathScheme, ctx.Bool(WitnessCrossCheckFlag.Name), tracer, runConfig(ctx), func(res error, chain *core.BlockChain) {
			if ctx.Bool(DumpFlag.Name) {
				if s, _ := chain.State(); s != nil {
					result.State = dump(s)
				}
			}
			// Capture final state root for end marker
			if chain != nil {
				root := chain.CurrentBlock().Root
				finalRoot = &root
			}
		}); err != nil {
			result.Pass, result.Error = false, err.Error()
		}

		// Always assign fork (regardless of pass/fail or tracer)
		result.Fork = test.Network()
		// Assign root if test succeeded
		if result.Pass && finalRoot != nil {
			result.Root = finalRoot
		}
		result.Rejections = test.Rejections

		// When fuzzing, write results after every block
		if ctx.Bool(FuzzFlag.Name) {
			report(ctx, []testResult{*result})
		}
		results = append(results, *result)
	}
	return results, nil
}
