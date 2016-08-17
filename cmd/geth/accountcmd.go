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

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
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

Keys are stored under <DATADIR>/keystore.
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

func accountList(ctx *cli.Context) error {
	stack := utils.MakeNode(ctx, clientIdentifier, verString)
	for i, acct := range stack.AccountManager().Accounts() {
		fmt.Printf("Account #%d: {%x} %s\n", i, acct.Address, acct.File)
	}
	return nil
}

// tries unlocking the specified account a few times.
func unlockAccount(ctx *cli.Context, accman *accounts.Manager, address string, i int, passwords []string) (accounts.Account, string) {
	account, err := utils.MakeAddress(accman, address)
	if err != nil {
		utils.Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := getPassPhrase(prompt, false, i, passwords)
		err = accman.Unlock(account, password)
		if err == nil {
			glog.V(logger.Info).Infof("Unlocked account %x", account.Address)
			return account, password
		}
		if err, ok := err.(*accounts.AmbiguousAddrError); ok {
			glog.V(logger.Info).Infof("Unlocked account %x", account.Address)
			return ambiguousAddrRecovery(accman, err, password), password
		}
		if err != accounts.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	utils.Fatalf("Failed to unlock account %s (%v)", address, err)
	return accounts.Account{}, ""
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
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}
	return password
}

func ambiguousAddrRecovery(am *accounts.Manager, err *accounts.AmbiguousAddrError, auth string) accounts.Account {
	fmt.Printf("Multiple key files exist for address %x:\n", err.Addr)
	for _, a := range err.Matches {
		fmt.Println("  ", a.File)
	}
	fmt.Println("Testing your passphrase against all of them...")
	var match *accounts.Account
	for _, a := range err.Matches {
		if err := am.Unlock(a, auth); err == nil {
			match = &a
			break
		}
	}
	if match == nil {
		utils.Fatalf("None of the listed files could be unlocked.")
	}
	fmt.Printf("Your passphrase unlocked %s\n", match.File)
	fmt.Println("In order to avoid this warning, you need to remove the following duplicate key files:")
	for _, a := range err.Matches {
		if a != *match {
			fmt.Println("  ", a.File)
		}
	}
	return *match
}

// accountCreate creates a new account into the keystore defined by the CLI flags.
func accountCreate(ctx *cli.Context) error {
	stack := utils.MakeNode(ctx, clientIdentifier, verString)
	password := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, utils.MakePasswordList(ctx))

	account, err := stack.AccountManager().NewAccount(password)
	if err != nil {
		utils.Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("Address: {%x}\n", account.Address)
	return nil
}

// accountUpdate transitions an account from a previous format to the current
// one, also providing the possibility to change the pass-phrase.
func accountUpdate(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		utils.Fatalf("No accounts specified to update")
	}
	stack := utils.MakeNode(ctx, clientIdentifier, verString)
	account, oldPassword := unlockAccount(ctx, stack.AccountManager(), ctx.Args().First(), 0, nil)
	newPassword := getPassPhrase("Please give a new password. Do not forget this password.", true, 0, nil)
	if err := stack.AccountManager().Update(account, oldPassword, newPassword); err != nil {
		utils.Fatalf("Could not update the account: %v", err)
	}
	return nil
}

func importWallet(ctx *cli.Context) error {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	keyJson, err := ioutil.ReadFile(keyfile)
	if err != nil {
		utils.Fatalf("Could not read wallet file: %v", err)
	}

	stack := utils.MakeNode(ctx, clientIdentifier, verString)
	passphrase := getPassPhrase("", false, 0, utils.MakePasswordList(ctx))
	acct, err := stack.AccountManager().ImportPreSaleKey(keyJson, passphrase)
	if err != nil {
		utils.Fatalf("%v", err)
	}
	fmt.Printf("Address: {%x}\n", acct.Address)
	return nil
}

func accountImport(ctx *cli.Context) error {
	keyfile := ctx.Args().First()
	if len(keyfile) == 0 {
		utils.Fatalf("keyfile must be given as argument")
	}
	key, err := crypto.LoadECDSA(keyfile)
	if err != nil {
		utils.Fatalf("Failed to load the private key: %v", err)
	}
	stack := utils.MakeNode(ctx, clientIdentifier, verString)
	passphrase := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, utils.MakePasswordList(ctx))
	acct, err := stack.AccountManager().ImportECDSA(key, passphrase)
	if err != nil {
		utils.Fatalf("Could not create the account: %v", err)
	}
	fmt.Printf("Address: {%x}\n", acct.Address)
	return nil
}
