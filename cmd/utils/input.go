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

package utils

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	// "strings"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"gopkg.in/urfave/cli.v1"
	// "github.com/peterh/liner"
)

// tries unlocking the specified account a few times.
func UnlockAccount(ctx *cli.Context, accman *accounts.Manager, address string, i int, passwords []string) (accounts.Account, string) {
	account, err := MakeAddress(accman, address)
	if err != nil {
		Fatalf("Could not list accounts: %v", err)
	}
	for trials := 0; trials < 3; trials++ {
		prompt := fmt.Sprintf("Unlocking account %s | Attempt %d/%d", address, trials+1, 3)
		password := GetPassPhrase(prompt, false, i, passwords)
		err = accman.Unlock(account, password)
		if err == nil {
			glog.V(logger.Info).Infof("Unlocked account %x", account.Address)
			return account, password
		}
		if err, ok := err.(*accounts.AmbiguousAddrError); ok {
			glog.V(logger.Info).Infof("Unlocked account %x", account.Address)
			return AmbiguousAddrRecovery(accman, err, password), password
		}
		if err != accounts.ErrDecrypt {
			// No need to prompt again if the error is not decryption-related.
			break
		}
	}
	// All trials expended to unlock account, bail out
	Fatalf("Failed to unlock account %s (%v)", address, err)
	return accounts.Account{}, ""
}

// getPassPhrase retrieves the passwor associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
func GetPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
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
		Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			Fatalf("Passphrases do not match")
		}
	}
	return password
}

func AmbiguousAddrRecovery(am *accounts.Manager, err *accounts.AmbiguousAddrError, auth string) accounts.Account {
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
		Fatalf("None of the listed files could be unlocked.")
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
