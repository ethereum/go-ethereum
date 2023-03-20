package cli

import (
	"os"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/ethereum/go-ethereum/internal/cli/server"
)

// DumpconfigCommand is for exporting user provided flags into a config file
type DumpconfigCommand struct {
	*Meta2
}

// MarkDown implements cli.MarkDown interface
func (p *DumpconfigCommand) MarkDown() string {
	items := []string{
		"# Dumpconfig",
		"The ```bor dumpconfig <your-favourite-flags>``` command will export the user provided flags into a configuration file",
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (c *DumpconfigCommand) Help() string {
	return `Usage: bor dumpconfig <your-favourite-flags>

  This command will will export the user provided flags into a configuration file`
}

// Synopsis implements the cli.Command interface
func (c *DumpconfigCommand) Synopsis() string {
	return "Export configuration file"
}

// TODO: add flags for file location and format (toml, json, hcl) of the configuration file.

// Run implements the cli.Command interface
func (c *DumpconfigCommand) Run(args []string) int {
	// Initialize an empty command instance to get flags
	command := server.Command{}
	flags := command.Flags()

	if err := flags.Parse(args); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	userConfig := command.GetConfig()

	// convert the big.Int and time.Duration fields to their corresponding Raw fields
	userConfig.JsonRPC.RPCEVMTimeoutRaw = userConfig.JsonRPC.RPCEVMTimeout.String()
	userConfig.JsonRPC.HttpTimeout.ReadTimeoutRaw = userConfig.JsonRPC.HttpTimeout.ReadTimeout.String()
	userConfig.JsonRPC.HttpTimeout.WriteTimeoutRaw = userConfig.JsonRPC.HttpTimeout.WriteTimeout.String()
	userConfig.JsonRPC.HttpTimeout.IdleTimeoutRaw = userConfig.JsonRPC.HttpTimeout.IdleTimeout.String()
	userConfig.TxPool.RejournalRaw = userConfig.TxPool.Rejournal.String()
	userConfig.TxPool.LifeTimeRaw = userConfig.TxPool.LifeTime.String()
	userConfig.Sealer.GasPriceRaw = userConfig.Sealer.GasPrice.String()
	userConfig.Sealer.RecommitRaw = userConfig.Sealer.Recommit.String()
	userConfig.Gpo.MaxPriceRaw = userConfig.Gpo.MaxPrice.String()
	userConfig.Gpo.IgnorePriceRaw = userConfig.Gpo.IgnorePrice.String()
	userConfig.Cache.RejournalRaw = userConfig.Cache.Rejournal.String()
	userConfig.Cache.TrieTimeoutRaw = userConfig.Cache.TrieTimeout.String()

	if err := toml.NewEncoder(os.Stdout).Encode(userConfig); err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	return 0
}
