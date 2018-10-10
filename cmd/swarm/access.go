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
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/api/client"
	"gopkg.in/urfave/cli.v1"
)

var salt = make([]byte, 32)

func init() {
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
}

func accessNewPass(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 1 {
		utils.Fatalf("Expected 1 argument - the ref")
	}

	var (
		ae        *api.AccessEntry
		accessKey []byte
		err       error
		ref       = args[0]
		password  = getPassPhrase("", 0, makePasswordList(ctx))
		dryRun    = ctx.Bool(SwarmDryRunFlag.Name)
	)
	accessKey, ae, err = api.DoPassword(ctx, password, salt)
	if err != nil {
		utils.Fatalf("error getting session key: %v", err)
	}
	m, err := api.GenerateAccessControlManifest(ctx, ref, accessKey, ae)
	if dryRun {
		err = printManifests(m, nil)
		if err != nil {
			utils.Fatalf("had an error printing the manifests: %v", err)
		}
	} else {
		err = uploadManifests(ctx, m, nil)
		if err != nil {
			utils.Fatalf("had an error uploading the manifests: %v", err)
		}
	}
}

func accessNewPK(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 1 {
		utils.Fatalf("Expected 1 argument - the ref")
	}

	var (
		ae               *api.AccessEntry
		sessionKey       []byte
		err              error
		ref              = args[0]
		privateKey       = getPrivKey(ctx)
		granteePublicKey = ctx.String(SwarmAccessGrantKeyFlag.Name)
		dryRun           = ctx.Bool(SwarmDryRunFlag.Name)
	)
	sessionKey, ae, err = api.DoPK(ctx, privateKey, granteePublicKey, salt)
	if err != nil {
		utils.Fatalf("error getting session key: %v", err)
	}
	m, err := api.GenerateAccessControlManifest(ctx, ref, sessionKey, ae)
	if dryRun {
		err = printManifests(m, nil)
		if err != nil {
			utils.Fatalf("had an error printing the manifests: %v", err)
		}
	} else {
		err = uploadManifests(ctx, m, nil)
		if err != nil {
			utils.Fatalf("had an error uploading the manifests: %v", err)
		}
	}
}

func accessNewACT(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 1 {
		utils.Fatalf("Expected 1 argument - the ref")
	}

	var (
		ae                   *api.AccessEntry
		actManifest          *api.Manifest
		accessKey            []byte
		err                  error
		ref                  = args[0]
		pkGrantees           = []string{}
		passGrantees         = []string{}
		pkGranteesFilename   = ctx.String(SwarmAccessGrantKeysFlag.Name)
		passGranteesFilename = ctx.String(utils.PasswordFileFlag.Name)
		privateKey           = getPrivKey(ctx)
		dryRun               = ctx.Bool(SwarmDryRunFlag.Name)
	)
	if pkGranteesFilename == "" && passGranteesFilename == "" {
		utils.Fatalf("you have to provide either a grantee public-keys file or an encryption passwords file (or both)")
	}

	if pkGranteesFilename != "" {
		bytes, err := ioutil.ReadFile(pkGranteesFilename)
		if err != nil {
			utils.Fatalf("had an error reading the grantee public key list")
		}
		pkGrantees = strings.Split(strings.Trim(string(bytes), "\n"), "\n")
	}

	if passGranteesFilename != "" {
		bytes, err := ioutil.ReadFile(passGranteesFilename)
		if err != nil {
			utils.Fatalf("could not read password filename: %v", err)
		}
		passGrantees = strings.Split(strings.Trim(string(bytes), "\n"), "\n")
	}
	accessKey, ae, actManifest, err = api.DoACT(ctx, privateKey, salt, pkGrantees, passGrantees)
	if err != nil {
		utils.Fatalf("error generating ACT manifest: %v", err)
	}

	if err != nil {
		utils.Fatalf("error getting session key: %v", err)
	}
	m, err := api.GenerateAccessControlManifest(ctx, ref, accessKey, ae)
	if err != nil {
		utils.Fatalf("error generating root access manifest: %v", err)
	}

	if dryRun {
		err = printManifests(m, actManifest)
		if err != nil {
			utils.Fatalf("had an error printing the manifests: %v", err)
		}
	} else {
		err = uploadManifests(ctx, m, actManifest)
		if err != nil {
			utils.Fatalf("had an error uploading the manifests: %v", err)
		}
	}
}

func printManifests(rootAccessManifest, actManifest *api.Manifest) error {
	js, err := json.Marshal(rootAccessManifest)
	if err != nil {
		return err
	}
	fmt.Println(string(js))

	if actManifest != nil {
		js, err := json.Marshal(actManifest)
		if err != nil {
			return err
		}
		fmt.Println(string(js))
	}
	return nil
}

func uploadManifests(ctx *cli.Context, rootAccessManifest, actManifest *api.Manifest) error {
	bzzapi := strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
	client := client.NewClient(bzzapi)

	var (
		key string
		err error
	)
	if actManifest != nil {
		key, err = client.UploadManifest(actManifest, false)
		if err != nil {
			return err
		}

		rootAccessManifest.Entries[0].Access.Act = key
	}
	key, err = client.UploadManifest(rootAccessManifest, false)
	if err != nil {
		return err
	}
	fmt.Println(key)
	return nil
}

// makePasswordList reads password lines from the file specified by the global --password flag
// and also by the same subcommand --password flag.
// This function ia a fork of utils.MakePasswordList to lookup cli context for subcommand.
// Function ctx.SetGlobal is not setting the global flag value that can be accessed
// by ctx.GlobalString using the current version of cli package.
func makePasswordList(ctx *cli.Context) []string {
	path := ctx.GlobalString(utils.PasswordFileFlag.Name)
	if path == "" {
		path = ctx.String(utils.PasswordFileFlag.Name)
		if path == "" {
			return nil
		}
	}
	text, err := ioutil.ReadFile(path)
	if err != nil {
		utils.Fatalf("Failed to read password file: %v", err)
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}
