package cli

import (
	"strings"

	"github.com/mitchellh/cli"
)

type Account struct {
	UI cli.Ui
}

// MarkDown implements cli.MarkDown interface
func (a *Account) MarkDown() string {
	items := []string{
		"# Account",
		"The ```account``` command groups actions to interact with accounts:",
		"- [```account new```](./account_new.md): Create a new account in the Bor client.",
		"- [```account list```](./account_list.md): List the wallets in the Bor client.",
		"- [```account import```](./account_import.md): Import an account to the Bor client.",
	}

	return strings.Join(items, "\n\n")
}

// Help implements the cli.Command interface
func (a *Account) Help() string {
	return `Usage: bor account <subcommand>

  This command groups actions to interact with accounts.
  
  List the running deployments:

    $ bor account new
  
  Display the status of a specific deployment:

    $ bor account import
    
  List the imported accounts in the keystore:
    
    $ bor account list`
}

// Synopsis implements the cli.Command interface
func (a *Account) Synopsis() string {
	return "Interact with accounts"
}

// Run implements the cli.Command interface
func (a *Account) Run(args []string) int {
	return cli.RunResultHelp
}
