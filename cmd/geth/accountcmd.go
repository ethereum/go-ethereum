// Copyright 2016 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"io/ioutil"

	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
)

var (
	walletCommand = cli.Command{
		Name:  "wallet",
		Usage: "ethereum presale wallet",
		Subcommands: []cli.Command{
			{
				Action: importWallet,
				Name:   "import",
				Usage:  "import ethereum presale wallet",
			},
		},
		Description: `

    get wallet import /path/to/my/presale.wallet

will prompt for your password and imports your ether presale account.
It can be used non-interactively with the --password option taking a
passwordfile as argument containing the wallet password in plaintext.

`}
	accountCommand = cli.Command{
		Action: accountList,
		Name:   "account",
		Usage:  "manage accounts",
		Description: `

Manage accounts lets you create new accounts, list all existing accounts,
import a private key into a new account.

'            help' shows a list of subcommands or help for one subcommand.

It supports interactive mode, when you are prompted for password as well as
non-interactive mode where passwords are supplied via a given password file.
Non-interactive mode is only meant for scripted use on test networks or known
safe environments.

Make sure you remember the password you gave when creating a new account (with
either new or import). Without it you are not able to unlock your account.

Note that exporting your key in unencrypted format is NOT supported.

Keys are stored under <DATADIR>/keys.
It is safe to transfer the entire directory or the individual keys therein
between ethereum nodes by simply copying.
Make sure you backup your keys regularly.

In order to use your account to send transactions, you need to unlock them using
the '--unlock' option. The argument is a space separated list of addresses or
indexes. If used non-interactively with a passwordfile, the file should contain
the respective passwords one per line. If you unlock n accounts and the password
file contains less than n entries, then the last password is meant to apply to
all remaining accounts.

And finally. DO NOT FORGET YOUR PASSWORD.
`,
		Subcommands: []cli.Command{
			{
				Action: accountList,
				Name:   "list",
				Usage:  "print account addresses",
			},
			{
				Action: accountCreate,
				Name:   "new",
				Usage:  "create a new account",
				Description: `

    ethereum account new

Creates a new account. Prints the address.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the --password flag:

    ethereum --password <passwordfile> account new

Note, this is meant to be used for testing only, it is a bad idea to save your
password to file or expose in any other way.
					`,
			},
			{
				Action: accountUpdate,
				Name:   "update",
				Usage:  "update an existing account",
				Description: `

    ethereum account update <address>

Update an existing account.

The account is saved in the newest version in encrypted format, you are prompted
for a passphrase to unlock the account and another to save the updated file.

This same command can therefore be used to migrate an account of a deprecated
format to the newest format or change the password for an account.

For non-interactive use the passphrase can be specified with the --password flag:

    ethereum --password <passwordfile> account update <address>

Since only one password can be given, only format update can be performed,
changing your password is only possible interactively.
					`,
			},
			{
				Action: accountImport,
				Name:   "import",
				Usage:  "import a private key into a new account",
				Description: `

    ethereum account import <keyfile>

Imports an unencrypted private key from <keyfile> and creates a new account.
Prints the address.

The keyfile is assumed to contain an unencrypted private key in hexadecimal format.

The account is saved in encrypted format, you are prompted for a passphrase.

You must remember this passphrase to unlock your account in the future.

For non-interactive use the passphrase can be specified with the -password flag:

    ethereum --password <passwordfile> account import <keyfile>

Note:
As you can directly copy your encrypted accounts to another ethereum instance,
this import mechanism is not needed when you transfer an account between
nodes.
					`,
			},
		},
	}
)

func accountList(ctx *cli.Context) {
	accman := utils.MakeAccountManager(ctx)
	accts, err := accman.Accounts()
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	for i, acct := range accts {
		fmt.Printf("Account #%d: %x\n", i, acct)
	}
}

// tries unlocking the specified account a few times.
func unlockAccount(ctx *cli.Context, accman *accounts.Manager, address string, i int, passwords []string) (common.Address, string) {
	account, err := utils.MakeAddress(accman, address)
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := getPassPhrase(prompt, false, i, passwords)
		if err := accman.Unlock(account, password); err == nil {
			return account, password
		}
	}
	// All trials expended to unlock account, bail out
	utils.Fatalf("Failed to unlock account: %s", address)
	return common.Address{}, ""
}

// getPassPhrase retrieves the passwor associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
func getPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	fmt.Println(prompt)
	password, err := utils.Stdin.PasswordPrompt("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := utils.Stdin.PasswordPrompt("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}
	return password
}

// accountCreate creates a new account into the keystore defined by the CLI flags.
func accountCreate(ctx *cli.Context) {
	accman := utils.MakeAccountManager(ctx)
	password := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, utils.MakePasswordList(ctx))

	account, err := accman.NewAccount(password)
	if err != nil {
		utils.Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("Address: %x\n", account)
}

// accountUpdate transitions an account from a previous format to the current
// one, also providing the possibility to change the pass-phrase.
func accountUpdate(ctx *cli.Context) {
	if len(ctx.Args()) == 0 {
		utils.Fatalf("No accounts specified to update")
	}
	accman := utils.MakeAccountManager(ctx)

	account, oldPassword := unlockAccount(ctx, accman, ctx.Args().First(), 0, nil)
	newPassword := getPassPhrase("Please give a new password. Do not forget this password.", true, 0, nil)
	if err := accman.Update(account, oldPassword, newPassword); err != nil {
		utils.Fatalf("Could not update the account: %v", err)
	}
}

func importWallet(ctx *cli.Context) {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	keyJson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		utils.Fatalf("Could not read wallet file: %v", err)
	}

	accman := utils.MakeAccountManager(ctx)
	passphrase := getPassPhrase("", false, 0, utils.MakePasswordList(ctx))

	acct, err := accman.ImportPreSaleKey(keyJson, passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: %x\n", acct)
}

func accountImport(ctx *cli.Context) {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	accman := utils.MakeAccountManager(ctx)
	passphrase := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, utils.MakePasswordList(ctx))
	acct, err := accman.Import(keyfile, passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: %x\n", acct)
}
