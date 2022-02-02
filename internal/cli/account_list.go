package cli

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/internal/cli/flagset"
)

type AccountListCommand struct {
	*Meta
}

// Help implements the cli.Command interface
func (a *AccountListCommand) Help() string {
	return `Usage: bor account list

  List the local accounts.

  ` + a.Flags().Help()
}

func (a *AccountListCommand) Flags() *flagset.Flagset {
	return a.NewFlagSet("account list")
}

// Synopsis implements the cli.Command interface
func (a *AccountListCommand) Synopsis() string {
	return "List the local accounts"
}

// Run implements the cli.Command interface
func (a *AccountListCommand) Run(args []string) int {
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
	a.UI.Output(formatAccounts(keystore.Accounts()))
	return 0
}

func formatAccounts(accts []accounts.Account) string {
	if len(accts) == 0 {
		return "No accounts found"
	}

	rows := make([]string, len(accts)+1)
	rows[0] = "Index|Address"
	for i, d := range accts {
		rows[i+1] = fmt.Sprintf("%d|%s",
			i,
			d.Address.String())
	}
	return formatList(rows)
}
