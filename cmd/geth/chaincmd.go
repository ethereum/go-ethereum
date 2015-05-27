package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/logger/glog"
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

func importChain(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		utils.Fatalf("This command requires an argument.")
	}
	chain, blockDB, stateDB, extraDB := utils.GetChain(ctx)
	start := time.Now()
	if err := utils.ImportChain(chain, ctx.Args().First()); err != nil {
		utils.Fatalf("Import error: %v\n", err)
	}
	flushAll(blockDB, stateDB, extraDB)
	fmt.Printf("Import done in %v", time.Since(start))
}

func exportChain(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		utils.Fatalf("This command requires an argument.")
	}
	chain, _, _, _ := utils.GetChain(ctx)
	start := time.Now()
	if err := utils.ExportChain(chain, ctx.Args().First()); err != nil {
		utils.Fatalf("Export error: %v\n", err)
	}
	fmt.Printf("Export done in %v", time.Since(start))
}

func removeDB(ctx *cli.Context) {
	confirm, err := utils.PromptConfirm("Remove local databases?")
	if err != nil {
		utils.Fatalf("%v", err)
	}

	if confirm {
		fmt.Println("Removing chain and state databases...")
		start := time.Now()

		os.RemoveAll(filepath.Join(ctx.GlobalString(utils.DataDirFlag.Name), "blockchain"))
		os.RemoveAll(filepath.Join(ctx.GlobalString(utils.DataDirFlag.Name), "state"))

		fmt.Printf("Removed in %v\n", time.Since(start))
	} else {
		fmt.Println("Operation aborted")
	}
}

func upgradeDB(ctx *cli.Context) {
	glog.Infoln("Upgrading blockchain database")

	chain, blockDB, stateDB, extraDB := utils.GetChain(ctx)
	v, _ := blockDB.Get([]byte("BlockchainVersion"))
	bcVersion := int(common.NewValue(v).Uint())
	if bcVersion == 0 {
		bcVersion = core.BlockChainVersion
	}

	// Export the current chain.
	filename := fmt.Sprintf("blockchain_%d_%s.chain", bcVersion, time.Now().Format("20060102_150405"))
	exportFile := filepath.Join(ctx.GlobalString(utils.DataDirFlag.Name), filename)
	if err := utils.ExportChain(chain, exportFile); err != nil {
		utils.Fatalf("Unable to export chain for reimport %s\n", err)
	}
	flushAll(blockDB, stateDB, extraDB)
	os.RemoveAll(filepath.Join(ctx.GlobalString(utils.DataDirFlag.Name), "blockchain"))
	os.RemoveAll(filepath.Join(ctx.GlobalString(utils.DataDirFlag.Name), "state"))

	// Import the chain file.
	chain, blockDB, stateDB, extraDB = utils.GetChain(ctx)
	blockDB.Put([]byte("BlockchainVersion"), common.NewValue(core.BlockChainVersion).Bytes())
	err := utils.ImportChain(chain, exportFile)
	flushAll(blockDB, stateDB, extraDB)
	if err != nil {
		utils.Fatalf("Import error %v (a backup is made in %s, use the import command to import it)\n", err, exportFile)
	} else {
		os.Remove(exportFile)
		glog.Infoln("Import finished")
	}
}

func dump(ctx *cli.Context) {
	chain, _, stateDB, _ := utils.GetChain(ctx)
	for _, arg := range ctx.Args() {
		var block *types.Block
		if hashish(arg) {
			block = chain.GetBlock(common.HexToHash(arg))
		} else {
			num, _ := strconv.Atoi(arg)
			block = chain.GetBlockByNumber(uint64(num))
		}
		if block == nil {
			fmt.Println("{}")
			utils.Fatalf("block not found")
		} else {
			state := state.New(block.Root(), stateDB)
			fmt.Printf("%s\n", state.Dump())
		}
	}
}

// hashish returns true for strings that look like hashes.
func hashish(x string) bool {
	_, err := strconv.Atoi(x)
	return err != nil
}

func flushAll(dbs ...common.Database) {
	for _, db := range dbs {
		db.Flush()
		db.Close()
	}
}
