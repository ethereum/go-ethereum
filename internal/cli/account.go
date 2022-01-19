package cli

import "github.com/mitchellh/cli"

type Account struct {
	UI cli.Ui
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
