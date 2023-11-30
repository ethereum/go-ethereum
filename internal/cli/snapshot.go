// Snapshot related commands

package cli

import (
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/pruner"
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"

	"github.com/mitchellh/cli"
)

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
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *SnapshotCommand) Help() string {
	return `Usage: bor snapshot <subcommand>

  This command groups snapshot related actions.

  Prune the state trie:

    $ bor snapshot prune-state`
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
		"# Prune state",
		"The ```bor snapshot prune-state``` command will prune historical state data with the help of the state snapshot. All trie nodes and contract codes that do not belong to the specified	version state will be deleted from the database. After pruning, only two version states are available: genesis and the specific one.",
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

	chaindb, err := node.OpenDatabaseWithFreezer(chaindataPath, int(c.cache), dbHandles, c.datadirAncient, "", false, rawdb.ExtraDBConfig{})

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
