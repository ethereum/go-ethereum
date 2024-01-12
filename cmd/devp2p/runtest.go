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
	"os"

	"github.com/ethereum/go-ethereum/cmd/devp2p/internal/v4test"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/internal/utesting"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
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

	// for eth/snap tests
	testChainDirFlag = &cli.StringFlag{
		Name:     "chain",
		Usage:    "Test chain directory (required)",
		Category: flags.TestingCategory,
	}
	testNodeFlag = &cli.StringFlag{
		Name:     "node",
		Usage:    "Peer-to-Peer endpoint (ENR) of the test node (required)",
		Category: flags.TestingCategory,
	}
	testNodeJWTFlag = &cli.StringFlag{
		Name:     "jwtsecret",
		Usage:    "JWT secret for the engine API of the test node (required)",
		Category: flags.TestingCategory,
		Value:    "0x7365637265747365637265747365637265747365637265747365637265747365",
	}
	testNodeEngineFlag = &cli.StringFlag{
		Name:     "engineapi",
		Usage:    "Engine API endpoint of the test node (required)",
		Category: flags.TestingCategory,
	}

	// These two are specific to the discovery tests.
	testListen1Flag = &cli.StringFlag{
		Name:     "listen1",
		Usage:    "IP address of the first tester",
		Value:    v4test.Listen1,
		Category: flags.TestingCategory,
	}
	testListen2Flag = &cli.StringFlag{
		Name:     "listen2",
		Usage:    "IP address of the second tester",
		Value:    v4test.Listen2,
		Category: flags.TestingCategory,
	}
)

func runTests(ctx *cli.Context, tests []utesting.Test) error {
	// Filter test cases.
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
