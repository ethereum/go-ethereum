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
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"regexp"
	"slices"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/urfave/cli/v2"
)

var blockTestCommand = &cli.Command{
	Action:    blockTestCmd,
	Name:      "blocktest",
	Usage:     "Executes the given blockchain tests",
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
	if len(path) == 0 {
		return errors.New("path argument required")
	}
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
		result := &testResult{Name: name, Pass: true}
		if err := tests[name].Run(false, rawdb.HashScheme, ctx.Bool(WitnessCrossCheckFlag.Name), tracer, func(res error, chain *core.BlockChain) {
			if ctx.Bool(DumpFlag.Name) {
				if s, _ := chain.State(); s != nil {
					result.State = dump(s)
				}
			}
		}); err != nil {
			result.Pass, result.Error = false, err.Error()
		}
		results = append(results, *result)
	}
	return results, nil
}
