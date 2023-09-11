package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
	"github.com/ethereum/go-ethereum/internal/cli/server"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"

	"github.com/mitchellh/cli"
)

// RemoveDBCommand is for removing blockchain and state databases
type RemoveDBCommand struct {
	*Meta2

	datadir string
}

const (
	chaindataPath        string = "chaindata"
	ancientPath          string = "ancient"
	trieCacheJournalPath string = "triecache"
	lightchaindataPath   string = "lightchaindata"
)

// MarkDown implements cli.MarkDown interface
func (c *RemoveDBCommand) MarkDown() string {
	items := []string{
		"# RemoveDB",
		"The ```bor removedb``` command will remove the blockchain and state databases at the given datadir location",
		c.Flags().MarkDown(),
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *RemoveDBCommand) Help() string {
	return `Usage: bor removedb <datadir>

  This command will remove the blockchain and state databases at the given datadir location`
}

// Synopsis implements the cli.Command interface
func (c *RemoveDBCommand) Synopsis() string {
	return "Remove blockchain and state databases"
}

func (c *RemoveDBCommand) Flags() *flagset.Flagset {
	flags := c.NewFlagSet("removedb")

	flags.StringFlag(&flagset.StringFlag{
		Name:  "datadir",
		Value: &c.datadir,
		Usage: "Path of the data directory to store information",
	})

	return flags
}

// Run implements the cli.Command interface
func (c *RemoveDBCommand) Run(args []string) int {
	flags := c.Flags()

	// parse datadir
	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	datadir := c.datadir
	if datadir == "" {
		datadir = server.DefaultDataDir()
	}

	// create ethereum node config with just the datadir
	nodeCfg := &node.Config{DataDir: datadir}

	// Remove the full node state database
	path := nodeCfg.ResolvePath(chaindataPath)
	if common.FileExist(path) {
		confirmAndRemoveDB(c.UI, path, "full node state database")
	} else {
		log.Info("Full node state database missing", "path", path)
	}

	// Remove the full node ancient database
	// Note: The old cli used DatabaseFreezer path from config if provided explicitly
	// We don't have access to eth config and hence we assume it to be
	// under the "chaindata" folder.
	path = filepath.Join(nodeCfg.ResolvePath(chaindataPath), ancientPath)
	if common.FileExist(path) {
		confirmAndRemoveDB(c.UI, path, "full node ancient database")
	} else {
		log.Info("Full node ancient database missing", "path", path)
	}

	// Remove the light node database
	path = nodeCfg.ResolvePath(lightchaindataPath)
	if common.FileExist(path) {
		confirmAndRemoveDB(c.UI, path, "light node database")
	} else {
		log.Info("Light node database missing", "path", path)
	}

	return 0
}

// confirmAndRemoveDB prompts the user for a last confirmation and removes the
// folder if accepted.
func confirmAndRemoveDB(ui cli.Ui, database string, kind string) {
	for {
		confirm, err := ui.Ask(fmt.Sprintf("Remove %s (%s)? [y/n]", kind, database))

		switch {
		case err != nil:
			ui.Output(err.Error())
			return
		case confirm != "":
			switch strings.ToLower(confirm) {
			case "y":
				start := time.Now()
				err = filepath.Walk(database, func(path string, info os.FileInfo, err error) error {
					// If we're at the top level folder, recurse into
					if path == database {
						return nil
					}
					// Delete all the files, but not subfolders
					if !info.IsDir() {
						return os.Remove(path)
					}
					return filepath.SkipDir
				})

				if err != nil && err != filepath.SkipDir {
					ui.Output(err.Error())
				} else {
					log.Info("Database successfully deleted", "path", database, "elapsed", common.PrettyDuration(time.Since(start)))
				}

				return
			case "n":
				log.Info("Database deletion skipped", "path", database)
				return
			}
		}
	}
}
