// Snapshot related commands

package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/pruner"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/prometheus/tsdb/fileutil"

	"github.com/mitchellh/cli"
)

var errPbssNotSupported = errors.New("ancient block pruning is not supporeted on path based storage scheme")

// SnapshotCommand is the command to group the snapshot commands
type SnapshotCommand struct {
	UI cli.Ui
}

// MarkDown implements cli.MarkDown interface
func (a *SnapshotCommand) MarkDown() string {
	items := []string{
		"# snapshot",
		"The ```snapshot``` command groups snapshot related actions:",
		"- [```snapshot prune-state```](./snapshot_prune-state.md): Prune state databases at the given datadir location.",
		"- [```snapshot prune-block```](./snapshot_prune-block.md): Prune ancient chaindata at the given datadir location.",
		"- [```snapshot inspect-ancient-db```](./snapshot_inspect-ancient-db.md): Inspect few fields in ancient datastore.",
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *SnapshotCommand) Help() string {
	return `Usage: bor snapshot <subcommand>

  This command groups snapshot related actions.

  Prune the state trie:

    $ bor snapshot prune-state

  Prune the ancient data:

    $ bor snapshot prune-block

  Inspect ancient DB pruning related fields:

    $ bor snapshot inspect-ancient-db`
}

// Synopsis implements the cli.Command interface
func (c *SnapshotCommand) Synopsis() string {
	return "Snapshot related commands"
}

// Run implements the cli.Command interface
func (c *SnapshotCommand) Run(args []string) int {
	return cli.RunResultHelp
}

type PruneStateCommand struct {
	*Meta

	datadirAncient   string
	cache            uint64
	cacheTrie        uint64
	cacheTrieJournal string
	bloomfilterSize  uint64
}

// MarkDown implements cli.MarkDown interface
func (c *PruneStateCommand) MarkDown() string {
	items := []string{
		`# Prune state`,
		`The ` + "```" + "bor snapshot prune-state" + "```" + ` command will prune historical state data
with the help of the state snapshot. All trie nodes and contract codes that do not belong to the
specified	version state will be deleted from the database. After pruning, only two version states
are available: genesis and the specific one.`,
		c.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *PruneStateCommand) Help() string {
	return `Usage: bor snapshot prune-state <datadir>

  This command will prune state databases at the given datadir location` + c.Flags().Help()
}

// Synopsis implements the cli.Command interface
func (c *PruneStateCommand) Synopsis() string {
	return "Prune state databases"
}

// Flags: datadir, datadir.ancient, cache.trie.journal, bloomfilter.size
func (c *PruneStateCommand) Flags() *flagset.Flagset {
	flags := c.NewFlagSet("prune-state")

	flags.StringFlag(&flagset.StringFlag{
		Name:    "datadir.ancient",
		Value:   &c.datadirAncient,
		Usage:   "Path of the ancient data directory to store information",
		Default: "",
	})

	flags.Uint64Flag(&flagset.Uint64Flag{
		Name:    "cache",
		Usage:   "Megabytes of memory allocated to internal caching",
		Value:   &c.cache,
		Default: 1024.0,
		Group:   "Cache",
	})

	flags.Uint64Flag(&flagset.Uint64Flag{
		Name:    "cache.trie",
		Usage:   "Percentage of cache memory allowance to use for trie caching",
		Value:   &c.cacheTrie,
		Default: 25,
		Group:   "Cache",
	})

	flags.StringFlag(&flagset.StringFlag{
		Name:    "cache.trie.journal",
		Value:   &c.cacheTrieJournal,
		Usage:   "Path of the trie journal directory to store information",
		Default: trieCacheJournalPath,
		Group:   "Cache",
	})

	flags.Uint64Flag(&flagset.Uint64Flag{
		Name:    "bloomfilter.size",
		Value:   &c.bloomfilterSize,
		Usage:   "Size of the bloom filter",
		Default: 2048,
	})

	return flags
}

// Run implements the cli.Command interface
func (c *PruneStateCommand) Run(args []string) int {
	flags := c.Flags()

	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	datadir := c.dataDir
	if datadir == "" {
		c.UI.Error("datadir is required")
		return 1
	}

	// Create the node
	node, err := node.New(&node.Config{
		DataDir: datadir,
	})

	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	dbHandles, err := server.MakeDatabaseHandles(0)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	chaindb, err := node.OpenDatabaseWithFreezer(chaindataPath, int(c.cache), dbHandles, c.datadirAncient, "", false, false, false)

	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	prunerconfig := pruner.Config{
		Datadir:   node.ResolvePath(""),
		BloomSize: c.bloomfilterSize,
	}

	pruner, err := pruner.NewPruner(chaindb, prunerconfig)
	if err != nil {
		log.Error("Failed to open snapshot tree", "err", err)
		return 1
	}

	if err = pruner.Prune(common.Hash{}); err != nil {
		log.Error("Failed to prune state", "err", err)
		return 1
	}

	return 0
}

type PruneBlockCommand struct {
	*Meta

	datadirAncient       string
	cache                int
	blockAmountReserved  uint64
	triesInMemory        int
	checkSnapshotWithMPT bool
}

// MarkDown implements cli.MarkDown interface
func (c *PruneBlockCommand) MarkDown() string {
	items := []string{
		"# Prune ancient blockchain",
		"The ```bor snapshot prune-block``` command will prune historical blockchain data stored in the ancientdb. The amount of blocks expected for remaining after prune can be specified via `block-amount-reserved` in this command, will prune and only remain the specified amount of old block data in ancientdb.",
		`
The brief workflow as below:

1. backup the the number of specified number of blocks backward in original ancientdb into new ancient_backup,
2. then delete the original ancientdb dir and rename the ancient_backup to original one for replacement,
3. finally assemble the statedb and new ancientdb together.

The purpose of doing it is because the block data will be moved into the ancient store when it becomes old enough (exceed the Threshold 90000), the disk usage will be very large over time, and is occupied mainly by ancientdb, so it's very necessary to do block data pruning, this feature will handle it.

Warning: This command only works with hash based storage scheme and doesn't work with path based storage scheme.`,
		c.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *PruneBlockCommand) Help() string {
	return `Usage: bor snapshot prune-block <datadir>

  This command will prune ancient blockchain data at the given datadir location` + c.Flags().Help()
}

// Synopsis implements the cli.Command interface
func (c *PruneBlockCommand) Synopsis() string {
	return "Prune ancient blockchain data"
}

// Flags: datadir, datadir.ancient, cache.trie.journal, bloomfilter.size
func (c *PruneBlockCommand) Flags() *flagset.Flagset {
	flags := c.NewFlagSet("prune-block")

	flags.StringFlag(&flagset.StringFlag{
		Name:    "datadir.ancient",
		Value:   &c.datadirAncient,
		Usage:   "Path of the old ancient data directory",
		Default: "",
	})

	flags.IntFlag(&flagset.IntFlag{
		Name:    "cache",
		Usage:   "Megabytes of memory allocated to internal caching",
		Value:   &c.cache,
		Default: 1024,
		Group:   "Cache",
	})
	flags.Uint64Flag(&flagset.Uint64Flag{
		Name:    "block-amount-reserved",
		Usage:   "Sets the expected reserved number of blocks for offline block prune",
		Value:   &c.blockAmountReserved,
		Default: 1024,
	})

	flags.IntFlag(&flagset.IntFlag{
		Name:    "cache.triesinmemory",
		Usage:   "Number of block states (tries) to keep in memory (default = 128)",
		Value:   &c.triesInMemory,
		Default: 128,
	})

	flags.BoolFlag(&flagset.BoolFlag{
		Name:  "check-snapshot-with-mpt",
		Value: &c.checkSnapshotWithMPT,
		Usage: "Enable checking between snapshot and MPT",
	})

	return flags
}

// Run implements the cli.Command interface
func (c *PruneBlockCommand) Run(args []string) int {
	flags := c.Flags()

	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	datadir := c.dataDir
	if datadir == "" {
		c.UI.Error("datadir is required")
		return 1
	}

	// Create the node
	node, err := node.New(&node.Config{
		DataDir: datadir,
	})

	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	defer node.Close()

	dbHandles, err := server.MakeDatabaseHandles(0)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	err = c.validateAgainstSnapshot(node, dbHandles)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	err = c.pruneBlock(node, dbHandles)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	return 0
}

// validateAgainstSnapshot checks if the MPT data and snapshot data matches with each other or not
func (c *PruneBlockCommand) validateAgainstSnapshot(stack *node.Node, dbHandles int) error {
	chaindb, err := stack.OpenDatabaseWithFreezer(chaindataPath, c.cache, dbHandles, c.datadirAncient, "", false, true, false)
	if err != nil {
		return fmt.Errorf("failed to accessdb %v", err)
	}
	defer chaindb.Close()

	// Check if we're using hash based scheme and not path based
	if rawdb.ReadStateScheme(chaindb) != rawdb.HashScheme {
		return errPbssNotSupported
	}

	if !c.checkSnapshotWithMPT {
		return nil
	}

	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		return errors.New("failed to load head block")
	}
	headHeader := headBlock.Header()

	snapconfig := snapshot.Config{
		CacheSize:  256,
		Recovery:   false,
		NoBuild:    true,
		AsyncBuild: false,
	}

	// Make sure the MPT and snapshot matches before pruning, otherwise the node can not start.
	snaptree, err := snapshot.New(snapconfig, chaindb, trie.NewDatabase(chaindb, trie.HashDefaults), headBlock.Root())
	if err != nil {
		log.Error("Unable to load snapshot", "err", err)
		return err // The relevant snapshot(s) might not exist
	}

	// Use the HEAD-(n-1) as the target root. The reason for picking it is:
	// - in most of the normal cases, the related state is available
	// - the probability of this layer being reorg is very low

	// Note that here (n) refers to `c.triesInMemory` which is a
	// configurable parameter.
	// Retrieve all snapshot layers from the current HEAD.
	// In theory there are n difflayers + 1 disk layer present,
	// so n diff layers are expected to be returned.
	layers := snaptree.Snapshots(headHeader.Root, c.triesInMemory, true)
	if len(layers) != c.triesInMemory {
		// Reject if the accumulated diff layers are less than n. It
		// means in most of normal cases, there is no associated state
		// with bottom-most diff layer.
		log.Error("snapshot layers != TriesInMemory", "err", err)
		return fmt.Errorf("snapshot not old enough yet: need %d more blocks", c.triesInMemory-len(layers))
	}
	// Use the bottom-most diff layer as the target
	targetRoot := layers[len(layers)-1].Root()

	// Ensure the root is really present. The weak assumption
	// is the presence of root can indicate the presence of the
	// entire trie.
	if blob := rawdb.ReadTrieNode(chaindb, common.Hash{}, nil, targetRoot, rawdb.HashScheme); len(blob) == 0 {
		// The special case is for clique based networks(rinkeby, goerli
		// and some other private networks), it's possible that two
		// consecutive blocks will have same root. In this case snapshot
		// difflayer won't be created. So HEAD-(n-1) may not paired with
		// head-(n-1) layer. Instead the paired layer is higher than the
		// bottom-most diff layer. Try to find the bottom-most snapshot
		// layer with state available.
		//
		// Note HEAD is ignored. Usually there is the associated
		// state available, but we don't want to use the topmost state
		// as the pruning target.
		for i := len(layers) - 2; i >= 1; i-- {
			if blob := rawdb.ReadTrieNode(chaindb, common.Hash{}, nil, layers[i].Root(), rawdb.HashScheme); len(blob) != 0 {
				targetRoot = layers[i].Root()
				log.Info("Selecting middle-layer as the pruning target", "root", targetRoot, "depth", i)
				return nil
			}
		}

		if blob := rawdb.ReadTrieNode(chaindb, common.Hash{}, nil, snaptree.DiskRoot(), rawdb.HashScheme); len(blob) != 0 {
			targetRoot = snaptree.DiskRoot()
			log.Info("Selecting disk-layer as the pruning target", "root", targetRoot)
			return nil
		}

		if len(layers) > 0 {
			log.Error("no snapshot paired state")
			return errors.New("no snapshot paired state")
		}

		return fmt.Errorf("associated state[%x] is not present", targetRoot)
	} else {
		if len(layers) > 0 {
			log.Info("Selecting bottom-most difflayer as the pruning target", "root", targetRoot, "height", headHeader.Number.Uint64()-uint64(len(layers)-1))
		} else {
			log.Info("Selecting user-specified state as the pruning target", "root", targetRoot)
		}
	}

	return nil
}

// checkDeletePermissions checks if the user has the permission to
// delete the given `path`.
func checkDeletePermissions(path string) (bool, error) {
	dirInfo, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	// Check if the user has write and execute permissions on the directory
	if dirInfo.Mode().Perm()&(0200|0100) == (0200 | 0100) {
		// Also check if the parent directory has write permissions because delete needs them
		parentDir := filepath.Dir(path)
		parentDirInfo, err := os.Stat(parentDir)
		if err != nil {
			return false, err
		}
		if parentDirInfo.Mode().Perm()&0200 != 0 {
			return true, nil
		}
	}
	return false, nil
}

// pruneBlock is the entry point for the ancient pruning process. Based on the user specified
// params, it will prune the ancient data. It also handles the case where the pruning process
// was interrupted earlier.
func (c *PruneBlockCommand) pruneBlock(stack *node.Node, fdHandles int) error {
	name := "chaindata"

	oldAncientPath := c.datadirAncient

	switch {
	case oldAncientPath == "":
		oldAncientPath = filepath.Join(stack.ResolvePath(name), "ancient")
	case !filepath.IsAbs(oldAncientPath):
		oldAncientPath = stack.ResolvePath(oldAncientPath)
	}

	path, _ := filepath.Split(oldAncientPath)
	if path == "" {
		return errors.New("prune failed, did not specify the AncientPath")
	}

	newAncientPath := filepath.Join(path, "ancient_back")

	// Check if we have delete permissions on the ancient datadir path beforehand
	allow, err := checkDeletePermissions(oldAncientPath)
	if err != nil {
		log.Error("Failed to check delete permissions for ancient datadir", "path", oldAncientPath, "err", err)
		return err
	}
	if !allow {
		return fmt.Errorf("user doesn't have delete permissions on ancient datadir: %s", oldAncientPath)
	}

	blockpruner := pruner.NewBlockPruner(stack, oldAncientPath, newAncientPath, c.blockAmountReserved)

	lock, exist, err := fileutil.Flock(filepath.Join(oldAncientPath, "PRUNEFLOCK"))
	if err != nil {
		log.Error("file lock error", "err", err)
		return err
	}

	if exist {
		defer func() {
			_ = lock.Release()
		}()
		log.Info("File lock existed, waiting for prune recovery and continue", "err", err)

		if err := blockpruner.RecoverInterruption("chaindata", c.cache, fdHandles, "", false); err != nil {
			log.Error("Pruning failed", "err", err)
			return err
		}

		log.Info("Block prune successfully")

		return nil
	}

	if _, err := os.Stat(newAncientPath); err == nil {
		// No file lock found for old ancientDB but new ancientDB exsisted, indicating the geth was interrupted
		// after old ancientDB removal, this happened after backup successfully, so just rename the new ancientDB
		if err := blockpruner.AncientDbReplacer(); err != nil {
			return err
		}

		log.Info("Block prune successfully")

		return nil
	}

	if err := blockpruner.BlockPruneBackup(name, c.cache, fdHandles, "", false, false); err != nil {
		return err
	}

	log.Info("Block backup successfully")

	// After backup successfully, rename the new ancientdb name to the original one, and delete the old ancientdb
	if err := blockpruner.AncientDbReplacer(); err != nil {
		return err
	}

	if err = lock.Release(); err != nil {
		log.Error("Unable to release lock on file", "err", err)

		return err
	}

	log.Info("Block prune successfully")

	return nil
}

type InspectAncientDbCommand struct {
	*Meta

	datadirAncient string
}

// MarkDown implements cli.MarkDown interface
func (c *InspectAncientDbCommand) MarkDown() string {
	items := []string{
		"# Inspect ancient DB for block pruning",
		"The ```bor snapshot inspect-ancient-db``` command will inspect few fields in the ancient datastore using the given datadir location.",
		`
This command prints the following information which is useful for block-pruning rounds:

1. Offset / Start block number (from kvDB).
2. Amount of items in the ancientdb.
3. Last block number written in ancientdb.
`,
		c.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *InspectAncientDbCommand) Help() string {
	return `Usage: bor snapshot inspect-ancient-db <datadir>

  This command will inspect few fields in the ancient datastore using the given datadir location` + c.Flags().Help()
}

// Synopsis implements the cli.Command interface
func (c *InspectAncientDbCommand) Synopsis() string {
	return "Inspect fields in the ancient blockchain data"
}

// Flags: datadir, datadir.ancient, cache.trie.journal, bloomfilter.size
func (c *InspectAncientDbCommand) Flags() *flagset.Flagset {
	flags := c.NewFlagSet("inspect-ancient-db")

	flags.StringFlag(&flagset.StringFlag{
		Name:    "datadir.ancient",
		Value:   &c.datadirAncient,
		Usage:   "Path of the old ancient data directory",
		Default: "",
	})

	return flags
}

// Run implements the cli.Command interface
func (c *InspectAncientDbCommand) Run(args []string) int {
	flags := c.Flags()

	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	datadir := c.dataDir
	if datadir == "" {
		c.UI.Error("datadir is required")
		return 1
	}

	// Create the node
	node, err := node.New(&node.Config{
		DataDir: datadir,
	})

	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}
	defer node.Close()

	dbHandles, err := server.MakeDatabaseHandles(0)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	err = c.inspectAncientDb(node, dbHandles)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	return 0
}

func (c *InspectAncientDbCommand) inspectAncientDb(stack *node.Node, dbHandles int) error {
	chaindb, err := stack.OpenDatabaseWithFreezer(chaindataPath, 1024, dbHandles, c.datadirAncient, "", false, true, false)
	if err != nil {
		return err
	}
	defer chaindb.Close()

	return rawdb.AncientInspect(chaindb)
}
