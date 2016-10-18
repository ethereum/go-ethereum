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
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gopkg.in/urfave/cli.v1"
)

var (
	importCommand = cli.Command{
		Action: importChain,
		Name:   "import",
		Usage:  `import a blockchain file`,
	}
	exportCommand = cli.Command{
		Action: exportChain,
		Name:   "export",
		Usage:  `export blockchain into file`,
		Description: `
Requires a first argument of the file to write to.
Optional second and third arguments control the first and
last block to write. In this mode, the file will be appended
if already existing.
		`,
	}
	upgradedbCommand = cli.Command{
		Action: upgradeDB,
		Name:   "upgradedb",
		Usage:  "upgrade chainblock database",
	}
	removedbCommand = cli.Command{
		Action: removeDB,
		Name:   "removedb",
		Usage:  "Remove blockchain and state databases",
	}
	dumpCommand = cli.Command{
		Action: dump,
		Name:   "dump",
		Usage:  `dump a specific block from storage`,
		Description: `
The arguments are interpreted as block numbers or hashes.
Use "ethereum dump 0" to dump the genesis block.
`,
	}
)

func importChain(ctx *cli.Context) error {
	if len(ctx.Args()) != 1 {
		utils.Fatalf("This command requires an argument.")
	}
	if ctx.GlobalBool(utils.TestNetFlag.Name) {
		state.StartingNonce = 1048576 // (2**20)
	}
	stack := makeFullNode(ctx)
	chain, chainDb := utils.MakeChain(ctx, stack)
	defer chainDb.Close()

	// Import the chain
	start := time.Now()
	if err := utils.ImportChain(chain, ctx.Args().First()); err != nil {
		utils.Fatalf("Import error: %v", err)
	}
	fmt.Printf("Import done in %v, compacting...\n", time.Since(start))

	// Compact the entire database to more accurately measure disk io and print the stats
	if db, ok := chainDb.(*ethdb.LDBDatabase); ok {
		start = time.Now()
		if err := db.LDB().CompactRange(util.Range{}); err != nil {
			utils.Fatalf("Compaction failed: %v", err)
		}
		fmt.Printf("Compaction done in %v.\n", time.Since(start))

		stats, err := db.LDB().GetProperty("leveldb.stats")
		if err != nil {
			utils.Fatalf("Failed to read database stats: %v", err)
		}
		fmt.Println(stats)
	}
	return nil
}

func exportChain(ctx *cli.Context) error {
	if len(ctx.Args()) < 1 {
		utils.Fatalf("This command requires an argument.")
	}
	stack := makeFullNode(ctx)
	chain, _ := utils.MakeChain(ctx, stack)
	start := time.Now()

	var err error
	fp := ctx.Args().First()
	if len(ctx.Args()) < 3 {
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
	fmt.Printf("Export done in %v", time.Since(start))
	return nil
}

func removeDB(ctx *cli.Context) error {
	stack := utils.MakeNode(ctx, clientIdentifier, gitCommit)
	dbdir := stack.ResolvePath("chaindata")
	if !common.FileExist(dbdir) {
		fmt.Println(dbdir, "does not exist")
		return nil
	}

	fmt.Println(dbdir)
	confirm, err := console.Stdin.PromptConfirm("Remove this database?")
	switch {
	case err != nil:
		utils.Fatalf("%v", err)
	case !confirm:
		fmt.Println("Operation aborted")
	default:
		fmt.Println("Removing...")
		start := time.Now()
		os.RemoveAll(dbdir)
		fmt.Printf("Removed in %v\n", time.Since(start))
	}
	return nil
}

func upgradeDB(ctx *cli.Context) error {
	glog.Infoln("Upgrading blockchain database")

	stack := utils.MakeNode(ctx, clientIdentifier, gitCommit)
	chain, chainDb := utils.MakeChain(ctx, stack)
	bcVersion := core.GetBlockChainVersion(chainDb)
	if bcVersion == 0 {
		bcVersion = core.BlockChainVersion
	}

	// Export the current chain.
	filename := fmt.Sprintf("blockchain_%d_%s.chain", bcVersion, time.Now().Format("20060102_150405"))
	exportFile := filepath.Join(ctx.GlobalString(utils.DataDirFlag.Name), filename)
	if err := utils.ExportChain(chain, exportFile); err != nil {
		utils.Fatalf("Unable to export chain for reimport %s", err)
	}
	chainDb.Close()
	if dir := dbDirectory(chainDb); dir != "" {
		os.RemoveAll(dir)
	}

	// Import the chain file.
	chain, chainDb = utils.MakeChain(ctx, stack)
	core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	err := utils.ImportChain(chain, exportFile)
	chainDb.Close()
	if err != nil {
		utils.Fatalf("Import error %v (a backup is made in %s, use the import command to import it)", err, exportFile)
	} else {
		os.Remove(exportFile)
		glog.Infoln("Import finished")
	}
	return nil
}

func dbDirectory(db ethdb.Database) string {
	ldb, ok := db.(*ethdb.LDBDatabase)
	if !ok {
		return ""
	}
	return ldb.Path()
}

func dump(ctx *cli.Context) error {
	stack := makeFullNode(ctx)
	chain, chainDb := utils.MakeChain(ctx, stack)
	for _, arg := range ctx.Args() {
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
			state, err := state.New(block.Root(), chainDb)
			if err != nil {
				utils.Fatalf("could not create new state: %v", err)
			}
			fmt.Printf("%s\n", state.Dump())
		}
	}
	chainDb.Close()
	return nil
}

// hashish returns true for strings that look like hashes.
func hashish(x string) bool {
	_, err := strconv.Atoi(x)
	return err != nil
}

func closeAll(dbs ...ethdb.Database) {
	for _, db := range dbs {
		db.Close()
	}
}
