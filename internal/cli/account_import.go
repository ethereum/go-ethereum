package cli

import (
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
)

type AccountImportCommand struct {
	*Meta
}

// Help implements the cli.Command interface
func (a *AccountImportCommand) Help() string {
	return `Usage: bor account import

  Import a private key into a new account.

  Import an account:

    $ bor account import key.json

  ` + a.Flags().Help()
}

func (a *AccountImportCommand) Flags() *flagset.Flagset {
	return a.NewFlagSet("account import")
}

// Synopsis implements the cli.Command interface
func (a *AccountImportCommand) Synopsis() string {
	return "Import a private key into a new account"
}

// Run implements the cli.Command interface
func (a *AccountImportCommand) Run(args []string) int {
	flags := a.Flags()
	if err := flags.Parse(args); err != nil {
		a.UI.Error(err.Error())
		return 1
	}

	args = flags.Args()
	if len(args) != 1 {
		a.UI.Error("Expected one argument")
		return 1
	}
	key, err := crypto.LoadECDSA(args[0])
	if err != nil {
		a.UI.Error(fmt.Sprintf("Failed to load the private key '%s': %v", args[0], err))
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

	acct, err := keystore.ImportECDSA(key, password)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	a.UI.Output(fmt.Sprintf("Account created: %s", acct.Address.String()))
	return 0
}
