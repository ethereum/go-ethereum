// Copyright 2020 The go-ethereum Authors
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
	"embed"
	"io/fs"
	"os"
	"slices"

	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

//go:embed queries
var builtinTestFiles embed.FS

var (
	runTestCommand = &cli.Command{
		Name:      "test",
		Usage:     "Runs workload tests against an RPC endpoint",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    runTestCmd,
		Flags: []cli.Flag{
			testPatternFlag,
			testTAPFlag,
			testSlowFlag,
			testSepoliaFlag,
			testMainnetFlag,
			filterQueryFileFlag,
			historyTestFileFlag,
		},
	}
	testPatternFlag = &cli.StringFlag{
		Name:     "run",
		Usage:    "Pattern of test suite(s) to run",
		Category: flags.TestingCategory,
	}
	testTAPFlag = &cli.BoolFlag{
		Name:     "tap",
		Usage:    "Output test results in TAP format",
		Category: flags.TestingCategory,
	}
	testSlowFlag = &cli.BoolFlag{
		Name:     "slow",
		Usage:    "Enable slow tests",
		Value:    false,
		Category: flags.TestingCategory,
	}
	testSepoliaFlag = &cli.BoolFlag{
		Name:     "sepolia",
		Usage:    "Use test cases for sepolia network",
		Category: flags.TestingCategory,
	}
	testMainnetFlag = &cli.BoolFlag{
		Name:     "mainnet",
		Usage:    "Use test cases for mainnet network",
		Category: flags.TestingCategory,
	}
)

// testConfig holds the parameters for testing.
type testConfig struct {
	client          *client
	fsys            fs.FS
	filterQueryFile string
	historyTestFile string
}

func testConfigFromCLI(ctx *cli.Context) (cfg testConfig) {
	flags.CheckExclusive(ctx, testMainnetFlag, testSepoliaFlag)
	if (ctx.IsSet(testMainnetFlag.Name) || ctx.IsSet(testSepoliaFlag.Name)) && ctx.IsSet(filterQueryFileFlag.Name) {
		exit(filterQueryFileFlag.Name + " cannot be used with " + testMainnetFlag.Name + " or " + testSepoliaFlag.Name)
	}

	// configure ethclient
	cfg.client = makeClient(ctx)

	// configure test files
	switch {
	case ctx.Bool(testMainnetFlag.Name):
		cfg.fsys = builtinTestFiles
		cfg.filterQueryFile = "queries/filter_queries_mainnet.json"
		cfg.historyTestFile = "queries/history_mainnet.json"
	case ctx.Bool(testSepoliaFlag.Name):
		cfg.fsys = builtinTestFiles
		cfg.filterQueryFile = "queries/filter_queries_sepolia.json"
		cfg.historyTestFile = "queries/history_sepolia.json"
	default:
		cfg.fsys = os.DirFS(".")
		cfg.filterQueryFile = ctx.String(filterQueryFileFlag.Name)
		cfg.historyTestFile = ctx.String(historyTestFileFlag.Name)
	}
	return cfg
}

func runTestCmd(ctx *cli.Context) error {
	cfg := testConfigFromCLI(ctx)
	filterSuite := newFilterTestSuite(cfg)
	historySuite := newHistoryTestSuite(cfg)

	// Filter test cases.
	tests := filterSuite.allTests()
	tests = append(tests, historySuite.allTests()...)
	if ctx.IsSet(testPatternFlag.Name) {
		tests = utesting.MatchTests(tests, ctx.String(testPatternFlag.Name))
	}
	if !ctx.Bool(testSlowFlag.Name) {
		tests = slices.DeleteFunc(tests, func(test utesting.Test) bool {
			return test.Slow
		})
	}

	// Disable logging unless explicitly enabled.
	if !ctx.IsSet("verbosity") && !ctx.IsSet("vmodule") {
		log.SetDefault(log.NewLogger(log.DiscardHandler()))
	}

	// Run the tests.
	var run = utesting.RunTests
	if ctx.Bool(testTAPFlag.Name) {
		run = utesting.RunTAP
	}
	results := run(tests, os.Stdout)
	if utesting.CountFailures(results) > 0 {
		os.Exit(1)
	}
	return nil
}
