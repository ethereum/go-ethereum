package cli

import (
	"github.com/mitchellh/cli"
)

// ChainCommand is the command to group the peers commands
type ChainCommand struct {
	UI cli.Ui
}

// Help implements the cli.Command interface
func (c *ChainCommand) Help() string {
	return `Usage: bor chain <subcommand>

  This command groups actions to interact with the chain.
	
  Set the new head of the chain:
  
    $ bor chain sethead <number>`
}

// Synopsis implements the cli.Command interface
func (c *ChainCommand) Synopsis() string {
	return "Interact with the chain"
}

// Run implements the cli.Command interface
func (c *ChainCommand) Run(args []string) int {
	return cli.RunResultHelp
}
