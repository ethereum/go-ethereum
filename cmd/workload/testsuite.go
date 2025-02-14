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
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

var (
	runTestCommand = &cli.Command{
		Name:      "test",
		Usage:     "Runs workload tests against an RPC endpoint",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    runTestCmd,
		Flags: []cli.Flag{
			testPatternFlag,
			testTAPFlag,
			filterQueryFileFlag,
			filterErrorFileFlag,
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
)

type testSuite struct {
	ec             *ethclient.Client
	finalizedBlock int64
	filterTest
}

func newTestSuite(ctx *cli.Context) *testSuite {
	s := &testSuite{ec: makeEthClient(ctx)}
	s.getFinalizedBlock()
	s.filterTest.initFilterTest(ctx)
	return s
}

func (s *testSuite) getFinalizedBlock() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	header, err := s.ec.HeaderByNumber(ctx, big.NewInt(int64(rpc.FinalizedBlockNumber)))
	if err != nil {
		exit(fmt.Errorf("could not fetch finalized header (error: %v)", err))
	}
	s.finalizedBlock = header.Number.Int64()
}

func (s *testSuite) allTests() []utesting.Test {
	return []utesting.Test{
		{Name: "Filter/ShortRange", Fn: s.filterShortRange},
		{Name: "Filter/LongRange", Fn: s.filterLongRange},
		{Name: "Filter/FullRange", Fn: s.filterFullRange},
	}
}

func runTestCmd(ctx *cli.Context) error {
	s := newTestSuite(ctx)
	// Filter test cases.
	tests := s.allTests()
	if ctx.IsSet(testPatternFlag.Name) {
		tests = utesting.MatchTests(tests, ctx.String(testPatternFlag.Name))
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
