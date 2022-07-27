package cli

import (
	"reflect"
	"strings"

	"github.com/naoina/toml"

	"github.com/ethereum/go-ethereum/internal/cli/server"
)

// These settings ensure that TOML keys use the same names as Go struct fields.
var tomlSettings = toml.Config{
	NormFieldName: func(rt reflect.Type, key string) string {
		return key
	},
	FieldToKey: func(rt reflect.Type, field string) string {
		return field
	},
}

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
	userConfig.TxPool.RejournalRaw = userConfig.TxPool.Rejournal.String()
	userConfig.TxPool.LifeTimeRaw = userConfig.TxPool.LifeTime.String()
	userConfig.Sealer.GasPriceRaw = userConfig.Sealer.GasPrice.String()
	userConfig.Gpo.MaxPriceRaw = userConfig.Gpo.MaxPrice.String()
	userConfig.Gpo.IgnorePriceRaw = userConfig.Gpo.IgnorePrice.String()
	userConfig.Cache.RejournalRaw = userConfig.Cache.Rejournal.String()

	// Currently, the configurations (userConfig) is exported into `toml` file format.
	out, err := tomlSettings.Marshal(&userConfig)
	if err != nil {
		c.UI.Error(err.Error())
		return 1
	}

	c.UI.Output(string(out))

	return 0
}
