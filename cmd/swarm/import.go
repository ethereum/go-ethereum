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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/ethereum/go-ethereum/cmd/utils"

	"github.com/ethereum/go-ethereum/swarm/api"
	"github.com/ethereum/go-ethereum/swarm/storage"

	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
)

func import_ldb(ctx *cli.Context) {
	var importdatadir string
	var importbzzaccount string
	var configdata []byte
	var sourceconfig api.Config
	var targetconfig api.Config

	var sourcedatadirfull string
	var targetdatadirfull string

	var err error

	args := ctx.Args()
	if len(args) == 0 {
		utils.Fatalf("need at least one argument <source-datadir> [<source-bzzaccount>]")
	}

	importdatadir = args[0]
	if len(args) > 0 {
		importbzzaccount = args[1]
	}

	var (
		bzzaccount = ctx.GlobalString(SwarmAccountFlag.Name)
		datadir    = ctx.GlobalString(utils.DataDirFlag.Name)
	)

	if importdatadir == "" {
		utils.Fatalf("--importdir must be specificed")
	}
	if datadir == "" {
		utils.Fatalf("--datadir must be specificed")
	}

	sourcedatadirfull = fmt.Sprintf("%s/swarm/bzz-%s", path.Clean(importdatadir), importbzzaccount)
	targetdatadirfull = fmt.Sprintf("%s/swarm/bzz-%s", path.Clean(datadir), bzzaccount)

	configdata, err = ioutil.ReadFile(sourcedatadirfull + "/config.json")
	if err != nil {
		utils.Fatalf("Could not open source config file '%s'", sourcedatadirfull+"/config.json")
	}
	err = json.Unmarshal(configdata, &sourceconfig)
	if err != nil {
		utils.Fatalf("Corrupt or invalid source config file '%s'", sourcedatadirfull+"/config.json")
	}
	log.Trace(fmt.Sprintf("Sourceconfig has bzzkey %v", sourceconfig.BzzKey))

	configdata, err = ioutil.ReadFile(targetdatadirfull + "/config.json")
	if err != nil {
		utils.Fatalf("Could not open target config file '%s'", targetdatadirfull+"/config.json")
	}
	err = json.Unmarshal(configdata, &targetconfig)
	if err != nil {
		utils.Fatalf("Corrupt or invalid source config file '%s'", targetdatadirfull+"/config.json")
	}

	chunkcount, err := storage.Import(sourceconfig.ChunkDbPath, targetconfig.ChunkDbPath, sourceconfig.BzzKey, targetconfig.BzzKey)
	if err != nil {
		utils.Fatalf("import failed: %s", err)
	}

	log.Trace(fmt.Sprintf("Chunks imported: %d", chunkcount))

}
