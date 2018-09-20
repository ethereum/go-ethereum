// Copyright 2018 The go-ethereum Authors
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
	"io/ioutil"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/contracts/registrar"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"gopkg.in/urfave/cli.v1"
)

// setupClient creates a client with specified remote URL.
func setupClient(ctx *cli.Context) *ethclient.Client {
	url := ctx.GlobalString(clientURLFlag.Name)
	client, err := ethclient.Dial(url)
	if err != nil {
		utils.Fatalf("Failed to setup ethereum client at url %s: %v", url, err)
	}
	log.Info("Setup ethereum client", "URL", url)
	return client
}

// setupDialContext creates a rpc client with specified node URL.
func setupDialContext(ctx *cli.Context) *rpc.Client {
	url := ctx.GlobalString(clientURLFlag.Name)
	client, err := rpc.Dial(url)
	if err != nil {
		utils.Fatalf("Failed to setup rpc client at url %s: %v", url, err)
	}
	log.Info("Setup rpc client", "URL", url)
	return client
}

// setupContract creates a registrar contract instance with specified
// contract address or the default contracts for mainnet or testnet.
func setupContract(client *rpc.Client) *registrar.Registrar {
	var addr string
	err := client.Call(&addr, "les_getCheckpointContractAddress")
	if err != nil {
		utils.Fatalf("Failed to fetch checkpoint contract address, err %v", err)
	}
	contractAddr := common.HexToAddress(addr)
	if contractAddr == (common.Address{}) {
		utils.Fatalf("No specified registrar contract address")
	}
	contract, err := registrar.NewRegistrar(contractAddr, ethclient.NewClient(client))
	if err != nil {
		utils.Fatalf("Failed to setup registrar contract %s: %v", contractAddr, err)
	}
	log.Info("Setup registrar contract", "address", contractAddr)
	return contract
}

// promptPassphrase prompts the user for a passphrase.
// Set confirmation to true to require the user to confirm the passphrase.
func promptPassphrase(confirmation bool) string {
	passphrase, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		utils.Fatalf("Failed to read passphrase: %v", err)
	}

	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			utils.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if passphrase != confirm {
			utils.Fatalf("Passphrases do not match")
		}
	}

	return passphrase
}

// getPassphrase obtains a passphrase given by the user. It first checks the
// --password command line flag and ultimately prompts the user for a
// passphrase.
func getPassphrase(ctx *cli.Context) string {
	passphraseFile := ctx.String(utils.PasswordFileFlag.Name)
	if passphraseFile != "" {
		content, err := ioutil.ReadFile(passphraseFile)
		if err != nil {
			utils.Fatalf("Failed to read passphrase file '%s': %v",
				passphraseFile, err)
		}
		return strings.TrimRight(string(content), "\r\n")
	}

	// Otherwise prompt the user for the passphrase.
	return promptPassphrase(false)
}

// getPrivateKey retrieves the user key through specified key file.
func getPrivateKey(ctx *cli.Context) *keystore.Key {
	// Read key from file.
	keyFile := ctx.GlobalString(keyFileFlag.Name)
	keyJson, err := ioutil.ReadFile(keyFile)
	if err != nil {
		utils.Fatalf("Failed to read the keyfile at '%s': %v", keyFile, err)
	}
	// Decrypt key with passphrase.
	passphrase := getPassphrase(ctx)
	key, err := keystore.DecryptKey(keyJson, passphrase)
	if err != nil {
		utils.Fatalf("Failed to decrypt user key '%s': %v", keyFile, err)
	}
	return key
}
