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
	"github.com/ethereum/go-ethereum/tests"
	"github.com/urfave/cli/v2"
)

var blockTestCommand = &cli.Command{
	Action:    blockTestCmd,
	Name:      "blocktest",
	Usage:     "Executes the given blockchain tests. Filenames can be fed via standard input (batch mode) or as an argument (one-off execution).",
	ArgsUsage: "<path>",
	Flags: slices.Concat([]cli.Flag{
		DumpFlag,
		HumanReadableFlag,
		RunFlag,
		WitnessCrossCheckFlag,
	}, traceFlags),
}

func blockTestCmd(ctx *cli.Context) error {
	path := ctx.Args().First()

	// If path is provided, run the tests at that path.
	if len(path) != 0 {
		var (
			collected = collectFiles(path)
			results   []testResult
		)
		for _, fname := range collected {
			r, err := runBlockTest(ctx, fname)
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
		results, err := runBlockTest(ctx, fname)
		if err != nil {
			return err
		}
		report(ctx, results)
	}
	return nil
}

// traceEndMarker represents the final status of a blocktest when tracing is enabled.
// It is written as the last line of trace output in JSONL format to signal completion.
type traceEndMarker struct {
	TestEnd traceEndDetails `json:"testEnd"`
}

type traceEndDetails struct {
	Name  string `json:"name"`
	Pass  bool   `json:"pass"`
	Fork  string `json:"fork"`
	Root  string `json:"root,omitempty"`
	Error string `json:"error,omitempty"`
}

// writeTraceEndMarker writes a blocktest end marker to stderr in JSONL format.
// This provides a clear delimiter for trace parsers (e.g., goevmlab) to know when
// the trace output for a specific test is complete, enabling proper batched processing.
func writeTraceEndMarker(name string, pass bool, fork string, root *common.Hash, errMsg string) {
	details := traceEndDetails{
		Name: name,
		Pass: pass,
		Fork: fork,
	}
	if root != nil {
		details.Root = root.Hex()
	}
	if !pass && errMsg != "" {
		details.Error = errMsg
	}
	marker := traceEndMarker{TestEnd: details}
	if data, err := json.Marshal(marker); err == nil {
		fmt.Fprintf(os.Stderr, "%s\n", data)
	}
}

func runBlockTest(ctx *cli.Context, fname string) ([]testResult, error) {
	src, err := os.ReadFile(fname)
	if err != nil {
		return nil, err
	}
	var tests map[string]*tests.BlockTest
	if err = json.Unmarshal(src, &tests); err != nil {
		return nil, err
	}
	re, err := regexp.Compile(ctx.String(RunFlag.Name))
	if err != nil {
		return nil, fmt.Errorf("invalid regex -%s: %v", RunFlag.Name, err)
	}
	tracer := tracerFromFlags(ctx)

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
		if err := test.Run(false, rawdb.PathScheme, ctx.Bool(WitnessCrossCheckFlag.Name), tracer, func(res error, chain *core.BlockChain) {
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

		// When tracing, write end marker to delimit trace output for this test
		if tracer != nil {
			writeTraceEndMarker(result.Name, result.Pass, result.Fork, finalRoot, result.Error)
		}

		results = append(results, *result)
	}
	return results, nil
}
