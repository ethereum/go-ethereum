// Copyright 2025 The go-ethereum Authors
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
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
	traceGenerateCommand = &cli.Command{
		Name:      "tracegen",
		Usage:     "Generates tests for state tracing",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    generateTraceTests,
		Flags: []cli.Flag{
			traceTestFileFlag,
			traceTestResultOutputFlag,
			traceTestStartBlockFlag,
			traceTestEndBlockFlag,
		},
	}

	traceTestFileFlag = &cli.StringFlag{
		Name:     "trace-tests",
		Usage:    "JSON file containing trace test queries",
		Value:    "trace_tests.json",
		Category: flags.TestingCategory,
	}
	traceTestResultOutputFlag = &cli.StringFlag{
		Name:     "trace-output",
		Usage:    "Folder containing detailed trace output files",
		Value:    "",
		Category: flags.TestingCategory,
	}
	traceTestStartBlockFlag = &cli.IntFlag{
		Name:     "trace-start",
		Usage:    "The number of starting block for tracing (included)",
		Category: flags.TestingCategory,
	}
	traceTestEndBlockFlag = &cli.IntFlag{
		Name:     "trace-end",
		Usage:    "The number of ending block for tracing (excluded)",
		Category: flags.TestingCategory,
	}
	traceTestInvalidOutputFlag = &cli.StringFlag{
		Name:     "trace-invalid",
		Usage:    "Folder containing the mismatched trace output files",
		Value:    "",
		Category: flags.TestingCategory,
	}
)

func generateTraceTests(clictx *cli.Context) error {
	var (
		client     = makeClient(clictx)
		outputFile = clictx.String(traceTestFileFlag.Name)
		outputDir  = clictx.String(traceTestResultOutputFlag.Name)
		startBlock = clictx.Int(traceTestStartBlockFlag.Name)
		endBlock   = clictx.Int(traceTestEndBlockFlag.Name)
		ctx        = context.Background()
		test       = new(traceTest)
	)
	latest, err := client.Eth.BlockNumber(ctx)
	if err != nil {
		exit(err)
	}
	if startBlock > endBlock {
		exit(fmt.Errorf("invalid block range for tracing, start: %d, end: %d", startBlock, endBlock))
	}
	if endBlock-startBlock == 0 {
		exit(fmt.Errorf("invalid block range for tracing, start: %d, end: %d", startBlock, endBlock))
	}
	if latest < uint64(startBlock) || latest < uint64(endBlock) {
		exit(fmt.Errorf("node seems not synced, latest block is %d", latest))
	}
	// Get blocks and assign block info into the test
	var (
		start  = time.Now()
		logged = time.Now()
		failed int
	)
	log.Info("Trace transactions around the chain tip", "head", latest, "start", startBlock, "end", endBlock)

	for i := startBlock; i < endBlock; i++ {
		header, err := client.Eth.HeaderByNumber(ctx, big.NewInt(int64(i)))
		if err != nil {
			exit(err)
		}
		config, configName := randomTraceOption()
		result, err := client.Geth.TraceBlock(ctx, header.Hash(), config)
		if err != nil {
			failed += 1
			continue
		}
		blob, err := json.Marshal(result)
		if err != nil {
			failed += 1
			continue
		}
		test.BlockHashes = append(test.BlockHashes, header.Hash())
		test.TraceConfigs = append(test.TraceConfigs, *config)
		test.ResultHashes = append(test.ResultHashes, crypto.Keccak256Hash(blob))
		writeTraceResult(outputDir, header.Hash(), result, configName)

		if time.Since(logged) > time.Second*8 {
			logged = time.Now()
			log.Info("Tracing blocks", "executed", len(test.BlockHashes), "failed", failed, "elapsed", common.PrettyDuration(time.Since(start)))
		}
	}
	log.Info("Traced blocks", "executed", len(test.BlockHashes), "failed", failed, "elapsed", common.PrettyDuration(time.Since(start)))

	// Write output file.
	writeJSON(outputFile, test)
	return nil
}

func randomTraceOption() (*tracers.TraceConfig, string) {
	x := rand.Intn(10)
	if x == 0 {
		// default options for struct-logger, with stack and storage capture
		// enabled
		return &tracers.TraceConfig{
			Config: &logger.Config{},
		}, "structDefault"
	}
	if x >= 1 && x <= 3 {
		// struct-logger with storage capture enabled
		return &tracers.TraceConfig{
			Config: &logger.Config{
				DisableStack: true,
			},
		}, "structStorage"
	}
	// Native tracer
	loggers := []string{"callTracer", "4byteTracer", "flatCallTracer", "muxTracer", "noopTracer", "prestateTracer"}
	return &tracers.TraceConfig{
		Tracer: &loggers[x-4],
	}, loggers[x-4]
}

func writeTraceResult(dir string, hash common.Hash, result any, configName string) {
	if dir == "" {
		return
	}
	// Ensure the directory exists
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		exit(fmt.Errorf("failed to create directories: %w", err))
	}
	name := filepath.Join(dir, configName+"_"+hash.String())
	file, err := os.Create(name)
	if err != nil {
		exit(fmt.Errorf("error creating %s: %v", name, err))
		return
	}
	defer file.Close()

	data, _ := json.MarshalIndent(result, "", "    ")
	_, err = file.Write(data)
	if err != nil {
		exit(fmt.Errorf("error writing %s: %v", name, err))
		return
	}
}
