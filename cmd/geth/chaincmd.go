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

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/syndtr/goleveldb/leveldb/util"
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
	pruningCommand = cli.Command{
		Action: pruneDB,
		Name:   "prune",
		Usage:  "Prunes database of old state information",
	}
)

func pruneDB(ctx *cli.Context) {
	chainDb := utils.MakeChainDatabase(ctx)
	defer chainDb.Close()

	// create a new temporary database to which we'll copy
	// the required state information. This DB will be
	// removed once finished.
	const pruningDB = "data_pruning_process"
	fresh := utils.MustOpenDatabase(ctx, pruningDB)
	defer func() {
		fresh.Close()
		os.RemoveAll(filepath.Join(utils.MustMakeDataDir(ctx), pruningDB))
	}()

	/*
		if db, ok := chainDb.(*ethdb.LDBDatabase); ok {
			iter := db.NewIterator()
			for iter.Next() {
				key := iter.Key()
				fmt.Printf("(%-4d) %-6d %x\n", len(key), len(iter.Value()), key)
				//db.Delete(key)
			}
			iter.Release()
		}
		return
	*/

	tbegin := time.Now()
	glog.V(logger.Info).Infoln("Starting pruning process (this may take a while)")

	// Fetch the current head block on which we'll determine the
	// "required state information" (current - 256 blocks ago)
	headHash := core.GetHeadBlockHash(chainDb)
	if (headHash == common.Hash{}) {
		utils.Fatalf("pruning: no HEAD block found")
	}
	headBlock := core.GetBlock(chainDb, headHash)

	glog.V(logger.Info).Infoln("Copying relevant state data")

	// Fetch each block and start the copying process. The copying process will use the
	// state iterator to check whether it needs to copy over state nodes, if a node is
	// missing it will fetch it from the database and puts it in our new fresh database.
	for blockno := headBlock.NumberU64() - 1500; blockno <= headBlock.NumberU64(); blockno++ {
		numhash := 0

		// fetch the block for the state root, this is the start of
		// the copying process
		hash := core.GetCanonicalHash(chainDb, blockno)
		if (hash == common.Hash{}) {
			utils.Fatalf("pruning: unable to find block %d for pruning session", blockno)
		}
		stateRoot := core.GetBlock(chainDb, hash).Root()

		// create a new sync state for the copying process
		sync := state.NewStateSync(stateRoot, fresh)
		// find missing nodes in batches of 256 and copy over the data.
		for missing := sync.Missing(256); len(missing) > 0; missing = sync.Missing(256) {
			syncRes := make([]trie.SyncResult, len(missing))
			for i, hash := range missing {
				node, _ := chainDb.Get(hash[:])

				syncRes[i] = trie.SyncResult{Hash: hash, Data: node}
			}
			sync.Process(syncRes)
			numhash += len(syncRes)
		}
	}

	glog.V(logger.Info).Infoln("Pruning old state data")

	var (
		tmp   = fresh.(*ethdb.LDBDatabase)
		chain = chainDb.(*ethdb.LDBDatabase)
	)
	// Deletion process. The deletion process will fetch the secure-keys
	// and deletes them from the database.
	presec := []byte("secure-key-")
	// Unfortunately our current implementation of the DB wrapper
	// does not allow us to create iterators whith filtering
	// options and therefor we fetch the leveldb instance and
	// create a iterator from there instead.
	{
		iter := chain.LDB().NewIterator(util.BytesPrefix(presec), nil)
		for iter.Next() {
			// delete each key
			chain.Delete(iter.Key())
		}
		iter.Release()
	}

	// delete old state information. this process is not yet finished
	// this process curerntly checks whether the key is 32 and **assumes**
	// it's state information.
	{
		iter := chain.NewIterator()
		for iter.Next() {
			key := iter.Key()

			has, _ := chain.LDB().Has(append(key, 0x01), nil)
			if len(key) == 32 && !has {
				chain.Delete(key)
			}
		}
		iter.Release()
	}

	glog.V(logger.Info).Infoln("Compacting database")
	// Compact the database again
	chain.LDB().CompactRange(util.Range{nil, nil})

	glog.V(logger.Info).Infoln("Moving new state data to database")

	// Copy the relevant state information back to the state db
	{
		batch := chain.NewBatch()
		iter := tmp.NewIterator()
		for iter.Next() {
			batch.Put(iter.Key(), iter.Value())
		}
		iter.Release()
		batch.Write()
	}

	glog.V(logger.Info).Infoln("Pruning process completed in:", time.Since(tbegin))
}

func importChain(ctx *cli.Context) {
	if len(ctx.Args()) != 1 {
		utils.Fatalf("This command requires an argument.")
	}
	chain, chainDb := utils.MakeChain(ctx)
	start := time.Now()
	err := utils.ImportChain(chain, ctx.Args().First())
	chainDb.Close()
	if err != nil {
		utils.Fatalf("Import error: %v", err)
	}
	fmt.Printf("Import done in %v", time.Since(start))
}

func exportChain(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		utils.Fatalf("This command requires an argument.")
	}
	chain, _ := utils.MakeChain(ctx)
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
}

func removeDB(ctx *cli.Context) {
	confirm, err := utils.Stdin.ConfirmPrompt("Remove local database?")
	if err != nil {
		utils.Fatalf("%v", err)
	}

	if confirm {
		fmt.Println("Removing chaindata...")
		start := time.Now()

		os.RemoveAll(filepath.Join(ctx.GlobalString(utils.DataDirFlag.Name), "chaindata"))

		fmt.Printf("Removed in %v\n", time.Since(start))
	} else {
		fmt.Println("Operation aborted")
	}
}

func upgradeDB(ctx *cli.Context) {
	glog.Infoln("Upgrading blockchain database")

	chain, chainDb := utils.MakeChain(ctx)
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
	os.RemoveAll(filepath.Join(ctx.GlobalString(utils.DataDirFlag.Name), "chaindata"))

	// Import the chain file.
	chain, chainDb = utils.MakeChain(ctx)
	core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	err := utils.ImportChain(chain, exportFile)
	chainDb.Close()
	if err != nil {
		utils.Fatalf("Import error %v (a backup is made in %s, use the import command to import it)", err, exportFile)
	} else {
		os.Remove(exportFile)
		glog.Infoln("Import finished")
	}
}

func dump(ctx *cli.Context) {
	chain, chainDb := utils.MakeChain(ctx)
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
			state, err := state.New(block.Root(), chainDb)
			if err != nil {
				utils.Fatalf("could not create new state: %v", err)
				return
			}
			fmt.Printf("%s\n", state.Dump())
		}
	}
	chainDb.Close()
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
