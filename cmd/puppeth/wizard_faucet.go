// Copyright 2017 The go-ethereum Authors
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
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/log"
)

// deployFaucet queries the user for various input on deploying a faucet, after
// which it executes it.
func (w *wizard) deployFaucet() {
	// Select the server to interact with
	server := w.selectServer()
	if server == "" {
		return
	}
	client := w.servers[server]

	// Retrieve any active faucet configurations from the server
	infos, err := checkFaucet(client, w.network)
	if err != nil {
		infos = &faucetInfos{
			node:    &nodeInfos{portFull: 30303, peersTotal: 25},
			port:    80,
			host:    client.server,
			amount:  1,
			minutes: 1440,
			tiers:   3,
		}
	}
	existed := err == nil

	infos.node.genesis, _ = json.MarshalIndent(w.conf.Genesis, "", "  ")
	infos.node.network = w.conf.Genesis.Config.ChainId.Int64()

	// Figure out which port to listen on
	fmt.Println()
	fmt.Printf("Which port should the faucet listen on? (default = %d)\n", infos.port)
	infos.port = w.readDefaultInt(infos.port)

	// Figure which virtual-host to deploy ethstats on
	if infos.host, err = w.ensureVirtualHost(client, infos.port, infos.host); err != nil {
		log.Error("Failed to decide on faucet host", "err", err)
		return
	}
	// Port and proxy settings retrieved, figure out the funding amount per period configurations
	fmt.Println()
	fmt.Printf("How many Ethers to release per request? (default = %d)\n", infos.amount)
	infos.amount = w.readDefaultInt(infos.amount)

	fmt.Println()
	fmt.Printf("How many minutes to enforce between requests? (default = %d)\n", infos.minutes)
	infos.minutes = w.readDefaultInt(infos.minutes)

	fmt.Println()
	fmt.Printf("How many funding tiers to feature (x2.5 amounts, x3 timeout)? (default = %d)\n", infos.tiers)
	infos.tiers = w.readDefaultInt(infos.tiers)
	if infos.tiers == 0 {
		log.Error("At least one funding tier must be set")
		return
	}
	// Accessing the reCaptcha service requires API authorizations, request it
	if infos.captchaToken != "" {
		fmt.Println()
		fmt.Println("Reuse previous reCaptcha API authorization (y/n)? (default = yes)")
		if w.readDefaultString("y") != "y" {
			infos.captchaToken, infos.captchaSecret = "", ""
		}
	}
	if infos.captchaToken == "" {
		// No previous authorization (or old one discarded)
		fmt.Println()
		fmt.Println("Enable reCaptcha protection against robots (y/n)? (default = no)")
		if w.readDefaultString("n") == "n" {
			log.Warn("Users will be able to requests funds via automated scripts")
		} else {
			// Captcha protection explicitly requested, read the site and secret keys
			fmt.Println()
			fmt.Printf("What is the reCaptcha site key to authenticate human users?\n")
			infos.captchaToken = w.readString()

			fmt.Println()
			fmt.Printf("What is the reCaptcha secret key to verify authentications? (won't be echoed)\n")
			infos.captchaSecret = w.readPassword()
		}
	}
	// Figure out where the user wants to store the persistent data
	fmt.Println()
	if infos.node.datadir == "" {
		fmt.Printf("Where should data be stored on the remote machine?\n")
		infos.node.datadir = w.readString()
	} else {
		fmt.Printf("Where should data be stored on the remote machine? (default = %s)\n", infos.node.datadir)
		infos.node.datadir = w.readDefaultString(infos.node.datadir)
	}
	// Figure out which port to listen on
	fmt.Println()
	fmt.Printf("Which TCP/UDP port should the light client listen on? (default = %d)\n", infos.node.portFull)
	infos.node.portFull = w.readDefaultInt(infos.node.portFull)

	// Set a proper name to report on the stats page
	fmt.Println()
	if infos.node.ethstats == "" {
		fmt.Printf("What should the node be called on the stats page?\n")
		infos.node.ethstats = w.readString() + ":" + w.conf.ethstats
	} else {
		fmt.Printf("What should the node be called on the stats page? (default = %s)\n", infos.node.ethstats)
		infos.node.ethstats = w.readDefaultString(infos.node.ethstats) + ":" + w.conf.ethstats
	}
	// Load up the credential needed to release funds
	if infos.node.keyJSON != "" {
		if key, err := keystore.DecryptKey([]byte(infos.node.keyJSON), infos.node.keyPass); err != nil {
			infos.node.keyJSON, infos.node.keyPass = "", ""
		} else {
			fmt.Println()
			fmt.Printf("Reuse previous (%s) funding account (y/n)? (default = yes)\n", key.Address.Hex())
			if w.readDefaultString("y") != "y" {
				infos.node.keyJSON, infos.node.keyPass = "", ""
			}
		}
	}
	for i := 0; i < 3 && infos.node.keyJSON == ""; i++ {
		fmt.Println()
		fmt.Println("Please paste the faucet's funding account key JSON:")
		infos.node.keyJSON = w.readJSON()

		fmt.Println()
		fmt.Println("What's the unlock password for the account? (won't be echoed)")
		infos.node.keyPass = w.readPassword()

		if _, err := keystore.DecryptKey([]byte(infos.node.keyJSON), infos.node.keyPass); err != nil {
			log.Error("Failed to decrypt key with given passphrase")
			infos.node.keyJSON = ""
			infos.node.keyPass = ""
		}
	}
	// Check if the user wants to run the faucet in debug mode (noauth)
	noauth := "n"
	if infos.noauth {
		noauth = "y"
	}
	fmt.Println()
	fmt.Printf("Permit non-authenticated funding requests (y/n)? (default = %v)\n", infos.noauth)
	infos.noauth = w.readDefaultString(noauth) != "n"

	// Try to deploy the faucet server on the host
	nocache := false
	if existed {
		fmt.Println()
		fmt.Printf("Should the faucet be built from scratch (y/n)? (default = no)\n")
		nocache = w.readDefaultString("n") != "n"
	}
	if out, err := deployFaucet(client, w.network, w.conf.bootLight, infos, nocache); err != nil {
		log.Error("Failed to deploy faucet container", "err", err)
		if len(out) > 0 {
			fmt.Printf("%s\n", out)
		}
		return
	}
	// All ok, run a network scan to pick any changes up
	w.networkStats()
}
