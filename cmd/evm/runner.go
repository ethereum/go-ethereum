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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	goruntime "runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/core/vm/runtime"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/urfave/cli/v2"
)

var runCommand = &cli.Command{
	Action:      runCmd,
	Name:        "run",
	Usage:       "Run arbitrary evm binary",
	ArgsUsage:   "<code>",
	Description: `The run command runs arbitrary EVM code.`,
	Flags: slices.Concat([]cli.Flag{
		BenchFlag,
		CodeFileFlag,
		CreateFlag,
		GasFlag,
		GenesisFlag,
		InputFlag,
		InputFileFlag,
		PriceFlag,
		ReceiverFlag,
		SenderFlag,
		ValueFlag,
		StatDumpFlag,
		DumpFlag,
	}, traceFlags),
}

var (
	CodeFileFlag = &cli.StringFlag{
		Name:     "codefile",
		Usage:    "File containing EVM code. If '-' is specified, code is read from stdin ",
		Category: flags.VMCategory,
	}
	CreateFlag = &cli.BoolFlag{
		Name:     "create",
		Usage:    "Indicates the action should be create rather than call",
		Category: flags.VMCategory,
	}
	GasFlag = &cli.Uint64Flag{
		Name:     "gas",
		Usage:    "Gas limit for the evm",
		Value:    10000000000,
		Category: flags.VMCategory,
	}
	GenesisFlag = &cli.StringFlag{
		Name:     "prestate",
		Usage:    "JSON file with prestate (genesis) config",
		Category: flags.VMCategory,
	}
	InputFlag = &cli.StringFlag{
		Name:     "input",
		Usage:    "Input for the EVM",
		Category: flags.VMCategory,
	}
	InputFileFlag = &cli.StringFlag{
		Name:     "inputfile",
		Usage:    "File containing input for the EVM",
		Category: flags.VMCategory,
	}
	PriceFlag = &flags.BigFlag{
		Name:     "price",
		Usage:    "Price set for the evm",
		Value:    new(big.Int),
		Category: flags.VMCategory,
	}
	ReceiverFlag = &cli.StringFlag{
		Name:     "receiver",
		Usage:    "The transaction receiver (execution context)",
		Category: flags.VMCategory,
	}
	SenderFlag = &cli.StringFlag{
		Name:     "sender",
		Usage:    "The transaction origin",
		Category: flags.VMCategory,
	}
	ValueFlag = &flags.BigFlag{
		Name:     "value",
		Usage:    "Value set for the evm",
		Value:    new(big.Int),
		Category: flags.VMCategory,
	}
)

// readGenesis will read the given JSON format genesis file and return
// the initialized Genesis structure
func readGenesis(genesisPath string) *core.Genesis {
	// Make sure we have a valid genesis JSON
	if len(genesisPath) == 0 {
		utils.Fatalf("Must supply path to genesis JSON file")
	}
	file, err := os.Open(genesisPath)
	if err != nil {
		utils.Fatalf("Failed to read genesis file: %v", err)
	}
	defer file.Close()

	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		utils.Fatalf("invalid genesis file: %v", err)
	}
	return genesis
}

type execStats struct {
	Time           time.Duration `json:"time"`           // The execution Time.
	Allocs         int64         `json:"allocs"`         // The number of heap allocations during execution.
	BytesAllocated int64         `json:"bytesAllocated"` // The cumulative number of bytes allocated during execution.
	GasUsed        uint64        `json:"gasUsed"`        // the amount of gas used during execution
}

func timedExec(bench bool, execFunc func() ([]byte, uint64, error)) ([]byte, execStats, error) {
	if bench {
		testing.Init()
		// Do one warm-up run
		output, gasUsed, err := execFunc()
		result := testing.Benchmark(func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				haveOutput, haveGasUsed, haveErr := execFunc()
				if !bytes.Equal(haveOutput, output) {
					panic(fmt.Sprintf("output differs\nhave %x\nwant %x\n", haveOutput, output))
				}
				if haveGasUsed != gasUsed {
					panic(fmt.Sprintf("gas differs, have %v want %v", haveGasUsed, gasUsed))
				}
				if haveErr != err {
					panic(fmt.Sprintf("err differs, have %v want %v", haveErr, err))
				}
			}
		})
		// Get the average execution time from the benchmarking result.
		// There are other useful stats here that could be reported.
		stats := execStats{
			Time:           time.Duration(result.NsPerOp()),
			Allocs:         result.AllocsPerOp(),
			BytesAllocated: result.AllocedBytesPerOp(),
			GasUsed:        gasUsed,
		}
		return output, stats, err
	}
	var memStatsBefore, memStatsAfter goruntime.MemStats
	goruntime.ReadMemStats(&memStatsBefore)
	t0 := time.Now()
	output, gasUsed, err := execFunc()
	duration := time.Since(t0)
	goruntime.ReadMemStats(&memStatsAfter)
	stats := execStats{
		Time:           duration,
		Allocs:         int64(memStatsAfter.Mallocs - memStatsBefore.Mallocs),
		BytesAllocated: int64(memStatsAfter.TotalAlloc - memStatsBefore.TotalAlloc),
		GasUsed:        gasUsed,
	}
	return output, stats, err
}

func runCmd(ctx *cli.Context) error {
	var (
		tracer      *tracing.Hooks
		prestate    *state.StateDB
		chainConfig *params.ChainConfig
		sender      = common.BytesToAddress([]byte("sender"))
		receiver    = common.BytesToAddress([]byte("receiver"))
		preimages   = ctx.Bool(DumpFlag.Name)
		blobHashes  []common.Hash  // TODO (MariusVanDerWijden) implement blob hashes in state tests
		blobBaseFee = new(big.Int) // TODO (MariusVanDerWijden) implement blob fee in state tests
	)
	tracer = tracerFromFlags(ctx)
	initialGas := ctx.Uint64(GasFlag.Name)
	genesisConfig := new(core.Genesis)
	genesisConfig.GasLimit = initialGas
	if ctx.String(GenesisFlag.Name) != "" {
		genesisConfig = readGenesis(ctx.String(GenesisFlag.Name))
		if genesisConfig.GasLimit != 0 {
			initialGas = genesisConfig.GasLimit
		}
	} else {
		genesisConfig.Config = params.AllDevChainProtocolChanges
	}

	db := rawdb.NewMemoryDatabase()
	triedb := triedb.NewDatabase(db, &triedb.Config{
		Preimages: preimages,
		HashDB:    hashdb.Defaults,
	})
	defer triedb.Close()
	genesis := genesisConfig.MustCommit(db, triedb)
	sdb := state.NewDatabase(triedb, nil)
	prestate, _ = state.New(genesis.Root(), sdb)
	chainConfig = genesisConfig.Config

	if ctx.String(SenderFlag.Name) != "" {
		sender = common.HexToAddress(ctx.String(SenderFlag.Name))
	}

	if ctx.String(ReceiverFlag.Name) != "" {
		receiver = common.HexToAddress(ctx.String(ReceiverFlag.Name))
	}

	var code []byte
	codeFileFlag := ctx.String(CodeFileFlag.Name)
	hexcode := ctx.Args().First()

	// The '--codefile' flag overrides code in state
	if codeFileFlag == "-" {
		// If - is specified, it means that code comes from stdin
		// Try reading from stdin
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Printf("Could not load code from stdin: %v\n", err)
			os.Exit(1)
		}
		hexcode = string(input)
	} else if codeFileFlag != "" {
		// Codefile with hex assembly
		input, err := os.ReadFile(codeFileFlag)
		if err != nil {
			fmt.Printf("Could not load code from file: %v\n", err)
			os.Exit(1)
		}
		hexcode = string(input)
	}

	hexcode = strings.TrimSpace(hexcode)
	if len(hexcode)%2 != 0 {
		fmt.Printf("Invalid input length for hex data (%d)\n", len(hexcode))
		os.Exit(1)
	}
	code = common.FromHex(hexcode)

	runtimeConfig := runtime.Config{
		Origin:      sender,
		State:       prestate,
		GasLimit:    initialGas,
		GasPrice:    flags.GlobalBig(ctx, PriceFlag.Name),
		Value:       flags.GlobalBig(ctx, ValueFlag.Name),
		Difficulty:  genesisConfig.Difficulty,
		Time:        genesisConfig.Timestamp,
		Coinbase:    genesisConfig.Coinbase,
		BlockNumber: new(big.Int).SetUint64(genesisConfig.Number),
		BaseFee:     genesisConfig.BaseFee,
		BlobHashes:  blobHashes,
		BlobBaseFee: blobBaseFee,
		EVMConfig: vm.Config{
			Tracer: tracer,
		},
	}

	if chainConfig != nil {
		runtimeConfig.ChainConfig = chainConfig
	} else {
		runtimeConfig.ChainConfig = params.AllEthashProtocolChanges
	}

	var hexInput []byte
	if inputFileFlag := ctx.String(InputFileFlag.Name); inputFileFlag != "" {
		var err error
		if hexInput, err = os.ReadFile(inputFileFlag); err != nil {
			fmt.Printf("could not load input from file: %v\n", err)
			os.Exit(1)
		}
	} else {
		hexInput = []byte(ctx.String(InputFlag.Name))
	}
	hexInput = bytes.TrimSpace(hexInput)
	if len(hexInput)%2 != 0 {
		fmt.Println("input length must be even")
		os.Exit(1)
	}
	input := common.FromHex(string(hexInput))

	var execFunc func() ([]byte, uint64, error)
	if ctx.Bool(CreateFlag.Name) {
		input = append(code, input...)
		execFunc = func() ([]byte, uint64, error) {
			// don't mutate the state!
			runtimeConfig.State = prestate.Copy()
			output, _, gasLeft, err := runtime.Create(input, &runtimeConfig)
			return output, gasLeft, err
		}
	} else {
		if len(code) > 0 {
			prestate.SetCode(receiver, code)
		}
		execFunc = func() ([]byte, uint64, error) {
			// don't mutate the state!
			runtimeConfig.State = prestate.Copy()
			output, gasLeft, err := runtime.Call(receiver, input, &runtimeConfig)
			return output, initialGas - gasLeft, err
		}
	}

	bench := ctx.Bool(BenchFlag.Name)
	output, stats, err := timedExec(bench, execFunc)

	if ctx.Bool(DumpFlag.Name) {
		root, err := runtimeConfig.State.Commit(genesisConfig.Number, true, false)
		if err != nil {
			fmt.Printf("Failed to commit changes %v\n", err)
			return err
		}
		dumpdb, err := state.New(root, sdb)
		if err != nil {
			fmt.Printf("Failed to open statedb %v\n", err)
			return err
		}
		fmt.Println(string(dumpdb.Dump(nil)))
	}

	if ctx.Bool(DebugFlag.Name) {
		if logs := runtimeConfig.State.Logs(); len(logs) > 0 {
			fmt.Fprintln(os.Stderr, "### LOGS")
			writeLogs(os.Stderr, logs)
		}
	}

	if bench || ctx.Bool(StatDumpFlag.Name) {
		fmt.Fprintf(os.Stderr, `EVM gas used:    %d
execution time:  %v
allocations:     %d
allocated bytes: %d
`, stats.GasUsed, stats.Time, stats.Allocs, stats.BytesAllocated)
	}
	if tracer == nil {
		fmt.Printf("%#x\n", output)
		if err != nil {
			fmt.Printf(" error: %v\n", err)
		}
	}

	return nil
}

// writeLogs writes vm logs in a readable format to the given writer
func writeLogs(writer io.Writer, logs []*types.Log) {
	for _, log := range logs {
		fmt.Fprintf(writer, "LOG%d: %x bn=%d txi=%x\n", len(log.Topics), log.Address, log.BlockNumber, log.TxIndex)

		for i, topic := range log.Topics {
			fmt.Fprintf(writer, "%08d  %x\n", i, topic)
		}
		fmt.Fprint(writer, hex.Dump(log.Data))
		fmt.Fprintln(writer)
	}
}
