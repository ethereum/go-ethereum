// Copyright 2015 The go-ethereum Authors
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
	"fmt"
	"os"
	"runtime"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/XinFinOrg/XDPoSChain/cmd/utils"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	xdc_genesis "github.com/XinFinOrg/XDPoSChain/genesis"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/urfave/cli/v2"
)

var (
	initCommand = &cli.Command{
		Action:    initGenesis,
		Name:      "init",
		Usage:     "Bootstrap and initialize a new genesis block",
		ArgsUsage: "<genesisPath>",
		Flags:     slices.Concat(utils.NetworkFlags, utils.DatabaseFlags),
		Description: `
The init command initializes a new genesis block and definition for the network.
This is a destructive action and changes the network in which you will be
participating.

It expects the genesis file or the network name [ mainnet | testnet | devnet ] as argument.`,
	}
	importCommand = &cli.Command{
		Action:    importChain,
		Name:      "import",
		Usage:     "Import a blockchain file",
		ArgsUsage: "<filename> (<filename 2> ... <filename N>) ",
		Flags: slices.Concat([]cli.Flag{
			utils.CacheFlag,
			utils.SyncModeFlag,
			utils.GCModeFlag,
			utils.CacheDatabaseFlag,
			utils.CacheGCFlag,
			utils.MetricsEnabledFlag,
			utils.MetricsEnabledExpensiveFlag,
			utils.MetricsEnableInfluxDBFlag,
			utils.MetricsEnableInfluxDBV2Flag,
			utils.MetricsInfluxDBEndpointFlag,
			utils.MetricsInfluxDBDatabaseFlag,
			utils.MetricsInfluxDBUsernameFlag,
			utils.MetricsInfluxDBPasswordFlag,
			utils.MetricsInfluxDBTagsFlag,
			utils.MetricsInfluxDBTokenFlag,
			utils.MetricsInfluxDBBucketFlag,
			utils.MetricsInfluxDBOrganizationFlag,
		}, utils.DatabaseFlags),
		Description: `
The import command imports blocks from an RLP-encoded form. The form can be one file
with several RLP-encoded blocks, or several files can be used.

If only one file is used, import error will result in failure. If several files are used,
processing will proceed even if an individual RLP-file import failure occurs.`,
	}
	exportCommand = &cli.Command{
		Action:    exportChain,
		Name:      "export",
		Usage:     "Export blockchain into file",
		ArgsUsage: "<filename> [<blockNumFirst> <blockNumLast>]",
		Flags: slices.Concat([]cli.Flag{
			utils.CacheFlag,
			utils.SyncModeFlag,
		}, utils.DatabaseFlags),
		Description: `
Requires a first argument of the file to write to.
Optional second and third arguments control the first and
last block to write. In this mode, the file will be appended
if already existing.`,
	}
	importPreimagesCommand = &cli.Command{
		Action:    importPreimages,
		Name:      "import-preimages",
		Usage:     "Import the preimage database from an RLP stream",
		ArgsUsage: "<datafile>",
		Flags: slices.Concat([]cli.Flag{
			utils.CacheFlag,
			utils.SyncModeFlag,
		}, utils.DatabaseFlags),
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
	The import-preimages command imports hash preimages from an RLP encoded stream.`,
	}
	exportPreimagesCommand = &cli.Command{
		Action:    exportPreimages,
		Name:      "export-preimages",
		Usage:     "Export the preimage database into an RLP stream",
		ArgsUsage: "<dumpfile>",
		Flags: slices.Concat([]cli.Flag{
			utils.CacheFlag,
			utils.SyncModeFlag,
		}, utils.DatabaseFlags),
		Description: `
The export-preimages command export hash preimages to an RLP encoded stream`,
	}
	dumpCommand = &cli.Command{
		Action:    dump,
		Name:      "dump",
		Usage:     "Dump a specific block from storage",
		ArgsUsage: "[<blockHash> | <blockNum>]...",
		Flags: slices.Concat([]cli.Flag{
			utils.CacheFlag,
			utils.SyncModeFlag,
		}, utils.DatabaseFlags),
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
The arguments are interpreted as block numbers or hashes.
Use "ethereum dump 0" to dump the genesis block.`,
	}
)

// initGenesis will initialise the given JSON format genesis file and writes it as
// the zero'd block (i.e. genesis) or will fail hard if it can't succeed.
func initGenesis(ctx *cli.Context) error {
	if ctx.Args().Len() != 1 {
		utils.Fatalf("need the genesis.json file or the network name [ mainnet | testnet | devnet ] as the only argument")
	}

	var err error
	genesis := new(core.Genesis)
	genesisPath := ctx.Args().First()
	if genesisPath == "mainnet" || genesisPath == "xinfin" {
		err = json.Unmarshal(xdc_genesis.MainnetGenesis, genesis)
	} else if genesisPath == "testnet" || genesisPath == "apothem" {
		err = json.Unmarshal(xdc_genesis.TestnetGenesis, genesis)
	} else if genesisPath == "devnet" {
		err = json.Unmarshal(xdc_genesis.DevnetGenesis, genesis)
	} else {
		if len(genesisPath) == 0 {
			utils.Fatalf("invalid path to genesis file")
		}
		var file *os.File
		file, err = os.Open(genesisPath)
		if err != nil {
			utils.Fatalf("Failed to read genesis file: %v", err)
		}
		defer file.Close()
		err = json.NewDecoder(file).Decode(genesis)
	}
	if err != nil {
		utils.Fatalf("invalid genesis json: %v", err)
	}

	if genesis.Config.ChainId != nil {
		common.CopyConstants(genesis.Config.ChainId.Uint64())
	}

	// Open an initialise both full and light databases
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	name := "chaindata"
	chaindb, err := stack.OpenDatabase(name, 0, 0, "", false)
	if err != nil {
		utils.Fatalf("Failed to open database: %v", err)
	}
	_, hash, err := core.SetupGenesisBlock(chaindb, genesis)
	if err != nil {
		utils.Fatalf("Failed to write genesis block: %v", err)
	}
	chaindb.Close()
	log.Info("Successfully wrote genesis state", "database", name, "hash", hash.String())
	return nil
}

func importChain(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		utils.Fatalf("This command requires an argument.")
	}
	stack, _, cfg := makeFullNode(ctx)
	defer stack.Close()

	// Start metrics export if enabled
	utils.SetupMetrics(&cfg.Metrics)

	chain, db := utils.MakeChain(ctx, stack, false)
	defer db.Close()

	// Start periodically gathering memory profiles
	var peakMemAlloc, peakMemSys uint64
	go func() {
		stats := new(runtime.MemStats)
		for {
			runtime.ReadMemStats(stats)
			if atomic.LoadUint64(&peakMemAlloc) < stats.Alloc {
				atomic.StoreUint64(&peakMemAlloc, stats.Alloc)
			}
			if atomic.LoadUint64(&peakMemSys) < stats.Sys {
				atomic.StoreUint64(&peakMemSys, stats.Sys)
			}
			time.Sleep(5 * time.Second)
		}
	}()
	// Import the chain
	start := time.Now()

	var importErr error

	if ctx.Args().Len() == 1 {
		if err := utils.ImportChain(chain, ctx.Args().First()); err != nil {
			importErr = err
			log.Error("Import error", "err", err)
		}
	} else {
		for _, arg := range ctx.Args().Slice() {
			if err := utils.ImportChain(chain, arg); err != nil {
				importErr = err
				log.Error("Import error", "file", arg, "err", err)
			}
		}
	}
	chain.Stop()
	fmt.Printf("Import done in %v.\n\n", time.Since(start))

	// Output pre-compaction stats mostly to see the import trashing
	showLeveldbStats(db)

	// Print the memory statistics used by the importing
	mem := new(runtime.MemStats)
	runtime.ReadMemStats(mem)

	fmt.Printf("Object memory: %.3f MB current, %.3f MB peak\n", float64(mem.Alloc)/1024/1024, float64(atomic.LoadUint64(&peakMemAlloc))/1024/1024)
	fmt.Printf("System memory: %.3f MB current, %.3f MB peak\n", float64(mem.Sys)/1024/1024, float64(atomic.LoadUint64(&peakMemSys))/1024/1024)
	fmt.Printf("Allocations:   %.3f million\n", float64(mem.Mallocs)/1000000)
	fmt.Printf("GC pause:      %v\n\n", time.Duration(mem.PauseTotalNs))

	if ctx.Bool(utils.NoCompactionFlag.Name) {
		return nil
	}

	// Compact the entire database to more accurately measure disk io and print the stats
	start = time.Now()
	fmt.Println("Compacting entire database...")
	if err := db.Compact(nil, nil); err != nil {
		utils.Fatalf("Compaction failed: %v", err)
	}
	fmt.Printf("Compaction done in %v.\n\n", time.Since(start))

	showLeveldbStats(db)
	return importErr
}

func exportChain(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		utils.Fatalf("This command requires an argument.")
	}

	stack, _, _ := makeFullNode(ctx)
	defer stack.Close()

	chain, db := utils.MakeChain(ctx, stack, true)
	defer db.Close()
	start := time.Now()

	var err error
	fp := ctx.Args().First()
	if ctx.Args().Len() < 3 {
		err = utils.ExportChain(chain, fp)
	} else {
		// This can be improved to allow for numbers larger than 9223372036854775807
		first, ferr := strconv.ParseInt(ctx.Args().Get(1), 10, 64)
		last, lerr := strconv.ParseInt(ctx.Args().Get(2), 10, 64)
		if ferr != nil || lerr != nil {
			utils.Fatalf("Export error in parsing parameters: block number not an integer\n")
		}
		if first < 0 || last < 0 {
			utils.Fatalf("Export error: block number must be greater than 0\n")
		}
		err = utils.ExportAppendChain(chain, fp, uint64(first), uint64(last))
	}

	if err != nil {
		utils.Fatalf("Export error: %v\n", err)
	}
	fmt.Printf("Export done in %v\n", time.Since(start))
	return nil
}

// importPreimages imports preimage data from the specified file.
func importPreimages(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		utils.Fatalf("This command requires an argument.")
	}

	stack, _, _ := makeFullNode(ctx)
	defer stack.Close()

	db := utils.MakeChainDatabase(ctx, stack, false)
	defer db.Close()

	start := time.Now()
	if err := utils.ImportPreimages(db, ctx.Args().First()); err != nil {
		utils.Fatalf("Export error: %v\n", err)
	}
	fmt.Printf("Export done in %v\n", time.Since(start))
	return nil
}

// exportPreimages dumps the preimage data to specified json file in streaming way.
func exportPreimages(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		utils.Fatalf("This command requires an argument.")
	}
	stack, _, _ := makeFullNode(ctx)
	defer stack.Close()

	db := utils.MakeChainDatabase(ctx, stack, true)
	defer db.Close()

	start := time.Now()
	if err := utils.ExportPreimages(db, ctx.Args().First()); err != nil {
		utils.Fatalf("Export error: %v\n", err)
	}
	fmt.Printf("Export done in %v\n", time.Since(start))
	return nil
}

func dump(ctx *cli.Context) error {
	stack, _, _ := makeFullNode(ctx)
	defer stack.Close()

	chain, chainDb := utils.MakeChain(ctx, stack, true)
	defer chainDb.Close()

	for _, arg := range ctx.Args().Slice() {
		var block *types.Block
		if hashish(arg) {
			block = chain.GetBlockByHash(common.HexToHash(arg))
		} else {
			num, _ := strconv.Atoi(arg)
			block = chain.GetBlockByNumber(uint64(num))
		}
		if block == nil {
			fmt.Println("{}")
			utils.Fatalf("block not found")
		} else {
			state, err := state.New(block.Root(), state.NewDatabase(chainDb))
			if err != nil {
				utils.Fatalf("could not create new state: %v", err)
			}
			fmt.Printf("%s\n", state.Dump())
		}
	}
	return nil
}

// hashish returns true for strings that look like hashes.
func hashish(x string) bool {
	_, err := strconv.Atoi(x)
	return err != nil
}
