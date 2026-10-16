// Copyright 2014 The go-ethereum Authors
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

// evm executes EVM code snippets.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/ethereum/go-ethereum/cmd/evm/internal/t8ntool"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/tests"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"

	// Force-load the tracer engines to trigger registration
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

// Some other nice-to-haves:
// * accumulate traces into an object to bundle with test
// * write tx identifier for trace before hand (blocktest only)
// * combine blocktest and statetest runner logic using unified test interface

const traceCategory = "TRACING"

var (
	// Test running flags.
	RunFlag = &cli.StringFlag{
		Name:  "run",
		Value: ".*",
		Usage: "Run only those tests matching the regular expression.",
	}
	BenchFlag = &cli.BoolFlag{
		Name:     "bench",
		Usage:    "benchmark the execution",
		Category: flags.VMCategory,
	}
	FuzzFlag = &cli.BoolFlag{
		Name:     "fuzz",
		Usage:    "adapts output format for fuzzing",
		Category: flags.VMCategory,
	}
	WitnessCrossCheckFlag = &cli.BoolFlag{
		Name:    "cross-check",
		Aliases: []string{"xc"},
		Usage:   "Cross-check stateful execution against stateless, verifying the witness generation.",
	}
	WorkersFlag = &cli.IntFlag{
		Name:  "workers",
		Usage: "Number of test files to run in parallel (0 = one per CPU)",
		Value: 1,
	}
	BALSequentialFlag = &cli.BoolFlag{
		Name:  "bal-sequential",
		Usage: "Execute every block sequentially, even one the EIP-7928 parallel processor could run. The delivered block access list is still validated.",
	}
	BALWithholdFlag = &cli.BoolFlag{
		Name:  "bal-withhold",
		Usage: "Never deliver the fixture's block access list, so every block executes without one, as a block received without its list does.",
	}
	CacheNoPrecompileFlag = &cli.BoolFlag{
		Name:  "cache.noprecompile",
		Usage: "Disable precompile result caching",
	}
	BALReportFlag = &cli.BoolFlag{
		Name:  "bal-report",
		Usage: "Print one JSON line per executed block on stderr, naming the processor that ran it (EIP-7928 parallel or sequential) and why parallel was ruled out.",
	}

	// Debugging flags.
	DumpFlag = &cli.BoolFlag{
		Name:  "dump",
		Usage: "dumps the state after the run",
	}
	HumanReadableFlag = &cli.BoolFlag{
		Name:  "human",
		Usage: "\"Human-readable\" output",
	}
	StatDumpFlag = &cli.BoolFlag{
		Name:  "statdump",
		Usage: "displays stack and heap memory information",
	}

	// Tracing flags.
	TraceFlag = &cli.BoolFlag{
		Name:     "trace",
		Usage:    "Enable tracing and output trace log.",
		Category: traceCategory,
	}
	TraceFormatFlag = &cli.StringFlag{
		Name:     "trace.format",
		Usage:    "Trace output format to use (json|struct|md)",
		Value:    "json",
		Category: traceCategory,
	}
	TraceDisableMemoryFlag = &cli.BoolFlag{
		Name:     "trace.nomemory",
		Aliases:  []string{"nomemory"},
		Value:    true,
		Usage:    "disable memory output",
		Category: traceCategory,
	}
	TraceDisableStackFlag = &cli.BoolFlag{
		Name:     "trace.nostack",
		Aliases:  []string{"nostack"},
		Usage:    "disable stack output",
		Category: traceCategory,
	}
	TraceDisableStorageFlag = &cli.BoolFlag{
		Name:     "trace.nostorage",
		Aliases:  []string{"nostorage"},
		Usage:    "disable storage output",
		Category: traceCategory,
	}
	TraceDisableReturnDataFlag = &cli.BoolFlag{
		Name:     "trace.noreturndata",
		Aliases:  []string{"noreturndata"},
		Value:    true,
		Usage:    "disable return data output",
		Category: traceCategory,
	}

	// Deprecated flags.
	DebugFlag = &cli.BoolFlag{
		Name:     "debug",
		Usage:    "output full trace logs (deprecated)",
		Hidden:   true,
		Category: traceCategory,
	}
	MachineFlag = &cli.BoolFlag{
		Name:     "json",
		Usage:    "output trace logs in machine readable format, json (deprecated)",
		Hidden:   true,
		Category: traceCategory,
	}
)

// Command definitions.
var (
	stateTransitionCommand = &cli.Command{
		Name:    "transition",
		Aliases: []string{"t8n"},
		Usage:   "Executes a full state transition",
		Action:  t8ntool.Transition,
		Flags: []cli.Flag{
			t8ntool.TraceFlag,
			t8ntool.TraceTracerFlag,
			t8ntool.TraceTracerConfigFlag,
			t8ntool.TraceEnableMemoryFlag,
			t8ntool.TraceDisableStackFlag,
			t8ntool.TraceEnableReturnDataFlag,
			t8ntool.TraceEnableCallFramesFlag,
			t8ntool.OutputBasedir,
			t8ntool.OutputAllocFlag,
			t8ntool.OutputResultFlag,
			t8ntool.OutputBodyFlag,
			t8ntool.InputAllocFlag,
			t8ntool.InputEnvFlag,
			t8ntool.InputTxsFlag,
			t8ntool.ForknameFlag,
			t8ntool.ChainIDFlag,
			t8ntool.RewardFlag,
			t8ntool.OpcodeCountFlag,
		},
	}

	transactionCommand = &cli.Command{
		Name:    "transaction",
		Aliases: []string{"t9n"},
		Usage:   "Performs transaction validation",
		Action:  t8ntool.Transaction,
		Flags: []cli.Flag{
			t8ntool.InputTxsFlag,
			t8ntool.ChainIDFlag,
			t8ntool.ForknameFlag,
		},
	}

	blockBuilderCommand = &cli.Command{
		Name:    "block-builder",
		Aliases: []string{"b11r"},
		Usage:   "Builds a block",
		Action:  t8ntool.BuildBlock,
		Flags: []cli.Flag{
			t8ntool.OutputBasedir,
			t8ntool.OutputBlockFlag,
			t8ntool.InputHeaderFlag,
			t8ntool.InputOmmersFlag,
			t8ntool.InputWithdrawalsFlag,
			t8ntool.InputTxsRlpFlag,
			t8ntool.SealCliqueFlag,
		},
	}
)

// traceFlags contains flags that configure tracing output.
var traceFlags = []cli.Flag{
	TraceFlag,
	TraceFormatFlag,
	TraceDisableStackFlag,
	TraceDisableMemoryFlag,
	TraceDisableStorageFlag,
	TraceDisableReturnDataFlag,

	// deprecated
	DebugFlag,
	MachineFlag,
}

var app = flags.NewApp("the evm command line interface")

func init() {
	app.Flags = debug.Flags
	app.Commands = []*cli.Command{
		runCommand,
		blockTestCommand,
		engineTestCommand,
		stateTestCommand,
		stateTransitionCommand,
		transactionCommand,
		blockBuilderCommand,
	}
	app.Before = func(ctx *cli.Context) error {
		flags.MigrateGlobalFlags(ctx)
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// tracerFromFlags parses the cli flags and returns the specified tracer.
func tracerFromFlags(ctx *cli.Context) *tracing.Hooks {
	config := &logger.Config{
		EnableMemory:     !ctx.Bool(TraceDisableMemoryFlag.Name),
		DisableStack:     ctx.Bool(TraceDisableStackFlag.Name),
		DisableStorage:   ctx.Bool(TraceDisableStorageFlag.Name),
		EnableReturnData: !ctx.Bool(TraceDisableReturnDataFlag.Name),
	}
	switch {
	case ctx.Bool(TraceFlag.Name):
		switch format := ctx.String(TraceFormatFlag.Name); format {
		case "struct":
			return logger.NewStreamingStructLogger(config, os.Stderr).Hooks()
		case "json":
			return logger.NewJSONLogger(config, os.Stderr)
		case "md", "markdown":
			return logger.NewMarkdownLogger(config, os.Stderr).Hooks()
		default:
			fmt.Fprintf(os.Stderr, "unknown trace format: %q\n", format)
			os.Exit(1)
			return nil
		}
	// Deprecated ways of configuring tracing.
	case ctx.Bool(MachineFlag.Name):
		return logger.NewJSONLogger(config, os.Stderr)
	case ctx.Bool(DebugFlag.Name):
		return logger.NewStreamingStructLogger(config, os.Stderr).Hooks()
	default:
		return nil
	}
}

// runConfig returns the block and engine test execution settings set by flags.
func runConfig(ctx *cli.Context) tests.RunConfig {
	return tests.RunConfig{
		Sequential:         ctx.Bool(BALSequentialFlag.Name),
		WithholdAccessList: ctx.Bool(BALWithholdFlag.Name),
		NoPrecompileCache:  ctx.Bool(CacheNoPrecompileFlag.Name),
	}
}

// collectFiles walks the given path. If the path is a directory, it will
// return a list of all accumulates all files with json extension.
// Otherwise (if path points to a file), it will return the path.
func collectFiles(path string) []string {
	var out []string
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		// User explicitly pointed out a file, ignore extension.
		return []string{path}
	}
	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(info.Name()) == ".json" {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return out
}

// collectFixtures walks the given paths for blocktest and enginetest. A
// directory contributes all files with json extension below it, except in a
// .meta directory, where EEST keeps files that are not fixtures; a path that
// points to a file is used as is. A path that does not exist or cannot be read
// is an error, returned before any test runs.
func collectFixtures(paths ...string) ([]string, error) {
	var out []string
	for _, path := range paths {
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			// User explicitly pointed out a file, ignore extension.
			out = append(out, path)
			continue
		}
		err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && info.Name() == ".meta" {
				return filepath.SkipDir
			}
			if !info.IsDir() && filepath.Ext(info.Name()) == ".json" {
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	for _, path := range out {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		f.Close()
	}
	return out, nil
}

// runFiles runs the test files with run, on as many files at once as the workers
// flag allows, and returns their results in file order. A file that cannot be
// run does not stop the others; the errors of all such files are returned.
func runFiles(ctx *cli.Context, files []string, run func(*cli.Context, string) ([]testResult, error)) ([]testResult, error) {
	workers := ctx.Int(WorkersFlag.Name)
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	var (
		results = make([][]testResult, len(files))
		errs    = make([]error, len(files))
		group   errgroup.Group
	)
	group.SetLimit(workers)
	for i, fname := range files {
		group.Go(func() error {
			results[i], errs[i] = run(ctx, fname)
			return nil
		})
	}
	group.Wait()
	return slices.Concat(results...), errors.Join(errs...)
}

// dump returns a state dump for the most current trie.
func dump(s *state.StateDB) *state.Dump {
	// A dump is not a state transition: report accounts exactly as they are.
	root := s.IntermediateRoot(params.Rules{})
	cpy, _ := state.New(root, s.Database())
	dump := cpy.RawDump(nil)
	return &dump
}
