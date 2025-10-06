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

// blocktestEndMarker represents the final status of a blocktest execution.
// It is written as the last line of trace output in JSONL format (single-line JSON).
type blocktestEndMarker struct {
	TestEnd blocktestEndDetails `json:"testEnd"`
}

type blocktestEndDetails struct {
	Name  string `json:"name"`
	Pass  bool   `json:"pass"`
	Fork  string `json:"fork,omitempty"`
	Root  string `json:"root,omitempty"`
	Error string `json:"error,omitempty"`
	V     int    `json:"v"` // Version: 1
}

// writeEndMarker writes the blocktest end marker to stderr in JSONL format.
// This marker indicates the final outcome of the test as a single-line JSON object.
func writeEndMarker(result *testResult, fork string, root *common.Hash) {
	details := blocktestEndDetails{
		Name: result.Name,
		Pass: result.Pass,
		Fork: fork,
		V:    1,
	}
	if !result.Pass && result.Error != "" {
		details.Error = result.Error
	}
	if root != nil {
		details.Root = root.Hex()
	}
	marker := blocktestEndMarker{TestEnd: details}
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

	// Suppress INFO logs when tracing to avoid polluting stderr
	if tracer != nil {
		log.SetDefault(log.NewLogger(log.DiscardHandler()))
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
		results = append(results, *result)

		// Write end marker when tracing is enabled
		if tracer != nil {
			fork := test.Network()
			writeEndMarker(result, fork, finalRoot)
		}
	}
	return results, nil
}
