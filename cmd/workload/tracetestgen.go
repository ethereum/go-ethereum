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
	defaultBlocksToTrace = 64 // the number of states assumed to be available

	traceGenerateCommand = &cli.Command{
		Name:      "tracegen",
		Usage:     "Generates tests for state tracing",
		ArgsUsage: "<RPC endpoint URL>",
		Action:    generateTraceTests,
		Flags: []cli.Flag{
			traceTestFileFlag,
			traceTestResultOutputFlag,
			traceTestBlockFlag,
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
		Usage:    "Folder containing the trace output files",
		Value:    "",
		Category: flags.TestingCategory,
	}
	traceTestBlockFlag = &cli.IntFlag{
		Name:     "trace-blocks",
		Usage:    "The number of blocks for tracing",
		Value:    defaultBlocksToTrace,
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
		blocks     = clictx.Int(traceTestBlockFlag.Name)
		ctx        = context.Background()
		test       = new(traceTest)
	)
	if outputDir != "" {
		err := os.MkdirAll(outputDir, os.ModePerm)
		if err != nil {
			return err
		}
	}
	latest, err := client.Eth.BlockNumber(ctx)
	if err != nil {
		exit(err)
	}
	if latest < uint64(blocks) {
		exit(fmt.Errorf("node seems not synced, latest block is %d", latest))
	}
	// Get blocks and assign block info into the test
	var (
		start  = time.Now()
		logged = time.Now()
		failed int
	)
	log.Info("Trace transactions around the chain tip", "head", latest, "blocks", blocks)

	for i := 0; i < blocks; i++ {
		number := latest - uint64(i)
		block, err := client.Eth.BlockByNumber(ctx, big.NewInt(int64(number)))
		if err != nil {
			exit(err)
		}
		for _, tx := range block.Transactions() {
			config, configName := randomTraceOption()
			result, err := client.Geth.TraceTransaction(ctx, tx.Hash(), config)
			if err != nil {
				failed += 1
				break
			}
			blob, err := json.Marshal(result)
			if err != nil {
				failed += 1
				break
			}
			test.TxHashes = append(test.TxHashes, tx.Hash())
			test.TraceConfigs = append(test.TraceConfigs, *config)
			test.ResultHashes = append(test.ResultHashes, crypto.Keccak256Hash(blob))
			writeTraceResult(outputDir, tx.Hash(), result, configName)
		}
		if time.Since(logged) > time.Second*8 {
			logged = time.Now()
			log.Info("Tracing transactions", "executed", len(test.TxHashes), "failed", failed, "elapsed", common.PrettyDuration(time.Since(start)))
		}
	}
	log.Info("Traced transactions", "executed", len(test.TxHashes), "failed", failed, "elapsed", common.PrettyDuration(time.Since(start)))

	// Write output file.
	writeJSON(outputFile, test)
	return nil
}

func randomTraceOption() (*tracers.TraceConfig, string) {
	x := rand.Intn(11)
	if x == 0 {
		// struct-logger, with all fields enabled, very heavy
		return &tracers.TraceConfig{
			Config: &logger.Config{
				EnableMemory:     true,
				EnableReturnData: true,
			},
		}, "structAll"
	}
	if x == 1 {
		// default options for struct-logger, with stack and storage capture
		// enabled
		return &tracers.TraceConfig{
			Config: &logger.Config{},
		}, "structDefault"
	}
	if x == 2 || x == 3 || x == 4 {
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
		Tracer: &loggers[x-5],
	}, loggers[x-5]
}

func writeTraceResult(dir string, hash common.Hash, result any, configName string) {
	if dir == "" {
		return
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
