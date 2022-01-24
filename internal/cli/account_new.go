package cli

import (
	"fmt"

	"github.com/ethereum/go-ethereum/internal/cli/flagset"
)

type AccountNewCommand struct {
	*Meta
}

// Help implements the cli.Command interface
func (a *AccountNewCommand) Help() string {
	return `Usage: bor account new

  Create a new local account.

  ` + a.Flags().Help()
}

func (a *AccountNewCommand) Flags() *flagset.Flagset {
	return a.NewFlagSet("account new")
}

// Synopsis implements the cli.Command interface
func (a *AccountNewCommand) Synopsis() string {
	return "Create a new local account"
}

// Run implements the cli.Command interface
func (a *AccountNewCommand) Run(args []string) int {
	flags := a.Flags()
	if err := flags.Parse(args); err != nil {
		a.UI.Error(err.Error())
		return 1
	}

	keystore, err := a.GetKeystore()
	if err != nil {
		a.UI.Error(fmt.Sprintf("Failed to get keystore: %v", err))
		return 1
	}

	password, err := a.AskPassword()
	if err != nil {
		a.UI.Error(err.Error())
		return 1
	}

	account, err := keystore.NewAccount(password)
	if err != nil {
		a.UI.Error(fmt.Sprintf("Failed to create new account: %v", err))
		return 1
	}

	a.UI.Output("\nYour new key was generated")
	a.UI.Output(fmt.Sprintf("Public address of the key:   %s", account.Address.Hex()))
	a.UI.Output(fmt.Sprintf("Path of the secret key file: %s", account.URL.Path))

	return 0
}
