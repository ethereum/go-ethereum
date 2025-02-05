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
	"encoding/json"
	"fmt"
	"io"
	"os"
	goruntime "runtime"
	"runtime/pprof"
	"time"

	"github.com/XinFinOrg/XDPoSChain/cmd/evm/internal/compiler"
	"github.com/XinFinOrg/XDPoSChain/cmd/utils"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/core/vm/runtime"
	"github.com/XinFinOrg/XDPoSChain/internal/flags"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/urfave/cli/v2"
)

var runCommand = &cli.Command{
	Action:      runCmd,
	Name:        "run",
	Usage:       "run arbitrary evm binary",
	ArgsUsage:   "<code>",
	Description: `The run command runs arbitrary EVM code.`,
}

// readGenesis will read the given JSON format genesis file and return
// the initialized Genesis structure
func readGenesis(genesisPath string) *core.Genesis {
	// Make sure we have a valid genesis JSON
	//genesisPath := ctx.Args().First()
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

func runCmd(ctx *cli.Context) error {
	logconfig := &vm.LogConfig{
		EnableMemory:     !ctx.Bool(DisableMemoryFlag.Name),
		DisableStack:     ctx.Bool(DisableStackFlag.Name),
		DisableStorage:   ctx.Bool(DisableStorageFlag.Name),
		EnableReturnData: !ctx.Bool(DisableReturnDataFlag.Name),
		Debug:            ctx.Bool(DebugFlag.Name),
	}

	var (
		tracer      vm.EVMLogger
		debugLogger *vm.StructLogger
		statedb     *state.StateDB
		chainConfig *params.ChainConfig
		sender      = common.StringToAddress("sender")
		receiver    = common.StringToAddress("receiver")
	)
	if ctx.Bool(MachineFlag.Name) {
		tracer = vm.NewJSONLogger(logconfig, os.Stdout)
	} else if ctx.Bool(DebugFlag.Name) {
		debugLogger = vm.NewStructLogger(logconfig)
		tracer = debugLogger
	} else {
		debugLogger = vm.NewStructLogger(logconfig)
	}

	if ctx.String(GenesisFlag.Name) != "" {
		gen := readGenesis(ctx.String(GenesisFlag.Name))
		db := rawdb.NewMemoryDatabase()
		genesis := gen.ToBlock(db)
		statedb, _ = state.New(genesis.Root(), state.NewDatabase(db))
		chainConfig = gen.Config
	} else {
		db := rawdb.NewMemoryDatabase()
		statedb, _ = state.New(common.Hash{}, state.NewDatabase(db))
	}
	if ctx.String(SenderFlag.Name) != "" {
		sender = common.HexToAddress(ctx.String(SenderFlag.Name))
	}
	statedb.CreateAccount(sender)

	if ctx.String(ReceiverFlag.Name) != "" {
		receiver = common.HexToAddress(ctx.String(ReceiverFlag.Name))
	}

	var (
		code []byte
		ret  []byte
		err  error
	)
	// The '--code' or '--codefile' flag overrides code in state
	if ctx.String(CodeFileFlag.Name) != "" {
		var hexcode []byte
		var err error
		// If - is specified, it means that code comes from stdin
		if ctx.String(CodeFileFlag.Name) == "-" {
			//Try reading from stdin
			if hexcode, err = io.ReadAll(os.Stdin); err != nil {
				fmt.Printf("Could not load code from stdin: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Codefile with hex assembly
			if hexcode, err = os.ReadFile(ctx.String(CodeFileFlag.Name)); err != nil {
				fmt.Printf("Could not load code from file: %v\n", err)
				os.Exit(1)
			}
		}
		code = common.Hex2Bytes(string(bytes.TrimRight(hexcode, "\n")))

	} else if ctx.String(CodeFlag.Name) != "" {
		code = common.Hex2Bytes(ctx.String(CodeFlag.Name))
	} else if fn := ctx.Args().First(); len(fn) > 0 {
		// EASM-file to compile
		src, err := os.ReadFile(fn)
		if err != nil {
			return err
		}
		bin, err := compiler.Compile(fn, src, false)
		if err != nil {
			return err
		}
		code = common.Hex2Bytes(bin)
	}

	initialGas := ctx.Uint64(GasFlag.Name)
	runtimeConfig := runtime.Config{
		Origin:   sender,
		State:    statedb,
		GasLimit: initialGas,
		GasPrice: flags.GlobalBig(ctx, PriceFlag.Name),
		Value:    flags.GlobalBig(ctx, ValueFlag.Name),
		EVMConfig: vm.Config{
			Tracer: tracer,
		},
	}

	if cpuProfilePath := ctx.String(CPUProfileFlag.Name); cpuProfilePath != "" {
		f, err := os.Create(cpuProfilePath)
		if err != nil {
			fmt.Println("could not create CPU profile: ", err)
			os.Exit(1)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			fmt.Println("could not start CPU profile: ", err)
			os.Exit(1)
		}
		defer pprof.StopCPUProfile()
	}

	if chainConfig != nil {
		runtimeConfig.ChainConfig = chainConfig
	}
	tstart := time.Now()
	var leftOverGas uint64
	if ctx.Bool(CreateFlag.Name) {
		input := append(code, common.Hex2Bytes(ctx.String(InputFlag.Name))...)
		ret, _, leftOverGas, err = runtime.Create(input, &runtimeConfig)
	} else {
		if len(code) > 0 {
			statedb.SetCode(receiver, code)
		}
		ret, leftOverGas, err = runtime.Call(receiver, common.Hex2Bytes(ctx.String(InputFlag.Name)), &runtimeConfig)
	}
	execTime := time.Since(tstart)

	if ctx.Bool(DumpFlag.Name) {
		statedb.IntermediateRoot(true)
		fmt.Println(string(statedb.Dump()))
	}

	if memProfilePath := ctx.String(MemProfileFlag.Name); memProfilePath != "" {
		f, err := os.Create(memProfilePath)
		if err != nil {
			fmt.Println("could not create memory profile: ", err)
			os.Exit(1)
		}
		if err := pprof.WriteHeapProfile(f); err != nil {
			fmt.Println("could not write memory profile: ", err)
			os.Exit(1)
		}
		f.Close()
	}

	if ctx.Bool(DebugFlag.Name) {
		if debugLogger != nil {
			fmt.Fprintln(os.Stderr, "#### TRACE ####")
			vm.WriteTrace(os.Stderr, debugLogger.StructLogs())
		}
		fmt.Fprintln(os.Stderr, "#### LOGS ####")
		vm.WriteLogs(os.Stderr, statedb.Logs())
	}

	if ctx.Bool(StatDumpFlag.Name) {
		var mem goruntime.MemStats
		goruntime.ReadMemStats(&mem)
		fmt.Fprintf(os.Stderr, `evm execution time: %v
heap objects:       %d
allocations:        %d
total allocations:  %d
GC calls:           %d
Gas used:           %d

`, execTime, mem.HeapObjects, mem.Alloc, mem.TotalAlloc, mem.NumGC, initialGas-leftOverGas)
	}
	if tracer != nil {
		tracer.CaptureEnd(ret, initialGas-leftOverGas, execTime, err)
	} else {
		fmt.Printf("%#x\n", ret)
		if err != nil {
			fmt.Printf(" error: %v\n", err)
		}
	}

	return nil
}
