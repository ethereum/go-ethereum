// Copyright 2017 The go-ethereum Authors
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
	"os"
	"regexp"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/urfave/cli/v2"
)

var (
	forkFlag = &cli.StringFlag{
		Name:     "statetest.fork",
		Usage:    "Only run tests for the specified fork.",
		Category: flags.VMCategory,
	}
	idxFlag = &cli.IntFlag{
		Name:     "statetest.index",
		Usage:    "The index of the subtest to run.",
		Category: flags.VMCategory,
		Value:    -1, // default to select all subtest indices
	}
)
var stateTestCommand = &cli.Command{
	Action:    stateTestCmd,
	Name:      "statetest",
	Usage:     "Executes the given state tests. Filenames can be fed via standard input (batch mode) or as an argument (one-off execution).",
	ArgsUsage: "<file>",
	Flags: slices.Concat([]cli.Flag{
		BenchFlag,
		DumpFlag,
		forkFlag,
		HumanReadableFlag,
		idxFlag,
		RunFlag,
	}, traceFlags),
}

func stateTestCmd(ctx *cli.Context) error {
	path := ctx.Args().First()

	// If path is provided, run the tests at that path.
	if len(path) != 0 {
		var (
			collected = collectFiles(path)
			results   []testResult
		)
		for _, fname := range collected {
			r, err := runStateTest(ctx, fname)
			if err != nil {
				return err
			}
			results = append(results, r...)
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
		results, err := runStateTest(ctx, fname)
		if err != nil {
			return err
		}
		report(ctx, results)
	}
	return nil
}

// runStateTest loads the state-test given by fname, and executes the test.
func runStateTest(ctx *cli.Context, fname string) ([]testResult, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var testsByName map[string]tests.StateTest
	if err := json.Unmarshal(src, &testsByName); err != nil {
		return nil, fmt.Errorf("unable to read test file %s: %w", fname, err)
	}

	cfg := vm.Config{Tracer: tracerFromFlags(ctx)}
	re, err := regexp.Compile(ctx.String(RunFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid regex -%s: %v", RunFlag.Name, err)
	}

	// Iterate over all the tests, run them and aggregate the results
	results := make([]testResult, 0, len(testsByName))
	for key, test := range testsByName {
		if !re.MatchString(key) {
			continue
		}
		for i, st := range test.Subtests() {
			if idx := ctx.Int(idxFlag.Name); idx != -1 && idx != i {
				// If specific index requested, skip all tests that do not match.
				continue
			}
			if fork := ctx.String(forkFlag.Name); fork != "" && st.Fork != fork {
				// If specific fork requested, skip all tests that do not match.
				continue
			}
			// Run the test and aggregate the result
			result := &testResult{Name: key, Fork: st.Fork, Pass: true}
			test.Run(st, cfg, false, rawdb.HashScheme, func(err error, state *tests.StateTestState) {
				var root common.Hash
				if state.StateDB != nil {
					root = state.StateDB.IntermediateRoot(false)
					result.Root = &root
					fmt.Fprintf(os.Stderr, "{\"stateRoot\": \"%#x\"}\n", root)
					// Dump any state to aid debugging.
					if ctx.Bool(DumpFlag.Name) {
						result.State = dump(state.StateDB)
					}
				}
				// Collect bench stats if requested.
				if ctx.Bool(BenchFlag.Name) {
					_, stats, _ := timedExec(true, func() ([]byte, uint64, error) {
						_, _, gasUsed, _ := test.RunNoVerify(st, cfg, false, rawdb.HashScheme)
						return nil, gasUsed, nil
					})
					result.Stats = &stats
				}
				if err != nil {
					// Test failed, mark as so.
					result.Pass, result.Error = false, err.Error()
					return
				}
			})
			results = append(results, *result)
		}
	}
	return results, nil
}
