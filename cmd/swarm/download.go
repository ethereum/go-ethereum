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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/api"
	swarm "github.com/ethereum/go-ethereum/swarm/api/client"
	"gopkg.in/urfave/cli.v1"
)

func download(ctx *cli.Context) {
	log.Debug("downloading content using swarm down")
	args := ctx.Args()
	dest := "."

	switch len(args) {
	case 0:
		utils.Fatalf("Usage: swarm down [options] <bzz locator> [<destination path>]")
	case 1:
		log.Trace(fmt.Sprintf("swarm down: no destination path - assuming working dir"))
	default:
		log.Trace(fmt.Sprintf("destination path arg: %s", args[1]))
		if absDest, err := filepath.Abs(args[1]); err == nil {
			dest = absDest
		} else {
			utils.Fatalf("could not get download path: %v", err)
		}
	}

	var (
		bzzapi      = strings.TrimRight(ctx.GlobalString(SwarmApiFlag.Name), "/")
		isRecursive = ctx.Bool(SwarmRecursiveFlag.Name)
		client      = swarm.NewClient(bzzapi)
	)

	if fi, err := os.Stat(dest); err == nil {
		if isRecursive && !fi.Mode().IsDir() {
			utils.Fatalf("destination path is not a directory!")
		}
	} else {
		if !os.IsNotExist(err) {
			utils.Fatalf("could not stat path: %v", err)
		}
	}

	uri, err := api.Parse(args[0])
	if err != nil {
		utils.Fatalf("could not parse uri argument: %v", err)
	}

	dl := func(credentials string) error {
		// assume behaviour according to --recursive switch
		if isRecursive {
			if err := client.DownloadDirectory(uri.Addr, uri.Path, dest, credentials); err != nil {
				if err == swarm.ErrUnauthorized {
					return err
				}
				return fmt.Errorf("directory %s: %v", uri.Path, err)
			}
		} else {
			// we are downloading a file
			log.Debug("downloading file/path from a manifest", "uri.Addr", uri.Addr, "uri.Path", uri.Path)

			err := client.DownloadFile(uri.Addr, uri.Path, dest, credentials)
			if err != nil {
				if err == swarm.ErrUnauthorized {
					return err
				}
				return fmt.Errorf("file %s from address: %s: %v", uri.Path, uri.Addr, err)
			}
		}
		return nil
	}
	if passwords := makePasswordList(ctx); passwords != nil {
		password := getPassPhrase(fmt.Sprintf("Downloading %s is restricted", uri), 0, passwords)
		err = dl(password)
	} else {
		err = dl("")
	}
	if err != nil {
		utils.Fatalf("download: %v", err)
	}
}
