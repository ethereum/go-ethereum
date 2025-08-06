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
	"errors"
	"io/fs"
	"os"

	"github.com/ethereum/go-ethereum/core/history"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
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
			testArchiveFlag,
			testSepoliaFlag,
			testMainnetFlag,
			filterQueryFileFlag,
			historyTestFileFlag,
			traceTestFileFlag,
			traceTestInvalidOutputFlag,
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
	testArchiveFlag = &cli.BoolFlag{
		Name:     "archive",
		Usage:    "Enable archive tests",
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
	client            *client
	fsys              fs.FS
	filterQueryFile   string
	historyTestFile   string
	historyPruneBlock *uint64
	traceTestFile     string
}

var errPrunedHistory = errors.New("attempt to access pruned history")

// validateHistoryPruneErr checks whether the given error is caused by access
// to history before the pruning threshold block (it is an rpc.Error with code 4444).
// In this case, errPrunedHistory is returned.
// If the error is a pruned history error that occurs when accessing a block past the
// historyPrune block, an error is returned.
// Otherwise, the original value of err is returned.
func validateHistoryPruneErr(err error, blockNum uint64, historyPruneBlock *uint64) error {
	if err != nil {
		if rpcErr, ok := err.(rpc.Error); ok && rpcErr.ErrorCode() == 4444 {
			if historyPruneBlock != nil && blockNum > *historyPruneBlock {
				return errors.New("pruned history error returned after pruning threshold")
			}
			return errPrunedHistory
		}
	}
	return err
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
		if ctx.IsSet(filterQueryFileFlag.Name) {
			cfg.filterQueryFile = ctx.String(filterQueryFileFlag.Name)
		} else {
			cfg.filterQueryFile = "queries/filter_queries_mainnet.json"
		}
		if ctx.IsSet(historyTestFileFlag.Name) {
			cfg.historyTestFile = ctx.String(historyTestFileFlag.Name)
		} else {
			cfg.historyTestFile = "queries/history_mainnet.json"
		}
		if ctx.IsSet(traceTestFileFlag.Name) {
			cfg.traceTestFile = ctx.String(traceTestFileFlag.Name)
		} else {
			cfg.traceTestFile = "queries/trace_mainnet.json"
		}
		cfg.historyPruneBlock = new(uint64)
		*cfg.historyPruneBlock = history.PrunePoints[params.MainnetGenesisHash].BlockNumber
	case ctx.Bool(testSepoliaFlag.Name):
		cfg.fsys = builtinTestFiles
		if ctx.IsSet(filterQueryFileFlag.Name) {
			cfg.filterQueryFile = ctx.String(filterQueryFileFlag.Name)
		} else {
			cfg.filterQueryFile = "queries/filter_queries_sepolia.json"
		}
		if ctx.IsSet(historyTestFileFlag.Name) {
			cfg.historyTestFile = ctx.String(historyTestFileFlag.Name)
		} else {
			cfg.historyTestFile = "queries/history_sepolia.json"
		}
		if ctx.IsSet(traceTestFileFlag.Name) {
			cfg.traceTestFile = ctx.String(traceTestFileFlag.Name)
		} else {
			cfg.traceTestFile = "queries/trace_sepolia.json"
		}
		cfg.historyPruneBlock = new(uint64)
		*cfg.historyPruneBlock = history.PrunePoints[params.SepoliaGenesisHash].BlockNumber
	default:
		cfg.fsys = os.DirFS(".")
		cfg.filterQueryFile = ctx.String(filterQueryFileFlag.Name)
		cfg.historyTestFile = ctx.String(historyTestFileFlag.Name)
		cfg.traceTestFile = ctx.String(traceTestFileFlag.Name)
	}
	return cfg
}

// workloadTest represents a single test in the workload. It's a wrapper
// of utesting.Test by adding a few additional attributes.
type workloadTest struct {
	utesting.Test

	archive bool // Flag whether the archive node (full state history) is required for this test
}

func newWorkLoadTest(name string, fn func(t *utesting.T)) workloadTest {
	return workloadTest{
		Test: utesting.Test{
			Name: name,
			Fn:   fn,
		},
	}
}

func newSlowWorkloadTest(name string, fn func(t *utesting.T)) workloadTest {
	t := newWorkLoadTest(name, fn)
	t.Slow = true
	return t
}

func newArchiveWorkloadTest(name string, fn func(t *utesting.T)) workloadTest {
	t := newWorkLoadTest(name, fn)
	t.archive = true
	return t
}

func filterTests(tests []workloadTest, pattern string, filterFn func(t workloadTest) bool) []utesting.Test {
	var utests []utesting.Test
	for _, t := range tests {
		if filterFn(t) {
			utests = append(utests, t.Test)
		}
	}
	if pattern == "" {
		return utests
	}
	return utesting.MatchTests(utests, pattern)
}

func runTestCmd(ctx *cli.Context) error {
	cfg := testConfigFromCLI(ctx)
	filterSuite := newFilterTestSuite(cfg)
	historySuite := newHistoryTestSuite(cfg)
	traceSuite := newTraceTestSuite(cfg, ctx)

	// Filter test cases.
	tests := filterSuite.allTests()
	tests = append(tests, historySuite.allTests()...)
	tests = append(tests, traceSuite.allTests()...)

	utests := filterTests(tests, ctx.String(testPatternFlag.Name), func(t workloadTest) bool {
		if t.Slow && !ctx.Bool(testSlowFlag.Name) {
			return false
		}
		if t.archive && !ctx.Bool(testArchiveFlag.Name) {
			return false
		}
		return true
	})

	// Disable logging unless explicitly enabled.
	if !ctx.IsSet("verbosity") && !ctx.IsSet("vmodule") {
		log.SetDefault(log.NewLogger(log.DiscardHandler()))
	}

	// Run the tests.
	var run = utesting.RunTests
	if ctx.Bool(testTAPFlag.Name) {
		run = utesting.RunTAP
	}
	results := run(utests, os.Stdout)
	if utesting.CountFailures(results) > 0 {
		os.Exit(1)
	}
	return nil
}
