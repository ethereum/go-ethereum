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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/urfave/cli/v2"
)

var stateTestCommand = &cli.Command{
	Action:    stateTestCmd,
	Name:      "statetest",
	Usage:     "Executes the given state tests. Filenames can be fed via standard input (batch mode) or as an argument (one-off execution).",
	ArgsUsage: "<file>",
	Flags: flags.Merge([]cli.Flag{
		DumpFlag,
		HumanReadableFlag,
		RunFlag,
	}, traceFlags),
}

func stateTestCmd(ctx *cli.Context) error {
	path := ctx.Args().First()

	// If path is provided, run the tests at that path.
	if len(path) != 0 {
		var (
			collected = collectJSONFiles(path)
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
		if err := runStateTest(ctx, fname); err != nil {
			return err
		}
	}
	return nil
}

// runStateTest loads the state-test given by fname, and executes the test.
func runStateTest(ctx *cli.Context, fname string) error {
	src, err := os.ReadFile(fname)
	if err != nil {
		return err
	}
	var testsByName map[string]tests.StateTest
	if err := json.Unmarshal(src, &testsByName); err != nil {
		return err
	}

	cfg := vm.Config{Tracer: tracerFromFlags(ctx)}
	re, err := regexp.Compile(ctx.String(RunFlag.Name))
	if err != nil {
		return fmt.Errorf("invalid regex -%s: %v", RunFlag.Name, err)
	}

	// Iterate over all the tests, run them and aggregate the results
	results := make([]testResult, 0, len(testsByName))
	for key, test := range testsByName {
		if !re.MatchString(key) {
			continue
		}
		for _, st := range test.Subtests() {
			// Run the test and aggregate the result
			result := &testResult{Name: key, Fork: st.Fork, Pass: true}
			test.Run(st, cfg, false, rawdb.HashScheme, func(err error, state *tests.StateTestState) {
				var root common.Hash
				if state.StateDB != nil {
					root = state.StateDB.IntermediateRoot(false)
					result.Root = &root
					// Dump any state to aid debugging.
					if ctx.Bool(DumpFlag.Name) {
						result.State = dump(state.StateDB)
					}
				}
				if err != nil {
					// Test failed, mark as so
					result.Pass, result.Error = false, err.Error()
					return
				}
			})
			results = append(results, *result)
		}
	}
	report(ctx, results)
	return nil
}
