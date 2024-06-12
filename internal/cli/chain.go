package cli

import (
	"strings"

	"github.com/mitchellh/cli"
)

// ChainCommand is the command to group the peers commands
type ChainCommand struct {
	UI cli.Ui
}

// MarkDown implements cli.MarkDown interface
func (c *ChainCommand) MarkDown() string {
	items := []string{
		"# Chain",
		"The ```chain``` command groups actions to interact with the blockchain in the client:",
		"- [```chain sethead```](./chain_sethead.md): Set the current chain to a certain block.",
		"- [```chain watch```](./chain_watch.md): Watch the chainHead, reorg and fork events in real-time.",
	}

	return strings.Join(items, "\n\n")
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
